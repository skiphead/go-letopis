package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config — корневая структура конфигурации
type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	DBConfig DBConfig       `yaml:"db_config"`
}

// TelegramConfig — настройки Telegram Bot API
type TelegramConfig struct {
	Address string `yaml:"address"`
	Token   string `yaml:"token"`
}

// DBConfig — конфигурация подключения к БД
type DBConfig struct {
	Schema   string   `yaml:"schema"`
	DBName   string   `yaml:"dbname"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Nodes    []string `yaml:"nodes"`
}

// Load загружает и парсит конфигурацию из YAML-файла
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return &cfg, nil
}

// Validate проверяет обязательные поля конфигурации
func (c *Config) Validate() error {
	if c.Telegram.Token == "" {
		return fmt.Errorf("telegram.secret_key is required")
	}

	if c.DBConfig.DBName == "" {
		return fmt.Errorf("db_config.dbname is required")
	}
	if c.DBConfig.User == "" {
		return fmt.Errorf("db_config.user is required")
	}
	if c.DBConfig.Password == "" {
		return fmt.Errorf("db_config.password is required")
	}
	if len(c.DBConfig.Nodes) == 0 {
		return fmt.Errorf("db_config.nodes must contain at least one address")
	}

	return nil
}

// GetBaseURL возвращает базовый URL для запросов к Telegram Bot API
func (t *TelegramConfig) GetBaseURL() string {
	addr := t.Address
	if addr == "" {
		addr = "https://api.telegram.org"
	}
	addr = strings.TrimSuffix(addr, "/api")
	addr = strings.TrimSuffix(addr, "/")
	return fmt.Sprintf("%s/bot%s", addr, t.Token)
}
