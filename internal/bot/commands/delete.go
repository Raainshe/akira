package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
)

// HandleDeleteCommand handles the /delete Discord command
func HandleDeleteCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	// Get command options
	data := i.ApplicationCommandData()

	var query string
	var deleteFiles bool = false

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "query":
			query = option.StringValue()
		case "delete_files":
			deleteFiles = option.BoolValue()
		}
	}

	// Validate query
	if query == "" {
		respondWithError(s, i, "Query is required (torrent name or hash)")
		return
	}

	// Get torrents to find matches
	ctx := context.Background()
	torrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	// Find matching torrents
	var matchingTorrents []string
	var torrentNames []string

	for _, torrent := range torrents {
		// Check if query matches hash (case insensitive)
		if strings.EqualFold(torrent.Hash, query) {
			matchingTorrents = append(matchingTorrents, torrent.Hash)
			torrentNames = append(torrentNames, torrent.Name)
		} else if strings.Contains(strings.ToLower(torrent.Name), strings.ToLower(query)) {
			// Check if query matches name (case insensitive partial match)
			matchingTorrents = append(matchingTorrents, torrent.Hash)
			torrentNames = append(torrentNames, torrent.Name)
		}
	}

	if len(matchingTorrents) == 0 {
		respondWithError(s, i, fmt.Sprintf("No torrents found matching '%s'", query))
		return
	}

	// Delete torrents
	err = torrentService.DeleteTorrents(ctx, matchingTorrents, deleteFiles)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to delete torrents: %v", err))
		return
	}

	// Stop tracking for seeding service
	if seedingService != nil {
		for _, hash := range matchingTorrents {
			err = seedingService.StopTracking(hash)
			if err != nil {
				// Log error but don't fail the command
				fmt.Printf("Warning: Failed to stop tracking torrent %s: %v\n", hash, err)
			}
		}
	}

	// Create success response
	var content strings.Builder
	content.WriteString(fmt.Sprintf("✅ **Deleted %d Torrent(s)**\n\n", len(matchingTorrents)))

	if deleteFiles {
		content.WriteString("🗑️ **Files were also deleted**\n\n")
	} else {
		content.WriteString("📁 **Files were preserved**\n\n")
	}

	content.WriteString("**Deleted Torrents:**\n")
	for i, name := range torrentNames {
		if i >= 10 { // Limit to 10 names
			content.WriteString(fmt.Sprintf("... and %d more\n", len(torrentNames)-10))
			break
		}
		content.WriteString(fmt.Sprintf("• %s\n", name))
	}

	embed := createSuccessEmbed("🗑️ Torrents Deleted", content.String())

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send delete response: %v\n", err)
	}
}
