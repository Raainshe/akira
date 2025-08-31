package models

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
	"github.com/raainshe/akira/internal/tui/shared"
	"github.com/raainshe/akira/internal/tui/styles"
)

// Placeholder models for other views
// These will be implemented in subsequent steps

// TorrentsModel represents the torrent list view
type TorrentsModel struct {
	selectedIndex int
	scrollOffset  int
	filter        string
	sortBy        string
	sortDesc      bool
}

func NewTorrentsModel() TorrentsModel {
	return TorrentsModel{
		sortBy: "name", // Default sort by name
	}
}

func (m TorrentsModel) Update(msg tea.Msg) (TorrentsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				if m.selectedIndex < m.scrollOffset {
					m.scrollOffset = m.selectedIndex
				}
			}
		case "down", "j":
			// We'll handle max in the View method when we know torrent count
			m.selectedIndex++
		case "home", "g":
			m.selectedIndex = 0
			m.scrollOffset = 0
		case "end", "G":
			// Will be handled in View when we know torrent count
		case "n":
			// Sort by name
			if m.sortBy == "name" {
				m.sortDesc = !m.sortDesc
			} else {
				m.sortBy = "name"
				m.sortDesc = false
			}
		case "s":
			// Sort by size
			if m.sortBy == "size" {
				m.sortDesc = !m.sortDesc
			} else {
				m.sortBy = "size"
				m.sortDesc = true // Default descending for size
			}
		case "p":
			// Sort by progress
			if m.sortBy == "progress" {
				m.sortDesc = !m.sortDesc
			} else {
				m.sortBy = "progress"
				m.sortDesc = true // Default descending for progress
			}
		case "d":
			// Sort by download speed
			if m.sortBy == "dlspeed" {
				m.sortDesc = !m.sortDesc
			} else {
				m.sortBy = "dlspeed"
				m.sortDesc = true // Default descending for speed
			}
		}
	}
	return m, nil
}

