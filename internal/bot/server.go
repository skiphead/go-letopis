package bot

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skiphead/go-letopis/internal/infra/config"
	"github.com/skiphead/go-letopis/internal/usecase"
	"gopkg.in/telebot.v3"
)

// Constants for bot configuration.
const (
	MaxAudioSize           = 20 * 1024 * 1024
	JobTimeout             = 10 * time.Minute
	ShutdownTimeout        = 30 * time.Second
	WorkerShutdownTimeout  = 5 * time.Second
	CleanupInterval        = 1 * time.Hour
	CleanupMaxAge          = 24 * time.Hour
	ActiveFilesLogInterval = 5 * time.Second
	DefaultWorkerCount     = 5
	DirPermissions         = 0755
)

// Bot represents the main bot with dependencies and state.
type Bot struct {
	*telebot.Bot

	cfg            *config.Config
	userUseCase    usecase.UserUseCase
	meetingUseCase usecase.MeetingUseCase
	aiUseCase      usecase.AIUseCase
	tempDir        string
	activeFiles    sync.Map
	logger         *slog.Logger
	jobQueue       chan *processJob
	wg             sync.WaitGroup
	cancel         atomic.Pointer[context.CancelFunc]
	workerCount    int
}

// New creates and initializes a new bot.
func New(cfg *config.Config, useCase usecase.AIUseCase, userUseCase usecase.UserUseCase, meetingUseCase usecase.MeetingUseCase, logger *slog.Logger) (*Bot, error) {
	if err := validateDependencies(cfg, useCase, logger); err != nil {
		return nil, err
	}

	botClient, err := createTelebot(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create telebot: %w", err)
	}

	tempDir := resolveTempDir(cfg.Bot.TmpDir)
	if err := ensureDirExists(tempDir); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	workerCount := resolveWorkerCount(cfg.Bot.WorkerCount)

	bot := &Bot{
		Bot:            botClient,
		cfg:            cfg,
		aiUseCase:      useCase,
		userUseCase:    userUseCase,
		meetingUseCase: meetingUseCase,
		tempDir:        tempDir,
		activeFiles:    sync.Map{},
		logger:         logger.With(slog.String("component", "bot")),
		jobQueue:       make(chan *processJob, workerCount*2),
		workerCount:    workerCount,
	}

	bot.registerHandlers()
	return bot, nil
}

// validateDependencies checks that all required dependencies are provided.
func validateDependencies(cfg *config.Config, useCase usecase.AIUseCase, logger *slog.Logger) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if useCase == nil {
		return errors.New("ai use case is required") // Fixed message
	}
	if logger == nil {
		return errors.New("logger is required")
	}
	return nil
}

// createTelebot creates and configures a new telebot instance.
func createTelebot(cfg *config.Config, logger *slog.Logger) (*telebot.Bot, error) {
	settings := cfg.Telegram.ToTelebotSettings()
	clientTimeout := 180*time.Second + 10*time.Second

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Telegram.Insecure,
	}

	settings.Client = &http.Client{
		Timeout: clientTimeout,
		Transport: &http.Transport{
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   20 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 180 * time.Second,
		},
	}
	return telebot.NewBot(settings)
}

// resolveTempDir determines the temporary directory path.
func resolveTempDir(cfgValue string) string {
	if cfgValue != "" {
		return cfgValue
	}
	return "./tmp/audio"
}

// resolveWorkerCount determines the number of workers to use.
func resolveWorkerCount(cfgValue int) int {
	if cfgValue > 0 {
		return cfgValue
	}
	return DefaultWorkerCount
}

// ensureDirExists creates the directory if it does not exist.
func ensureDirExists(path string) error {
	return os.MkdirAll(path, DirPermissions)
}

// Start starts the bot: workers, cleanup routine, and telebot main loop.
func (b *Bot) Start(ctx context.Context) {
	b.logger.Info("Bot starting...")

	ctx, cancel := context.WithCancel(ctx)
	b.cancel.Store(&cancel)

	b.startWorkers(ctx)
	go b.cleanupRoutine(ctx)

	// Run telebot in a goroutine since it blocks
	go func() {
		b.logger.Info("Starting telebot main loop...")
		// Bot.Start() blocks until Stop() is called
		b.Bot.Start()
		b.logger.Info("Telebot main loop stopped")

		// When the bot stops, trigger the shutdown sequence
		cancel()
	}()

	<-ctx.Done()
	b.Stop()
}

// startWorkers launches the worker goroutines.
func (b *Bot) startWorkers(ctx context.Context) {
	b.wg.Add(b.workerCount)
	for i := 0; i < b.workerCount; i++ {
		go b.worker(ctx, i)
	}
}

// cleanupRoutine periodically cleans up old temporary files.
func (b *Bot) cleanupRoutine(ctx context.Context) {
	logger := b.logger.With(slog.String("goroutine", "cleanup"))
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cleanup routine stopping")
			return
		case <-ticker.C:
			if err := b.CleanupOldTempFiles(ctx, CleanupMaxAge); err != nil {
				logger.Warn("Cleanup error", slog.String("error", err.Error()))
			}
		}
	}
}

// Stop performs graceful shutdown: cancels jobs, waits for workers, stops the bot.
func (b *Bot) Stop() {
	b.logger.Info("Bot stopping...")

	b.cancelContext()
	close(b.jobQueue)
	b.waitForActiveFilesWithTimeout()
	b.waitForWorkersWithTimeout()

	b.Bot.Stop()
	b.logger.Info("Bot stopped")
}

// cancelContext cancels the main context if it exists.
func (b *Bot) cancelContext() {
	if cancelPtr := b.cancel.Load(); cancelPtr != nil {
		(*cancelPtr)()
	}
}

// waitForActiveFilesWithTimeout waits for active files with a timeout.
func (b *Bot) waitForActiveFilesWithTimeout() {
	done := make(chan struct{})
	go func() {
		b.waitForActiveFiles()
		close(done)
	}()

	select {
	case <-done:
		b.logger.Info("All active files processed")
	case <-time.After(ShutdownTimeout):
		b.logger.Warn("Timeout waiting for active files")
	}
}

// waitForWorkersWithTimeout waits for workers to finish with a timeout.
func (b *Bot) waitForWorkersWithTimeout() {
	waitCh := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		b.logger.Info("All workers stopped")
	case <-time.After(WorkerShutdownTimeout):
		b.logger.Warn("Timeout waiting for workers")
	}
}

// waitForActiveFiles waits until all active files are processed.
func (b *Bot) waitForActiveFiles() {
	backoff := 100 * time.Millisecond
	const maxBackoff = 1 * time.Second
	lastLog := time.Now()
	lastCount := -1

	for {
		count := b.countActiveFiles()
		if count == 0 {
			return
		}

		b.logActiveFilesWait(count, lastCount, lastLog, backoff)
		lastCount = count
		lastLog = time.Now()

		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
		}
	}
}

// countActiveFiles returns the number of currently active files.
func (b *Bot) countActiveFiles() int {
	count := 0
	b.activeFiles.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// logActiveFilesWait logs the status of waiting for active files.
func (b *Bot) logActiveFilesWait(count, lastCount int, lastLog time.Time, backoff time.Duration) {
	if count != lastCount || time.Since(lastLog) > ActiveFilesLogInterval {
		b.logger.Info("Waiting for active files",
			slog.Int("count", count),
			slog.Duration("backoff", backoff),
		)
	}
}
