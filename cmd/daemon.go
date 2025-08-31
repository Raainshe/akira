package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/raainshe/akira/internal/bot"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
	"github.com/spf13/cobra"
)

const (
	pidFile = "akira.pid"
)

// NewDaemonCommand creates the daemon command
func NewDaemonCommand(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) *cobra.Command {

	var daemonConfig struct {
		foreground bool
		pidFile    string
	}

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start the Discord bot daemon",
		Long: `Start the Discord bot daemon that runs in the background.
		
The daemon will:
- Start the Discord bot with slash commands
- Run the seeding service in the background
- Handle graceful shutdown on SIGINT/SIGTERM
- Create a PID file for process management`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon(ctx, cfg, torrentService, diskService, seedingService, qbClient, daemonConfig)
		},
	}

	cmd.Flags().BoolVarP(&daemonConfig.foreground, "foreground", "f", false, "Run in foreground (don't daemonize)")
	cmd.Flags().StringVarP(&daemonConfig.pidFile, "pid-file", "p", pidFile, "PID file location")

	return cmd
}

// NewStatusCommand creates the status command
func NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		Long:  "Check if the Akira daemon is running and show its status",
		RunE:  runStatus,
	}
}

// NewStopCommand creates the stop command
func NewStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		Long:  "Stop the running Akira daemon gracefully",
		RunE:  runStop,
	}
}

// NewRestartCommand creates the restart command
func NewRestartCommand(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) *cobra.Command {

	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the daemon",
		Long:  "Stop the running daemon and start it again",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestart(ctx, cfg, torrentService, diskService, seedingService, qbClient)
		},
	}
}

// displayAkiraBanner displays the AKIRA ASCII art banner with proper alignment
func displayAkiraBanner() {
	fmt.Println(`
    ╔══════════════════════════════════════════════════════════════╗
    ║                                                              ║
    ║     █████  ██  ██ ██ ██  ██ ██████  █████                 ║
    ║    ██   ██ ██ ██  ██ ██ ██ ██   ██ ██   ██                ║
    ║    ███████ █████   ██ █████ ██████  ███████                ║
    ║    ██   ██ ██ ██  ██ ██ ██ ██   ██ ██   ██                ║
    ║    ██   ██ ██  ██ ██ ██  ██ ██   ██ ██   ██                ║
    ║                                                              ║
    ║           🌟 Torrent Management Discord Bot 🌟               ║
    ║                                                              ║
    ║     Discord Bot Daemon Starting...                           ║
    ║     PID: ` + fmt.Sprintf("%-6d", os.Getpid()) + `                                    ║
    ║     Time: ` + time.Now().Format("2006-01-02 15:04:05") + `                    ║
    ║                                                              ║
    ╚══════════════════════════════════════════════════════════════╝
`)
}

func runDaemon(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client,
	daemonConfig struct {
		foreground bool
		pidFile    string
	}) error {

	// Check if daemon is already running
	if isDaemonRunning(daemonConfig.pidFile) {
		return fmt.Errorf("daemon is already running (PID file exists: %s)", daemonConfig.pidFile)
	}

	// Create logger
	logger := logging.GetLogger()

	// Create Discord bot
	discordBot, err := bot.NewBot(cfg, torrentService, diskService, seedingService, qbClient)
	if err != nil {
		return fmt.Errorf("failed to create Discord bot: %w", err)
	}

	// Create context for graceful shutdown
	daemonCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create PID file
	if err := createPIDFile(daemonConfig.pidFile); err != nil {
		return fmt.Errorf("failed to create PID file: %w", err)
	}
	defer removePIDFile(daemonConfig.pidFile)

	// Display AKIRA ASCII art banner
	displayAkiraBanner()

	logger.Info("Starting Akira daemon", map[string]interface{}{
		"pid_file":   daemonConfig.pidFile,
		"foreground": daemonConfig.foreground,
	})

	// Start Discord bot
	if err := discordBot.Start(); err != nil {
		return fmt.Errorf("failed to start Discord bot: %w", err)
	}

	// Ensure seeding service is stopped before starting
	if seedingService.IsRunning() {
		logger.Info("Stopping existing seeding service")
		if err := seedingService.Stop(); err != nil {
			logger.Error("Error stopping existing seeding service", map[string]interface{}{
				"error": err.Error(),
			})
		}
		// Give it a moment to stop
		time.Sleep(100 * time.Millisecond)
	}

	// Start seeding service in background
	go func() {
		logger.Info("Starting seeding service")
		if err := seedingService.Start(daemonCtx); err != nil {
			logger.Error("Seeding service error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Daemon started successfully", map[string]interface{}{
		"discord_guilds": len(cfg.Discord.GuildIDs),
		"pid":            os.Getpid(),
	})

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", map[string]interface{}{
			"signal": sig.String(),
		})
	case <-daemonCtx.Done():
		logger.Info("Received shutdown context")
	}

	// Graceful shutdown
	logger.Info("Shutting down daemon...")

	// Stop Discord bot
	if err := discordBot.Stop(); err != nil {
		logger.Error("Error stopping Discord bot", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Cancel context to stop seeding service
	cancel()

	logger.Info("Daemon stopped successfully")
	return nil
}

// isDaemonRunning checks if the daemon is already running
func isDaemonRunning(pidFile string) bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is running
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// createPIDFile creates a PID file with the current process ID
func createPIDFile(pidFile string) error {
	pid := os.Getpid()
	data := fmt.Sprintf("%d\n", pid)
	return os.WriteFile(pidFile, []byte(data), 0644)
}

// removePIDFile removes the PID file
func removePIDFile(pidFile string) {
	os.Remove(pidFile)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if isDaemonRunning(pidFile) {
		data, _ := os.ReadFile(pidFile)
		pid := strings.TrimSpace(string(data))
		fmt.Printf("✅ Daemon is running (PID: %s)\n", pid)
		return nil
	}

	fmt.Println("❌ Daemon is not running")
	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	if !isDaemonRunning(pidFile) {
		return fmt.Errorf("daemon is not running")
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	fmt.Printf("🔄 Sent SIGTERM to daemon (PID: %d)\n", pid)
	fmt.Println("Waiting for graceful shutdown...")

	// Wait for process to exit (up to 10 seconds)
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			// Process has exited
			removePIDFile(pidFile)
			fmt.Println("✅ Daemon stopped successfully")
			return nil
		}
	}

	// Force kill if still running
	fmt.Println("⚠️  Daemon not responding, sending SIGKILL...")
	err = process.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("failed to send SIGKILL: %w", err)
	}

	removePIDFile(pidFile)
	fmt.Println("✅ Daemon force stopped")
	return nil
}

func runRestart(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) error {

	fmt.Println("🔄 Restarting daemon...")

	// Stop daemon if running
	if isDaemonRunning(pidFile) {
		// Create a temporary command to call runStop
		tempCmd := &cobra.Command{}
		if err := runStop(tempCmd, []string{}); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		// Wait a moment for cleanup
		time.Sleep(2 * time.Second)
	}

	// Start daemon
	daemonConfig := struct {
		foreground bool
		pidFile    string
	}{
		foreground: false,
		pidFile:    pidFile,
	}

	return runDaemon(ctx, cfg, torrentService, diskService, seedingService, qbClient, daemonConfig)
}
