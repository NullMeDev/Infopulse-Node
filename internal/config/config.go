// internal/config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NullMeDev/Infopulse-Node/internal/models"
)

// Config represents application configuration
type Config struct {
	LogFilePath         string                      `json:"logFilePath"`
	DBFilePath          string                      `json:"dbFilePath"`
	CommandPrefix       string                      `json:"commandPrefix"`
	BotToken            string                      `json:"-"` // Loaded from secrets file
	FetchTimeoutSeconds int                         `json:"fetchTimeoutSeconds"`
	MaxConcurrentFetches int                        `json:"maxConcurrentFetches"`
	AutopostEnabled     bool                        `json:"autopostEnabled"`
	AutopostChannels    map[models.Category]string  `json:"autopostChannels"`
	FeedSources         []models.FeedSource         `json:"feedSources"`
}

// Secrets represents sensitive configuration
type Secrets struct {
	BotToken string `json:"botToken"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Load main config
	config := &Config{
		LogFilePath:         "./logs/infopulse.log",
		DBFilePath:          "./data/intelligence.db",
		CommandPrefix:       "!",
		FetchTimeoutSeconds: 30,
		MaxConcurrentFetches: 5,
		AutopostEnabled:     true,
		AutopostChannels:    make(map[models.Category]string),
		FeedSources:         []models.FeedSource{},
	}

	// Read config file
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse config
	if err := json.Unmarshal(configFile, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Load secrets
	secretsPath := filepath.Join(filepath.Dir(configPath), "secrets.json")
	secrets := &Secrets{}

	// Read secrets file
	secretsFile, err := os.ReadFile(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %v", err)
	}

	// Parse secrets
	if err := json.Unmarshal(secretsFile, secrets); err != nil {
		return nil, fmt.Errorf("failed to parse secrets file: %v", err)
	}

	// Copy secrets to config
	config.BotToken = secrets.BotToken

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// validateConfig ensures configuration is valid
func validateConfig(config *Config) error {
	if config.BotToken == "" {
		return fmt.Errorf("bot token is required")
	}

	if config.CommandPrefix == "" {
		config.CommandPrefix = "!"
	}

	if config.FetchTimeoutSeconds <= 0 {
		config.FetchTimeoutSeconds = 30
	}

	if config.MaxConcurrentFetches <= 0 {
		config.MaxConcurrentFetches = 5
	}

	// Ensure directories exist
	logDir := filepath.Dir(config.LogFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	dbDir := filepath.Dir(config.DBFilePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %v", err)
	}

	return nil
}
