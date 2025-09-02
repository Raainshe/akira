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
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		respondWithError(s, i, "Invalid page number")
		return
	}

	// Create filter (assume no filter for now)
	torrentFilter := &core.TorrentFilter{
		Limit: 10, // Show 10 torrents per page for Discord
	}

	// Get all torrents first to calculate total pages
	ctx := context.Background()
	allTorrents, err := torrentService.GetTorrents(ctx, torrentFilter)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	// Calculate total pages based on all matching torrents
	totalPages := 1
	if len(allTorrents) > 0 {
		totalPages = (len(allTorrents) + torrentFilter.Limit - 1) / torrentFilter.Limit
	}

	// Ensure page is within valid range
	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	// Calculate offset for pagination
	offset := (page - 1) * torrentFilter.Limit

	// Get torrents for the current page
	var pageTorrents []qbittorrent.Torrent
	if offset < len(allTorrents) {
		end := offset + torrentFilter.Limit
		if end > len(allTorrents) {
			end = len(allTorrents)
		}
		pageTorrents = allTorrents[offset:end]
	}

	// Format response
	content := formatTorrentList(pageTorrents, page, totalPages)

	// Create embed
	embed := createInfoEmbed("📋 Torrent List", content)

	// Add pagination components if needed
	var components []discordgo.MessageComponent
	if totalPages > 1 {
		components = createPaginationComponents(page, totalPages)
	}

	// Update the message instead of creating a new response
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		fmt.Printf("Failed to update torrents response: %v\n", err)
		// Fallback to responding with error
		respondWithError(s, i, fmt.Sprintf("Failed to update page: %v", err))
	}
}