func (m TorrentsModel) View(cache interface{}, width, height int) string {
	// Type assert the cache
	appCache, ok := cache.(*shared.CachedData)
	if !ok {
		return fmt.Sprintf("Error: Invalid cache type. Expected *CachedData, got %T", cache)
	}
	if appCache == nil {
		return "Loading torrent data..."
	}

	if len(appCache.Torrents) == 0 {
		return "No torrents found.\n\nAdd a torrent using the 'Add Magnet' view (press 3) or the CLI command:\nakira add <magnet-uri>"
	}

	// Sort torrents
	torrents := make([]qbittorrent.Torrent, len(appCache.Torrents))
	copy(torrents, appCache.Torrents)
	m.sortTorrents(torrents)

	// Adjust selection bounds
	if m.selectedIndex >= len(torrents) {
		m.selectedIndex = len(torrents) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}

	// Calculate visible area
	visibleHeight := height - 6 // Reserve space for header, help text, etc.
	if m.selectedIndex >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.selectedIndex - visibleHeight + 1
	}
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}

	// Build the table
	var content []string

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	header := fmt.Sprintf("%-30s %-8s %-8s %-10s %-8s %-12s %s",
		"Name", "Size", "Progress", "Speed", "ETA", "State", "Ratio")
	content = append(content, headerStyle.Render(header))
	content = append(content, strings.Repeat("─", width-4))

	// Torrent rows
	endIndex := m.scrollOffset + visibleHeight
	if endIndex > len(torrents) {
		endIndex = len(torrents)
	}

	for i := m.scrollOffset; i < endIndex; i++ {
		torrent := torrents[i]
		row := m.formatTorrentRow(torrent, i == m.selectedIndex, width-4)
		content = append(content, row)
	}

	// Add padding if needed
	for len(content) < visibleHeight+2 {
		content = append(content, "")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	help := "↑/↓: Navigate • N: Sort by Name • S: Sort by Size • P: Sort by Progress • D: Sort by Speed"
	content = append(content, "")
	content = append(content, helpStyle.Render(help))

	// Status
	statusStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	sortIndicator := "↑"
	if m.sortDesc {
		sortIndicator = "↓"
	}
	status := fmt.Sprintf("Showing %d-%d of %d torrents • Sorted by %s %s • Selected: %d",
		m.scrollOffset+1, endIndex, len(torrents), m.sortBy, sortIndicator, m.selectedIndex+1)
	content = append(content, statusStyle.Render(status))

	// Ensure we don't exceed the total height
	if len(content) > height {
		content = content[:height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// sortTorrents sorts the torrent slice based on current sort settings
func (m TorrentsModel) sortTorrents(torrents []qbittorrent.Torrent) {
	sort.Slice(torrents, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case "name":
			less = strings.ToLower(torrents[i].Name) < strings.ToLower(torrents[j].Name)
		case "size":
			less = torrents[i].Size < torrents[j].Size
		case "progress":
			less = torrents[i].Progress < torrents[j].Progress
		case "dlspeed":
			less = torrents[i].Dlspeed < torrents[j].Dlspeed
		default:
			less = strings.ToLower(torrents[i].Name) < strings.ToLower(torrents[j].Name)
		}

		if m.sortDesc {
			return !less
		}
		return less
	})
}

// formatTorrentRow formats a single torrent row for display
func (m TorrentsModel) formatTorrentRow(torrent qbittorrent.Torrent, isSelected bool, maxWidth int) string {
	// Format basic info
	name := m.truncateString(torrent.Name, 28)
	size := m.formatBytes(torrent.Size)
	progress := fmt.Sprintf("%.1f%%", torrent.Progress*100)
	speed := m.formatSpeed(torrent.Dlspeed)
	eta := m.formatETA(torrent.Eta)
	state := m.formatState(torrent.State)
	ratio := fmt.Sprintf("%.2f", torrent.Ratio)

	// Create progress bar
	progressBar := m.createProgressBar(torrent.Progress*100, 10)

	// Format the row
	row := fmt.Sprintf("%-28s %-8s %s %-8s %-8s %-8s %-12s %s",
		name, size, progressBar, progress, speed, eta, state, ratio)

	// Apply selection styling
	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Foreground(styles.Background).
			Background(styles.Primary).
			Bold(true)
		return selectedStyle.Render(row)
	}

	// Apply state-based coloring
	stateStyle := styles.GetStateStyle(string(torrent.State))
	return stateStyle.Render(row)
}

// Helper functions
func (m TorrentsModel) truncateString(s string, maxLen int) string {
	// Use lipgloss.Width to account for character width variations (emojis, CJK, etc.)
	if lipgloss.Width(s) <= maxLen {
		return s
	}

	// Truncate based on actual display width
	for i := len(s) - 1; i >= 0; i-- {
		if lipgloss.Width(s[:i]) <= maxLen-3 {
			return s[:i] + "..."
		}
	}

	return "..."
}

func (m TorrentsModel) formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

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

func (m TorrentsModel) formatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}
	return m.formatBytes(bytesPerSecond) + "/s"
}

