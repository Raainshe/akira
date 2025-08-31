package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// Response utilities for Discord commands

// createEmbed creates a basic Discord embed
func createEmbed(title, description string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Akira Torrent Manager",
		},
	}
}

// createSuccessEmbed creates a success embed
func createSuccessEmbed(title, description string) *discordgo.MessageEmbed {
	return createEmbed(title, description, 0x00FF00) // Green
}

// createErrorEmbed creates an error embed
func createErrorEmbed(title, description string) *discordgo.MessageEmbed {
	return createEmbed(title, description, 0xFF0000) // Red
}

// createInfoEmbed creates an info embed
func createInfoEmbed(title, description string) *discordgo.MessageEmbed {
	return createEmbed(title, description, 0x0099FF) // Blue
}

// createWarningEmbed creates a warning embed
func createWarningEmbed(title, description string) *discordgo.MessageEmbed {
	return createEmbed(title, description, 0xFFA500) // Orange
}

// formatTorrentList formats torrents for Discord display
func formatTorrentList(torrents []qbittorrent.Torrent, page, totalPages int) string {
	if len(torrents) == 0 {
		return "No torrents found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("**Page %d/%d**\n\n", page, totalPages))

	for i, torrent := range torrents {
		// Truncate name if too long
		name := torrent.Name
		if len(name) > 50 {
			name = name[:47] + "..."
		}

		// Format progress
		progress := "0%"
		if torrent.Size > 0 {
			progress = fmt.Sprintf("%.1f%%", float64(torrent.Downloaded)/float64(torrent.Size)*100)
		}

		// Format speed
		speed := "0 B/s"
		if torrent.Dlspeed > 0 {
			speed = formatBytes(torrent.Dlspeed) + "/s"
		} else if torrent.Upspeed > 0 {
			speed = formatBytes(torrent.Upspeed) + "/s"
		}

		// Format state with emoji
		state := getStateEmoji(torrent.State) + " " + string(torrent.State)

		builder.WriteString(fmt.Sprintf("**%d.** %s\n", i+1, name))
		builder.WriteString(fmt.Sprintf("   %s | %s | %s\n", state, progress, speed))
		builder.WriteString(fmt.Sprintf("   Size: %s | Hash: `%s`\n\n", formatBytes(torrent.Size), torrent.Hash[:8]))
	}

	return builder.String()
}

// formatDiskUsage formats disk usage for Discord display
func formatDiskUsage(diskInfo *core.DiskInfo) string {
	var builder strings.Builder

	// Calculate usage percentage
	usagePercent := diskInfo.UsedPercent

	// Choose color based on usage
	usageBar := getUsageBar(usagePercent)

	builder.WriteString(fmt.Sprintf("**%s**\n", diskInfo.Path))
	builder.WriteString(fmt.Sprintf("%s\n", usageBar))
	builder.WriteString(fmt.Sprintf("Used: %s / %s (%.1f%%)\n\n",
		formatBytes(diskInfo.Used),
		formatBytes(diskInfo.Total),
		usagePercent))

	return builder.String()
}

// formatSeedingStatus formats seeding status for Discord display
func formatSeedingStatus(status core.SeedingStatus) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("**Tracked Torrents:** %d\n", status.TrackedTorrents))
	builder.WriteString(fmt.Sprintf("**Active Seeding:** %d\n", status.ActiveSeeding))
	builder.WriteString(fmt.Sprintf("**Completed Seeding:** %d\n", status.CompletedSeeding))
	builder.WriteString(fmt.Sprintf("**Overdue Seeding:** %d\n\n", status.OverdueSeeding))

	if len(status.Details) > 0 {
		builder.WriteString("**Tracked Torrents:**\n")
		count := 0
		for _, torrent := range status.Details {
			if count >= 10 { // Limit to 10 torrents
				builder.WriteString(fmt.Sprintf("... and %d more\n", len(status.Details)-10))
				break
			}
			builder.WriteString(fmt.Sprintf("• %s (%s)\n", torrent.Name, formatBytes(int64(torrent.SeedingDuration))))
			count++
		}
	}

	return builder.String()
}

// formatLogs formats logs for Discord display
func formatLogs(logs []string, level string) string {
	if len(logs) == 0 {
		return "No logs found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("**Recent Logs (%s)**\n\n", level))

	// Limit to prevent Discord embed description overflow (4096 chars)
	maxChars := 3500 // Leave some buffer
	currentChars := len(builder.String())

	for i, log := range logs {
		if i >= 20 { // Limit to 20 log lines
			builder.WriteString(fmt.Sprintf("... and %d more lines\n", len(logs)-20))
			break
		}

		// Truncate log line if it's too long
		logLine := log
		if len(logLine) > 200 {
			logLine = logLine[:197] + "..."
		}

		formattedLine := fmt.Sprintf("`%s`\n", logLine)
		if currentChars+len(formattedLine) > maxChars {
			builder.WriteString(fmt.Sprintf("... and %d more lines\n", len(logs)-i))
			break
		}

		builder.WriteString(formattedLine)
		currentChars += len(formattedLine)
	}

	return builder.String()
}

// formatBytes formats bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getStateEmoji returns emoji for torrent state
func getStateEmoji(state qbittorrent.TorrentState) string {
	switch state {
	case qbittorrent.StateDownloading:
		return "⬇️"
	case qbittorrent.StateUploading:
		return "⬆️"
	case qbittorrent.StatePausedDL, qbittorrent.StatePausedUP:
		return "⏸️"
	case qbittorrent.StateCheckingUP, qbittorrent.StateCheckingDL:
		return "✅"
	case qbittorrent.StateError:
		return "❌"
	default:
		return "❓"
	}
}

// getServiceStatusEmoji returns emoji for service status
func getServiceStatusEmoji(isRunning bool) string {
	if isRunning {
		return "🟢 Running"
	}
	return "🔴 Stopped"
}

// getUsageBar creates a visual usage bar
func getUsageBar(percent float64) string {
	const barLength = 20
	filled := int(percent / 100 * barLength)
	if filled > barLength {
		filled = barLength
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

	// Add color indicator
	if percent >= 90 {
		return "🔴 " + bar
	} else if percent >= 70 {
		return "🟡 " + bar
	} else {
		return "🟢 " + bar
	}
}

// truncateString truncates string to fit Discord limits
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// respondWithError sends an error response to Discord
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Failed to send error response: %v\n", err)
	}
}

// createPaginationComponents creates pagination buttons
func createPaginationComponents(currentPage, totalPages int) []discordgo.MessageComponent {
	if totalPages <= 1 {
		return nil
	}

	var components []discordgo.MessageComponent
	row := discordgo.ActionsRow{}

	// Previous button
	if currentPage > 1 {
		row.Components = append(row.Components, discordgo.Button{
			Label:    "◀️ Previous",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("page_%d", currentPage-1),
		})
	}

	// Page indicator
	row.Components = append(row.Components, discordgo.Button{
		Label:    fmt.Sprintf("Page %d/%d", currentPage, totalPages),
		Style:    discordgo.SecondaryButton,
		CustomID: "page_info",
		Disabled: true,
	})

	// Next button
	if currentPage < totalPages {
		row.Components = append(row.Components, discordgo.Button{
			Label:    "Next ▶️",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("page_%d", currentPage+1),
		})
	}

	components = append(components, row)
	return components
}
