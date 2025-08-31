package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	var downloadingOnly bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "📋 List torrents",
		Long: `📋 List torrents with filtering and formatting options

This command displays all torrents with a beautiful table format including:
- Progress bars and completion status
- Download/upload speeds and ETA
- Color-coded states (downloading, seeding, paused, error)
- Filtering by category, state, and activity
- JSON output for scripting

Examples:
  akira list                           # Show all torrents
  akira list --category movies         # Show only movies
  akira list --seeding-only           # Show only seeding torrents
  akira list --downloading            # Show only downloading torrents
  akira list --state downloading      # Show only downloading (alternative)
  akira list --json                   # JSON output for scripts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCommand(ctx, torrentService, category, state, seedingOnly, downloadingOnly, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "filter by category (series, movies, anime)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "filter by state (downloading, seeding, paused, error)")
	cmd.Flags().BoolVar(&seedingOnly, "seeding-only", false, "show only seeding torrents")
	cmd.Flags().BoolVar(&downloadingOnly, "downloading", false, "show only downloading torrents")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output in JSON format")

	return cmd
}

// NewAddCommand creates the add command
func NewAddCommand(ctx context.Context, torrentService *core.TorrentService, seedingService *core.SeedingService) *cobra.Command {
	var category string
	var path string

	cmd := &cobra.Command{
		Use:   "add <magnet-uri>",
		Short: "➕ Add torrent",
		Long: `➕ Add a new torrent from magnet URI

This command adds a torrent to qBittorrent with validation and feedback:
- Validates magnet URI format and info hash
- Validates category selection (series, movies, anime)
- Supports custom save path override
- Shows detailed torrent information after adding
- Provides progress tracking guidance

Examples:
  akira add "magnet:?xt=urn:btih:..."                    # Add with default settings
  akira add "magnet:?xt=urn:btih:..." --category movies  # Add to movies category
  akira add "magnet:?xt=urn:btih:..." --path /custom     # Add with custom path`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			magnetURI := args[0]
			return runAddCommand(ctx, torrentService, seedingService, magnetURI, category, path)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "category (series, movies, anime)")
	cmd.Flags().StringVarP(&path, "path", "p", "", "custom save path")

	return cmd
}

