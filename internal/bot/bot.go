package bot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// Bot represents the Discord bot instance
type Bot struct {
	session        *discordgo.Session
	config         *config.Config
	logger         *logging.Logger
	torrentService *core.TorrentService
	diskService    *core.DiskService
	seedingService *core.SeedingService
	qbClient       *qbittorrent.Client
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewBot creates a new Discord bot instance
func NewBot(cfg *config.Config, torrentService *core.TorrentService, diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) (*Bot, error) {
	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.Discord.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Create logger for bot
	logger := logging.GetDiscordLogger()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		session:        session,
		config:         cfg,
		logger:         logger,
		torrentService: torrentService,
		diskService:    diskService,
		seedingService: seedingService,
		qbClient:       qbClient,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Set up event handlers
	bot.setupEventHandlers()

	return bot, nil
}

// setupEventHandlers configures Discord event handlers
func (b *Bot) setupEventHandlers() {
	// Bot ready event
	b.session.AddHandler(b.handleReady)

	// Interaction create event (slash commands)
	b.session.AddHandler(b.handleInteractionCreate)
}

// handleReady is called when the bot is ready
func (b *Bot) handleReady(s *discordgo.Session, event *discordgo.Ready) {
	b.logger.Info("Discord bot is ready", map[string]interface{}{
		"bot_id":   event.User.ID,
		"username": event.User.Username,
		"guilds":   len(event.Guilds),
	})

	// Set bot status
	err := s.UpdateGameStatus(0, "Managing torrents")
	if err != nil {
		b.logger.Error("Failed to update bot status", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// handleInteractionCreate handles slash command interactions
func (b *Bot) handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Only handle slash commands
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	// Route to appropriate command handler
	switch i.ApplicationCommandData().Name {
	case "torrents":
		b.handleTorrentsCommand(s, i)
	case "add":
		b.handleAddCommand(s, i)
	case "delete":
		b.handleDeleteCommand(s, i)
	case "disk":
		b.handleDiskCommand(s, i)
	case "logs":
		b.handleLogsCommand(s, i)
	case "seeding-status":
		b.handleSeedingStatusCommand(s, i)
	case "stop-seeding":
		b.handleStopSeedingCommand(s, i)
	case "help":
		b.handleHelpCommand(s, i)
	default:
		b.handleUnknownCommand(s, i)
	}
}

// RegisterCommands registers slash commands with Discord
func (b *Bot) RegisterCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "torrents",
			Description: "List all torrents with filters and pagination",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "filter",
					Description: "Filter torrents (all, downloading, seeding, paused)",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "All", Value: "all"},
						{Name: "Downloading", Value: "downloading"},
						{Name: "Seeding", Value: "seeding"},
						{Name: "Paused", Value: "paused"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "page",
					Description: "Page number (default: 1)",
					Required:    false,
				},
			},
		},
		{
			Name:        "add",
			Description: "Add a magnet link or torrent file",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "magnet",
					Description: "Magnet link to add",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "category",
					Description: "Category for the torrent",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Default", Value: "default"},
						{Name: "Movies", Value: "movies"},
						{Name: "Series", Value: "series"},
						{Name: "Anime", Value: "anime"},
					},
				},
			},
		},
		{
			Name:        "delete",
			Description: "Delete torrents by name or hash",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "query",
					Description: "Torrent name or hash to delete",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "delete_files",
					Description: "Also delete downloaded files",
					Required:    false,
				},
			},
		},
		{
			Name:        "disk",
			Description: "Show disk usage statistics",
		},
		{
			Name:        "logs",
			Description: "Show recent application logs",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "level",
					Description: "Filter logs by level",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "All", Value: "all"},
						{Name: "Error", Value: "error"},
						{Name: "Warning", Value: "warning"},
						{Name: "Info", Value: "info"},
						{Name: "Debug", Value: "debug"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "lines",
					Description: "Number of log lines to show (default: 10)",
					Required:    false,
				},
			},
		},
		{
			Name:        "seeding-status",
			Description: "Show seeding status and statistics",
		},
		{
			Name:        "stop-seeding",
			Description: "Stop seeding for a specific torrent",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "torrent",
					Description: "Torrent name or hash",
					Required:    true,
				},
			},
		},
		{
			Name:        "help",
			Description: "Show available commands and usage",
		},
	}

	// Register commands for each guild (server)
	for _, guildID := range b.config.Discord.GuildIDs {
		b.logger.Info("Registering commands for guild", map[string]interface{}{
			"guild_id": guildID,
		})

		_, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, guildID, commands)
		if err != nil {
			return fmt.Errorf("failed to register commands for guild %s: %w", guildID, err)
		}
	}

	b.logger.Info("Successfully registered Discord commands", map[string]interface{}{
		"command_count": len(commands),
		"guild_count":   len(b.config.Discord.GuildIDs),
	})

	return nil
}

// Start starts the Discord bot
func (b *Bot) Start() error {
	b.logger.Info("Starting Discord bot", map[string]interface{}{
		"guild_count": len(b.config.Discord.GuildIDs),
	})

	// Open Discord connection
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Register slash commands
	err = b.RegisterCommands()
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	b.logger.Info("Discord bot started successfully")

	return nil
}

// Stop gracefully stops the Discord bot
func (b *Bot) Stop() error {
	b.logger.Info("Stopping Discord bot")

	// Cancel context to signal shutdown
	b.cancel()

	// Close Discord session
	if b.session != nil {
		err := b.session.Close()
		if err != nil {
			b.logger.Error("Error closing Discord session", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
	}

	b.logger.Info("Discord bot stopped successfully")
	return nil
}

// WaitForShutdown waits for shutdown signal
func (b *Bot) WaitForShutdown() {
	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal or context cancellation
	select {
	case <-stop:
		b.logger.Info("Received shutdown signal")
	case <-b.ctx.Done():
		b.logger.Info("Received shutdown context")
	}

	// Stop the bot
	err := b.Stop()
	if err != nil {
		b.logger.Error("Error stopping bot", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// Command handlers (to be implemented)
func (b *Bot) handleTorrentsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement torrents command
	b.respondWithError(s, i, "Torrents command not yet implemented")
}

func (b *Bot) handleAddCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement add command
	b.respondWithError(s, i, "Add command not yet implemented")
}

func (b *Bot) handleDeleteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement delete command
	b.respondWithError(s, i, "Delete command not yet implemented")
}

func (b *Bot) handleDiskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement disk command
	b.respondWithError(s, i, "Disk command not yet implemented")
}

func (b *Bot) handleLogsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement logs command
	b.respondWithError(s, i, "Logs command not yet implemented")
}

func (b *Bot) handleSeedingStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement seeding status command
	b.respondWithError(s, i, "Seeding status command not yet implemented")
}

func (b *Bot) handleStopSeedingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement stop seeding command
	b.respondWithError(s, i, "Stop seeding command not yet implemented")
}

func (b *Bot) handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: Implement help command
	b.respondWithError(s, i, "Help command not yet implemented")
}

func (b *Bot) handleUnknownCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.respondWithError(s, i, "Unknown command")
}

// respondWithError sends an error response to Discord
func (b *Bot) respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		b.logger.Error("Failed to send error response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}
