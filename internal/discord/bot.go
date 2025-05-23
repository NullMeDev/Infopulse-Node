// internal/discord/bot.go
package discord

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/NullMeDev/Infopulse-Node/internal/config"
	"github.com/NullMeDev/Infopulse-Node/internal/feeds"
	"github.com/NullMeDev/Infopulse-Node/internal/logger"
	"github.com/NullMeDev/Infopulse-Node/internal/models"
	"github.com/bwmarrin/discordgo"
)

// Bot represents a Discord bot
type Bot struct {
	session  *discordgo.Session
	config   *config.Config
	engine   *feeds.Engine
	logger   *logger.Logger
	commands map[string]CommandHandler
}

// CommandHandler is a function that handles a command
type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error

// NewBot creates a new Discord bot
func NewBot(cfg *config.Config, engine *feeds.Engine, logger *logger.Logger) (*Bot, error) {
	// Create a new Discord session
	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %v", err)
	}

	// Create bot instance
	bot := &Bot{
		session:  session,
		config:   cfg,
		engine:   engine,
		logger:   logger,
		commands: make(map[string]CommandHandler),
	}

	// Register message handler
	session.AddHandler(bot.messageHandler)

	// Register commands
	bot.registerCommands()

	return bot, nil
}

// Start starts the Discord bot
func (b *Bot) Start() error {
	// Open Discord connection
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %v", err)
	}

	b.logger.Info("Bot", "Discord bot started")
	return nil
}

// Stop stops the Discord bot
func (b *Bot) Stop() error {
	b.logger.Info("Bot", "Stopping Discord bot")
	return b.session.Close()
}

// Run runs the bot and blocks until interrupted
func (b *Bot) Run() error {
	// Start the bot
	if err := b.Start(); err != nil {
		return err
	}

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Stop the bot
	return b.Stop()
}

// messageHandler handles incoming Discord messages
func (b *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots (including ourselves)
	if m.Author.Bot {
		return
	}

	// Check if message starts with command prefix
	if len(m.Content) > 0 && string(m.Content[0]) == b.config.CommandPrefix {
		b.handleCommand(s, m)
	}
}

// handleCommand processes a command message
func (b *Bot) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Parse command and arguments
	command, args := parseCommand(m.Content[len(b.config.CommandPrefix):])

	// Log command
	b.logger.Info("Bot", fmt.Sprintf("Command received: %s %v from %s", 
		command, args, m.Author.Username))

	// Look up command handler
	handler, exists := b.commands[command]
	if !exists {
		// Unknown command
		s.ChannelMessageSend(m.ChannelID, 
			fmt.Sprintf("Unknown command: %s. Type %shelp for available commands.", 
				command, b.config.CommandPrefix))
		return
	}

	// Execute command
	if err := handler(s, m, args); err != nil {
		// Command error
		s.ChannelMessageSend(m.ChannelID, 
			fmt.Sprintf("Error executing command: %v", err))
		b.logger.Error("Bot", fmt.Sprintf("Command error: %v", err))
	}
}

// parseCommand splits a message into command and arguments
func parseCommand(content string) (string, []string) {
	// TODO: Implement proper command parsing with quoted arguments
	// For now, just split by space
	// Command is the first word, args are the rest
	return "help", []string{} // Placeholder
}

// registerCommands registers all command handlers
func (b *Bot) registerCommands() {
	// Register help command
	b.commands["help"] = b.helpCommand
	
	// Register intelligence commands
	b.commands["latest"] = b.latestCommand
	b.commands["intel"] = b.intelCommand
	b.commands["cybersec"] = b.categoryCommand(models.CategoryCybersec)
	b.commands["aitools"] = b.categoryCommand(models.CategoryAITools)
	b.commands["opensource"] = b.categoryCommand(models.CategoryOpenSource)
	b.commands["infosec"] = b.categoryCommand(models.CategoryInfosecNews)
	
	// Register admin commands
	b.commands["status"] = b.statusCommand
	b.commands["refresh"] = b.refreshCommand
}

// Command handlers

// helpCommand handles the help command
func (b *Bot) helpCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "Infopulse Node Help",
		Description: "Available commands:",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  b.config.CommandPrefix + "help",
				Value: "Show this help message",
			},
			{
				Name:  b.config.CommandPrefix + "latest [count]",
				Value: "Show latest intelligence items",
			},
			{
				Name:  b.config.CommandPrefix + "intel <id>",
				Value: "Show details for a specific intelligence item",
			},
			{
				Name:  b.config.CommandPrefix + "cybersec [count]",
				Value: "Show latest cybersecurity intelligence",
			},
			{
				Name:  b.config.CommandPrefix + "aitools [count]",
				Value: "Show latest AI tools intelligence",
			},
			{
				Name:  b.config.CommandPrefix + "opensource [count]",
				Value: "Show latest open source intelligence",
			},
			{
				Name:  b.config.CommandPrefix + "infosec [count]",
				Value: "Show latest infosec news",
			},
			{
				Name:  b.config.CommandPrefix + "status",
				Value: "Show bot status",
			},
			{
				Name:  b.config.CommandPrefix + "refresh",
				Value: "Force refresh of intelligence feeds (admin only)",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Infopulse Node v1.0",
		},
	}

	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	return err
}

// latestCommand handles the latest command
func (b *Bot) latestCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	// Get latest intel
	items := b.engine.GetLatestIntel("", 10) // Default limit to 10
	
	// Create embed
	embed := createIntelEmbed("Latest Intelligence", items)
	
	// Send embed
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	return err
}

// intelCommand handles the intel command
func (b *Bot) intelCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	// TODO: Implement
	return nil
}

// categoryCommand creates a command handler for a specific category
func (b *Bot) categoryCommand(category models.Category) CommandHandler {
	return func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
		// Get intel for category
		items := b.engine.GetLatestIntel(category, 10) // Default limit to 10
		
		// Create embed
		embed := createIntelEmbed(fmt.Sprintf("%s Intelligence", category), items)
		
		// Send embed
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return err
	}
}

// statusCommand handles the status command
func (b *Bot) statusCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	// Get stats
	totalItems := b.engine.GetTotalCount()
	
	// Create embed
	embed := &discordgo.MessageEmbed{
		Title: "Infopulse Node Status",
		Color: 0x0000ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Total Intelligence Items",
				Value: fmt.Sprintf("%d", totalItems),
			},
			{
				Name:  "Feed Sources",
				Value: fmt.Sprintf("%d", len(b.config.FeedSources)),
			},
			{
				Name:  "Auto-posting",
				Value: fmt.Sprintf("%v", b.config.AutopostEnabled),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Infopulse Node v1.0",
		},
	}
	
	// Send embed
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	return err
}

// refreshCommand handles the refresh command
func (b *Bot) refreshCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	// Check if user has admin role
	if !b.isAdmin(m.Member) {
		return fmt.Errorf("you do not have permission to use this command")
	}
	
	// TODO: Trigger a manual refresh of feeds
	
	// Send response
	_, err := s.ChannelMessageSend(m.ChannelID, "Refreshing intelligence feeds...")
	return err
}

// isAdmin checks if a user has an admin role
func (b *Bot) isAdmin(member *discordgo.Member) bool {
	if member == nil {
		return false
	}
	
	for _, roleID := range member.Roles {
		// TODO: Check if role is in admin roles list
		// For now, just return true
	}
	
	return true
}