// NewDeleteCommand creates the delete command
func NewDeleteCommand(ctx context.Context, torrentService *core.TorrentService, seedingService *core.SeedingService) *cobra.Command {
	var hash string
	var namePattern string
	var category string
	var deleteFiles bool
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [flags]",
		Short: "🗑️  Delete torrents",
		Long: `🗑️  Delete torrents with optional file removal

This command deletes torrents from qBittorrent with safety confirmations:
- Delete by specific hash or name pattern
- Filter by category for batch operations
- Option to delete files or keep them on disk
- Safety confirmation prompts (unless --force is used)
- Detailed progress and result feedback

Examples:
  akira delete --hash abc123...                    # Delete specific torrent
  akira delete --name "Ubuntu"                     # Delete torrents matching "Ubuntu"
  akira delete --category movies                   # Delete all torrents in movies category
  akira delete --hash abc123... --delete-files    # Delete torrent and its files
  akira delete --name "Ubuntu" --force            # Skip confirmation prompt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteCommand(ctx, torrentService, seedingService, hash, namePattern, category, deleteFiles, force)
		},
	}

	cmd.Flags().StringVar(&hash, "hash", "", "specific torrent hash to delete")
	cmd.Flags().StringVar(&namePattern, "name", "", "delete torrents matching name pattern")
	cmd.Flags().StringVar(&category, "category", "", "delete all torrents in category")
	cmd.Flags().BoolVar(&deleteFiles, "delete-files", false, "also delete downloaded files")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}

// NewDiskCommand creates the disk space command
func NewDiskCommand(ctx context.Context, diskService *core.DiskService) *cobra.Command {
	var path string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "disk",
		Short: "💾 Check disk space",
		Long: `💾 Check disk space usage for configured paths

This command displays disk usage information with beautiful progress bars:
- Visual progress bars showing usage percentage
- Color-coded health indicators (healthy, warning, critical)
- Human-readable sizes (GB, TB) with precise percentages
- Summary statistics for multiple paths
- JSON output for scripting and automation

Examples:
  akira disk                    # Show all configured paths
  akira disk --path /custom     # Check specific path
  akira disk --json            # JSON output for scripts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiskCommand(ctx, diskService, path, jsonOutput)
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

	// Add flags for status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "📊 Show seeding status",
		Long: `📊 Show detailed seeding status and tracking information

This command displays comprehensive seeding management information:
- Service status and health
- Statistics for tracked torrents
- Individual torrent seeding progress
- Time remaining and completion status

Examples:
  akira seeding status                    # Show overview
  akira seeding status --detailed         # Show detailed torrent list  
  akira seeding status --json            # Export as JSON`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			detailed, _ := cmd.Flags().GetBool("detailed")
			return runSeedingStatusCommand(ctx, seedingService, jsonOutput, detailed)
		},
	}
	statusCmd.Flags().BoolP("json", "j", false, "output in JSON format")
	statusCmd.Flags().BoolP("detailed", "d", false, "show detailed torrent information")

	// Add subcommands
	cmd.AddCommand(
		statusCmd,
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
	category, state string, seedingOnly, downloadingOnly, jsonOutput bool) error {

	// Validate conflicting flags
	if seedingOnly && downloadingOnly {
		return fmt.Errorf("cannot use both --seeding-only and --downloading flags together")
	}

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

	// Apply downloading-only filter
	if downloadingOnly {
		// Set filter to show downloading states
		filter.States = []qbittorrent.TorrentState{
			qbittorrent.StateDownloading,
			qbittorrent.StateMetaDL,
			qbittorrent.StateStalledDL,
			qbittorrent.StateQueuedDL,
			qbittorrent.StateForcedDL,
			qbittorrent.StateCheckingDL,
			qbittorrent.StateAllocating,
		}
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

// NewDownloadingCommand creates a dedicated downloading torrents command
func NewDownloadingCommand(ctx context.Context, torrentService *core.TorrentService) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "downloading",
		Short: "⬇️  Show downloading torrents",
		Long: `⬇️  Show only torrents that are currently downloading

This command is a shortcut for 'akira list --downloading' and displays:
- All torrents in downloading states (downloading, metadata fetch, queued, etc.)
- Progress bars and download speeds
- ETA and completion information
- Color-coded status indicators

Examples:
  akira downloading                # Show downloading torrents
  akira downloading --json         # JSON output for scripts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Call runListCommand with downloading filter enabled
			return runListCommand(ctx, torrentService, "", "", false, true, jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output in JSON format")

	return cmd
}

// runDiskCommand implements the disk space command functionality
func runDiskCommand(ctx context.Context, diskService *core.DiskService,
	customPath string, jsonOutput bool) error {

	var diskInfos []*cli.DiskSpaceInfo

	if customPath != "" {
		// Check specific path
		diskSpace, err := diskService.GetDiskSpace(ctx, customPath)
		if err != nil {
			return fmt.Errorf("failed to get disk space for path '%s': %w", customPath, err)
		}

		info := cli.ConvertDiskSpaceInfo(customPath, diskSpace.Used, diskSpace.Free, diskSpace.Total)
		diskInfos = append(diskInfos, info)

	} else {
		// Get all configured qBittorrent paths from environment
		// We'll use a simple approach and check common paths for now
		paths := []string{
			os.Getenv("QBITTORRENT_DEFAULT_SAVE_PATH"),
			os.Getenv("QBITTORRENT_SERIES_SAVE_PATH"),
			os.Getenv("QBITTORRENT_MOVIES_SAVE_PATH"),
			os.Getenv("QBITTORRENT_ANIME_SAVE_PATH"),
		}

		// Remove duplicates and empty paths
		uniquePaths := make(map[string]bool)
		var validPaths []string

		for _, path := range paths {
			if path != "" && !uniquePaths[path] {
				uniquePaths[path] = true
				validPaths = append(validPaths, path)
			}
		}

		// Get disk space for each unique path
		for _, path := range validPaths {
			diskSpace, err := diskService.GetDiskSpace(ctx, path)
			if err != nil {
				// Log error but continue with other paths
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to get disk space for '%s': %v\n", path, err)
				continue
			}

			info := cli.ConvertDiskSpaceInfo(path, diskSpace.Used, diskSpace.Free, diskSpace.Total)
			diskInfos = append(diskInfos, info)
		}

		if len(diskInfos) == 0 {
			return fmt.Errorf("no valid paths found to check disk space")
		}
	}

	// Print results
	return cli.PrintDiskSpaceInfo(diskInfos, jsonOutput)
}

// runAddCommand implements the add magnet command functionality
func runAddCommand(ctx context.Context, torrentService *core.TorrentService, seedingService *core.SeedingService,
	magnetURI, category, customPath string) error {

	// Step 1: Validate magnet URI
	fmt.Printf("🔍 %s\n", cli.ColorHeader.Sprint("Validating magnet URI..."))

	magnetInfo, err := cli.ExtractMagnetInfo(magnetURI)
	if err != nil {
		cli.PrintAddResult(false, nil, category, customPath, err)
		return err
	}

	fmt.Printf("✅ Valid magnet URI found\n")
	fmt.Printf("   Name: %s\n", magnetInfo.DisplayName)
	fmt.Printf("   Hash: %s\n", magnetInfo.Hash)
	fmt.Printf("   Trackers: %d\n\n", len(magnetInfo.Trackers))

	// Step 2: Validate category
	if category != "" {
		fmt.Printf("🏷️  %s\n", cli.ColorHeader.Sprint("Validating category..."))

		if err := cli.ValidateCategory(category); err != nil {
			cli.PrintAddResult(false, magnetInfo, category, customPath, err)
			return err
		}

		fmt.Printf("✅ Category '%s' is valid\n\n", category)
	}

	// Step 3: Validate custom path if provided
	if customPath != "" {
		fmt.Printf("📁 %s\n", cli.ColorHeader.Sprint("Validating custom path..."))

		if _, err := os.Stat(customPath); err != nil {
			pathErr := fmt.Errorf("custom path does not exist or is not accessible: %w", err)
			cli.PrintAddResult(false, magnetInfo, category, customPath, pathErr)
			return pathErr
		}

		fmt.Printf("✅ Custom path '%s' is accessible\n\n", customPath)
	}

	// Step 4: Add torrent to qBittorrent
	fmt.Printf("⬇️  %s\n", cli.ColorHeader.Sprint("Adding torrent to qBittorrent..."))

	// Create add request
	addRequest := &core.AddTorrentRequest{
		MagnetURI: magnetURI,
		Category:  category,
		SavePath:  customPath,
	}

	// Add the torrent
	err = torrentService.AddMagnet(ctx, addRequest)
	if err != nil {
		cli.PrintAddResult(false, magnetInfo, category, customPath, err)
		return fmt.Errorf("failed to add torrent: %w", err)
	}

	// Step 5: Start seeding tracking
	fmt.Printf("🌱 %s\n", cli.ColorHeader.Sprint("Starting seeding tracking..."))

	err = seedingService.StartTracking(ctx, magnetInfo.Hash, magnetInfo.DisplayName)
	if err != nil {
		// Don't fail the whole operation if seeding tracking fails
		fmt.Printf("⚠️  Warning: Failed to start seeding tracking: %v\n", err)
	} else {
		fmt.Printf("✅ Seeding tracking started\n\n")
	}

	// Step 6: Success!
	cli.PrintAddResult(true, magnetInfo, category, customPath, nil)
	return nil
}

// runDeleteCommand implements the delete torrent command functionality
func runDeleteCommand(ctx context.Context, torrentService *core.TorrentService, seedingService *core.SeedingService,
	hash, namePattern, category string, deleteFiles, force bool) error {

	// Step 1: Validate input parameters
	if hash == "" && namePattern == "" && category == "" {
		return fmt.Errorf("must specify one of: --hash, --name, or --category")
	}

	if (hash != "" && namePattern != "") || (hash != "" && category != "") || (namePattern != "" && category != "") {
		return fmt.Errorf("can only specify one of: --hash, --name, or --category")
	}

	// Step 2: Find torrents to delete
	fmt.Printf("🔍 %s\n", cli.ColorHeader.Sprint("Finding torrents to delete..."))

	var torrentsToDelete []qbittorrent.Torrent
	var err error

	if hash != "" {
		// Delete by specific hash
		torrent, err := torrentService.FindTorrentByHash(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to find torrent: %w", err)
		}
		torrentsToDelete = []qbittorrent.Torrent{*torrent}
		fmt.Printf("✅ Found torrent: %s\n\n", torrent.Name)

	} else if namePattern != "" {
		// Delete by name pattern
		torrents, err := torrentService.FindTorrentsByPattern(ctx, namePattern)
		if err != nil {
			return fmt.Errorf("failed to search torrents: %w", err)
		}
		if len(torrents) == 0 {
			return fmt.Errorf("no torrents found matching pattern '%s'", namePattern)
		}
		torrentsToDelete = torrents
		fmt.Printf("✅ Found %d torrent(s) matching '%s'\n\n", len(torrents), namePattern)

	} else if category != "" {
		// Delete by category
		filter := &core.TorrentFilter{
			Category: category,
		}
		torrents, err := torrentService.GetTorrents(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get torrents by category: %w", err)
		}
		if len(torrents) == 0 {
			return fmt.Errorf("no torrents found in category '%s'", category)
		}
		torrentsToDelete = torrents
		fmt.Printf("✅ Found %d torrent(s) in category '%s'\n\n", len(torrents), category)
	}

	// Step 3: Get confirmation (unless forced)
	var confirmed bool
	if force {
		fmt.Printf("⚡ %s\n\n", cli.ColorDownloading.Sprint("Force mode enabled - skipping confirmation"))
		confirmed = true
	} else {
		// Convert to pointer slice for PrintDeleteConfirmation
		torrentPtrs := make([]*qbittorrent.Torrent, len(torrentsToDelete))
		for i := range torrentsToDelete {
			torrentPtrs[i] = &torrentsToDelete[i]
		}
		confirmed = cli.PrintDeleteConfirmation(torrentPtrs, deleteFiles)
	}

	if !confirmed {
		fmt.Println("❌ Deletion cancelled by user")
		return nil
	}

	// Step 4: Delete torrents
	fmt.Printf("🗑️  %s\n", cli.ColorHeader.Sprint("Deleting torrents..."))

	// Extract hashes
	hashes := make([]string, len(torrentsToDelete))
	for i, torrent := range torrentsToDelete {
		hashes[i] = torrent.Hash
	}

	// Perform deletion
	err = torrentService.DeleteTorrents(ctx, hashes, deleteFiles)
	if err != nil {
		// For now, treat as complete failure
		failed := make(map[string]error)
		for _, hash := range hashes {
			failed[hash] = err
		}
		cli.PrintDeleteResult([]string{}, failed, deleteFiles)
		return fmt.Errorf("failed to delete torrents: %w", err)
	}

	// Step 5: Stop seeding tracking for deleted torrents
	fmt.Printf("🛑 %s\n", cli.ColorHeader.Sprint("Stopping seeding tracking..."))

	stoppedCount := 0
	for _, hash := range hashes {
		err := seedingService.StopTracking(hash)
		if err != nil {
			// Don't fail the whole operation, just log the warning
			fmt.Printf("⚠️  Warning: Failed to stop seeding tracking for %s: %v\n", hash[:16]+"...", err)
		} else {
			stoppedCount++
		}
	}

	if stoppedCount > 0 {
		fmt.Printf("✅ Stopped seeding tracking for %d torrent(s)\n\n", stoppedCount)
	}

	// Step 6: Success!
	cli.PrintDeleteResult(hashes, map[string]error{}, deleteFiles)
	return nil
}

// runSeedingStatusCommand implements the seeding status command functionality
func runSeedingStatusCommand(ctx context.Context, seedingService *core.SeedingService,
	jsonOutput, detailed bool) error {

	// Get seeding service status
	fmt.Printf("🔍 %s\n", cli.ColorHeader.Sprint("Checking seeding service status..."))

	if !seedingService.IsRunning() {
		fmt.Printf("❌ %s\n", cli.ColorError.Sprint("Seeding service is not running"))
		return fmt.Errorf("seeding service is not running")
	}

	fmt.Printf("✅ Seeding service is running\n\n")

	// Get detailed seeding status
	status, err := seedingService.GetSeedingStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeding status: %w", err)
	}

	// Output in JSON format if requested
	if jsonOutput {
		return outputSeedingStatusJSON(status)
	}

	// Output in human-readable format
	return outputSeedingStatusHuman(status, detailed)
}

// runForceStopSeeding handles force stopping seeding for a specific torrent
func runForceStopSeeding(ctx context.Context, seedingService *core.SeedingService, hash string) error {
	fmt.Printf("🛑 %s\n", cli.ColorHeader.Sprintf("Force stopping seeding for %s...", hash[:16]+"..."))

	err := seedingService.ForceStopSeeding(ctx, []string{hash})
	if err != nil {
		fmt.Printf("❌ %s\n", cli.ColorError.Sprintf("Failed to force stop seeding: %v", err))
		return fmt.Errorf("failed to force stop seeding: %w", err)
	}

	fmt.Printf("✅ %s\n", cli.ColorSeeding.Sprint("Successfully force stopped seeding"))
	return nil
}

// outputSeedingStatusJSON outputs seeding status in JSON format
func outputSeedingStatusJSON(status *core.SeedingStatus) error {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal seeding status to JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// outputSeedingStatusHuman outputs seeding status in human-readable format
func outputSeedingStatusHuman(status *core.SeedingStatus, detailed bool) error {
	// Service overview
	fmt.Printf("🌱 %s\n\n", cli.ColorHeader.Sprint("Seeding Service Overview"))

	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("   Tracked Torrents: %d\n", status.TrackedTorrents)
	fmt.Printf("   Active Seeding: %d\n", status.ActiveSeeding)
	fmt.Printf("   Completed Seeding: %d\n", status.CompletedSeeding)
	fmt.Printf("   Overdue Seeding: %d\n", status.OverdueSeeding)

	if status.TotalDownloadTime > 0 {
		fmt.Printf("   Total Download Time: %s\n", formatDuration(status.TotalDownloadTime))
	}
	if status.TotalSeedingTime > 0 {
		fmt.Printf("   Total Seeding Time: %s\n", formatDuration(status.TotalSeedingTime))
	}

	fmt.Printf("   Last Checked: %s\n", status.LastChecked.Format("2006-01-02 15:04:05"))

	// Show detailed torrent information if requested
	if detailed && len(status.Details) > 0 {
		fmt.Printf("\n📋 %s\n\n", cli.ColorHeader.Sprint("Tracked Torrents"))

		for hash, torrentStatus := range status.Details {
			fmt.Printf("🔗 %s\n", hash[:16]+"...")
			fmt.Printf("   Name: %s\n", torrentStatus.Name)

			if torrentStatus.DownloadDuration > 0 {
				fmt.Printf("   Download Time: %s\n", formatDuration(torrentStatus.DownloadDuration))
			}
			if torrentStatus.SeedingDuration > 0 {
				fmt.Printf("   Seeding Time: %s\n", formatDuration(torrentStatus.SeedingDuration))
			}
			if torrentStatus.SeedingLimit > 0 {
				fmt.Printf("   Seeding Limit: %s\n", formatDuration(torrentStatus.SeedingLimit))
			}
			if torrentStatus.TimeRemaining > 0 {
				fmt.Printf("   Time Remaining: %s\n", formatDuration(torrentStatus.TimeRemaining))
			}

			// Status indicator
			if torrentStatus.AutoStopped {
				fmt.Printf("   Status: %s\n", cli.ColorSeeding.Sprint("✅ Seeding Complete (Auto-stopped)"))
			} else if torrentStatus.IsOverdue {
				fmt.Printf("   Status: %s\n", cli.ColorError.Sprint("⏰ Overdue"))
			} else {
				fmt.Printf("   Status: %s\n", cli.ColorDownloading.Sprint("🌱 Active Seeding"))
			}

			if torrentStatus.CurrentState != "" {
				fmt.Printf("   Current State: %s\n", torrentStatus.CurrentState)
			}

			fmt.Println()
		}
	} else if len(status.Details) > 0 {
		fmt.Printf("\n💡 Use '%s' to see detailed torrent information\n",
			cli.ColorDownloading.Sprint("akira seeding --detailed"))
	}

	// Summary message
	if status.TrackedTorrents == 0 {
		fmt.Printf("\n📭 No torrents are currently being tracked for seeding\n")
		fmt.Printf("💡 Add torrents with '%s' to start seeding tracking\n",
			cli.ColorDownloading.Sprint("akira add"))
	}

	return nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		days := int(d.Hours() / 24)
		hours := d.Hours() - float64(days*24)
		if hours < 1 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd %.1fh", days, hours)
	}
}
