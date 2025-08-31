package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/raainshe/akira/internal/qbittorrent"
)

// Colors for different torrent states
var (
	ColorDownloading = color.New(color.FgBlue, color.Bold)
	ColorSeeding     = color.New(color.FgGreen, color.Bold)
	ColorPaused      = color.New(color.FgYellow, color.Bold)
	ColorError       = color.New(color.FgRed, color.Bold)
	ColorCompleted   = color.New(color.FgCyan, color.Bold)
	ColorHeader      = color.New(color.FgWhite, color.Bold)
)

// Progress bar characters
const (
	ProgressFull  = "█"
	ProgressEmpty = "░"
	ProgressWidth = 20
)

// TorrentTableRow represents a row in the torrent table
type TorrentTableRow struct {
	Name     string  `json:"name"`
	Size     string  `json:"size"`
	Progress float64 `json:"progress"`
	Speed    string  `json:"speed"`
	ETA      string  `json:"eta"`
	State    string  `json:"state"`
	Ratio    float64 `json:"ratio,omitempty"`
	Category string  `json:"category,omitempty"`
	Hash     string  `json:"hash"`
}

// FormatBytes converts bytes to human readable format
func FormatBytes(bytes int64) string {
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

// FormatSpeed converts bytes per second to human readable format
func FormatSpeed(bytesPerSec int64) string {
	if bytesPerSec == 0 {
		return "0 B/s"
	}
	return FormatBytes(bytesPerSec) + "/s"
}

// FormatDuration converts seconds to human readable duration
func FormatDuration(seconds int64) string {
	if seconds <= 0 {
		return "∞"
	}

	duration := time.Duration(seconds) * time.Second

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm %ds", int(duration.Minutes()), int(duration.Seconds())%60)
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(duration.Hours()), int(duration.Minutes())%60)
	} else {
		days := int(duration.Hours()) / 24
		hours := int(duration.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}

// CreateProgressBar creates a Unicode progress bar
func CreateProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}

	filled := int(progress * float64(width))
	empty := width - filled

	bar := strings.Repeat(ProgressFull, filled) + strings.Repeat(ProgressEmpty, empty)
	return fmt.Sprintf("%s %.1f%%", bar, progress*100)
}

// GetStateColor returns the appropriate color for a torrent state
func GetStateColor(state string) *color.Color {
	stateLower := strings.ToLower(state)
	switch stateLower {
	case strings.ToLower(string(qbittorrent.StateDownloading)),
		strings.ToLower(string(qbittorrent.StateStalledDL)),
		strings.ToLower(string(qbittorrent.StateMetaDL)):
		return ColorDownloading
	case strings.ToLower(string(qbittorrent.StateUploading)),
		strings.ToLower(string(qbittorrent.StateStalledUP)):
		return ColorSeeding
	case strings.ToLower(string(qbittorrent.StatePausedDL)),
		strings.ToLower(string(qbittorrent.StatePausedUP)):
		return ColorPaused
	case strings.ToLower(string(qbittorrent.StateError)),
		strings.ToLower(string(qbittorrent.StateMissingFiles)):
		return ColorError
	case strings.ToLower(string(qbittorrent.StateQueuedDL)),
		strings.ToLower(string(qbittorrent.StateQueuedUP)),
		strings.ToLower(string(qbittorrent.StateAllocating)):
		return ColorCompleted
	default:
		return color.New(color.Reset)
	}
}

// GetStateIcon returns an emoji icon for the torrent state
func GetStateIcon(state string) string {
	stateLower := strings.ToLower(state)
	switch stateLower {
	case strings.ToLower(string(qbittorrent.StateDownloading)),
		strings.ToLower(string(qbittorrent.StateStalledDL)),
		strings.ToLower(string(qbittorrent.StateMetaDL)):
		return "📥"
	case strings.ToLower(string(qbittorrent.StateUploading)),
		strings.ToLower(string(qbittorrent.StateStalledUP)):
		return "🌱"
	case strings.ToLower(string(qbittorrent.StatePausedDL)),
		strings.ToLower(string(qbittorrent.StatePausedUP)):
		return "⏸️"
	case strings.ToLower(string(qbittorrent.StateError)),
		strings.ToLower(string(qbittorrent.StateMissingFiles)):
		return "❌"
	case strings.ToLower(string(qbittorrent.StateQueuedDL)),
		strings.ToLower(string(qbittorrent.StateQueuedUP)),
		strings.ToLower(string(qbittorrent.StateAllocating)):
		return "⏳"
	default:
		return "❓"
	}
}

