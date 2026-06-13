package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

const recategorizePageSize = 25

var recategorizeCategories = []struct {
	value       string
	label       string
	description string
}{
	{value: "movies", label: "Movies", description: "Move to movies save path"},
	{value: "series", label: "Series", description: "Move to series save path"},
	{value: "anime", label: "Anime", description: "Move to anime save path"},
	{value: "default", label: "Default", description: "Move to default save path"},
}

func updateRecategorizeMessage(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func createRecategorizePaginationComponents(currentPage, totalPages int) []discordgo.MessageComponent {
	if totalPages <= 1 {
		return nil
	}

	row := discordgo.ActionsRow{}

	if currentPage > 1 {
		row.Components = append(row.Components, discordgo.Button{
			Label:    "◀️ Previous",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("recategorize_page|%d", currentPage-1),
		})
	}

	row.Components = append(row.Components, discordgo.Button{
		Label:    fmt.Sprintf("Page %d/%d", currentPage, totalPages),
		Style:    discordgo.SecondaryButton,
		CustomID: "recategorize_page_info",
		Disabled: true,
	})

	if currentPage < totalPages {
		row.Components = append(row.Components, discordgo.Button{
			Label:    "Next ▶️",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("recategorize_page|%d", currentPage+1),
		})
	}

	return []discordgo.MessageComponent{row}
}

func showRecategorizeTorrentPage(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, page int) error {
	ctx := context.Background()

	torrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return err
	}

	if len(torrents) == 0 {
		respondWithError(s, i, "No torrents found")
		return fmt.Errorf("no torrents found")
	}

	totalTorrents := len(torrents)
	totalPages := (totalTorrents + recategorizePageSize - 1) / recategorizePageSize

	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * recategorizePageSize
	end := offset + recategorizePageSize
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

		category := torrentService.GetTorrentCategory(torrent)
		description := fmt.Sprintf("%s | %s | %s", category, formatBytes(int64(torrent.Size)), string(torrent.State))
		if len(description) > 100 {
			description = description[:97] + "..."
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       name,
			Value:       fmt.Sprintf("%s|%d", torrent.Hash, idx),
			Description: description,
		})
	}

	minValues := 1
	maxValues := 1
	selectMenu := discordgo.SelectMenu{
		CustomID:    "recategorize_torrent_select",
		Placeholder: "Select a torrent to recategorize",
		MinValues:   &minValues,
		MaxValues:   maxValues,
		Options:     options,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
	}
	if pagination := createRecategorizePaginationComponents(page, totalPages); pagination != nil {
		components = append(components, pagination...)
	}

	var embedTitle string
	var embedDescription string
	if totalPages > 1 {
		embedTitle = fmt.Sprintf("📂 Recategorize Torrent (Page %d/%d)", page, totalPages)
		embedDescription = fmt.Sprintf(
			"Select the torrent you want to move to a different category.\n\n**Showing:** %d-%d of %d torrent(s)\nUse the buttons below to navigate pages.",
			offset+1, end, totalTorrents,
		)
	} else {
		embedTitle = "📂 Recategorize Torrent"
		embedDescription = fmt.Sprintf(
			"Select the torrent you want to move to a different category.\n\n**Available:** %d torrent(s)",
			totalTorrents,
		)
	}

	embed := createInfoEmbed(embedTitle, embedDescription)
	return updateRecategorizeMessage(s, i, embed, components)
}

// HandleRecategorizeCommand handles the /recategorize Discord command
func HandleRecategorizeCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, _ *config.Config) {
	_ = showRecategorizeTorrentPageInitial(s, i, torrentService, 1)
}