func (m TorrentsModel) formatETA(eta int64) string {
	if eta <= 0 || eta == 8640000 { // qBittorrent uses 8640000 for infinite
		return "∞"
	}

	duration := time.Duration(eta) * time.Second
	if duration.Hours() >= 24 {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if duration.Hours() >= 1 {
		return fmt.Sprintf("%.0fh", duration.Hours())
	} else {
		return fmt.Sprintf("%.0fm", duration.Minutes())
	}
}

func (m TorrentsModel) formatState(state qbittorrent.TorrentState) string {
	switch state {
	case qbittorrent.StateDownloading:
		return "📥 Down"
	case qbittorrent.StateUploading:
		return "🌱 Seed"
	case qbittorrent.StateMetaDL:
		return "📥 Meta"
	case qbittorrent.StatePausedDL, qbittorrent.StatePausedUP:
		return "⏸️  Paused"
	case qbittorrent.StateError:
		return "❌ Error"
	case qbittorrent.StateStalledDL:
		return "📥 Stall"
	case qbittorrent.StateStalledUP:
		return "🌱 Stall"
	case qbittorrent.StateQueuedDL:
		return "📥 Queue"
	case qbittorrent.StateQueuedUP:
		return "🌱 Queue"
	case qbittorrent.StateCheckingDL, qbittorrent.StateCheckingUP:
		return "🔍 Check"
	default:
		return string(state)
	}
}

func (m TorrentsModel) createProgressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	var color lipgloss.Color
	switch {
	case percentage >= 100:
		color = styles.Success
	case percentage >= 50:
		color = styles.Primary
	case percentage >= 25:
		color = styles.Warning
	default:
		color = styles.Error
	}

	style := lipgloss.NewStyle().Foreground(color)
	return style.Render(bar)
}

// Note: CachedData and AppStats are defined in dashboard.go to avoid circular imports

// AddMagnetModel represents the add magnet form view (removed)
type AddMagnetModel struct{}

func NewAddMagnetModel() AddMagnetModel {
	return AddMagnetModel{}
}

func (m AddMagnetModel) Update(msg tea.Msg) (AddMagnetModel, tea.Cmd) {
	return m, nil
}

func (m AddMagnetModel) View(cache interface{}, width, height int) string {
	return "🚧 Add Magnet functionality removed from TUI\n\nUse the CLI command instead:\n  akira add <magnet-uri> [--category <category>]"
}

// SeedingModel represents the seeding management view
type SeedingModel struct {
	selectedTorrent int
	scrollOffset    int
}

func NewSeedingModel() SeedingModel {
	return SeedingModel{}
}

func (m SeedingModel) Update(msg tea.Msg) (SeedingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedTorrent > 0 {
				m.selectedTorrent--
			}
		case "down", "j":
			// Will be handled in View when we know torrent count
			m.selectedTorrent++
		case "home", "g":
			m.selectedTorrent = 0
		case "end", "G":
			// Will be handled in View when we know torrent count
		}
	}
	return m, nil
}

