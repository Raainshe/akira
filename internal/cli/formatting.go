package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
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

// DiskSpaceInfo represents disk space information for display
type DiskSpaceInfo struct {
	Path        string       `json:"path"`
	Used        int64        `json:"used_bytes"`
	Free        int64        `json:"free_bytes"`
	Total       int64        `json:"total_bytes"`
	UsedStr     string       `json:"used"`
	FreeStr     string       `json:"free"`
	TotalStr    string       `json:"total"`
	Percentage  float64      `json:"percentage"`
	HealthColor *color.Color `json:"-"`
	HealthText  string       `json:"health"`
}

// CreateDiskProgressBar creates a progress bar for disk usage
func CreateDiskProgressBar(percentage float64, width int) string {
	if percentage < 0 {
		percentage = 0
	} else if percentage > 100 {
		percentage = 100
	}

	filled := int((percentage / 100.0) * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("%s %.1f%%", bar, percentage)
}

// GetDiskHealthColor returns color based on disk usage percentage
func GetDiskHealthColor(percentage float64) (*color.Color, string) {
	switch {
	case percentage >= 95.0:
		return ColorError, "🔴 CRITICAL"
	case percentage >= 90.0:
		return color.New(color.FgRed), "🟠 LOW SPACE"
	case percentage >= 80.0:
		return color.New(color.FgYellow), "🟡 WARNING"
	case percentage >= 70.0:
		return color.New(color.FgBlue), "🔵 MODERATE"
	default:
		return ColorSeeding, "🟢 HEALTHY"
	}
}

// ConvertDiskSpaceInfo converts raw disk data to display format
func ConvertDiskSpaceInfo(path string, used, free, total int64) *DiskSpaceInfo {
	percentage := 0.0
	if total > 0 {
		percentage = (float64(used) / float64(total)) * 100.0
	}

	healthColor, healthText := GetDiskHealthColor(percentage)

	return &DiskSpaceInfo{
		Path:        path,
		Used:        used,
		Free:        free,
		Total:       total,
		UsedStr:     FormatBytes(used),
		FreeStr:     FormatBytes(free),
		TotalStr:    FormatBytes(total),
		Percentage:  percentage,
		HealthColor: healthColor,
		HealthText:  healthText,
	}
}

// PrintDiskSpaceInfo prints beautiful disk space information
func PrintDiskSpaceInfo(diskInfos []*DiskSpaceInfo, jsonOutput bool) error {
	if len(diskInfos) == 0 {
		fmt.Println("💾 No disk information available")
		return nil
	}

	// JSON output
	if jsonOutput {
		jsonData, err := json.MarshalIndent(diskInfos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Progress bar output
	fmt.Printf("💾 %s\n\n", ColorHeader.Sprintf("Disk Space Overview"))

	totalUsed := int64(0)
	totalFree := int64(0)
	totalSpace := int64(0)
	criticalCount := 0
	warningCount := 0

	for _, info := range diskInfos {
		// Print path header
		fmt.Printf("📁 %s\n", ColorHeader.Sprint(info.Path))

		// Create progress bar
		progressBar := CreateDiskProgressBar(info.Percentage, 60)

		// Print progress bar
		fmt.Printf("%s\n", progressBar)

		// Print details with health color
		fmt.Printf("%s used • %s free • %s total • %s\n\n",
			info.UsedStr,
			info.FreeStr,
			info.TotalStr,
			info.HealthColor.Sprint(info.HealthText))

		// Accumulate totals
		totalUsed += info.Used
		totalFree += info.Free
		totalSpace += info.Total

		// Count health status
		if info.Percentage >= 95.0 {
			criticalCount++
		} else if info.Percentage >= 80.0 {
			warningCount++
		}
	}

	// Print summary
	if len(diskInfos) > 1 {
		overallPercentage := 0.0
		if totalSpace > 0 {
			overallPercentage = (float64(totalUsed) / float64(totalSpace)) * 100.0
		}

		// Get overall health status
		_, _ = GetDiskHealthColor(overallPercentage)

		fmt.Printf("📊 %s\n", ColorHeader.Sprintf("Summary"))
		fmt.Printf("💾 Total: %s used • %s free • %s total (%.1f%%)\n",
			FormatBytes(totalUsed),
			FormatBytes(totalFree),
			FormatBytes(totalSpace),
			overallPercentage)

		if criticalCount > 0 {
			fmt.Printf("🔴 %s: %d critical paths\n", ColorError.Sprint("ATTENTION"), criticalCount)
		}
		if warningCount > 0 {
			fmt.Printf("🟡 %s: %d paths need attention\n", color.New(color.FgYellow).Sprint("WARNING"), warningCount)
		}
		if criticalCount == 0 && warningCount == 0 {
			fmt.Printf("🟢 %s: All paths healthy\n", ColorSeeding.Sprint("STATUS"))
		}
	}

	return nil
}

// ValidateMagnetURI validates a magnet URI format
func ValidateMagnetURI(magnetURI string) error {
	if magnetURI == "" {
		return fmt.Errorf("magnet URI cannot be empty")
	}

	// Check if it starts with magnet:
	if !strings.HasPrefix(magnetURI, "magnet:") {
		return fmt.Errorf("invalid magnet URI: must start with 'magnet:'")
	}

	// Parse as URL to validate structure
	parsedURL, err := url.Parse(magnetURI)
	if err != nil {
		return fmt.Errorf("invalid magnet URI format: %w", err)
	}

	// Check for required xt parameter (exact topic - the hash)
	query := parsedURL.Query()
	if !query.Has("xt") {
		return fmt.Errorf("invalid magnet URI: missing 'xt' parameter (info hash)")
	}

	xt := query.Get("xt")
	if !strings.HasPrefix(xt, "urn:btih:") {
		return fmt.Errorf("invalid magnet URI: 'xt' parameter must start with 'urn:btih:'")
	}

	// Extract hash and validate length
	hash := strings.TrimPrefix(xt, "urn:btih:")
	if len(hash) != 32 && len(hash) != 40 {
		return fmt.Errorf("invalid magnet URI: info hash must be 32 or 40 characters (got %d)", len(hash))
	}

	return nil
}

// ValidateCategory validates a torrent category
func ValidateCategory(category string) error {
	if category == "" {
		return nil // Empty category is allowed (uses default)
	}

	validCategories := []string{"series", "movies", "anime"}
	categoryLower := strings.ToLower(category)

	for _, valid := range validCategories {
		if categoryLower == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid category '%s'. Valid categories: %v", category, validCategories)
}

// ExtractMagnetInfo extracts useful information from a magnet URI
func ExtractMagnetInfo(magnetURI string) (*MagnetInfo, error) {
	if err := ValidateMagnetURI(magnetURI); err != nil {
		return nil, err
	}

	parsedURL, _ := url.Parse(magnetURI) // Already validated above
	query := parsedURL.Query()

	// Extract info hash
	xt := query.Get("xt")
	hash := strings.TrimPrefix(xt, "urn:btih:")

	// Extract display name
	displayName := query.Get("dn")
	if displayName == "" {
		displayName = "Unknown"
	}

	// Extract trackers
	var trackers []string
	for _, tr := range query["tr"] {
		if tr != "" {
			trackers = append(trackers, tr)
		}
	}

	return &MagnetInfo{
		Hash:        hash,
		DisplayName: displayName,
		Trackers:    trackers,
		OriginalURI: magnetURI,
	}, nil
}

// MagnetInfo represents extracted information from a magnet URI
type MagnetInfo struct {
	Hash        string   `json:"hash"`
	DisplayName string   `json:"display_name"`
	Trackers    []string `json:"trackers"`
	OriginalURI string   `json:"original_uri"`
}

// PrintAddResult prints the result of adding a torrent
func PrintAddResult(success bool, magnetInfo *MagnetInfo, category, customPath string, err error) {
	if !success {
		fmt.Printf("❌ %s\n", ColorError.Sprintf("Failed to add torrent"))
		if err != nil {
			fmt.Printf("   Error: %v\n", err)
		}
		return
	}

	fmt.Printf("✅ %s\n\n", ColorSeeding.Sprintf("Torrent added successfully!"))

	// Show torrent details
	fmt.Printf("📋 %s\n", ColorHeader.Sprintf("Torrent Details"))
	fmt.Printf("   Name: %s\n", magnetInfo.DisplayName)
	fmt.Printf("   Hash: %s\n", magnetInfo.Hash)

	if category != "" {
		fmt.Printf("   Category: %s\n", category)
	}

	if customPath != "" {
		fmt.Printf("   Save Path: %s\n", customPath)
	}

	if len(magnetInfo.Trackers) > 0 {
		fmt.Printf("   Trackers: %d found\n", len(magnetInfo.Trackers))
		for i, tracker := range magnetInfo.Trackers {
			if i >= 3 { // Show only first 3 trackers
				fmt.Printf("   ... and %d more\n", len(magnetInfo.Trackers)-3)
				break
			}
			// Truncate long tracker URLs
			if len(tracker) > 60 {
				tracker = tracker[:57] + "..."
			}
			fmt.Printf("   • %s\n", tracker)
		}
	}

	fmt.Printf("\n💡 Use '%s' to check download progress\n", ColorDownloading.Sprint("akira list"))
}

// PrintDeleteConfirmation prints a confirmation prompt for torrent deletion
func PrintDeleteConfirmation(torrents []*qbittorrent.Torrent, deleteFiles bool) bool {
	if len(torrents) == 0 {
		fmt.Println("❌ No torrents found to delete")
		return false
	}

	fmt.Printf("⚠️  %s\n\n", ColorError.Sprintf("DELETION CONFIRMATION"))

	if len(torrents) == 1 {
		torrent := torrents[0]
		fmt.Printf("📋 Torrent to delete:\n")
		fmt.Printf("   Name: %s\n", torrent.Name)
		fmt.Printf("   Hash: %s\n", torrent.Hash)
		fmt.Printf("   Size: %s\n", FormatBytes(torrent.Size))
		fmt.Printf("   State: %s %s\n", GetStateIcon(string(torrent.State)), GetStateName(string(torrent.State)))
	} else {
		fmt.Printf("📋 %d torrents to delete:\n", len(torrents))
		for i, torrent := range torrents {
			if i >= 5 { // Show only first 5
				fmt.Printf("   ... and %d more torrents\n", len(torrents)-5)
				break
			}
			fmt.Printf("   • %s (%s)\n", torrent.Name, FormatBytes(torrent.Size))
		}
	}

	fmt.Printf("\n🗑️  Action: ")
	if deleteFiles {
		fmt.Printf("%s\n", ColorError.Sprint("DELETE TORRENTS AND FILES"))
		fmt.Printf("   ⚠️  This will permanently delete all downloaded files!\n")
	} else {
		fmt.Printf("%s\n", ColorDownloading.Sprint("DELETE TORRENTS ONLY"))
		fmt.Printf("   ℹ️  Downloaded files will be kept on disk\n")
	}

	fmt.Printf("\n❓ Are you sure you want to continue? (y/N): ")

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// PrintDeleteResult prints the result of torrent deletion
func PrintDeleteResult(successful []string, failed map[string]error, deleteFiles bool) {
	if len(successful) == 0 && len(failed) == 0 {
		fmt.Println("❌ No torrents were processed")
		return
	}

	// Print successful deletions
	if len(successful) > 0 {
		actionText := "deleted"
		if deleteFiles {
			actionText = "deleted (with files)"
		}

		fmt.Printf("✅ %s\n\n", ColorSeeding.Sprintf("Successfully %s %d torrent(s)", actionText, len(successful)))

		for i, hash := range successful {
			if i >= 5 { // Show only first 5
				fmt.Printf("   ... and %d more\n", len(successful)-5)
				break
			}
			fmt.Printf("   ✓ %s\n", hash[:16]+"...") // Show first 16 chars of hash
		}
	}

	// Print failed deletions
	if len(failed) > 0 {
		fmt.Printf("\n❌ %s\n\n", ColorError.Sprintf("Failed to delete %d torrent(s)", len(failed)))

		i := 0
		for hash, err := range failed {
			if i >= 5 { // Show only first 5
				fmt.Printf("   ... and %d more errors\n", len(failed)-5)
				break
			}
			fmt.Printf("   ✗ %s: %v\n", hash[:16]+"...", err)
			i++
		}
	}

	// Summary
	total := len(successful) + len(failed)
	if len(successful) > 0 && len(failed) == 0 {
		fmt.Printf("\n🎉 All %d torrent(s) deleted successfully!\n", total)
	} else if len(successful) == 0 && len(failed) > 0 {
		fmt.Printf("\n💥 All %d deletion(s) failed\n", total)
	} else if len(successful) > 0 && len(failed) > 0 {
		fmt.Printf("\n⚠️  Partial success: %d succeeded, %d failed\n", len(successful), len(failed))
	}
}
