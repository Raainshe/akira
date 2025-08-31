package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
)

// HandleSeedingStatusCommand handles the /seeding-status Discord command
func HandleSeedingStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate, seedingService *core.SeedingService) {
	ctx := context.Background()

	// Get seeding status
	status, err := seedingService.GetSeedingStatus(ctx)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get seeding status: %v", err))
		return
	}

	// Format response
	content := formatSeedingStatus(*status)

	// Create embed
	embed := createInfoEmbed("🌱 Seeding Status", content)

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send seeding status response: %v\n", err)
	}
}

// HandleStopSeedingCommand handles the /stop-seeding Discord command
func HandleStopSeedingCommand(s *discordgo.Session, i *discordgo.InteractionCreate, seedingService *core.SeedingService) {
	// Get command options
	data := i.ApplicationCommandData()

	var torrentQuery string

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "torrent":
			torrentQuery = option.StringValue()
		}
	}

	// Validate query
	if torrentQuery == "" {
		respondWithError(s, i, "Torrent name or hash is required")
		return
	}

	// Get seeding status to find the torrent
	ctx := context.Background()
	status, err := seedingService.GetSeedingStatus(ctx)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get seeding status: %v", err))
		return
	}

	// Find matching torrent
	var matchingHash string
	var matchingName string

	for hash, torrent := range status.Details {
		// Check if query matches hash (case insensitive)
		if strings.EqualFold(hash, torrentQuery) {
			matchingHash = hash
			matchingName = torrent.Name
			break
		} else if strings.Contains(strings.ToLower(torrent.Name), strings.ToLower(torrentQuery)) {
			// Check if query matches name (case insensitive partial match)
			matchingHash = hash
			matchingName = torrent.Name
			break
		}
	}

	if matchingHash == "" {
		respondWithError(s, i, fmt.Sprintf("No tracked torrent found matching '%s'", torrentQuery))
		return
	}

	// Stop tracking the torrent
	err = seedingService.StopTracking(matchingHash)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to stop tracking torrent: %v", err))
		return
	}

	// Create success response
	content := fmt.Sprintf("✅ **Stopped Tracking Torrent**\n\n"+
		"**Name:** %s\n"+
		"**Hash:** `%s`\n\n"+
		"The torrent is no longer being tracked for seeding management.",
		matchingName,
		matchingHash[:8])

	embed := createSuccessEmbed("🛑 Seeding Stopped", content)

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send stop seeding response: %v\n", err)
	}
}