func (m SeedingModel) View(cache interface{}, width, height int) string {
	// Reserve space for title, help text, and spacing (4 lines total)
	reservedHeight := 4
	availableHeight := height - reservedHeight

	// Type assert the cache
	appCache, ok := cache.(*shared.CachedData)
	if !ok || appCache == nil {
		return "Loading seeding data..."
	}

	if appCache.SeedingInfo == nil {
		return "No seeding information available.\n\nMake sure the seeding service is running."
	}

	var content []string

	// Title
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	content = append(content, titleStyle.Render("🌱 Seeding Management"))

	// Service status
	content = append(content, m.renderServiceStatus(appCache.SeedingInfo, width-4))

	// Tracked torrents
	if len(appCache.SeedingInfo.Details) > 0 {
		content = append(content, m.renderTrackedTorrents(appCache.SeedingInfo, width-4, availableHeight-2))
	} else {
		noDataStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		content = append(content, noDataStyle.Render("No tracked torrents found."))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	help := "↑/↓: Navigate • Home/End: Jump to start/end"
	content = append(content, helpStyle.Render(help))

	// Ensure we don't exceed the total height
	if len(content) > height {
		content = content[:height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m SeedingModel) renderServiceStatus(info *core.SeedingStatus, width int) string {
	var lines []string

	// Service status
	statusStyle := lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	lines = append(lines, fmt.Sprintf("Service Status: %s", statusStyle.Render("🟢 RUNNING")))

	// Statistics
	statsStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	lines = append(lines, fmt.Sprintf("Tracked Torrents: %s", statsStyle.Render(fmt.Sprintf("%d", info.TrackedTorrents))))
	lines = append(lines, fmt.Sprintf("Active Seeding: %s", statsStyle.Render(fmt.Sprintf("%d", info.ActiveSeeding))))
	lines = append(lines, fmt.Sprintf("Completed Seeding: %s", statsStyle.Render(fmt.Sprintf("%d", info.CompletedSeeding))))

	if info.OverdueSeeding > 0 {
		warningStyle := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
		lines = append(lines, fmt.Sprintf("Overdue: %s", warningStyle.Render(fmt.Sprintf("%d", info.OverdueSeeding))))
	}

	// Time statistics
	timeStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Total Download Time: %s", timeStyle.Render(m.formatDuration(info.TotalDownloadTime))))
	lines = append(lines, fmt.Sprintf("Total Seeding Time: %s", timeStyle.Render(m.formatDuration(info.TotalSeedingTime))))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m SeedingModel) renderTrackedTorrents(info *core.SeedingStatus, width, maxHeight int) string {
	var content []string

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	content = append(content, headerStyle.Render("Tracked Torrents:"))
	content = append(content, "")

	// Adjust selection bounds
	if m.selectedTorrent >= len(info.Details) {
		m.selectedTorrent = len(info.Details) - 1
	}
	if m.selectedTorrent < 0 {
		m.selectedTorrent = 0
	}

	// Calculate visible area
	visibleHeight := maxHeight - 3 // Reserve space for header and help
	if m.selectedTorrent >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.selectedTorrent - visibleHeight + 1
	}
	if m.selectedTorrent < m.scrollOffset {
		m.scrollOffset = m.selectedTorrent
	}

	// Torrent entries
	endIndex := m.scrollOffset + visibleHeight
	if endIndex > len(info.Details) {
		endIndex = len(info.Details)
	}

	index := 0
	for hash, status := range info.Details {
		if index >= m.scrollOffset && index < endIndex {
			content = append(content, m.renderTorrentStatus(hash, status, index == m.selectedTorrent, width))
		}
		index++
	}

	// Status
	statusStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	content = append(content, "")
	content = append(content, statusStyle.Render(fmt.Sprintf("Showing %d-%d of %d tracked torrents • Selected: %d",
		m.scrollOffset+1, endIndex, len(info.Details), m.selectedTorrent+1)))

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m SeedingModel) renderTorrentStatus(hash string, status *core.SeedingTorrentStatus, isSelected bool, width int) string {
	// Format the torrent info
	name := m.truncateString(status.Name, 30)
	downloadTime := m.formatDuration(status.DownloadDuration)
	seedingTime := m.formatDuration(status.SeedingDuration)
	timeRemaining := m.formatDuration(status.TimeRemaining)

	// Status indicator
	var statusIcon string
	var statusColor lipgloss.Color
	if status.IsOverdue {
		statusIcon = "⚠️"
		statusColor = styles.Warning
	} else if status.TimeRemaining <= 0 {
		statusIcon = "✅"
		statusColor = styles.Success
	} else {
		statusIcon = "⏳"
		statusColor = styles.Info
	}

	line := fmt.Sprintf("%s %s | %s | DL: %s | Seed: %s | Remaining: %s",
		statusIcon, name, hash[:8], downloadTime, seedingTime, timeRemaining)

	// Apply selection styling
	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Foreground(styles.Background).
			Background(styles.Primary).
			Bold(true)
		return selectedStyle.Render(line)
	}

	// Apply status-based coloring
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))
	return statusStyle.Render(line)
}

func (m SeedingModel) formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	if d.Hours() >= 24 {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if d.Hours() >= 1 {
		return fmt.Sprintf("%.0fh", d.Hours())
	} else if d.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
}

