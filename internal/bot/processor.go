package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"
	"unique"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"gopkg.in/telebot.v3"
)

// processJob represents a task for processing audio or voice messages.
type processJob struct {
	ctx       context.Context
	cancel    context.CancelFunc
	chatID    int64
	userID    int64
	file      telebot.File
	fileName  string
	mimeType  string
	fileSize  int64
	duration  int
	caption   string
	fileType  string // "audio" or "voice"
	tempPath  string // path to temporary file
	createdAt time.Time
}

// stepContext holds state between processing steps.
type stepContext struct {
	tempPath string
	media    *entity.Media
}

// File type handles for efficient comparison.
var (
	handleAudio = unique.Make("audio")
	handleVoice = unique.Make("voice")
)

// Step processing errors.
var (
	ErrJobCancelled   = errors.New("job cancelled")
	ErrDownloadFailed = errors.New("download failed")
	ErrSaveFailed     = errors.New("save failed")
	ErrInvalidFile    = errors.New("invalid file")
)

// worker runs the main worker loop, extracting jobs from the queue and processing them.
func (b *Bot) worker(ctx context.Context, id int) {
	defer b.wg.Done()

	logger := b.logger.With(slog.Int("worker_id", id))
	logger.Info("Worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping due to context cancellation")
			return
		case job, ok := <-b.jobQueue:
			if !ok {
				logger.Info("Worker stopping due to closed queue")
				return
			}
			b.processJob(job, logger)
		}
	}
}

// processJob processes a single job as a chain of steps.
func (b *Bot) processJob(job *processJob, logger *slog.Logger) {
	defer b.ensureJobCleanup(job)
	logger = b.enrichLogger(logger, job)
	defer b.recoverFromPanic(job, logger)

	steps := []struct {
		name string
		fn   func(*processJob, *slog.Logger) error
	}{
		{"check_cancelled", b.stepCheckCancelled},
		{"download_file", b.stepDownloadFile},
		{"check_cancelled_after_download", b.stepCheckCancelledAfterDownload},
		{"convert_and_save", b.stepConvertAndSave},
		{"notify_success", b.stepNotifySuccess},
	}

	stepCtx := &stepContext{}

	for _, step := range steps {
		select {
		case <-job.ctx.Done():
			b.handleStepError(job, logger, ErrJobCancelled, step.name, stepCtx)
			return
		default:
			if err := b.runStep(job, logger, step.name, step.fn, stepCtx); err != nil {
				b.handleStepError(job, logger, err, step.name, stepCtx)
				return
			}
		}
	}
}

// runStep executes a step with logging and timing.
func (b *Bot) runStep(job *processJob, logger *slog.Logger, stepName string,
	stepFn func(*processJob, *slog.Logger) error, ctx *stepContext) error {

	start := time.Now()
	err := stepFn(job, logger)

	attrs := []slog.Attr{
		slog.String("step", stepName),
		slog.Duration("duration", time.Since(start)),
	}

	if err != nil {
		logger.LogAttrs(context.Background(), slog.LevelError, "Step failed", attrs...)
		return err
	}

	logger.LogAttrs(context.Background(), slog.LevelDebug, "Step completed", attrs...)
	return nil
}

// stepCheckCancelled checks if the job has been cancelled.
func (b *Bot) stepCheckCancelled(job *processJob, logger *slog.Logger) error {
	select {
	case <-job.ctx.Done():
		cause := context.Cause(job.ctx)
		reason := "unknown"
		if cause != nil {
			reason = cause.Error()
		}
		logger.Info("Job cancelled before processing",
			slog.String("reason", reason))
		return ErrJobCancelled
	default:
		return nil
	}
}

// stepDownloadFile downloads the file from Telegram.
func (b *Bot) stepDownloadFile(job *processJob, logger *slog.Logger) error {
	tempPath, err := b.downloadFile(job, logger)
	if err != nil {
		logger.Error("Download failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %w", ErrDownloadFailed, err)
	}

	job.tempPath = tempPath
	return nil
}

// stepCheckCancelledAfterDownload checks for cancellation after download and cleans up if needed.
func (b *Bot) stepCheckCancelledAfterDownload(job *processJob, logger *slog.Logger) error {
	select {
	case <-job.ctx.Done():
		logger.Info("Job cancelled after download, cleaning up")
		if job.tempPath != "" {
			b.cleanupTempFile(job.tempPath, logger)
		}
		return ErrJobCancelled
	default:
		if job.tempPath != "" {
			b.markFileActive(job.tempPath)
		}
		return nil
	}
}

