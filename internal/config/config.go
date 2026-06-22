package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Telegram TelegramConfig `yaml:"telegram"`
	Matrix   MatrixConfig   `yaml:"matrix"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Address string `yaml:"address"`
}

// TelegramConfig holds Telegram bot credentials.
type TelegramConfig struct {
	Enabled  bool   `yaml:"enabled"`
	ProxyURL string `yaml:"proxy_url"`
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

// MatrixConfig holds Matrix credentials (reserved for future use).
type MatrixConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Homeserver  string `yaml:"homeserver"`
	UserID      string `yaml:"user_id"`
	AccessToken string `yaml:"access_token"`
	RoomID      string `yaml:"room_id"`
}

// Load reads configuration from the given YAML path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.setDefaults()
	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.Server.Address == "" {
		c.Server.Address = ":8080"
	}
}
