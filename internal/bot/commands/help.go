package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandleHelpCommand handles the /help Discord command
func HandleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	content := "**Akira — Discord commands for qBittorrent**\n\n" +
		"Akira connects Discord to your qBittorrent instance. Use it to list and manage torrents, " +
		"add magnets, fix categories, and check disk space or logs. Seeding time limits run " +
		"automatically in the background (no Discord command required).\n\n" +
		"**Commands**\n" +
		"• `/torrents [filter]` — List torrents (filter: all, downloading, seeding, paused)\n" +
		"• `/add magnet [category]` — Add a magnet; the reply updates with live progress until done\n" +
		"• `/delete` — Select torrents from a menu, confirm, then remove torrent and files\n" +
		"• `/recategorize` — Select a torrent, pick a new category, confirm move to that save path\n" +
		"• `/disk` — Disk usage for configured download paths (includes a chart)\n" +
		"• `/logs [level] [lines]` — Recent bot activity log (default: 10 lines)\n" +
		"• `/help` — Show this message\n\n" +
		"**Examples**\n" +
		"• `/torrents filter:downloading`\n" +
		"• `/add magnet:?xt=urn:btih:... category:series`\n" +
		"• `/recategorize` — interactive flow to move a torrent to movies/series/anime/default\n" +
		"• `/logs level:error lines:20`\n\n" +
		"**Notes**\n" +
		"• Categories: default, movies, series, anime (each maps to a save path in your env config)\n" +
		"• `/delete` and `/recategorize` use select menus and confirmation buttons\n" +
		"• Seeding stops automatically after download time × `SEEDING_TIME_MULTIPLIER`"

	embed := createInfoEmbed("Help & Commands", content)

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
