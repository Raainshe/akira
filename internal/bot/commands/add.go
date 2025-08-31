package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// HandleAddCommand handles the /add Discord command
func HandleAddCommand(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService, config *config.Config) {
	// Get command options
	data := i.ApplicationCommandData()

	var magnetURI string
	var category string = "default"

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "magnet":
			magnetURI = option.StringValue()
		case "category":
			category = option.StringValue()
		}
	}

	// Validate magnet URI
	if magnetURI == "" {
		respondWithError(s, i, "Magnet URI is required")
		return
	}

	if !isValidMagnetURI(magnetURI) {
		respondWithError(s, i, "Invalid magnet URI format")
		return
	}

	// Create add request
	request := &core.AddTorrentRequest{
		MagnetURI: magnetURI,
		Category:  category,
	}

	// Add torrent
	ctx := context.Background()
	torrent, err := torrentService.AddMagnet(ctx, request)
	if err != nil {
		// Check if it's a qBittorrent API error
		if apiErr, ok := err.(*qbittorrent.APIError); ok {
			respondWithError(s, i, fmt.Sprintf("qBittorrent Error: %s", apiErr.Details))
		} else {
			respondWithError(s, i, fmt.Sprintf("Failed to add torrent: %v", err))
		}
		return
	}

	// Create initial success response
	var content string
	if torrent != nil {
		// We have torrent information - show initial progress
		content = formatTorrentProgress(torrent, 0, 0) // 0 elapsed, 0 remaining for initial
	} else {
		// No torrent info available - show basic confirmation
		content = fmt.Sprintf("✅ **Torrent Added Successfully!**\n\n"+
			"**Magnet URI:** `%s`\n"+
			"**Category:** %s\n\n"+
			"⏳ **Fetching torrent information...**\n"+
			"Live progress will start once torrent details are available.",
			truncateString(magnetURI, 50),
			category)
	}

	embed := createSuccessEmbed("📥 Torrent Added", content)

	// Send initial response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send add response: %v\n", err)
		return
	}

	// If we have torrent info, start automatic progress tracking
	if torrent != nil {
		go trackTorrentProgress(s, i, torrentService, seedingService, config, torrent.Hash, torrent.Name)
	} else {
		// Try to find the torrent after a delay and start tracking
		go findAndTrackTorrent(s, i, torrentService, seedingService, config, magnetURI, category)
	}
}

// isValidMagnetURI checks if a string is a valid magnet URI
func isValidMagnetURI(uri string) bool {
	return len(uri) > 20 && uri[:20] == "magnet:?xt=urn:btih:"
}

