package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// HandleLogsCommand handles the /logs Discord command
func HandleLogsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get command options
	data := i.ApplicationCommandData()

	var level string = "all"
	var lines int = 10

	// Parse options
	for _, option := range data.Options {
		switch option.Name {
		case "level":
			level = option.StringValue()
		case "lines":
			lines = int(option.IntValue())
		}
	}

	// Validate lines
	if lines < 1 || lines > 50 {
		lines = 10
	}

	// Get logs
	logLines, err := getRecentLogs(level, lines)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to get logs: %v", err))
		return
	}

	// Format response
	content := formatLogs(logLines, level)

	// Create embed
	embed := createInfoEmbed("📋 Recent Logs", content)

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("Failed to send logs response: %v\n", err)
	}
}

// getRecentLogs reads recent log lines from bot_activity.log
func getRecentLogs(level string, maxLines int) ([]string, error) {
	// Try to find the log file
	logFile := "bot_activity.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		// Try in logs directory
		logFile = filepath.Join("logs", "bot_activity.log")
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			return []string{"No log file found. Bot activity logs will appear here once the bot is running."}, nil
		}
	}

	// Open log file
	file, err := os.Open(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read all lines
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Reverse to get newest first
	for i, j := 0, len(allLines)-1; i < j; i, j = i+1, j-1 {
		allLines[i], allLines[j] = allLines[j], allLines[i]
	}

	// Filter by level if needed
	var filteredLines []string
	if level != "all" {
		for _, line := range allLines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(level)) {
				filteredLines = append(filteredLines, line)
			}
		}
	} else {
		filteredLines = allLines
	}

	// Limit to requested number of lines
	if len(filteredLines) > maxLines {
		filteredLines = filteredLines[:maxLines]
	}

	return filteredLines, nil
}