func (m SeedingModel) truncateString(s string, maxLen int) string {
	// Use lipgloss.Width to account for character width variations (emojis, CJK, etc.)
	if lipgloss.Width(s) <= maxLen {
		return s
	}

	// Truncate based on actual display width
	for i := len(s) - 1; i >= 0; i-- {
		if lipgloss.Width(s[:i]) <= maxLen-3 {
			return s[:i] + "..."
		}
	}

	return "..."
}

// DiskModel represents the disk usage view
type DiskModel struct {
	selectedPath int
}

func NewDiskModel() DiskModel {
	return DiskModel{}
}

func (m DiskModel) Update(msg tea.Msg) (DiskModel, tea.Cmd) {
	return m, nil
}

func (m DiskModel) View(cache interface{}, width, height int) string {
	// Reserve space for title and help text (3 lines total)
	reservedHeight := 3
	availableHeight := height - reservedHeight

	// Type assert the cache
	appCache, ok := cache.(*shared.CachedData)
	if !ok || appCache == nil {
		return "Loading disk usage data..."
	}

	if len(appCache.DiskInfo) == 0 {
		return "No disk information available.\n\nMake sure the configured paths exist and are accessible."
	}

	var content []string

	// Title
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	content = append(content, titleStyle.Render("💾 Disk Usage"))

	// Disk information for each path
	pathCount := 0
	for path, diskInfo := range appCache.DiskInfo {
		if diskInfo != nil && pathCount < availableHeight/4 { // Limit paths to fit in available space
			section := m.renderDiskInfo(path, diskInfo, width-4)
			content = append(content, section)
			pathCount++
		}
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	help := "Disk usage updates every 15 seconds"
	content = append(content, helpStyle.Render(help))

	// Ensure we don't exceed the total height
	if len(content) > height {
		content = content[:height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m DiskModel) renderDiskInfo(path string, diskInfo *core.DiskInfo, width int) string {
	percentage := float64(diskInfo.Used) / float64(diskInfo.Total) * 100

	// Health status
	var healthColor lipgloss.Color
	var healthText string

	switch {
	case percentage < 70:
		healthColor = styles.Success
		healthText = "🟢 HEALTHY"
	case percentage < 85:
		healthColor = styles.Warning
		healthText = "🟡 WARNING"
	case percentage < 95:
		healthColor = styles.Error
		healthText = "🟠 CRITICAL"
	default:
		healthColor = styles.Error
		healthText = "🔴 FULL"
	}

	// Format sizes
	totalStr := m.formatBytes(diskInfo.Total)
	usedStr := m.formatBytes(diskInfo.Used)
	freeStr := m.formatBytes(diskInfo.Free)

	// Create progress bar
	progressBar := m.createDiskProgressBar(percentage, 50)

	// Build the section
	var lines []string

	// Path header
	pathStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	lines = append(lines, pathStyle.Render(fmt.Sprintf("📁 %s", path)))

	// Progress bar with percentage
	lines = append(lines, fmt.Sprintf("%s %.1f%%", progressBar, percentage))

	// Size information
	infoStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	lines = append(lines, infoStyle.Render(fmt.Sprintf("Used: %s • Free: %s • Total: %s", usedStr, freeStr, totalStr)))

	// Health status
	healthStyle := lipgloss.NewStyle().Foreground(healthColor).Bold(true)
	lines = append(lines, fmt.Sprintf("Status: %s", healthStyle.Render(healthText)))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m DiskModel) createDiskProgressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	var color lipgloss.Color
	switch {
	case percentage < 70:
		color = styles.Success
	case percentage < 85:
		color = styles.Warning
	default:
		color = styles.Error
	}

	style := lipgloss.NewStyle().Foreground(color)
	return style.Render(bar)
}

func (m DiskModel) formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

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

// LogEntry represents a JSON log entry
type LogEntry struct {
	Component string                 `json:"component"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	Time      string                 `json:"time"`
	Extra     map[string]interface{} `json:"-"`
}

// LogsModel represents the logs viewer
type LogsModel struct {
	scrollOffset int
	selectedLine int
	filterLevel  string
	followMode   bool
	lastLogCount int
}

func NewLogsModel() LogsModel {
	return LogsModel{
		filterLevel: "all",
		followMode:  true, // Start in follow mode by default
	}
}

func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedLine > 0 {
				m.selectedLine--
			}
		case "down", "j":
			// Will be handled in View when we know line count
			m.selectedLine++
		case "home", "g":
			m.selectedLine = 0
		case "end", "G":
			// Will be handled in View when we know line count
		case "f":
			// Toggle follow mode
			m.followMode = !m.followMode
		case "l":
			// Cycle through filter levels
			levels := []string{"all", "error", "warn", "info", "debug"}
			currentIndex := -1
			for i, level := range levels {
				if level == m.filterLevel {
					currentIndex = i
					break
				}
			}
			if currentIndex >= 0 {
				m.filterLevel = levels[(currentIndex+1)%len(levels)]
			}
		}
	}
	return m, nil
}

func (m LogsModel) View(cache interface{}, width, height int) string {
	// Reserve space for header, filter, status, and help text (5 lines total)
	reservedHeight := 5
	availableHeight := height - reservedHeight

	// Get real log content from file
	logs := m.getSimulatedLogs()
	filteredLogs := m.filterLogs(logs, m.filterLevel)

	// Handle follow mode - auto-scroll to top (newest logs) if new logs appear
	if m.followMode && len(filteredLogs) > m.lastLogCount {
		m.selectedLine = 0
		m.scrollOffset = 0
	}
	m.lastLogCount = len(filteredLogs)

	// Build the content
	var content []string

	// Title and filter info
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	filterStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	content = append(content, titleStyle.Render("📋 Application Logs"))

	// Status line with filter and follow mode
	statusLine := fmt.Sprintf("Filter: %s | Follow: %s | Newest First", m.filterLevel, map[bool]string{true: "ON", false: "OFF"}[m.followMode])
	content = append(content, filterStyle.Render(statusLine))

	if len(filteredLogs) == 0 {
		noDataStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		content = append(content, noDataStyle.Render("No logs found for the selected filter level."))
	} else {
		// Adjust selection bounds
		if m.selectedLine >= len(filteredLogs) {
			m.selectedLine = len(filteredLogs) - 1
		}
		if m.selectedLine < 0 {
			m.selectedLine = 0
		}

		// Calculate visible area (respecting reserved height)
		if m.selectedLine >= m.scrollOffset+availableHeight {
			m.scrollOffset = m.selectedLine - availableHeight + 1
		}
		if m.selectedLine < m.scrollOffset {
			m.scrollOffset = m.selectedLine
		}

		// Display logs
		endIndex := m.scrollOffset + availableHeight
		if endIndex > len(filteredLogs) {
			endIndex = len(filteredLogs)
		}

		for i := m.scrollOffset; i < endIndex; i++ {
			logLine := filteredLogs[i]
			if i == m.selectedLine {
				selectedStyle := lipgloss.NewStyle().
					Foreground(styles.Background).
					Background(styles.Primary).
					Bold(true)
				content = append(content, selectedStyle.Render(logLine))
			} else {
				// Apply color coding based on log level
				coloredLine := m.colorCodeLogLine(logLine)
				content = append(content, coloredLine)
			}
		}
	}

	// Status line
	statusStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	if len(filteredLogs) > 0 {
		status := fmt.Sprintf("Showing %d-%d of %d log entries • Selected: %d",
			m.scrollOffset+1, m.scrollOffset+len(content)-2, len(filteredLogs), m.selectedLine+1)
		content = append(content, statusStyle.Render(status))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	help := "↑/↓: Navigate • F: Toggle follow • L: Change filter • Home/End: Jump to newest/oldest"
	content = append(content, helpStyle.Render(help))

	// Ensure we don't exceed the total height
	if len(content) > height {
		content = content[:height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m LogsModel) getSimulatedLogs() []string {
	// Read from actual log file
	logFile := "bot_activity.log"

	// Try to read the log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		// If file doesn't exist or can't be read, return a helpful message
		return []string{
			fmt.Sprintf("[ERROR] Could not read log file '%s': %v", logFile, err),
			"",
			"[INFO] Make sure the log file exists and is readable.",
			"[INFO] The application will create this file when logging is enabled.",
		}
	}

	// Split content into lines and filter out empty lines
	lines := strings.Split(string(content), "\n")
	var logLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as JSON log entry
		formattedLine, err := m.parseJSONLogLine(line)
		if err != nil {
			// If JSON parsing fails, use the raw line
			logLines = append(logLines, line)
		} else {
			logLines = append(logLines, formattedLine)
		}
	}

	// If no log lines found, return a message
	if len(logLines) == 0 {
		return []string{
			"[INFO] Log file is empty or contains no valid log entries.",
			"[INFO] Logs will appear here as the application runs.",
		}
	}

	// Reverse the order to show newest logs first
	for i, j := 0, len(logLines)-1; i < j; i, j = i+1, j-1 {
		logLines[i], logLines[j] = logLines[j], logLines[i]
	}

	return logLines
}

func (m LogsModel) parseJSONLogLine(line string) (string, error) {
	// Parse the JSON log entry
	var logEntry LogEntry
	err := json.Unmarshal([]byte(line), &logEntry)
	if err != nil {
		return "", err
	}

	// Parse time
	var timeStr string
	if logEntry.Time != "" {
		if t, err := time.Parse(time.RFC3339, logEntry.Time); err == nil {
			timeStr = t.Format("15:04:05")
		} else {
			timeStr = logEntry.Time
		}
	}

	// Format the log line - use full level names for better readability
	var levelDisplay string
	switch strings.ToLower(logEntry.Level) {
	case "warn", "warning":
		levelDisplay = "WARNING"
	case "error":
		levelDisplay = "ERROR"
	case "info":
		levelDisplay = "INFO"
	case "debug":
		levelDisplay = "DEBUG"
	default:
		levelDisplay = strings.ToUpper(logEntry.Level)
	}

	component := logEntry.Component
	if component == "" {
		component = "main"
	}

	// Create the formatted log line
	formatted := fmt.Sprintf("%s [%s] %s: %s", timeStr, levelDisplay, component, logEntry.Message)

	return formatted, nil
}

func (m LogsModel) filterLogs(logs []string, level string) []string {
	if level == "all" {
		return logs
	}

	var filtered []string

	// Map filter levels to the actual log level patterns
	levelMap := map[string]string{
		"error":   "ERROR",
		"warn":    "WARNING",
		"warning": "WARNING", // Handle both "warn" and "warning"
		"info":    "INFO",
		"debug":   "DEBUG",
	}

	searchPattern := fmt.Sprintf("[%s]", levelMap[level])
	for _, log := range logs {
		if strings.Contains(log, searchPattern) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func (m LogsModel) colorCodeLogLine(logLine string) string {
	// Apply color coding based on log level
	if strings.Contains(logLine, "[ERROR]") {
		errorStyle := lipgloss.NewStyle().Foreground(styles.Error)
		return errorStyle.Render(logLine)
	} else if strings.Contains(logLine, "[WARNING]") {
		warningStyle := lipgloss.NewStyle().Foreground(styles.Warning)
		return warningStyle.Render(logLine)
	} else if strings.Contains(logLine, "[INFO]") {
		infoStyle := lipgloss.NewStyle().Foreground(styles.Info)
		return infoStyle.Render(logLine)
	} else if strings.Contains(logLine, "[DEBUG]") {
		debugStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		return debugStyle.Render(logLine)
	}

	// Default color for unknown levels
	return logLine
}
