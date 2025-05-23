// cmd/main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NullMeDev/Infopulse-Node/internal/config"
	"github.com/NullMeDev/Infopulse-Node/internal/discord"
	"github.com/NullMeDev/Infopulse-Node/internal/feeds"
	"github.com/NullMeDev/Infopulse-Node/internal/logger"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config/config.json", "Path to configuration file")
	logPath := flag.String("log", "", "Path to log file (overrides config)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override log path if specified
	if *logPath != "" {
		cfg.LogFilePath = *logPath
	}

	// Create logger
	log, err := logger.NewLogger(cfg.LogFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// Log startup
	log.Info("Main", "Infopulse Node starting up")

	// Create feed engine
	engine, err := feeds.NewEngine(cfg, log)
	if err != nil {
		log.Critical("Main", fmt.Sprintf("Error creating feed engine: %v", err))
		os.Exit(1)
	}

	// Start feed engine
	if err := engine.Start(); err != nil {
		log.Critical("Main", fmt.Sprintf("Error starting feed engine: %v", err))
		os.Exit(1)
	}
	defer engine.Stop()

	// Create Discord bot
	bot, err := discord.NewBot(cfg, engine, log)
	if err != nil {
		log.Critical("Main", fmt.Sprintf("Error creating Discord bot: %v", err))
		os.Exit(1)
	}

	// Run bot (blocks until shutdown)
	if err := bot.Run(); err != nil {
		log.Critical("Main", fmt.Sprintf("Error running Discord bot: %v", err))
		os.Exit(1)
	}

	log.Info("Main", "Infopulse Node shutting down")
}
