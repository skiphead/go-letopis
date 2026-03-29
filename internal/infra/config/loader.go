package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure.
type Config struct {
	Bot          Bot            `yaml:"bot"`
	Telegram     TelegramConfig `yaml:"telegram"`
	SaluteSpeech OAuth          `yaml:"salute_speech"`
	GigaChat     OAuth          `yaml:"giga_chat"`
	DBConfig     DBConfig       `yaml:"db_config"`
}

// Bot contains bot-specific configuration.
type Bot struct {
	WorkerCount int    `yaml:"worker_count"`
	TmpDir      string `yaml:"tmp_dir"`
}

// TelegramConfig contains Telegram Bot API settings.
type TelegramConfig struct {
	Address        string        `yaml:"address"`
	Insecure       bool          `yaml:"insecure"`
	Token          string        `yaml:"token"`
	PollInterval   time.Duration `yaml:"poll_interval"`
	ParseMode      string        `yaml:"parse_mode"`
	AllowedUpdates []string      `yaml:"allowed_updates"`
}

// OAuth contains OAuth 2.0 client credentials.
type OAuth struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// DBConfig contains database connection settings.
type DBConfig struct {
	Host              string        `yaml:"host" env:"DB_HOST" envDefault:"localhost"`
	Port              int           `yaml:"port" env:"DB_PORT" envDefault:"5432"`
	User              string        `yaml:"user" env:"DB_USER" envDefault:"postgres"`
	Password          string        `yaml:"password" env:"DB_PASSWORD"`
	DBName            string        `yaml:"dbname" env:"DB_NAME" envDefault:"letopis"`
	Schema            string        `yaml:"schema" env:"DB_SCHEMA" envDefault:"public"`
	SSLMode           string        `yaml:"sslmode" env:"DB_SSLMODE" envDefault:"disable"`
	MaxConns          int32         `yaml:"max_conns" env:"DB_MAX_CONNS" envDefault:"25"`
	MinConns          int32         `yaml:"min_conns" env:"DB_MIN_CONNS" envDefault:"5"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime" env:"DB_MAX_CONN_LIFETIME" envDefault:"1h"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time" env:"DB_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period" env:"DB_HEALTH_CHECK_PERIOD" envDefault:"1m"`
	PingTimeout       time.Duration `yaml:"ping_timeout" env:"DB_PING_TIMEOUT" envDefault:"5s"`
}

// Load loads and parses the configuration from a YAML file with environment variable overrides.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply environment variable overrides for secrets
	if err = cfg.overrideWithEnv(); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Set defaults first
	cfg.setDefaults()

	// Then validate
	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return &cfg, nil
}

// overrideWithEnv overrides configuration values with environment variables.
func (c *Config) overrideWithEnv() error {
	// Telegram token
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		c.Telegram.Token = token
	}

	// Salute Speech credentials
	if clientID := os.Getenv("SALUTE_SPEECH_CLIENT_ID"); clientID != "" {
		c.SaluteSpeech.ClientID = clientID
	}
	if clientSecret := os.Getenv("SALUTE_SPEECH_CLIENT_SECRET"); clientSecret != "" {
		c.SaluteSpeech.ClientSecret = clientSecret
	}

	// GigaChat credentials
	if clientID := os.Getenv("GIGACHAT_CLIENT_ID"); clientID != "" {
		c.GigaChat.ClientID = clientID
	}
	if clientSecret := os.Getenv("GIGACHAT_CLIENT_SECRET"); clientSecret != "" {
		c.GigaChat.ClientSecret = clientSecret
	}

	// Database credentials
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		c.DBConfig.Password = dbPassword
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		c.DBConfig.User = dbUser
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		c.DBConfig.DBName = dbName
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		c.DBConfig.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		port, err := strconv.Atoi(dbPort)
		if err != nil {
			return fmt.Errorf("invalid DB_PORT value: %q, expected integer", dbPort)
		}
		c.DBConfig.Port = port
	}
	if dbSSLMode := os.Getenv("DB_SSLMODE"); dbSSLMode != "" {
		c.DBConfig.SSLMode = dbSSLMode
	}

	return nil
}

