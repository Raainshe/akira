package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// HandleTorrentsCommand handles the /torrents Discord command
func HandleTorrentsCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	// Get command options
	data := i.ApplicationCommandData()

	var filter string

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "filter":
			filter = option.StringValue()
		}
	}

	// Create filter (no limit - get all matching torrents)
	torrentFilter := &core.TorrentFilter{}

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

	// Get all matching torrents (no limit)
	ctx := context.Background()
	torrents, err := torrentService.GetTorrents(ctx, torrentFilter)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	if len(torrents) == 0 {
		embed := createInfoEmbed("📋 Torrent List", "No torrents found.")
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
		if err != nil {
			fmt.Printf("Failed to send torrents response: %v\n", err)
		}
		return
	}

	// Format all torrents
	content := formatTorrentListAll(torrents)

	// Discord embed description limit is 4096 characters
	// Split content into chunks if needed (leave buffer for formatting)
	const maxEmbedLength = 4000
	chunks := splitContent(content, maxEmbedLength)

	// Determine filter title
	filterTitle := "All Torrents"
	if filter != "" {
		filterTitle = fmt.Sprintf("%s Torrents", strings.Title(filter))
	}

	// Send initial response with first chunk
	firstChunk := chunks[0]
	if len(chunks) > 1 {
		firstChunk = fmt.Sprintf("%s\n\n*Part 1/%d*", firstChunk, len(chunks))
	}

	embed := createInfoEmbed(fmt.Sprintf("📋 %s", filterTitle), firstChunk)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send torrents response: %v\n", err)
		return
	}

	// Send follow-up messages for remaining chunks
	for idx, chunk := range chunks[1:] {
		partNum := idx + 2
		chunkWithHeader := fmt.Sprintf("%s\n\n*Part %d/%d*", chunk, partNum, len(chunks))
		followUpEmbed := createInfoEmbed(fmt.Sprintf("📋 %s (continued)", filterTitle), chunkWithHeader)

		_, followErr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{followUpEmbed},
		})
		if followErr != nil {
			fmt.Printf("Failed to send follow-up message %d: %v\n", partNum, followErr)
			// Try channel message as fallback
			_, _ = s.ChannelMessageSendEmbed(i.ChannelID, followUpEmbed)
		}
	}
}