// trackTorrentProgress automatically tracks torrent progress until completion
func trackTorrentProgress(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService, config *config.Config, hash, torrentName string) {
	ctx := context.Background()
	startTime := time.Now()

	// Update every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Track until completion or for maximum 30 minutes
	maxDuration := 30 * time.Minute
	endTime := startTime.Add(maxDuration)

	for {
		select {
		case <-ticker.C:
			// Check if we should stop
			if time.Now().After(endTime) {
				// Send final update
				finalContent := fmt.Sprintf("⏰ **Progress tracking completed**\n\n"+
					"**%s**\n\n"+
					"Live progress updates have stopped after 30 minutes.\n"+
					"Use `/progress \"%s\"` to continue tracking if needed.",
					torrentName, torrentName)
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
				finalContent := fmt.Sprintf("❌ **Torrent not found**\n\n"+
					"**%s**\n\n"+
					"The torrent may have been deleted or is no longer available.",
					torrentName)
				embed := createInfoEmbed("📊 Torrent Progress - Error", finalContent)

				_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					fmt.Printf("Failed to send error progress update: %v\n", err)
				}
				return
			}

			// Check if torrent is completed
			if torrent.Progress >= 1.0 {
				// Torrent is complete!
				elapsed := int(time.Since(startTime).Seconds())
				content := formatTorrentProgress(torrent, elapsed, 0)
				content += "\n\n🎉 **Torrent completed!** Starting automatic seeding management..."
				embed := createSuccessEmbed("✅ Torrent Completed!", content)

				_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					fmt.Printf("Failed to update completion status: %v\n", err)
				}

				// Start automatic seeding management
				go func() {
					// Start tracking the torrent for seeding management
					downloadDuration := time.Duration(elapsed) * time.Second
					if err := seedingService.StartTracking(ctx, hash, torrent.Name); err != nil {
						fmt.Printf("Failed to start seeding tracking: %v\n", err)
						return
					}

					// Mark the torrent as completed in the seeding service
					if err := seedingService.MarkTorrentCompleted(ctx, hash, downloadDuration); err != nil {
						fmt.Printf("Failed to mark torrent as completed: %v\n", err)
						return
					}

					// Calculate seeding duration based on download time and multiplier
					seedingDuration := time.Duration(float64(downloadDuration) * config.Seeding.TimeMultiplier)

					// Update message to show seeding management info
					content := formatTorrentProgress(torrent, elapsed, 0)
					content += fmt.Sprintf("\n\n🌱 **Seeding Management Started!**\n"+
						"**Download Time:** %s\n"+
						"**Seeding Duration:** %s\n"+
						"**Auto-stop Time:** %s",
						formatDuration(downloadDuration),
						formatDuration(seedingDuration),
						time.Now().Add(seedingDuration).Format("2006-01-02 15:04:05"))

					embed := createSuccessEmbed("🌱 Seeding Management Active", content)
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &[]*discordgo.MessageEmbed{embed},
					})
				}()

				// Continue tracking for 2 more minutes after completion
				completionTime := time.Now()
				for time.Since(completionTime) < 2*time.Minute {
					time.Sleep(10 * time.Second)

					// Get final stats
					torrent, err := torrentService.FindTorrentByHash(ctx, hash)
					if err != nil {
						break
					}

					elapsed := int(time.Since(startTime).Seconds())
					content := formatTorrentProgress(torrent, elapsed, 0)
					content += "\n\n🎉 **Torrent completed!** Seeding management is active."
					embed := createSuccessEmbed("✅ Torrent Completed!", content)

					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &[]*discordgo.MessageEmbed{embed},
					})
				}
				return
			}

			// Calculate elapsed time
			elapsed := int(time.Since(startTime).Seconds())

			// Update progress message
			content := formatTorrentProgress(torrent, elapsed, 0)
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

// findAndTrackTorrent tries to find a torrent by hash and start tracking
func findAndTrackTorrent(s *discordgo.Session, i *discordgo.InteractionCreate, torrentService *core.TorrentService, seedingService *core.SeedingService, config *config.Config, magnetURI, category string) {
	ctx := context.Background()

	// Extract hash from magnet URI
	hash, err := extractHashFromMagnet(magnetURI)
	if err != nil {
		// Can't extract hash, can't track
		return
	}

	// Try to find the torrent for up to 2 minutes
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for time.Since(startTime) < 2*time.Minute {
		<-ticker.C

		// Try to find the torrent
		torrent, err := torrentService.FindTorrentByHash(ctx, hash)
		if err == nil && torrent != nil {
			// Found it! Start tracking
			trackTorrentProgress(s, i, torrentService, seedingService, config, torrent.Hash, torrent.Name)
			return
		}
	}

	// Couldn't find the torrent after 2 minutes
	finalContent := fmt.Sprintf("⚠️ **Torrent not found**\n\n"+
		"**Magnet URI:** `%s`\n"+
		"**Category:** %s\n\n"+
		"Could not find torrent information after 2 minutes.\n"+
		"The torrent may still be processing or there was an issue.",
		truncateString(magnetURI, 50),
		category)
	embed := createWarningEmbed("⚠️ Torrent Status Unknown", finalContent)

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

// extractHashFromMagnet extracts the info hash from a magnet URI
func extractHashFromMagnet(magnetURI string) (string, error) {
	// Simple extraction - look for the hash after "urn:btih:"
	start := strings.Index(magnetURI, "urn:btih:")
	if start == -1 {
		return "", fmt.Errorf("invalid magnet URI: missing hash")
	}

	start += 9 // Length of "urn:btih:"
	end := strings.Index(magnetURI[start:], "&")
	if end == -1 {
		end = len(magnetURI)
	} else {
		end += start
	}

	hash := magnetURI[start:end]
	if len(hash) != 32 && len(hash) != 40 {
		return "", fmt.Errorf("invalid hash length: %d", len(hash))
	}

	return hash, nil
}
