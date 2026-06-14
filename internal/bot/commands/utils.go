package commands

import (
	"fmt"
	"regexp"
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

// formatTorrentList formats torrents for Discord display (legacy - with pagination)
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

// formatTorrentListAll formats all torrents for Discord display (no pagination)
func formatTorrentListAll(torrents []qbittorrent.Torrent) string {
	if len(torrents) == 0 {
		return "No torrents found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("**Total: %d torrent(s)**\n\n", len(torrents)))

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

		// Safe hash substring
		hashDisplay := torrent.Hash
		if len(hashDisplay) >= 8 {
			hashDisplay = hashDisplay[:8]
		}
		builder.WriteString(fmt.Sprintf("   Size: %s | Hash: `%s`\n\n", formatBytes(torrent.Size), hashDisplay))
	}

	return builder.String()
}

// splitContent splits content into chunks that fit within maxLength
// Tries to split at newlines to avoid breaking in the middle of a line
func splitContent(content string, maxLength int) []string {
	if len(content) <= maxLength {
		return []string{content}
	}

	var chunks []string
	lines := strings.Split(content, "\n")
	var currentChunk strings.Builder

	for _, line := range lines {
		// Check if adding this line would exceed the limit
		potentialLength := currentChunk.Len() + len(line) + 1 // +1 for newline
		if potentialLength > maxLength && currentChunk.Len() > 0 {
			// Save current chunk and start new one
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}

		// If a single line is too long, truncate it
		if len(line) > maxLength {
			line = line[:maxLength-3] + "..."
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
	}

	// Add the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// formatDiskUsage formats disk usage for Discord display
func formatDiskUsage(diskInfo *core.DiskInfo) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("**%s**\n", diskInfo.Path))
	if diskInfo.Total == 0 {
		builder.WriteString(fmt.Sprintf("Available: %s\n\n", formatBytes(diskInfo.Free)))
		return builder.String()
	}

	usageBar := getUsageBar(diskInfo.UsedPercent)
	builder.WriteString(fmt.Sprintf("%s\n", usageBar))
	builder.WriteString(fmt.Sprintf("Used: %s / %s (%.1f%%)\n\n",
		formatBytes(diskInfo.Used),
		formatBytes(diskInfo.Total),
		diskInfo.UsedPercent))

	return builder.String()
}

// formatTorrentProgress formats torrent progress for Discord display
func formatTorrentProgress(torrent *qbittorrent.Torrent, elapsed, remaining int) string {
	progress := "0%"
	if torrent.Size > 0 {
		percentage := float64(torrent.Downloaded) / float64(torrent.Size) * 100
		if percentage > 100 {
			percentage = 100
		}
		progress = fmt.Sprintf("%.1f%%", percentage)
	}

	downloadSpeed := "0 B/s"
	if torrent.Dlspeed > 0 {
		downloadSpeed = formatBytes(torrent.Dlspeed) + "/s"
	}

	uploadSpeed := "0 B/s"
	if torrent.Upspeed > 0 {
		uploadSpeed = formatBytes(torrent.Upspeed) + "/s"
	}

	eta := "Unknown"
	if torrent.Eta > 0 {
		eta = formatDuration(time.Duration(torrent.Eta) * time.Second)
	}

	ratio := "0.00"
	if torrent.Ratio > 0 {
		ratio = fmt.Sprintf("%.2f", torrent.Ratio)
	}

	progressBar := createProgressBar(float64(torrent.Downloaded), float64(torrent.Size))

	content := fmt.Sprintf("📥 **%s**\n\n"+
		"%s\n\n"+
		"**Progress:** %s\n"+
		"**Download Speed:** %s\n"+
		"**Upload Speed:** %s\n"+
		"**ETA:** %s\n"+
		"**Ratio:** %s\n"+
		"**Peers:** %d/%d (Seeds: %d)\n"+
		"**State:** %s\n\n"+
		"**Size:** %s\n"+
		"**Downloaded:** %s\n"+
		"**Uploaded:** %s",
		torrent.Name,
		progressBar,
		progress,
		downloadSpeed,
		uploadSpeed,
		eta,
		ratio,
		torrent.NumLeechs, torrent.NumIncomplete, torrent.NumSeeds,
		getStateEmoji(torrent.State)+" "+string(torrent.State),
		formatBytes(torrent.Size),
		formatBytes(torrent.Downloaded),
		formatBytes(torrent.Uploaded))

	if elapsed > 0 || remaining > 0 {
		content += fmt.Sprintf("\n\n⏱️ **Tracking:** %ds elapsed", elapsed)
		if remaining > 0 {
			content += fmt.Sprintf(", %ds remaining", remaining)
		}
	}

	return content
}

func createProgressBar(current, total float64) string {
	const barLength = 20

	if total <= 0 {
		return strings.Repeat("░", barLength)
	}

	percentage := current / total
	if percentage > 1.0 {
		percentage = 1.0
	}
	filled := int(percentage * barLength)
	if filled > barLength {
		filled = barLength
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)
	return fmt.Sprintf("`%s` %.1f%%", bar, percentage*100)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
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

	// Parse and format each log line
	for i, log := range logs {
		if i >= 20 { // Limit to 20 log lines
			builder.WriteString(fmt.Sprintf("... and %d more lines\n", len(logs)-20))
			break
		}

		// Parse the log line and format it cleanly
		formattedLine := parseAndFormatLogLine(log)

		// Truncate if too long
		if len(formattedLine) > 200 {
			formattedLine = formattedLine[:197] + "..."
		}

		// Check if adding this line would exceed Discord's limit
		if currentChars+len(formattedLine)+1 > maxChars {
			builder.WriteString(fmt.Sprintf("... and %d more lines\n", len(logs)-i))
			break
		}

		builder.WriteString(formattedLine + "\n")
		currentChars += len(formattedLine) + 1
	}

	return builder.String()
}

// parseAndFormatLogLine parses a logrus JSON log line and formats it for Discord
func parseAndFormatLogLine(logLine string) string {
	// Remove ANSI color codes
	cleanLine := removeANSICodes(logLine)

	// Try to parse as JSON first (structured logrus output)
	if strings.Contains(cleanLine, `"level"`) && strings.Contains(cleanLine, `"msg"`) {
		return parseStructuredLog(cleanLine)
	}

	// If not structured, try to clean up common logrus text format
	return cleanLogrusText(cleanLine)
}

// removeANSICodes removes ANSI escape sequences from the log line
func removeANSICodes(line string) string {
	// Remove ANSI color codes like [31m, [0m, etc.
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(line); i++ {
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			inEscape = true
			i++ // Skip the [
			continue
		}

		if inEscape {
			if line[i] == 'm' {
				inEscape = false
			}
			continue
		}

		result.WriteByte(line[i])
	}

	return result.String()
}