// GetStateName returns a human-readable state name
func GetStateName(state string) string {
	stateLower := strings.ToLower(state)
	switch stateLower {
	case strings.ToLower(string(qbittorrent.StateDownloading)):
		return "Downloading"
	case strings.ToLower(string(qbittorrent.StateUploading)):
		return "Seeding"
	case strings.ToLower(string(qbittorrent.StateStalledDL)):
		return "Stalled DL"
	case strings.ToLower(string(qbittorrent.StateStalledUP)):
		return "Stalled UP"
	case strings.ToLower(string(qbittorrent.StatePausedDL)):
		return "Paused DL"
	case strings.ToLower(string(qbittorrent.StatePausedUP)):
		return "Paused UP"
	case strings.ToLower(string(qbittorrent.StateError)):
		return "Error"
	case strings.ToLower(string(qbittorrent.StateMissingFiles)):
		return "Missing Files"
	case strings.ToLower(string(qbittorrent.StateQueuedDL)):
		return "Queued DL"
	case strings.ToLower(string(qbittorrent.StateQueuedUP)):
		return "Queued UP"
	case strings.ToLower(string(qbittorrent.StateAllocating)):
		return "Allocating"
	case strings.ToLower(string(qbittorrent.StateMetaDL)):
		return "Metadata DL"
	default:
		return strings.Title(state)
	}
}

// ConvertTorrentToTableRow converts a qBittorrent torrent to table row
func ConvertTorrentToTableRow(torrent *qbittorrent.Torrent) *TorrentTableRow {
	// Calculate ETA
	var eta string
	if torrent.Dlspeed > 0 && torrent.Progress < 1.0 {
		remainingBytes := torrent.Size - int64(float64(torrent.Size)*torrent.Progress)
		etaSeconds := remainingBytes / torrent.Dlspeed
		eta = FormatDuration(etaSeconds)
	} else {
		eta = "∞"
	}

	// Format state with icon
	stateIcon := GetStateIcon(string(torrent.State))
	stateName := GetStateName(string(torrent.State))
	stateText := fmt.Sprintf("%s %s", stateIcon, stateName)

	return &TorrentTableRow{
		Name:     torrent.Name,
		Size:     FormatBytes(torrent.Size),
		Progress: torrent.Progress,
		Speed:    FormatSpeed(torrent.Dlspeed),
		ETA:      eta,
		State:    stateText,
		Ratio:    torrent.Ratio,
		Category: torrent.Category,
		Hash:     torrent.Hash,
	}
}

// PrintTorrentTable prints a beautiful table of torrents
func PrintTorrentTable(torrents []*qbittorrent.Torrent, jsonOutput bool) error {
	if len(torrents) == 0 {
		fmt.Println("📭 No torrents found")
		return nil
	}

	// Convert torrents to table rows
	rows := make([]*TorrentTableRow, len(torrents))
	for i, torrent := range torrents {
		rows[i] = ConvertTorrentToTableRow(torrent)
	}

	// JSON output
	if jsonOutput {
		jsonData, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Custom table output with colors
	fmt.Printf("📊 %s\n\n", ColorHeader.Sprintf("Torrents"))

	// Print header
	fmt.Printf("%-40s %-8s %-20s %-10s %-10s %s\n",
		ColorHeader.Sprint("Name"),
		ColorHeader.Sprint("Size"),
		ColorHeader.Sprint("Progress"),
		ColorHeader.Sprint("Speed"),
		ColorHeader.Sprint("ETA"),
		ColorHeader.Sprint("State"))

	fmt.Println(strings.Repeat("─", 100))

	// Add rows with colors
	for _, row := range rows {
		// Create progress bar
		progressBar := CreateProgressBar(row.Progress, 15)

		// Truncate name if too long
		name := row.Name
		if len(name) > 37 {
			name = name[:34] + "..."
		}

		// Print row with colors
		fmt.Printf("%-40s %-8s %-20s %-10s %-10s %s\n",
			name,
			row.Size,
			progressBar,
			row.Speed,
			row.ETA,
			row.State)
	}

	fmt.Println()

	// Print summary
	downloading := 0
	seeding := 0
	paused := 0
	errored := 0

	for _, torrent := range torrents {
		state := strings.ToLower(string(torrent.State))
		switch {
		case strings.Contains(state, "downloading") || strings.Contains(state, "stalleddl"):
			downloading++
		case strings.Contains(state, "uploading") || strings.Contains(state, "stalledup"):
			seeding++
		case strings.Contains(state, "paused"):
			paused++
		case strings.Contains(state, "error"):
			errored++
		}
	}

	// Print summary
	fmt.Printf("📊 %s\n", ColorHeader.Sprintf("Summary"))
	fmt.Printf("📥 %s: %d  🌱 %s: %d  ⏸️  %s: %d  ❌ %s: %d  📋 %s: %d\n",
		ColorDownloading.Sprint("Downloading"), downloading,
		ColorSeeding.Sprint("Seeding"), seeding,
		ColorPaused.Sprint("Paused"), paused,
		ColorError.Sprint("Errored"), errored,
		ColorHeader.Sprint("Total"), len(torrents))

	return nil
}