func showRecategorizeTorrentPageInitial(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, page int) error {
	ctx := context.Background()

	torrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return err
	}

	if len(torrents) == 0 {
		embed := createInfoEmbed("📂 Recategorize Torrent", "No torrents found.")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
		return fmt.Errorf("no torrents found")
	}

	totalTorrents := len(torrents)
	totalPages := (totalTorrents + recategorizePageSize - 1) / recategorizePageSize

	offset := (page - 1) * recategorizePageSize
	end := offset + recategorizePageSize
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

		category := torrentService.GetTorrentCategory(torrent)
		description := fmt.Sprintf("%s | %s | %s", category, formatBytes(int64(torrent.Size)), string(torrent.State))
		if len(description) > 100 {
			description = description[:97] + "..."
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       name,
			Value:       fmt.Sprintf("%s|%d", torrent.Hash, idx),
			Description: description,
		})
	}

	minValues := 1
	maxValues := 1
	selectMenu := discordgo.SelectMenu{
		CustomID:    "recategorize_torrent_select",
		Placeholder: "Select a torrent to recategorize",
		MinValues:   &minValues,
		MaxValues:   maxValues,
		Options:     options,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
	}
	if pagination := createRecategorizePaginationComponents(page, totalPages); pagination != nil {
		components = append(components, pagination...)
	}

	var embedTitle string
	var embedDescription string
	if totalPages > 1 {
		embedTitle = fmt.Sprintf("📂 Recategorize Torrent (Page %d/%d)", page, totalPages)
		embedDescription = fmt.Sprintf(
			"Select the torrent you want to move to a different category.\n\n**Showing:** %d-%d of %d torrent(s)\nUse the buttons below to navigate pages.",
			offset+1, end, totalTorrents,
		)
	} else {
		embedTitle = "📂 Recategorize Torrent"
		embedDescription = fmt.Sprintf(
			"Select the torrent you want to move to a different category.\n\n**Available:** %d torrent(s)",
			totalTorrents,
		)
	}

	embed := createInfoEmbed(embedTitle, embedDescription)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	return err
}

// HandleRecategorizePagination handles pagination for the torrent list
func HandleRecategorizePagination(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	data := i.MessageComponentData()
	if data.CustomID == "recategorize_page_info" {
		return
	}

	parts := strings.Split(data.CustomID, "|")
	if len(parts) < 2 {
		respondWithError(s, i, "Invalid pagination data")
		return
	}

	page, err := strconv.Atoi(parts[1])
	if err != nil {
		respondWithError(s, i, "Invalid page number")
		return
	}

	_ = showRecategorizeTorrentPage(s, i, torrentService, page)
}

// HandleRecategorizeTorrentSelect handles torrent selection
func HandleRecategorizeTorrentSelect(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		respondWithError(s, i, "No torrent selected")
		return
	}

	parts := strings.Split(data.Values[0], "|")
	if len(parts) < 1 || parts[0] == "" {
		respondWithError(s, i, "Invalid torrent selection")
		return
	}

	hash := strings.TrimSpace(parts[0])
	showRecategorizeCategorySelection(s, i, torrentService, hash)
}

func showRecategorizeCategorySelection(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, hash string) {
	ctx := context.Background()

	torrent, err := torrentService.FindTorrentByHash(ctx, hash)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to find torrent: %v", err))
		return
	}

	currentCategory := torrentService.GetTorrentCategory(*torrent)
	var options []discordgo.SelectMenuOption
	for _, cat := range recategorizeCategories {
		if cat.value == currentCategory {
			continue
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       cat.label,
			Value:       cat.value,
			Description: cat.description,
		})
	}

	if len(options) == 0 {
		respondWithError(s, i, fmt.Sprintf("No other categories available (current: %s)", currentCategory))
		return
	}

	minValues := 1
	maxValues := 1
	selectMenu := discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("recategorize_category_select|%s", torrent.Hash),
		Placeholder: "Select the new category",
		MinValues:   &minValues,
		MaxValues:   maxValues,
		Options:     options,
	}

	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{selectMenu},
	}

	currentPath := torrent.SavePath
	content := fmt.Sprintf(
		"**Selected torrent:** %s\n\n**Current category:** %s\n**Current path:** `%s`\n\nSelect the category to move this torrent to.",
		torrent.Name,
		currentCategory,
		currentPath,
	)

	embed := createInfoEmbed("📂 Choose New Category", content)
	_ = updateRecategorizeMessage(s, i, embed, []discordgo.MessageComponent{actionRow})
}