// parseStructuredLog parses a structured logrus JSON log
func parseStructuredLog(jsonLine string) string {
	// Simple JSON parsing for common logrus fields
	// We'll extract the key fields and format them nicely

	// Extract level
	level := extractJSONField(jsonLine, "level")
	if level == "" {
		level = "INFO"
	}

	// Extract message
	msg := extractJSONField(jsonLine, "msg")
	if msg == "" {
		msg = "No message"
	}

	// Extract time
	time := extractJSONField(jsonLine, "time")
	if time == "" {
		time = "Unknown time"
	}

	// Extract component
	component := extractJSONField(jsonLine, "component")

	// Extract error if present
	error := extractJSONField(jsonLine, "error")

	// Format the log line
	var result strings.Builder

	// Add level emoji
	switch strings.ToUpper(level) {
	case "ERROR":
		result.WriteString("❌ ")
	case "WARN", "WARNING":
		result.WriteString("⚠️ ")
	case "INFO":
		result.WriteString("ℹ️ ")
	case "DEBUG":
		result.WriteString("🔍 ")
	default:
		result.WriteString("📝 ")
	}

	// Add level and time
	result.WriteString(fmt.Sprintf("**%s** | %s", strings.ToUpper(level), time))

	// Add component if present
	if component != "" {
		result.WriteString(fmt.Sprintf(" | **%s**", component))
	}

	result.WriteString("\n")

	// Add message
	result.WriteString(fmt.Sprintf("**Message:** %s", msg))

	// Add error if present
	if error != "" {
		result.WriteString(fmt.Sprintf("\n**Error:** %s", error))
	}

	return result.String()
}

// cleanLogrusText cleans up logrus text format logs
func cleanLogrusText(line string) string {
	// Remove common logrus prefixes and clean up the format
	// Example: time="2025-09-02T21:50:57+02:00" level=error msg="Authentication failed"

	// Remove time prefix if present
	if strings.HasPrefix(line, "time=") {
		// Find the end of the time field
		if idx := strings.Index(line, "level="); idx != -1 {
			line = line[idx:]
		}
	}

	// Clean up level field
	if strings.HasPrefix(line, "level=") {
		// Extract level
		levelEnd := strings.Index(line, " ")
		if levelEnd != -1 {
			level := line[6:levelEnd] // "level=" is 6 chars
			line = line[levelEnd+1:]

			// Add level emoji
			var levelEmoji string
			switch strings.ToUpper(level) {
			case "ERROR":
				levelEmoji = "❌ "
			case "WARN", "WARNING":
				levelEmoji = "⚠️ "
			case "INFO":
				levelEmoji = "ℹ️ "
			case "DEBUG":
				levelEmoji = "🔍 "
			default:
				levelEmoji = "📝 "
			}

			line = levelEmoji + "**" + strings.ToUpper(level) + "** " + line
		}
	}

	// Clean up msg field
	if strings.HasPrefix(line, "msg=") {
		line = strings.Replace(line, "msg=", "**Message:** ", 1)
	}

	// Clean up component field
	if strings.Contains(line, "component=") {
		// Extract component
		if idx := strings.Index(line, "component="); idx != -1 {
			componentStart := idx + 10 // "component=" is 10 chars
			componentEnd := strings.Index(line[componentStart:], " ")
			if componentEnd == -1 {
				componentEnd = len(line) - componentStart
			}
			component := line[componentStart : componentStart+componentEnd]

			// Replace component= with formatted version
			line = strings.Replace(line, "component="+component, "**Component:** "+component, 1)
		}
	}

	// Clean up error field
	if strings.Contains(line, "error=") {
		line = strings.Replace(line, "error=", "**Error:** ", 1)
	}

	return line
}

// extractJSONField extracts a field value from a JSON string
func extractJSONField(json, field string) string {
	// Simple JSON field extraction
	fieldPattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, field)

	// Use regex to find the field value
	re := regexp.MustCompile(fieldPattern)
	matches := re.FindStringSubmatch(json)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
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
