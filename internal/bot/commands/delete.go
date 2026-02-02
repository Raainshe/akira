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

	// Parse the comma-separated hashes and trim whitespace
	hashStrings := strings.Split(parts[1], ",")
	var selectedHashes []string
	for _, hashStr := range hashStrings {
		hash := strings.TrimSpace(hashStr)
		if hash != "" {
			selectedHashes = append(selectedHashes, hash)
		}
	}

	if len(selectedHashes) == 0 {
		respondWithError(s, i, "No torrents selected for deletion")
		return
	}

	// Get torrent names and states for the success message BEFORE deletion
	ctx := context.Background()
	allTorrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		// If we can't get torrent details for names, we'll still proceed with deletion
		// but use generic names in the response
		fmt.Printf("Warning: Failed to get torrent details for names: %v\n", err)
	}

	type torrentInfo struct {
		name  string
		state string
		hash  string
	}
	torrentInfoMap := make(map[string]*torrentInfo)

	for _, hash := range selectedHashes {
		nameFound := false
		for _, torrent := range allTorrents {
			// Case-insensitive hash comparison
			if strings.EqualFold(torrent.Hash, hash) {
				torrentInfoMap[hash] = &torrentInfo{
					name:  torrent.Name,
					state: string(torrent.State),
					hash:  torrent.Hash, // Use the actual hash from qBittorrent
				}
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
			torrentInfoMap[hash] = &torrentInfo{
				name:  fmt.Sprintf("Torrent (%s...)", hashPrefix),
				state: "unknown",
				hash:  hash,
			}
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

	// Normalize hashes to use the actual hash from qBittorrent (case-sensitive)
	// Also create a reverse map from normalized hash to original hash for lookups
	normalizedHashes := make([]string, 0, len(selectedHashes))
	normalizedToOriginal := make(map[string]string)
	for _, originalHash := range selectedHashes {
		if info, exists := torrentInfoMap[originalHash]; exists {
			normalizedHash := info.hash
			normalizedHashes = append(normalizedHashes, normalizedHash)
			normalizedToOriginal[normalizedHash] = originalHash
		} else {
			normalizedHashes = append(normalizedHashes, originalHash)
			normalizedToOriginal[originalHash] = originalHash
		}
	}

	// Pause torrents before deletion to ensure they can be deleted (some states may prevent deletion)
	// Pause individually so one failure doesn't stop others
	// Only pause torrents that we've verified exist in torrentInfoMap
	var pausedCount int
	var pauseFailedHashes []string
	for _, normalizedHash := range normalizedHashes {
		originalHash := normalizedToOriginal[normalizedHash]
		if info, exists := torrentInfoMap[originalHash]; exists {
			// Pause if torrent is in an active state (downloading, seeding, etc.)
			state := strings.ToLower(info.state)
			if state != "pauseddl" && state != "pausedup" && state != "error" {
				// Pause individually to avoid batch failures
				pauseErr := torrentService.PauseTorrents(ctx, []string{normalizedHash})
				if pauseErr != nil {
					fmt.Printf("Warning: Failed to pause torrent %s (%s) before deletion: %v\n", normalizedHash[:8], info.name, pauseErr)
					pauseFailedHashes = append(pauseFailedHashes, normalizedHash)
				} else {
					pausedCount++
				}
			}
		}
	}

	if pausedCount > 0 {
		fmt.Printf("Successfully paused %d torrent(s) before deletion\n", pausedCount)
	}
	if len(pauseFailedHashes) > 0 {
		fmt.Printf("Warning: Failed to pause %d torrent(s) before deletion, will attempt deletion anyway\n", len(pauseFailedHashes))
	}

	// Now perform the deletion (interaction is already acknowledged)
	err = torrentService.DeleteTorrents(ctx, normalizedHashes, true)
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

	// Verify which torrents were actually deleted by checking if they still exist
	remainingTorrents, verifyErr := torrentService.GetTorrents(ctx, nil)
	if verifyErr != nil {
		fmt.Printf("Warning: Failed to verify deletion: %v\n", verifyErr)
	}

	// Create a map of remaining torrent hashes for quick lookup
	remainingHashes := make(map[string]bool)
	if remainingTorrents != nil {
		for _, torrent := range remainingTorrents {
			remainingHashes[strings.ToLower(torrent.Hash)] = true
		}
	}

	// Determine which torrents were successfully deleted and which failed
	var deletedHashes []string
	var failedHashes []string
	var deletedNames []string
	var failedNames []string

	for _, normalizedHash := range normalizedHashes {
		hashLower := strings.ToLower(normalizedHash)
		originalHash := normalizedToOriginal[normalizedHash]
		if remainingHashes[hashLower] {
			// Torrent still exists - deletion failed
			failedHashes = append(failedHashes, normalizedHash)
			if info, exists := torrentInfoMap[originalHash]; exists {
				failedNames = append(failedNames, info.name)
			} else {
				hashPrefix := normalizedHash
				if len(normalizedHash) >= 8 {
					hashPrefix = normalizedHash[:8]
				}
				failedNames = append(failedNames, fmt.Sprintf("Torrent (%s...)", hashPrefix))
			}
		} else {
			// Torrent no longer exists - deletion succeeded
			deletedHashes = append(deletedHashes, normalizedHash)
			if info, exists := torrentInfoMap[originalHash]; exists {
				deletedNames = append(deletedNames, info.name)
			} else {
				hashPrefix := normalizedHash
				if len(normalizedHash) >= 8 {
					hashPrefix = normalizedHash[:8]
				}
				deletedNames = append(deletedNames, fmt.Sprintf("Torrent (%s...)", hashPrefix))
			}
		}
	}

	// Stop tracking for seeding service (only for successfully deleted torrents)
	if seedingService != nil {
		for _, hash := range deletedHashes {
			err = seedingService.StopTracking(hash)
			if err != nil {
				// Log error but don't fail the command
				fmt.Printf("Warning: Failed to stop tracking torrent %s: %v\n", hash, err)
			}
		}
	}

	// Create response based on results
	var content strings.Builder
	if len(deletedHashes) > 0 && len(failedHashes) == 0 {
		// All succeeded
		content.WriteString(fmt.Sprintf("✅ **Successfully Deleted %d Torrent(s)**\n\n", len(deletedHashes)))
		content.WriteString("🗑️ **Files were also deleted**\n\n")
		content.WriteString("**Deleted Torrents:**\n")

		for i, name := range deletedNames {
			if i >= 10 { // Limit to 10 names
				content.WriteString(fmt.Sprintf("... and %d more\n", len(deletedNames)-10))
				break
			}
			content.WriteString(fmt.Sprintf("• %s\n", name))
		}
	} else if len(deletedHashes) > 0 && len(failedHashes) > 0 {
		// Partial success
		content.WriteString(fmt.Sprintf("⚠️ **Partial Deletion: %d succeeded, %d failed**\n\n", len(deletedHashes), len(failedHashes)))
		content.WriteString("✅ **Successfully Deleted:**\n")
		for i, name := range deletedNames {
			if i >= 5 {
				content.WriteString(fmt.Sprintf("... and %d more\n", len(deletedNames)-5))
				break
			}
			content.WriteString(fmt.Sprintf("• %s\n", name))
		}
		content.WriteString("\n❌ **Failed to Delete:**\n")
		for i, name := range failedNames {
			if i >= 5 {
				content.WriteString(fmt.Sprintf("... and %d more\n", len(failedNames)-5))
				break
			}
			content.WriteString(fmt.Sprintf("• %s\n", name))
		}
		content.WriteString("\n*Some torrents may still be active or in a state that prevents deletion.*")
	} else {
		// All failed
		content.WriteString(fmt.Sprintf("❌ **Failed to Delete %d Torrent(s)**\n\n", len(failedHashes)))
		content.WriteString("**Failed Torrents:**\n")
		for i, name := range failedNames {
			if i >= 10 {
				content.WriteString(fmt.Sprintf("... and %d more\n", len(failedNames)-10))
				break
			}
			content.WriteString(fmt.Sprintf("• %s\n", name))
		}
		content.WriteString("\n*Torrents may be in a state that prevents deletion. Try pausing them first.*")
	}

	var embed *discordgo.MessageEmbed
	if len(failedHashes) == 0 {
		embed = createSuccessEmbed("🗑️ Torrents Deleted", content.String())
	} else if len(deletedHashes) > 0 {
		embed = createWarningEmbed("⚠️ Partial Deletion", content.String())
	} else {
		embed = createErrorEmbed("❌ Deletion Failed", content.String())
	}

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
	// Parse the selected category
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		respondWithError(s, i, "No category selected")
		return
	}

	selectedCategory := data.Values[0]

	// Validate category
	validCategories := []string{"movies", "series", "anime", "all"}
	isValid := false
	for _, valid := range validCategories {
		if selectedCategory == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		respondWithError(s, i, fmt.Sprintf("Invalid category '%s'. Valid categories: %v", selectedCategory, validCategories))
		return
	}

	// Get torrents based on selected category
	ctx := context.Background()
	var torrents []qbittorrent.Torrent
	var err error

	if selectedCategory == "all" {
		// Get all torrents
		torrents, err = torrentService.GetTorrents(ctx, nil)
	} else {
		// Get torrents by category
		filter := &core.TorrentFilter{
			Category: selectedCategory,
		}
		torrents, err = torrentService.GetTorrents(ctx, filter)
	}

	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	if len(torrents) == 0 {
		respondWithError(s, i, fmt.Sprintf("No torrents found in category '%s'", selectedCategory))
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
	// Limit to 2 torrents maximum due to Discord custom ID character limit (100 chars)
	const maxSelection = 2
	maxSelectable := len(options)
	if maxSelectable > maxSelection {
		maxSelectable = maxSelection
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    "delete_torrent_select",
		Placeholder: "Select torrents to delete (max 2)",
		MinValues:   &[]int{1}[0],
		MaxValues:   maxSelectable,
		Options:     options,
	}

	// Create the action row
	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{selectMenu},
	}

	// Create embed explaining the process
	var embedTitle string
	var embedDescription string

	const maxSelectionLimit = 2
	if totalTorrents <= maxOptions {
		embedTitle = fmt.Sprintf("🗑️ Delete Torrents - %s", strings.Title(selectedCategory))
		embedDescription = fmt.Sprintf("Select the torrents you want to delete from the **%s** category.\n\n**Note:** This will permanently delete both the torrent and all downloaded files.\n\n**Available:** %d torrent(s)\n*You can select up to %d torrent(s) at a time*", strings.Title(selectedCategory), totalTorrents, maxSelectionLimit)
	} else {
		embedTitle = fmt.Sprintf("🗑️ Delete Torrents - %s (Page 1)", strings.Title(selectedCategory))
		embedDescription = fmt.Sprintf("Select the torrents you want to delete from the **%s** category.\n\n**Note:** This will permanently delete both the torrent and all downloaded files.\n\n**Showing:** %d of %d torrent(s)\n*Only the first 25 torrents are shown due to Discord limits*\n*You can select up to %d torrent(s) at a time*", strings.Title(selectedCategory), maxOptions, totalTorrents, maxSelectionLimit)
	}

	embed := createInfoEmbed(embedTitle, embedDescription)

	// Update the message with the torrent selection menu
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actionRow},
		},
	})

	if err != nil {
		fmt.Printf("Failed to update delete response: %v\n", err)
		// Fallback to responding with error
		respondWithError(s, i, fmt.Sprintf("Failed to show torrents: %v", err))
	}
}