// Validate checks required configuration fields.
func (c *Config) Validate() error {
	// Validate Telegram configuration
	if c.Telegram.Token == "" {
		return fmt.Errorf("telegram.token is required")
	}

	// Warning for insecure mode (not a validation error, just a check)
	if c.Telegram.Insecure {
		_, err := fmt.Fprintf(os.Stderr, "WARNING: telegram.insecure is set to true. This is not recommended for production.\n")
		if err != nil {
			return err
		}
	}

	// Validate WorkerCount
	if c.Bot.WorkerCount <= 0 {
		return fmt.Errorf("bot.worker_count must be greater than 0, got: %d", c.Bot.WorkerCount)
	}

	// Validate PollInterval
	if c.Telegram.PollInterval <= 0 {
		return fmt.Errorf("telegram.poll_interval must be greater than 0, got: %v", c.Telegram.PollInterval)
	}

	// Validate ParseMode
	if c.Telegram.ParseMode != "" {
		validParseModes := map[string]bool{
			telebot.ModeDefault:    true,
			telebot.ModeHTML:       true,
			telebot.ModeMarkdown:   true,
			telebot.ModeMarkdownV2: true,
		}
		if !validParseModes[c.Telegram.ParseMode] {
			return fmt.Errorf("telegram.parse_mode must be one of: default, HTML, Markdown, MarkdownV2, got: %s", c.Telegram.ParseMode)
		}
	}

	// Validate database configuration
	if c.DBConfig.DBName == "" {
		return fmt.Errorf("db_config.dbname is required")
	}
	if c.DBConfig.User == "" {
		return fmt.Errorf("db_config.user is required")
	}
	if c.DBConfig.Password == "" {
		return fmt.Errorf("db_config.password is required")
	}
	if c.DBConfig.Host == "" {
		return fmt.Errorf("db_config.host is required")
	}
	if c.DBConfig.Port <= 0 {
		return fmt.Errorf("db_config.port must be greater than 0, got: %d", c.DBConfig.Port)
	}

	// Validate SSLMode
	validSSLmodes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if c.DBConfig.SSLMode != "" && !validSSLmodes[c.DBConfig.SSLMode] {
		return fmt.Errorf("db_config.sslmode must be one of: disable, require, verify-ca, verify-full, got: %s", c.DBConfig.SSLMode)
	}

	return nil
}

// setDefaults sets default values for configuration fields.
func (c *Config) setDefaults() {
	if c.Telegram.Address == "" {
		c.Telegram.Address = "https://api.telegram.org"
	}
	if c.Telegram.PollInterval == 0 {
		c.Telegram.PollInterval = 3 * time.Second
	}
	if c.Telegram.ParseMode == "" {
		c.Telegram.ParseMode = telebot.ModeHTML
	}
	if c.Telegram.AllowedUpdates == nil {
		c.Telegram.AllowedUpdates = []string{
			"message", "edited_message", "callback_query",
		}
	}
	if c.Bot.WorkerCount == 0 {
		c.Bot.WorkerCount = 1 // Default to 1 worker
	}
	if c.DBConfig.Port == 0 {
		c.DBConfig.Port = 5432
	}
	if c.DBConfig.Schema == "" {
		c.DBConfig.Schema = "public" // Default PostgreSQL schema
	}
	if c.DBConfig.SSLMode == "" {
		c.DBConfig.SSLMode = "disable" // Default SSL mode
	}
}

// DSN returns the PostgreSQL connection string.
func (db *DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.Host, db.Port, db.User, db.Password, db.DBName, db.SSLMode)
}

// GetBaseURL returns the base URL for Telegram Bot API requests.
func (t *TelegramConfig) GetBaseURL() string {
	addr := t.Address
	if addr == "" {
		addr = "https://api.telegram.org"
	}
	addr = strings.TrimSuffix(addr, "/api")
	addr = strings.TrimSuffix(addr, "/")
	return fmt.Sprintf("%s/bot%s", addr, t.Token)
}

// ToTelebotSettings converts the configuration to telebot settings.
func (t *TelegramConfig) ToTelebotSettings() telebot.Settings {
	return telebot.Settings{
		URL:       t.Address,
		Token:     t.Token,
		ParseMode: t.ParseMode,
		Poller: &telebot.LongPoller{
			Timeout:        t.PollInterval,
			AllowedUpdates: t.AllowedUpdates,
		},
	}
}
