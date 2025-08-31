package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// HandleTorrentsCommand handles the /torrents Discord command
func HandleTorrentsCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	// Get command options
	data := i.ApplicationCommandData()

	var filter string
	var page int = 1

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "filter":
			filter = option.StringValue()
		case "page":
			page = int(option.IntValue())
		}
	}

	// Validate page
	if page < 1 {
		page = 1
	}

	// Create filter
	torrentFilter := &core.TorrentFilter{
		Limit: 10, // Show 10 torrents per page for Discord
	}

	// Apply filter based on option
	switch filter {
	case "downloading":
		torrentFilter.States = []qbittorrent.TorrentState{
			qbittorrent.StateDownloading,
			qbittorrent.StateMetaDL,
			qbittorrent.StateForcedDL,
		}
	case "seeding":
		torrentFilter.States = []qbittorrent.TorrentState{
			qbittorrent.StateUploading,
			qbittorrent.StateForcedUP,
		}
	case "paused":
		torrentFilter.States = []qbittorrent.TorrentState{
			qbittorrent.StatePausedDL,
			qbittorrent.StatePausedUP,
		}
	}

	// Get torrents
	ctx := context.Background()
	torrents, err := torrentService.GetTorrents(ctx, torrentFilter)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	// Calculate pagination (simplified for now)
	totalPages := 1
	if len(torrents) > 0 {
		totalPages = (len(torrents) + torrentFilter.Limit - 1) / torrentFilter.Limit
	}

	// Format response
	content := formatTorrentList(torrents, page, totalPages)

	// Create embed
	embed := createInfoEmbed("📋 Torrent List", content)

	// Add pagination components if needed
	var components []discordgo.MessageComponent
	if totalPages > 1 {
		components = createPaginationComponents(page, totalPages)
	}

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})

	if err != nil {
		fmt.Printf("Failed to send torrents response: %v\n", err)
	}
}

// HandleTorrentsPagination handles pagination for torrents command
func HandleTorrentsPagination(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	// Extract page number from custom ID
	customID := i.MessageComponentData().CustomID
	pageStr := customID[5:] // Remove "page_" prefix
	_, err := strconv.Atoi(pageStr)
	if err != nil {
		respondWithError(s, i, "Invalid page number")
		return
	}

	// Re-run torrents command with new page
	HandleTorrentsCommand(s, i, torrentService)
}
