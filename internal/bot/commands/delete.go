package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// HandleDeleteCommand handles the /delete Discord command
func HandleDeleteCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	// Get all available torrents
	ctx := context.Background()
	torrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	if len(torrents) == 0 {
		respondWithError(s, i, "No torrents available to delete")
		return
	}

	// Discord limits select menus to 25 options
	const maxOptions = 25
	totalTorrents := len(torrents)
	
	// Create select menu options for torrents (limited to 25)
	var options []discordgo.SelectMenuOption
	var torrentsToShow []qbittorrent.Torrent
	
	if totalTorrents <= maxOptions {
		// If we have 25 or fewer torrents, show them all
		torrentsToShow = torrents
	} else {
		// If we have more than 25, show the first 25 and add pagination info
		torrentsToShow = torrents[:maxOptions]
	}

	for i, torrent := range torrentsToShow {
		// Truncate name if too long for Discord
		name := torrent.Name
		if len(name) > 100 {
			name = name[:97] + "..."
		}

		// Create a unique value that includes hash and index
		value := fmt.Sprintf("%s|%d", torrent.Hash, i)

		// Create description with size and state
		description := fmt.Sprintf("%s | %s", formatBytes(int64(torrent.Size)), string(torrent.State))
		if len(description) > 100 {
			description = description[:97] + "..."
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       name,
			Value:       value,
			Description: description,
		})
	}

	// Create the select menu with proper limits
	selectMenu := discordgo.SelectMenu{
		CustomID:    "delete_torrent_select",
		Placeholder: "Select torrents to delete",
		MinValues:   &[]int{1}[0],
		MaxValues:   len(options), // This will now be ≤ 25
		Options:     options,
	}

	// Create the action row
	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{selectMenu},
	}

	// Create embed explaining the process
	var embedTitle string
	var embedDescription string
	
	if totalTorrents <= maxOptions {
		embedTitle = "🗑️ Delete Torrents"
		embedDescription = fmt.Sprintf("Select the torrents you want to delete from the list below.\n\n**Note:** This will permanently delete both the torrent and all downloaded files.\n\n**Available:** %d torrent(s)", totalTorrents)
	} else {
		embedTitle = "🗑️ Delete Torrents (Page 1)"
		embedDescription = fmt.Sprintf("Select the torrents you want to delete from the list below.\n\n**Note:** This will permanently delete both the torrent and all downloaded files.\n\n**Showing:** %d of %d torrent(s)\n*Only the first 25 torrents are shown due to Discord limits*", maxOptions, totalTorrents)
	}

	embed := createInfoEmbed(embedTitle, embedDescription)

	// Send initial response with select menu
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actionRow},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send delete response: %v\n", err)
	}
}

// HandleDeleteTorrentSelect handles the torrent selection from the select menu
func HandleDeleteTorrentSelect(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	// Parse the selected values
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		respondWithError(s, i, "No torrents selected")
		return
	}

	// Extract torrent hashes and names
	var selectedHashes []string
	var selectedNames []string

	for _, value := range data.Values {
		parts := strings.Split(value, "|")
		if len(parts) >= 2 {
			hash := parts[0]
			selectedHashes = append(selectedHashes, hash)
		}
	}

	if len(selectedHashes) == 0 {
		respondWithError(s, i, "Invalid torrent selection")
		return
	}

	// Get torrent details for confirmation
	ctx := context.Background()
	allTorrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrent details: %v", err))
		return
	}

	// Build confirmation message
	var content strings.Builder
	content.WriteString(fmt.Sprintf("**You are about to delete %d torrent(s):**\n\n", len(selectedHashes)))

	for _, hash := range selectedHashes {
		for _, torrent := range allTorrents {
			if torrent.Hash == hash {
				selectedNames = append(selectedNames, torrent.Name)
				content.WriteString(fmt.Sprintf("• **%s**\n", torrent.Name))
				content.WriteString(fmt.Sprintf("  Size: %s | State: %s\n", formatBytes(int64(torrent.Size)), string(torrent.State)))
				break
			}
		}
	}

	content.WriteString("\n⚠️ **This action will:**\n")
	content.WriteString("• Permanently delete the torrent from qBittorrent\n")
	content.WriteString("• Permanently delete all downloaded files\n")
	content.WriteString("• Stop tracking in seeding service\n\n")
	content.WriteString("**Are you sure you want to proceed?**")

	// Create confirmation buttons
	// Store only the hashes to stay within Discord's 100 character custom ID limit
	confirmButton := discordgo.Button{
		Label:    "✅ Yes, Delete Everything",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("delete_confirm|%s", strings.Join(selectedHashes, ",")),
	}

	cancelButton := discordgo.Button{
		Label:    "❌ Cancel",
		Style:    discordgo.SecondaryButton,
		CustomID: "delete_cancel",
	}

	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{confirmButton, cancelButton},
	}

	// Create confirmation embed
	embed := createWarningEmbed("🗑️ Confirm Deletion", content.String())

	// Respond to the component interaction with the confirmation
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actionRow},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send confirmation: %v\n", err)
	}
}