// HandleRecategorizeCategorySelect handles new category selection and shows confirmation
func HandleRecategorizeCategorySelect(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, cfg *config.Config) {
	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, "|")
	if len(parts) < 2 {
		respondWithError(s, i, "Invalid category selection data")
		return
	}

	hash := parts[1]
	if len(data.Values) == 0 {
		respondWithError(s, i, "No category selected")
		return
	}

	newCategory := data.Values[0]
	ctx := context.Background()

	torrent, err := torrentService.FindTorrentByHash(ctx, hash)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to find torrent: %v", err))
		return
	}

	currentCategory := torrentService.GetTorrentCategory(*torrent)
	newSavePath := cfg.GetSavePathForCategory(newCategory)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("**Torrent:** %s\n\n", torrent.Name))
	content.WriteString(fmt.Sprintf("**From:** %s → `%s`\n", currentCategory, torrent.SavePath))
	content.WriteString(fmt.Sprintf("**To:** %s → `%s`\n\n", newCategory, newSavePath))

	if torrent.State == qbittorrent.StateMoving {
		content.WriteString("⚠️ This torrent is currently moving and cannot be recategorized.\n")
	} else if torrent.Progress < 1.0 {
		content.WriteString(fmt.Sprintf("**Progress:** %.1f%% (%s)\n\n", torrent.GetProgressPercentage(), torrent.GetStateDisplayName()))
		content.WriteString("⚠️ **This action will:**\n")
		content.WriteString("• Pause the torrent during the move\n")
		content.WriteString("• Move downloaded files to the new save path\n")
		content.WriteString("• Update the torrent category in qBittorrent\n\n")
	} else {
		content.WriteString(fmt.Sprintf("**State:** %s\n\n", torrent.GetStateDisplayName()))
		content.WriteString("⚠️ **This action will:**\n")
		content.WriteString("• Move all downloaded files to the new save path\n")
		content.WriteString("• Update the torrent category in qBittorrent\n\n")
	}

	if torrent.SavePath == newSavePath {
		content.WriteString("*Note: Source and target paths are the same — only the category label will change.*\n\n")
	}

	content.WriteString("**Are you sure you want to proceed?**")

	confirmButton := discordgo.Button{
		Label:    "✅ Confirm Move",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("recategorize_confirm|%s|%s", torrent.Hash, newCategory),
	}

	cancelButton := discordgo.Button{
		Label:    "❌ Cancel",
		Style:    discordgo.SecondaryButton,
		CustomID: "recategorize_cancel",
	}

	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{confirmButton, cancelButton},
	}

	embed := createWarningEmbed("📂 Confirm Recategorize", content.String())
	_ = updateRecategorizeMessage(s, i, embed, []discordgo.MessageComponent{actionRow})
}

// HandleRecategorizeConfirm executes the recategorize operation
func HandleRecategorizeConfirm(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, "|")
	if len(parts) < 3 {
		respondWithError(s, i, "Invalid confirmation data")
		return
	}

	hash := parts[1]
	newCategory := parts[2]

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to acknowledge recategorize: %v", err))
		return
	}

	ctx := context.Background()
	result, err := torrentService.RecategorizeTorrent(ctx, hash, newCategory)
	if err != nil {
		_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{createErrorEmbed("❌ Recategorize Failed", fmt.Sprintf("Failed to recategorize torrent: %v", err))},
		})
		if editErr != nil {
			_, _ = s.ChannelMessageSendEmbed(i.ChannelID, createErrorEmbed("❌ Recategorize Failed", fmt.Sprintf("Failed to recategorize torrent: %v", err)))
		}
		return
	}

	var content strings.Builder
	content.WriteString("✅ **Successfully recategorized torrent**\n\n")
	content.WriteString(fmt.Sprintf("**Torrent:** %s\n\n", result.Torrent.Name))
	content.WriteString(fmt.Sprintf("**Category:** %s → %s\n", result.OldCategory, result.NewCategory))
	content.WriteString(fmt.Sprintf("**Path:** `%s` → `%s`\n", result.OldSavePath, result.Torrent.SavePath))

	if result.FilesMoved {
		content.WriteString("\n📁 Files were moved to the new save path.")
	} else {
		content.WriteString("\n🏷️ Category label updated (files were already at the target path).")
	}

	embed := createSuccessEmbed("📂 Torrent Recategorized", content.String())
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		_, _ = s.ChannelMessageSendEmbed(i.ChannelID, embed)
	}
}

// HandleRecategorizeCancel handles cancellation of the recategorize operation
func HandleRecategorizeCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := createInfoEmbed("❌ Recategorize Cancelled", "The recategorize operation has been cancelled. No changes were made.")

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		fmt.Printf("Failed to send recategorize cancellation: %v\n", err)
	}
}
