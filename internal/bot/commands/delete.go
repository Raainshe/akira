package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

const deletePageSize = 25

var validDeleteCategories = []string{"movies", "series", "anime", "all"}

func isValidDeleteCategory(category string) bool {
	for _, valid := range validDeleteCategories {
		if category == valid {
			return true
		}
	}
	return false
}

func fetchTorrentsForDeleteCategory(ctx context.Context, torrentService *core.TorrentService, category string) ([]qbittorrent.Torrent, error) {
	if category == "all" {
		return torrentService.GetTorrents(ctx, nil)
	}
	return torrentService.GetTorrents(ctx, &core.TorrentFilter{
		Category: category,
	})
}

func createDeletePaginationComponents(category string, currentPage, totalPages int) []discordgo.MessageComponent {
	if totalPages <= 1 {
		return nil
	}

	row := discordgo.ActionsRow{}

	if currentPage > 1 {
		row.Components = append(row.Components, discordgo.Button{
			Label:    "◀️ Previous",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("delete_page|%s|%d", category, currentPage-1),
		})
	}

	row.Components = append(row.Components, discordgo.Button{
		Label:    fmt.Sprintf("Page %d/%d", currentPage, totalPages),
		Style:    discordgo.SecondaryButton,
		CustomID: "delete_page_info",
		Disabled: true,
	})

	if currentPage < totalPages {
		row.Components = append(row.Components, discordgo.Button{
			Label:    "Next ▶️",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("delete_page|%s|%d", category, currentPage+1),
		})
	}

	return []discordgo.MessageComponent{row}
}

// updateDeleteInteractionMessage updates the message for a component interaction.
// Component interactions must use UpdateMessage, not InteractionResponseEdit.
func updateDeleteInteractionMessage(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func showDeleteTorrentPage(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, category string, page int) error {
	ctx := context.Background()

	torrents, err := fetchTorrentsForDeleteCategory(ctx, torrentService, category)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return err
	}

	if len(torrents) == 0 {
		respondWithError(s, i, fmt.Sprintf("No torrents found in category '%s'", category))
		return fmt.Errorf("no torrents in category %s", category)
	}

	totalTorrents := len(torrents)
	totalPages := (totalTorrents + deletePageSize - 1) / deletePageSize

	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * deletePageSize
	end := offset + deletePageSize
	if end > totalTorrents {
		end = totalTorrents
	}
	torrentsToShow := torrents[offset:end]

	var options []discordgo.SelectMenuOption
	for idx, torrent := range torrentsToShow {
		name := torrent.Name
		if len(name) > 100 {
			name = name[:97] + "..."
		}

		value := fmt.Sprintf("%s|%d", torrent.Hash, idx)

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

	minValues := 1
	selectMenu := discordgo.SelectMenu{
		CustomID:    "delete_torrent_select",
		Placeholder: "Select torrents to delete",
		MinValues:   &minValues,
		MaxValues:   len(options),
		Options:     options,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
	}
	if pagination := createDeletePaginationComponents(category, page, totalPages); pagination != nil {
		components = append(components, pagination...)
	}

	categoryTitle := strings.Title(category)
	var embedTitle string
	var embedDescription string

	if totalPages > 1 {
		embedTitle = fmt.Sprintf("🗑️ Delete Torrents - %s (Page %d/%d)", categoryTitle, page, totalPages)
		embedDescription = fmt.Sprintf(
			"Select the torrents you want to delete from the **%s** category.\n\n**Note:** This will permanently delete both the torrent and all downloaded files.\n\n**Showing:** %d-%d of %d torrent(s)\nUse the buttons below to navigate pages.",
			categoryTitle, offset+1, end, totalTorrents,
		)
	} else {
		embedTitle = fmt.Sprintf("🗑️ Delete Torrents - %s", categoryTitle)
		embedDescription = fmt.Sprintf(
			"Select the torrents you want to delete from the **%s** category.\n\n**Note:** This will permanently delete both the torrent and all downloaded files.\n\n**Available:** %d torrent(s)",
			categoryTitle, totalTorrents,
		)
	}

	embed := createInfoEmbed(embedTitle, embedDescription)

	if err := updateDeleteInteractionMessage(s, i, embed, components); err != nil {
		fmt.Printf("Failed to update delete response: %v\n", err)
		return err
	}

	return nil
}

// HandleDeleteCommand handles the /delete Discord command
func HandleDeleteCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	// Show category selection first
	showCategorySelection(s, i)
}

