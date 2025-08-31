package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raainshe/akira/internal/cli"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
	"github.com/raainshe/akira/internal/tui"
)

// NewTUICommand creates the TUI command
func NewTUICommand(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) *cobra.Command {

	return &cobra.Command{
		Use:   "tui",
		Short: "🌟 Launch interactive TUI",
		Long:  "Launch the beautiful interactive Terminal User Interface for torrent management",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(ctx, cfg, torrentService, diskService, seedingService, qbClient)
		},
	}
}

// NewListCommand creates the list command
func NewListCommand(ctx context.Context, torrentService *core.TorrentService) *cobra.Command {
	var category string
	var state string
	var seedingOnly bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "📋 List torrents",
		Long: `📋 List torrents with filtering and formatting options

This command displays all torrents with a beautiful table format including:
- Progress bars and completion status
- Download/upload speeds and ETA
- Color-coded states (downloading, seeding, paused, error)
- Filtering by category and state
- JSON output for scripting

Examples:
  akira list                           # Show all torrents
  akira list --category movies         # Show only movies
  akira list --seeding-only           # Show only seeding torrents
  akira list --state downloading      # Show only downloading
  akira list --json                   # JSON output for scripts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCommand(ctx, torrentService, category, state, seedingOnly, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "filter by category (series, movies, anime)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "filter by state (downloading, seeding, paused, error)")
	cmd.Flags().BoolVar(&seedingOnly, "seeding-only", false, "show only seeding torrents")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output in JSON format")

	return cmd
}

// NewAddCommand creates the add command
func NewAddCommand(ctx context.Context, torrentService *core.TorrentService) *cobra.Command {
	var category string
	var path string

	cmd := &cobra.Command{
		Use:   "add <magnet-uri>",
		Short: "➕ Add torrent",
		Long:  "Add a new torrent from magnet URI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			magnetURI := args[0]
			// TODO: Implement add command
			fmt.Println("➕ Add command - Coming soon!")
			fmt.Printf("Magnet: %s\n", magnetURI)
			fmt.Printf("Category: %s, Path: %s\n", category, path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "category (series, movies, anime)")
	cmd.Flags().StringVarP(&path, "path", "p", "", "custom save path")

	return cmd
}

// NewDeleteCommand creates the delete command
func NewDeleteCommand(ctx context.Context, torrentService *core.TorrentService) *cobra.Command {
	var hash string
	var category string
	var deleteFiles bool
	var interactive bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "🗑️  Delete torrents",
		Long:  "Delete torrents with optional file removal",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement delete command
			fmt.Println("🗑️ Delete command - Coming soon!")
			fmt.Printf("Hash: %s, Category: %s, Delete files: %v, Interactive: %v\n",
				hash, category, deleteFiles, interactive)
			return nil
		},
	}

	cmd.Flags().StringVar(&hash, "hash", "", "specific torrent hash to delete")
	cmd.Flags().StringVarP(&category, "category", "c", "", "filter by category")
	cmd.Flags().BoolVar(&deleteFiles, "delete-files", false, "also delete downloaded files")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "interactive selection")

	return cmd
}

// NewDiskCommand creates the disk space command
func NewDiskCommand(ctx context.Context, diskService *core.DiskService) *cobra.Command {
	var path string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "disk",
		Short: "💾 Check disk space",
		Long:  "Check disk space usage for configured paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement disk command
			fmt.Println("💾 Disk command - Coming soon!")
			fmt.Printf("Path: %s, JSON: %v\n", path, jsonOutput)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "specific path to check")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output in JSON format")

	return cmd
}

// NewLogsCommand creates the logs command
func NewLogsCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	var tail int
	var follow bool
	var level string
	var component string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "📜 View logs",
		Long:  "View application logs with filtering options",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement logs command
			fmt.Println("📜 Logs command - Coming soon!")
			fmt.Printf("Tail: %d, Follow: %v, Level: %s, Component: %s\n",
				tail, follow, level, component)
			return nil
		},
	}

	cmd.Flags().IntVarP(&tail, "tail", "n", 50, "number of recent entries to show")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	cmd.Flags().StringVarP(&level, "level", "l", "", "filter by log level")
	cmd.Flags().StringVarP(&component, "component", "c", "", "filter by component")

	return cmd
}

// NewSeedingCommand creates the seeding command
func NewSeedingCommand(ctx context.Context, seedingService *core.SeedingService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seeding",
		Short: "🌱 Seeding management",
		Long:  "Manage automatic seeding with status and controls",
	}

	// Add subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "status",
			Short: "📊 Show seeding status",
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement seeding status
				fmt.Println("📊 Seeding status - Coming soon!")
				return nil
			},
		},
		&cobra.Command{
			Use:   "stop-all",
			Short: "⏹️  Stop all seeding",
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement stop-all
				fmt.Println("⏹️ Stop all seeding - Coming soon!")
				return nil
			},
		},
		&cobra.Command{
			Use:   "stats",
			Short: "📈 Show statistics",
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement stats
				fmt.Println("📈 Seeding statistics - Coming soon!")
				return nil
			},
		},
	)

	return cmd
}

// NewVersionCommand creates the version command
func NewVersionCommand(version, buildTime, gitCommit string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "📋 Show version information",
		Long:  "Display version, build time, and git commit information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("🌟 Akira Torrent Manager\n")
			fmt.Printf("Version: %s\n", version)
			fmt.Printf("Built: %s\n", buildTime)
			fmt.Printf("Commit: %s\n", gitCommit)
		},
	}
}

// runListCommand implements the list command functionality
func runListCommand(ctx context.Context, torrentService *core.TorrentService,
	category, state string, seedingOnly, jsonOutput bool) error {

	// Create filter options
	filter := &core.TorrentFilter{}

	// Apply category filter
	if category != "" {
		// Validate category
		validCategories := []string{"series", "movies", "anime"}
		categoryLower := strings.ToLower(category)
		isValid := false
		for _, valid := range validCategories {
			if categoryLower == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid category '%s'. Valid categories: %v", category, validCategories)
		}
		filter.Category = categoryLower
	}

	// Apply state filter
	if state != "" {
		stateLower := strings.ToLower(state)
		// Map user-friendly state names to qBittorrent states
		switch stateLower {
		case "downloading":
			filter.State = qbittorrent.StateDownloading
		case "seeding":
			filter.State = qbittorrent.StateUploading
		case "paused":
			filter.State = qbittorrent.StatePausedDL // Will also match paused upload
		case "error":
			filter.State = qbittorrent.StateError
		default:
			// Try to use the state directly as TorrentState
			filter.State = qbittorrent.TorrentState(state)
		}
	}

	// Apply seeding-only filter
	if seedingOnly {
		filter.State = qbittorrent.StateUploading
	}

	// Get torrents from service
	torrents, err := torrentService.GetTorrents(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get torrents: %w", err)
	}

	// Additional filtering for paused state (since qBittorrent has multiple paused states)
	if state != "" && strings.ToLower(state) == "paused" {
		var filteredTorrents []qbittorrent.Torrent
		for _, torrent := range torrents {
			if strings.Contains(strings.ToLower(string(torrent.State)), "paused") {
				filteredTorrents = append(filteredTorrents, torrent)
			}
		}
		torrents = filteredTorrents
	}

	// Convert to pointer slice for the table formatter
	torrentPtrs := make([]*qbittorrent.Torrent, len(torrents))
	for i := range torrents {
		torrentPtrs[i] = &torrents[i]
	}

	// Print results
	return cli.PrintTorrentTable(torrentPtrs, jsonOutput)
}
