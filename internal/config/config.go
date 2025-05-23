// internal/config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NullMeDev/Infopulse-Node/internal/models"
)

// Config represents the application configuration
type Config struct {
	// Bot settings
	BotToken            string `json:"-"` // Loaded from secrets.json
	AutopostEnabled     bool   `json:"autopostEnabled"`
	AutopostChannelID   string `json:"autopostChannelID"`
	AutopostIntervalHours int  `json:"autopostIntervalHours"`
	LogOnlyMode         bool   `json:"logOnlyMode"`
	
	// Logging settings
	LogFilePath         string `json:"logFilePath"`
	DBFilePath          string `json:"dbFilePath"`
	
	// Discord settings
	CommandPrefix       string `json:"commandPrefix"`
	AdminRoles          []string `json:"adminRoles"`
	
	// Feed settings
	MaxConcurrentFetches int `json:"maxConcurrentFetches"`
	FetchTimeoutSeconds  int `json:"fetchTimeoutSeconds"`
	
	// Feed sources (loaded separately)
	FeedSources         []models.FeedSource `json:"-"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		AutopostEnabled:      true,
		AutopostChannelID:    "",
		AutopostIntervalHours: 4,
		LogOnlyMode:          false,
		LogFilePath:          "/var/log/infopulse-node.log",
		DBFilePath:           "/data/state.db",
		CommandPrefix:        "!",
		AdminRoles:           []string{"admin", "moderator"},
		MaxConcurrentFetches: 5,
		FetchTimeoutSeconds:  30,
	}
}

// LoadConfig loads configuration from the specified path
func LoadConfig(configPath string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()
	
	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := SaveConfig(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
	} else {
		// Load existing config
		file, err := os.Open(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %v", err)
		}
		defer file.Close()
		
		if err := json.NewDecoder(file).Decode(config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
	}
	
	// Load secrets
	secretsPath := filepath.Join(filepath.Dir(configPath), "secrets.json")
	if _, err := os.Stat(secretsPath); !os.IsNotExist(err) {
		secrets := struct {
			BotToken string `json:"botToken"`
		}{}
		
		file, err := os.Open(secretsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open secrets file: %v", err)
		}
		defer file.Close()
		
		if err := json.NewDecoder(file).Decode(&secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secrets file: %v", err)
		}
		
		config.BotToken = secrets.BotToken
	}
	
	// Load feed sources
	feedsPath := filepath.Join(filepath.Dir(configPath), "feeds.json")
	if _, err := os.Stat(feedsPath); os.IsNotExist(err) {
		// Create default feeds file
		if err := saveFeedSources(getDefaultFeedSources(), feedsPath); err != nil {
			return nil, fmt.Errorf("failed to create default feeds: %v", err)
		}
	}
	
	// Load feed sources
	feeds, err := loadFeedSources(feedsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load feed sources: %v", err)
	}
	
	config.FeedSources = feeds
	
	return config, nil
}

// SaveConfig saves configuration to the specified path
func SaveConfig(config *Config, configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(config)
}

// Helper to load feed sources from file
func loadFeedSources(path string) ([]models.FeedSource, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open feeds file: %v", err)
	}
	defer file.Close()
	
	var sources []models.FeedSource
	if err := json.NewDecoder(file).Decode(&sources); err != nil {
		return nil, fmt.Errorf("failed to parse feeds file: %v", err)
	}
	
	return sources, nil
}

// Helper to save feed sources to file
func saveFeedSources(sources []models.FeedSource, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create feeds file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(sources)
}

// getDefaultFeedSources returns a set of default feed sources
func getDefaultFeedSources() []models.FeedSource {
	return []models.FeedSource{
		{
			ID:         "krebs",
			Name:       "Krebs on Security",
			URL:        "https://krebsonsecurity.com/feed/",
			Categories: []models.Category{models.CategoryCybersec, models.CategoryInfosecNews},
			FetchMethod: "rss",
			UpdateFreq: 60, // minutes
			Enabled:   true,
		},
		{
			ID:         "bleeping",
			Name:       "Bleeping Computer",
			URL:        "https://www.bleepingcomputer.com/feed/",
			Categories: []models.Category{models.CategoryCybersec, models.CategoryInfosecNews},
			FetchMethod: "rss",
			UpdateFreq: 60,
			Enabled:   true,
