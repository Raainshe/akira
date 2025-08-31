package tui

import (
	"context"
	"fmt"

	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// Run starts the Bubbletea TUI application
func Run(ctx context.Context, cfg *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) error {

	// TODO: Implement full Bubbletea TUI
	fmt.Println("🌟 Welcome to Akira TUI!")
	fmt.Println("📋 This is a placeholder - full TUI implementation coming next!")
	fmt.Println("💡 For now, use the CLI commands: akira list, akira add, etc.")

	return nil
}