// showCategorySelection shows the category selection menu
func showCategorySelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Create category selection menu
	selectMenu := discordgo.SelectMenu{
		CustomID:    "delete_category_select",
		Placeholder: "Select a category to delete torrents from",
		Options: []discordgo.SelectMenuOption{
			{
				Label:       "🎬 Movies",
				Value:       "movies",
				Description: "Delete torrents from the movies category",
			},
			{
				Label:       "📺 Series",
				Value:       "series",
				Description: "Delete torrents from the series category",
			},
			{
				Label:       "🌸 Anime",
				Value:       "anime",
				Description: "Delete torrents from the anime category",
			},
			{
				Label:       "🌐 All Categories",
				Value:       "all",
				Description: "Delete torrents from all categories",
			},
		},
	}

	// Create the action row
	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{selectMenu},
	}

	// Create embed explaining the process
	embed := createInfoEmbed(
		"🗑️ Delete Torrents - Category Selection",
		"First, select which category of torrents you want to delete from.\n\n**Available Categories:**\n• 🎬 **Movies** - Movie torrents\n• 📺 **Series** - TV series torrents\n• 🌸 **Anime** - Anime torrents\n• 🌐 **All Categories** - All torrents\n\nAfter selecting a category, you'll see a list of torrents to choose from.",
	)

	// Send initial response with category selection
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actionRow},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send category selection response: %v\n", err)
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
			hash := strings.TrimSpace(parts[0])
			if hash != "" {
				selectedHashes = append(selectedHashes, hash)
			}
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

	// Create confirmation buttons
	// Discord has a 100 character limit for custom IDs
	// Limit the number of hashes to ensure we stay within the limit
	// "delete_confirm|" = 15 chars, leaving 85 chars for hashes
	const maxCustomIDLength = 100
	const prefixLength = 15 // "delete_confirm|"
	const maxHashLength = maxCustomIDLength - prefixLength

	// Build custom ID, limiting hashes if needed
	customIDHashes := selectedHashes
	customIDValue := strings.Join(selectedHashes, ",")
	truncated := false
	if len(customIDValue) > maxHashLength {
		// Truncate to fit within limit
		// Try to include as many hashes as possible
		var truncatedHashes []string
		currentLength := 0
		for _, hash := range selectedHashes {
			// Estimate: hash length + comma
			estimatedLength := len(hash) + 1
			if currentLength+estimatedLength > maxHashLength {
				break
			}
			truncatedHashes = append(truncatedHashes, hash)
			currentLength += estimatedLength
		}
		customIDHashes = truncatedHashes
		customIDValue = strings.Join(truncatedHashes, ",")
		truncated = len(selectedHashes) > len(truncatedHashes)
	}

	// Build confirmation message - only show torrents that will actually be deleted
	var content strings.Builder
	torrentsToDelete := customIDHashes // Use the hashes that will actually be deleted
	content.WriteString(fmt.Sprintf("**You are about to delete %d torrent(s):**\n\n", len(torrentsToDelete)))

	if truncated {
		content.WriteString(fmt.Sprintf("⚠️ **Note:** %d torrent(s) selected, but only %d will be deleted due to Discord limits.\n\n", len(selectedHashes), len(torrentsToDelete)))
	}

	for _, hash := range torrentsToDelete {
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

	confirmButton := discordgo.Button{
		Label:    "✅ Yes, Delete Everything",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("delete_confirm|%s", customIDValue),
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
		Type: discordgo.InteractionResponseUpdateMessage,
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

	// Get torrent names for the success message BEFORE deletion
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
			hashPrefix := hash
			if len(hash) >= 8 {
				hashPrefix = hash[:8]
			}
			torrentNames = append(torrentNames, fmt.Sprintf("Torrent (%s...)", hashPrefix))
		}
	}

	// Acknowledge interaction immediately with deferred response to prevent token expiration
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		// If deferred response fails, try to send error and return
		respondWithError(s, i, fmt.Sprintf("Failed to acknowledge deletion: %v", err))
		return
	}

	// Now perform the deletion (interaction is already acknowledged)
	err = torrentService.DeleteTorrents(ctx, selectedHashes, true)
	if err != nil {
		// Try to update the deferred response with error
		_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{createErrorEmbed("❌ Deletion Failed", fmt.Sprintf("Failed to delete torrents: %v", err))},
		})
		if editErr != nil {
			// If edit fails, send follow-up message as last resort
			errorEmbed := createErrorEmbed("❌ Deletion Failed", fmt.Sprintf("Failed to delete torrents: %v", err))
			_, _ = s.ChannelMessageSendEmbed(i.ChannelID, errorEmbed)
		}
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

	// Update the deferred response with success
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{},
	})

	if err != nil {
		// If edit fails (interaction expired), send follow-up message as fallback
		fmt.Printf("Failed to update deletion response, sending follow-up: %v\n", err)
		_, followErr := s.ChannelMessageSendEmbed(i.ChannelID, embed)
		if followErr != nil {
			fmt.Printf("Failed to send follow-up message: %v\n", followErr)
		}
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

// HandleDeleteCategorySelect handles the category selection from the delete command
func HandleDeleteCategorySelect(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		respondWithError(s, i, "No category selected")
		return
	}

	selectedCategory := data.Values[0]
	if !isValidDeleteCategory(selectedCategory) {
		respondWithError(s, i, fmt.Sprintf("Invalid category '%s'. Valid categories: %v", selectedCategory, validDeleteCategories))
		return
	}

	showDeleteTorrentPage(s, i, torrentService, selectedCategory, 1)
}

// HandleDeletePagination handles page navigation for the delete torrent selection menu
func HandleDeletePagination(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "|")
	if len(parts) != 3 || parts[0] != "delete_page" {
		respondWithError(s, i, "Invalid pagination data")
		return
	}

	category := parts[1]
	if !isValidDeleteCategory(category) {
		respondWithError(s, i, fmt.Sprintf("Invalid category '%s'", category))
		return
	}

	page, err := strconv.Atoi(parts[2])
	if err != nil || page < 1 {
		respondWithError(s, i, "Invalid page number")
		return
	}

	showDeleteTorrentPage(s, i, torrentService, category, page)
}
