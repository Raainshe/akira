package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/raainshe/akira/internal/core"
)

// HandleDiskCommand handles the /disk Discord command
func HandleDiskCommand(s *discordgo.Session, i *discordgo.InteractionCreate, diskService *core.DiskService) {
	ctx := context.Background()

	// Get disk space for all configured paths
	diskSummary, err := diskService.GetAllDiskSpaces(ctx)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get disk information: %v", err))
		return
	}

	// Format response
	content := formatDiskSummary(diskSummary)

	// Create embed
	embed := createInfoEmbed("💾 Disk Usage", content)

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send disk response: %v\n", err)
	}
}

// formatDiskSummary formats disk summary for Discord display
func formatDiskSummary(summary *core.DiskSummary) string {
	if summary == nil || len(summary.Paths) == 0 {
		return "No disk information available."
	}

	var builder strings.Builder

	// Overall summary
	builder.WriteString(fmt.Sprintf("**Overall Summary:**\n"))
	builder.WriteString(fmt.Sprintf("Total Space: %s\n", formatBytes(summary.TotalSpace)))
	builder.WriteString(fmt.Sprintf("Total Used: %s\n", formatBytes(summary.TotalUsed)))
	builder.WriteString(fmt.Sprintf("Total Free: %s\n", formatBytes(summary.TotalFree)))
	builder.WriteString(fmt.Sprintf("Worst Health: %s\n\n", getHealthEmoji(summary.WorstHealth)))

	// Individual paths
	builder.WriteString("**Individual Paths:**\n")
	for path, diskInfo := range summary.Paths {
		usageBar := getUsageBar(diskInfo.UsedPercent)
		builder.WriteString(fmt.Sprintf("**%s**\n", path))
		builder.WriteString(fmt.Sprintf("%s\n", usageBar))
		builder.WriteString(fmt.Sprintf("Used: %s / %s (%.1f%%)\n\n",
			formatBytes(diskInfo.Used),
			formatBytes(diskInfo.Total),
			diskInfo.UsedPercent))
	}

	// Warnings if any
	if len(summary.WarningPaths) > 0 || len(summary.CriticalPaths) > 0 {
		builder.WriteString("**⚠️ Warnings:**\n")
		if len(summary.WarningPaths) > 0 {
			builder.WriteString(fmt.Sprintf("Warning paths: %s\n", strings.Join(summary.WarningPaths, ", ")))
		}
		if len(summary.CriticalPaths) > 0 {
			builder.WriteString(fmt.Sprintf("Critical paths: %s\n", strings.Join(summary.CriticalPaths, ", ")))
		}
	}

	return builder.String()
}

// getHealthEmoji returns emoji for disk health status
func getHealthEmoji(health core.DiskHealthStatus) string {
	switch health {
	case core.DiskHealthGood:
		return "🟢 Good"
	case core.DiskHealthWarning:
		return "🟡 Warning"
	case core.DiskHealthCritical:
		return "🟠 Critical"
	case core.DiskHealthDanger:
		return "🔴 Danger"
	default:
		return "❓ Unknown"
	}
}