// stepConvertAndSave converts audio and saves the processed file.
func (b *Bot) stepConvertAndSave(job *processJob, logger *slog.Logger) error {
	if job.tempPath == "" {
		return fmt.Errorf("%w: no file to save", ErrInvalidFile)
	}

	// Ensure cleanup after processing
	defer func() {
		if job.tempPath != "" {
			b.cleanupTempFile(job.tempPath, logger)
		}
	}()

	media := b.createMediaFromJob(job)
	resp, err := b.aiUseCase.Recognition(job.ctx, media)
	if err != nil {
		logger.Error("Recognition failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}

	if resp != "" {
		b.sendSafeToChat(media.ChatID, resp, telebot.ModeHTML)
	}

	logger.Info("File processed successfully",
		slog.Int("duration", job.duration),
		slog.Int64("size", job.fileSize),
		slog.String("path", job.tempPath))

	return nil
}

// stepNotifySuccess sends success notification to the user.
func (b *Bot) stepNotifySuccess(job *processJob, logger *slog.Logger) error {
	var msg string
	fileTypeHandle := unique.Make(job.fileType)

	switch fileTypeHandle {
	case handleAudio:
		msg = fmt.Sprintf(MessageAudioSaved,
			escapeHTML(job.fileName), job.duration, formatFileSize(job.fileSize))
	case handleVoice:
		msg = fmt.Sprintf(MessageVoiceSaved, job.duration)
	default:
		logger.Warn("Unknown file type in success notification",
			slog.String("file_type", job.fileType))
		msg = MessageInternalError
	}

	b.sendSafeToChat(job.chatID, msg, telebot.ModeHTML)
	return nil
}

// handleStepError handles errors that occur during step execution.
func (b *Bot) handleStepError(job *processJob, logger *slog.Logger, err error, stepName string, ctx *stepContext) {
	logger.Error("Job failed",
		slog.String("step", stepName),
		slog.String("error", err.Error()))

	switch {
	case errors.Is(err, ErrJobCancelled):
		return
	case errors.Is(err, ErrDownloadFailed):
		b.sendSafeToChat(job.chatID, b.getDownloadErrorMessage(job), telebot.ModeHTML)
	case errors.Is(err, ErrSaveFailed):
		b.sendSafeToChat(job.chatID, b.getSaveErrorMessage(job), telebot.ModeHTML)
	default:
		b.sendSafeToChat(job.chatID, MessageInternalError, telebot.ModeHTML)
	}

	// Clean up temp file if it exists
	if job.tempPath != "" {
		b.cleanupTempFile(job.tempPath, logger)
	}
}

// getDownloadErrorMessage returns the appropriate download error message.
func (b *Bot) getDownloadErrorMessage(job *processJob) string {
	if job.fileType == "audio" {
		return fmt.Sprintf(MessageAudioDownloadFailed, escapeHTML(job.fileName))
	}
	return MessageVoiceDownloadFailed
}

// getSaveErrorMessage returns the appropriate save error message.
func (b *Bot) getSaveErrorMessage(job *processJob) string {
	if job.fileType == "audio" {
		return MessageAudioSaveFailed
	}
	return MessageVoiceSaveFailed
}

// ensureJobCleanup ensures the job's context is properly cancelled.
func (b *Bot) ensureJobCleanup(job *processJob) {
	if job.cancel != nil {
		job.cancel()
	}
}

// enrichLogger adds job-specific fields to the logger.
func (b *Bot) enrichLogger(logger *slog.Logger, job *processJob) *slog.Logger {
	return logger.With(
		slog.String("handler", job.fileType),
		slog.Int64("chat_id", job.chatID),
		slog.Int64("user_id", job.userID),
		slog.String("filename", job.fileName),
	)
}

// downloadFile downloads the file to a temporary location.
func (b *Bot) downloadFile(job *processJob, logger *slog.Logger) (string, error) {
	tempPath, err := b.downloadAudioToTemp(job.ctx, &job.file, job.fileName, job.mimeType)
	if err != nil {
		logger.Error("Download failed", slog.String("error", err.Error()))
		return "", err
	}
	return tempPath, nil
}

// createMediaFromJob creates a Media entity from job data.
func (b *Bot) createMediaFromJob(job *processJob) *entity.Media {
	media := &entity.Media{
		ChatID:   job.chatID,
		UserID:   job.userID,
		MimeType: job.mimeType,
		FileSize: job.fileSize,
		Duration: job.duration,
		Type:     job.fileType,
		FilePath: job.tempPath,
		FileName: job.fileName,
	}

	fileTypeHandle := unique.Make(job.fileType)

	switch fileTypeHandle {
	case handleVoice:
		media.FileID = job.file.FileID
	case handleAudio:
		media.Caption = job.caption
	}

	return media
}

// recoverFromPanic recovers from panics in job handlers.
func (b *Bot) recoverFromPanic(job *processJob, logger *slog.Logger) {
	if r := recover(); r != nil {
		logger.Error("Panic in job handler",
			slog.Any("panic", r),
			slog.String("stack", string(debug.Stack())),
		)

		// Clean up temp file on panic
		if job.tempPath != "" {
			b.cleanupTempFile(job.tempPath, logger)
		}

		b.sendSafeToChat(job.chatID, MessageInternalError, telebot.ModeHTML)
	}
}

// cleanupTempFile removes a temporary file and is idempotent.
func (b *Bot) cleanupTempFile(path string, logger *slog.Logger) {
	if path == "" {
		return
	}

	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			logger.Error("Failed to cleanup temp file",
				slog.String("path", path),
				slog.String("error", err.Error()))
		}
		return
	}

	logger.Debug("Temp file cleaned up", slog.String("path", path))
	b.markFileInactive(path)
}
