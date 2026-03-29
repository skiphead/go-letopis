package initapp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/skiphead/go-letopis/internal/bot"
	storage "github.com/skiphead/go-letopis/internal/domain/repository"
	"github.com/skiphead/go-letopis/internal/infra/config"
	"github.com/skiphead/go-letopis/internal/infra/postgres"
	gigachatservice "github.com/skiphead/go-letopis/internal/services/gigachat"
	"github.com/skiphead/go-letopis/internal/services/salutespeech/salute"
	"github.com/skiphead/go-letopis/internal/usecase"
)

// App holds all initialized application components.
type App struct {
	Config       *config.Config
	Logger       *slog.Logger
	Bot          *bot.Bot
	UseCases     *UseCases
	Repositories *Repositories
	Clients      *Clients
}

// UseCases holds all application use cases.
type UseCases struct {
	AI      usecase.AIUseCase
	User    usecase.UserUseCase
	Meeting usecase.MeetingUseCase
}

// Repositories holds all data repositories.
type Repositories struct {
	Meeting storage.MeetingRepository
	User    storage.UserRepository
}

// Clients holds all external service clients.
type Clients struct {
	SaluteSpeech salute.Client
	GigaChat     gigachatservice.Client
}

// Initialize loads configuration and initializes all application components.
func Initialize(configPath string) (*App, error) {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger with proper output
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	logger.Info("config loaded", slog.String("path", configPath))
	logger.Info("logger initialized")

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Initialize clients
	clients, err := initializeClients(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Initialize database
	repos, err := initializeRepositories(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Initialize use cases
	useCases := initializeUseCases(repos, clients, logger)

	// Initialize bot
	telegramBot, err := bot.New(cfg, useCases.AI, useCases.User, useCases.Meeting, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	logger.Info("bot initialized")

	return &App{
		Config:       cfg,
		Logger:       logger,
		Bot:          telegramBot,
		UseCases:     useCases,
		Repositories: repos,
		Clients:      clients,
	}, nil
}

// validateConfig checks that all required configuration fields are present.
func validateConfig(cfg *config.Config) error {
	if cfg.Telegram.Token == "" {
		return fmt.Errorf("telegram.token is required")
	}
	if cfg.SaluteSpeech.ClientID == "" {
		return fmt.Errorf("salute_speech.client_id is required")
	}
	if cfg.SaluteSpeech.ClientSecret == "" {
		return fmt.Errorf("salute_speech.client_secret is required")
	}
	if cfg.GigaChat.ClientID == "" {
		return fmt.Errorf("giga_chat.client_id is required")
	}
	if cfg.GigaChat.ClientSecret == "" {
		return fmt.Errorf("giga_chat.client_secret is required")
	}
	if cfg.DBConfig.Host == "" {
		return fmt.Errorf("db_config.host is required")
	}
	if cfg.DBConfig.User == "" {
		return fmt.Errorf("db_config.user is required")
	}
	if cfg.DBConfig.DBName == "" {
		return fmt.Errorf("db_config.dbname is required")
	}
	return nil
}

// initializeClients initializes all external service clients.
func initializeClients(cfg *config.Config, logger *slog.Logger) (*Clients, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	logger.Info("initializing salute speech client")
	saluteSpeechClient, err := salute.NewClient(
		cfg.SaluteSpeech.ClientID,
		cfg.SaluteSpeech.ClientSecret,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Salute Speech client: %w", err)
	}

	logger.Info("initializing gigachat client")
	gigaChatClient, err := gigachatservice.NewClient(
		cfg.GigaChat.ClientID,
		cfg.GigaChat.ClientSecret,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GigaChat client: %w", err)
	}

	return &Clients{
		SaluteSpeech: saluteSpeechClient,
		GigaChat:     gigaChatClient,
	}, nil
}

// initializeRepositories initializes database connection and repositories.
func initializeRepositories(cfg *config.Config, logger *slog.Logger) (*Repositories, error) {
	logger.Info("initializing database connection pool")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := postgres.NewPool(ctx, cfg.DBConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	logger.Info("initializing repositories")
	meetingRepo := storage.NewMeetingRepository(pool, logger)
	userRepo := storage.NewUserRepository(pool, logger)

	logger.Info("storage initialized", slog.String("database", cfg.DBConfig.DBName))

	return &Repositories{
		Meeting: meetingRepo,
		User:    userRepo,
	}, nil
}

// initializeUseCases initializes all application use cases.
func initializeUseCases(repos *Repositories, clients *Clients, logger *slog.Logger) *UseCases {
	logger.Info("initializing use cases")

	aiUseCase := usecase.NewAIUseCase(
		repos.User,
		repos.Meeting,
		clients.SaluteSpeech,
		clients.GigaChat,
		logger,
	)

	userUseCase := usecase.NewUserUseCase(repos.User, logger)
	meetingUseCase := usecase.NewMeetingUseCase(repos.Meeting, logger)

	return &UseCases{
		AI:      aiUseCase,
		User:    userUseCase,
		Meeting: meetingUseCase,
	}
}

// Run starts the bot and waits for termination signals.
func (a *App) Run(ctx context.Context) error {
	// Start bot in goroutine
	go func() {
		a.Bot.Start(ctx)
	}()

	a.Logger.Info("bot is running... press Ctrl+C to stop")

	// Wait for context cancellation
	<-ctx.Done()
	a.Bot.Stop()
	a.Logger.Info("bot stopped gracefully")

	return nil
}
