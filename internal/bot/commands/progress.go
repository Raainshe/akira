package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// HandleProgressCommand handles the /progress Discord command
func HandleProgressCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService) {
	// Get command options
	data := i.ApplicationCommandData()

	var torrentQuery string
	var duration int = 60 // Default 60 seconds

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "torrent":
			torrentQuery = option.StringValue()
		case "duration":
			duration = int(option.IntValue())
		}
	}

	// Validate query
	if torrentQuery == "" {
		respondWithError(s, i, "Torrent name or hash is required")
		return
	}

	// Validate duration
	if duration < 10 || duration > 300 {
		duration = 60 // Default to 60 seconds
	}

	// Find the torrent
	ctx := context.Background()
	torrents, err := torrentService.GetTorrents(ctx, nil)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get torrents: %v", err))
		return
	}

	// Find matching torrent
	var targetTorrent *qbittorrent.Torrent
	for _, torrent := range torrents {
		// Check if query matches hash (case insensitive)
		if strings.EqualFold(torrent.Hash, torrentQuery) {
			targetTorrent = &torrent
			break
		} else if strings.Contains(strings.ToLower(torrent.Name), strings.ToLower(torrentQuery)) {
			// Check if query matches name (case insensitive partial match)
			targetTorrent = &torrent
			break
		}
	}

	if targetTorrent == nil {
		respondWithError(s, i, fmt.Sprintf("No torrent found matching '%s'", torrentQuery))
		return
	}

	// Send initial progress message
	initialContent := formatTorrentProgress(targetTorrent, 0, duration)
	embed := createInfoEmbed("📊 Torrent Progress", initialContent)

	// Send initial response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send progress response: %v\n", err)
		return
	}

	// Start live progress updates
	go updateProgressLive(s, i, torrentService, targetTorrent.Hash, duration)
}

// updateProgressLive updates the progress message live
func updateProgressLive(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, hash string, duration int) {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)

	// Update every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if we should stop
			if time.Now().After(endTime) {
				// Send final update
				finalContent := "⏰ **Progress tracking completed**\n\nLive progress updates have stopped."
				embed := createInfoEmbed("📊 Torrent Progress - Completed", finalContent)

				_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					fmt.Printf("Failed to send final progress update: %v\n", err)
				}
				return
			}

			// Get updated torrent info
			torrent, err := torrentService.FindTorrentByHash(ctx, hash)
			if err != nil {
				// Torrent might have been deleted
				finalContent := "❌ **Torrent not found**\n\nThe torrent may have been deleted or is no longer available."
				embed := createInfoEmbed("📊 Torrent Progress - Error", finalContent)

				_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					fmt.Printf("Failed to send error progress update: %v\n", err)
				}
				return
			}

			// Calculate elapsed time
			elapsed := int(time.Since(startTime).Seconds())
			remaining := duration - elapsed

			// Update progress message
			content := formatTorrentProgress(torrent, elapsed, remaining)
			embed := createInfoEmbed("📊 Live Torrent Progress", content)

			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				fmt.Printf("Failed to update progress: %v\n", err)
				return
			}
		}
	}
}

// formatTorrentProgress formats torrent progress for Discord display
func formatTorrentProgress(torrent *qbittorrent.Torrent, elapsed, remaining int) string {
	// Calculate progress percentage (cap at 100%)
	progress := "0%"
	if torrent.Size > 0 {
		percentage := float64(torrent.Downloaded) / float64(torrent.Size) * 100
		if percentage > 100 {
			percentage = 100
		}
		progress = fmt.Sprintf("%.1f%%", percentage)
	}

	// Format speeds
	downloadSpeed := "0 B/s"
	if torrent.Dlspeed > 0 {
		downloadSpeed = formatBytes(torrent.Dlspeed) + "/s"
	}

	uploadSpeed := "0 B/s"
	if torrent.Upspeed > 0 {
		uploadSpeed = formatBytes(torrent.Upspeed) + "/s"
	}

	// Format ETA
	eta := "Unknown"
	if torrent.Eta > 0 {
		eta = formatDuration(time.Duration(torrent.Eta) * time.Second)
	}

	// Format ratio
	ratio := "0.00"
	if torrent.Ratio > 0 {
		ratio = fmt.Sprintf("%.2f", torrent.Ratio)
	}

	// Create progress bar
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

	// Add tracking info if provided
	if elapsed > 0 || remaining > 0 {
		content += fmt.Sprintf("\n\n⏱️ **Tracking:** %ds elapsed", elapsed)
		if remaining > 0 {
			content += fmt.Sprintf(", %ds remaining", remaining)
		}
	}

	return content
}

// createProgressBar creates a visual progress bar
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

// formatDuration formats duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}
