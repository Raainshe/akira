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

	diskSummary, err := diskService.GetAllDiskSpaces(ctx)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get disk information: %v", err))
		return
	}

	content := formatDiskSummary(diskSummary)
	embed := createInfoEmbed("💾 Disk Space", content)

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

func formatDiskSummary(summary *core.DiskSummary) string {
	if summary == nil || len(summary.Drives) == 0 {
		return "No disk information available."
	}

	var builder strings.Builder

	builder.WriteString("**Overall Summary:**\n")
	builder.WriteString(fmt.Sprintf("Total available (unique drives): %s\n", formatBytes(summary.TotalFree)))
	builder.WriteString(fmt.Sprintf("Worst health: %s\n\n", getHealthEmoji(summary.WorstHealth)))

	builder.WriteString("**Drives:**\n")
	for _, driveID := range summary.DriveOrder {
		drive := summary.Drives[driveID]
		if drive == nil {
			continue
		}
		builder.WriteString(fmt.Sprintf("**%s**\n", drive.DriveID))
		builder.WriteString(fmt.Sprintf("  Available: %s  %s\n", formatBytes(drive.Free), getHealthEmoji(drive.Health)))
		if len(drive.Paths) > 0 {
			builder.WriteString(fmt.Sprintf("  Paths: %s\n", strings.Join(drive.Paths, ", ")))
		}
		builder.WriteString("\n")
	}

	if len(summary.WarningPaths) > 0 || len(summary.CriticalPaths) > 0 {
		builder.WriteString("**⚠️ Warnings:**\n")
		if len(summary.WarningPaths) > 0 {
			builder.WriteString(fmt.Sprintf("Warning drives: %s\n", strings.Join(summary.WarningPaths, ", ")))
		}
		if len(summary.CriticalPaths) > 0 {
			builder.WriteString(fmt.Sprintf("Critical drives: %s\n", strings.Join(summary.CriticalPaths, ", ")))
		}
	}

	return builder.String()
}

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
