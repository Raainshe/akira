package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// Run starts the Bubbletea TUI application
func Run(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) error {

	// Create the main TUI model
	model := NewAppModel(ctx, cfg, torrentService, diskService, seedingService, qbClient)

	// Create the Bubbletea program
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	_, err := program.Run()
	return err
}