// HandleDeleteConfirm handles the final confirmation to delete torrents
func HandleDeleteConfirm(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	// Parse the selected torrents from custom ID
	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, "|")
	if len(parts) < 2 {
		respondWithError(s, i, "Invalid confirmation data")
		return
	}

	// Parse the comma-separated hashes
	selectedHashes := strings.Split(parts[1], ",")
	if len(selectedHashes) == 0 {
		respondWithError(s, i, "No torrents selected for deletion")
		return
	}

	// Get torrent names for the success message
	ctx := context.Background()
	allTorrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		// If we can't get torrent names, we'll still proceed with deletion
		// but use generic names in the response
		fmt.Printf("Warning: Failed to get torrent details for names: %v\n", err)
	}

	var torrentNames []string
	for _, hash := range selectedHashes {
		nameFound := false
		for _, torrent := range allTorrents {
			if torrent.Hash == hash {
				torrentNames = append(torrentNames, torrent.Name)
				nameFound = true
				break
			}
		}
		// If we couldn't find the name, use a generic one
		if !nameFound {
			torrentNames = append(torrentNames, fmt.Sprintf("Torrent (%s...)", hash[:8]))
		}
	}

	// Delete torrents (always delete files)
	err = torrentService.DeleteTorrents(ctx, selectedHashes, true)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to delete torrents: %v", err))
		return
	}

	// Stop tracking for seeding service
	if seedingService != nil {
		for _, hash := range selectedHashes {
			err = seedingService.StopTracking(hash)
			if err != nil {
				// Log error but don't fail the command
				fmt.Printf("Warning: Failed to stop tracking torrent %s: %v\n", hash, err)
			}
		}
	}

	// Create success response using the names we collected before deletion
	var content strings.Builder
	content.WriteString(fmt.Sprintf("✅ **Successfully Deleted %d Torrent(s)**\n\n", len(selectedHashes)))
	content.WriteString("🗑️ **Files were also deleted**\n\n")
	content.WriteString("**Deleted Torrents:**\n")

	for i, name := range torrentNames {
		if i >= 10 { // Limit to 10 names
			content.WriteString(fmt.Sprintf("... and %d more\n", len(torrentNames)-10))
			break
		}
		content.WriteString(fmt.Sprintf("• %s\n", name))
	}

	embed := createSuccessEmbed("🗑️ Torrents Deleted", content.String())

	// Respond to the component interaction with success and remove components
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send success response: %v\n", err)
	}
}

// HandleDeleteCancel handles cancellation of the delete operation
func HandleDeleteCancel(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	embed := createInfoEmbed("❌ Deletion Cancelled", "The torrent deletion operation has been cancelled. No torrents were deleted.")

	// Respond to the component interaction with cancellation and remove components
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send cancellation response: %v\n", err)
	}
}
