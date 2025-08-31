package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/raainshe/akira/cmd"
	"github.com/raainshe/akira/internal/cache"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
	"github.com/raainshe/akira/internal/tui"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// AppServices holds all initialized services
type AppServices struct {
	Config         *config.Config
	Logger         *logging.Logger
	Cache          *cache.CacheManager
	QBClient       *qbittorrent.Client
	TorrentService *core.TorrentService
	DiskService    *core.DiskService
	SeedingService *core.SeedingService
}

func main() {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutting down gracefully...")
		cancel()
	}()

	// Initialize services
	services, err := initializeServices(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize services: %v\n", err)
		os.Exit(1)
	}

	// Create root command
	rootCmd := createRootCommand(ctx, services)

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Command failed: %v\n", err)
		cleanup(services)
		os.Exit(1)
	}

	// Cleanup services
	cleanup(services)
}

// createRootCommand creates the main Cobra root command
func createRootCommand(ctx context.Context, services *AppServices) *cobra.Command {
	var configFile string
	var logLevel string
	var verbose bool

	rootCmd := &cobra.Command{
		Use:   "akira",
		Short: "🌟 Akira - Beautiful Torrent Management CLI & TUI",
		Long: `🌟 Akira - Beautiful Torrent Management CLI & TUI

Akira provides both traditional CLI commands for automation and a beautiful 
interactive TUI for monitoring and management. Choose the interface that 
fits your workflow.

Examples:
  akira                    # Launch interactive TUI (default)
  akira tui               # Explicit TUI mode
  akira list              # Quick torrent listing
  akira add "magnet:..."  # Add torrent via CLI
  akira seeding status    # Check seeding status`,
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitCommit),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: Launch TUI
			return tui.Run(ctx, services.Config, services.TorrentService,
				services.DiskService, services.SeedingService, services.QBClient)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Handle global flags
			if configFile != "" {
				viper.SetConfigFile(configFile)
				if err := viper.ReadInConfig(); err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}
			}

			// Set log level based on flags
			if verbose {
				services.Logger.SetLevel(logrus.DebugLevel)
			} else if logLevel != "" {
				level, err := logrus.ParseLevel(logLevel)
				if err != nil {
					return fmt.Errorf("invalid log level: %w", err)
				}
				services.Logger.SetLevel(level)
			} else {
				// Default: only show warnings and errors for CLI commands
				services.Logger.SetLevel(logrus.WarnLevel)
			}

			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level (debug, info, warn, error) - default: warn")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (shows all logs)")

	// Add all subcommands
	rootCmd.AddCommand(
		cmd.NewTUICommand(ctx, services.Config, services.TorrentService, services.DiskService, services.SeedingService, services.QBClient),
		cmd.NewListCommand(ctx, services.TorrentService),
		cmd.NewDownloadingCommand(ctx, services.TorrentService),
		cmd.NewAddCommand(ctx, services.TorrentService, services.SeedingService),
		cmd.NewDeleteCommand(ctx, services.TorrentService, services.SeedingService),
		cmd.NewDiskCommand(ctx, services.DiskService),
		cmd.NewLogsCommand(ctx, services.Config),
		cmd.NewSeedingCommand(ctx, services.SeedingService),
		cmd.NewVersionCommand(version, buildTime, gitCommit),
	)

	return rootCmd
}

// initializeServices initializes all application services
func initializeServices(ctx context.Context) (*AppServices, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Temporarily override log level for quieter CLI initialization
	originalLogLevel := cfg.Logging.Level
	cfg.Logging.Level = "warn"

	// Initialize logging
	logger, err := logging.Initialize(&cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Restore original config (in case TUI mode needs it)
	cfg.Logging.Level = originalLogLevel

	mainLogger := logging.GetLogger()

	// Initialize cache
	cacheManager, err := cache.Initialize(&cfg.Cache)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize qBittorrent client
	qbClient, err := qbittorrent.NewClient(cfg.QBittorrent.URL, cfg.QBittorrent.Username, cfg.QBittorrent.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create qBittorrent client: %w", err)
	}

	// Test qBittorrent connection
	if err := qbClient.Login(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to qBittorrent: %w", err)
	}
	mainLogger.Info("✅ Connected to qBittorrent successfully")

	// Initialize core services
	torrentService := core.NewTorrentService(qbClient, cfg, cacheManager)
	diskService := core.NewDiskService(cfg, cacheManager)
	seedingService := core.NewSeedingService(cfg, torrentService, qbClient)

	// Start seeding service
	if err := seedingService.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start seeding service: %w", err)
	}
	mainLogger.Info("🌱 Seeding management service started")

	mainLogger.Info("✅ All services initialized successfully")

	return &AppServices{
		Config:         cfg,
		Logger:         logger,
		Cache:          cacheManager,
		QBClient:       qbClient,
		TorrentService: torrentService,
		DiskService:    diskService,
		SeedingService: seedingService,
	}, nil
}

// cleanup gracefully shuts down all services
func cleanup(services *AppServices) {
	if services == nil {
		return
	}

	mainLogger := logging.GetLogger()
	mainLogger.Info("🧹 Cleaning up services...")

	// Stop seeding service
	if services.SeedingService != nil {
		if err := services.SeedingService.Stop(); err != nil {
			mainLogger.WithError(err).Error("Failed to stop seeding service")
		} else {
			mainLogger.Info("✅ Seeding service stopped")
		}
	}

	// Logout from qBittorrent
	if services.QBClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := services.QBClient.Logout(ctx); err != nil {
			mainLogger.WithError(err).Warn("Failed to logout from qBittorrent")
		} else {
			mainLogger.Info("✅ Logged out from qBittorrent")
		}
	}

	// Shutdown cache
	if services.Cache != nil {
		services.Cache.Shutdown()
		mainLogger.Info("✅ Cache manager shutdown")
	}

	mainLogger.Info("✅ Cleanup completed")
}
