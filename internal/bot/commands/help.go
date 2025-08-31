package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandleHelpCommand handles the /help Discord command
func HandleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	content := "**🤖 Akira Torrent Manager - Discord Bot Commands**\n\n" +
		"**📋 Torrent Management:**\n" +
		"• `/torrents [filter] [page]` - List torrents with filtering and pagination\n" +
		"• `/add <magnet> [category]` - Add a magnet link with **automatic live progress tracking**\n" +
		"• `/delete <query> [delete_files]` - Delete torrents by name or hash\n" +
		"• `/progress <torrent> [duration]` - Show live progress for a specific torrent\n\n" +
		"**💾 System Information:**\n" +
		"• `/disk` - Show disk usage statistics\n" +
		"• `/logs [level] [lines]` - Show recent application logs\n" +
		"• `/seeding-status` - Show seeding service status and statistics\n\n" +
		"**🌱 Seeding Management:**\n" +
		"• `/stop-seeding <torrent>` - Stop tracking a specific torrent for seeding\n\n" +
		"**📖 Usage Examples:**\n" +
		"• `/torrents filter:downloading` - Show only downloading torrents\n" +
		"• `/add magnet:?xt=urn:btih:... category:movies` - Add movie torrent with live tracking\n" +
		"• `/delete \"My Movie\" delete_files:true` - Delete torrent and files\n" +
		"• `/progress \"My Movie\" duration:120` - Track progress for 2 minutes\n" +
		"• `/logs level:error lines:20` - Show last 20 error logs\n\n" +
		"**🔧 Filter Options:**\n" +
		"• **torrents filter:** all, downloading, seeding, paused\n" +
		"• **logs level:** all, error, warning, info, debug\n" +
		"• **category:** default, movies, series, anime\n\n" +
		"**💡 Tips:**\n" +
		"• Use partial names for torrent queries\n" +
		"• Hash queries are case-insensitive\n" +
		"• **Add command automatically starts live progress tracking**\n" +
		"• **Automatic seeding management** starts when torrents complete\n" +
		"• **Seeding duration** = Download time × SEEDING_TIME_MULTIPLIER\n" +
		"• Progress tracking updates every 5 seconds\n" +
		"• Live tracking continues until completion or 30 minutes\n" +
		"• Logs show newest entries first\n" +
		"• Disk usage includes health indicators"

	embed := createInfoEmbed("❓ Help & Commands", content)

	// Send response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send help response: %v\n", err)
	}
}
