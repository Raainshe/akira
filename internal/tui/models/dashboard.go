package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raainshe/akira/internal/qbittorrent"
	"github.com/raainshe/akira/internal/tui/shared"
	"github.com/raainshe/akira/internal/tui/styles"
)

// DashboardModel represents the dashboard view
type DashboardModel struct {
	scrollOffset int
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel() DashboardModel {
	return DashboardModel{}
}

// Update implements tea.Model for dashboard
func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "ctrl+down":
			m.scrollOffset++
		case "ctrl+home":
			m.scrollOffset = 0
		case "ctrl+end":
			// Will be set in View when we know the content height
			m.scrollOffset = 999 // Temporary large value
		case "pageup":
			// Scroll up by 5 lines
			if m.scrollOffset > 4 {
				m.scrollOffset -= 5
			} else {
				m.scrollOffset = 0
			}
		case "pagedown":
			// Scroll down by 5 lines
			m.scrollOffset += 5
		}
	}
	return m, nil
}

// Note: CachedData and AppStats are defined in app.go to avoid circular imports

// View renders the dashboard view
func (m DashboardModel) View(cache interface{}, width, height int) string {
	// Type assert the cache
	appCache, ok := cache.(*shared.CachedData)
	if !ok {
		return fmt.Sprintf("Error: Invalid cache type. Expected *CachedData, got %T", cache)
	}
	if appCache == nil {
		return "Loading dashboard data..."
	}

	var sections []string

	// Overview section
	sections = append(sections, m.renderOverview(appCache, width))

	// Recent activity section
	sections = append(sections, m.renderRecentActivity(appCache, width))

	// System status section
	sections = append(sections, m.renderSystemStatus(appCache, width))

	// Join all content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Apply scrolling
	return m.applyScrolling(fullContent, width, height)
}

func (m DashboardModel) renderOverview(cache *shared.CachedData, width int) string {
	title := "📊 Torrent Overview"

	var stats []string

	if cache.Stats != nil {
		// Create styles for colored output
		primaryStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		downloadingStyle := lipgloss.NewStyle().Foreground(styles.Downloading).Bold(true)
		seedingStyle := lipgloss.NewStyle().Foreground(styles.Seeding).Bold(true)
		pausedStyle := lipgloss.NewStyle().Foreground(styles.Paused).Bold(true)
		errorStyle := lipgloss.NewStyle().Foreground(styles.Error).Bold(true)

		stats = append(stats,
			fmt.Sprintf("📋 Total Torrents: %s", primaryStyle.Render(fmt.Sprintf("%d", cache.Stats.TotalTorrents))),
			fmt.Sprintf("📥 Downloading: %s", downloadingStyle.Render(fmt.Sprintf("%d", cache.Stats.ActiveDownloads))),
			fmt.Sprintf("🌱 Seeding: %s", seedingStyle.Render(fmt.Sprintf("%d", cache.Stats.ActiveSeeds))),
			fmt.Sprintf("⏸️  Paused: %s", pausedStyle.Render(fmt.Sprintf("%d", cache.Stats.PausedTorrents))),
			fmt.Sprintf("❌ Errored: %s", errorStyle.Render(fmt.Sprintf("%d", cache.Stats.ErroredTorrents))),
		)

		// Speed information
		downSpeed := m.formatSpeed(cache.Stats.TotalDownSpeed)
		upSpeed := m.formatSpeed(cache.Stats.TotalUpSpeed)

		infoStyle := lipgloss.NewStyle().Foreground(styles.Info).Bold(true)
		successStyle := lipgloss.NewStyle().Foreground(styles.Success).Bold(true)

		stats = append(stats, "")
		stats = append(stats,
			fmt.Sprintf("⬇️  Download Speed: %s", infoStyle.Render(downSpeed)),
			fmt.Sprintf("⬆️  Upload Speed: %s", successStyle.Render(upSpeed)),
		)

		if !cache.Stats.LastUpdate.IsZero() {
			mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
			stats = append(stats, "")
			stats = append(stats,
				fmt.Sprintf("🕒 Last Updated: %s",
					mutedStyle.Render(cache.Stats.LastUpdate.Format("15:04:05"))),
			)
		}
	} else {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		stats = append(stats, mutedStyle.Render("Loading statistics..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, stats...)

	// Use 95% of available width for cards to leave some margin
	cardWidth := int(float64(width) * 0.95)
	cardStyle := styles.CardStyle.Width(cardWidth).Height(12)
	return styles.WithBorder(cardStyle, title).Render(content)
}

func (m DashboardModel) renderRecentActivity(cache *shared.CachedData, width int) string {
	title := "🕒 Recent Activity"

	var activities []string

	if len(cache.Torrents) > 0 {
		// Show recent downloads and uploads
		var recentDownloads []string
		var recentSeeds []string

		for _, torrent := range cache.Torrents {
			if len(recentDownloads) < 3 && m.isDownloading(torrent.State) {
				progress := float64(torrent.Progress) * 100
				downloadingStyle := lipgloss.NewStyle().Foreground(styles.Downloading)
				recentDownloads = append(recentDownloads,
					fmt.Sprintf("📥 %s - %s",
						m.truncateString(torrent.Name, 25),
						downloadingStyle.Render(fmt.Sprintf("%.1f%%", progress)),
					),
				)
			}

			if len(recentSeeds) < 3 && m.isSeeding(torrent.State) {
				ratio := float64(torrent.Ratio)
				seedingStyle := lipgloss.NewStyle().Foreground(styles.Seeding)
				recentSeeds = append(recentSeeds,
					fmt.Sprintf("🌱 %s - %s",
						m.truncateString(torrent.Name, 25),
						seedingStyle.Render(fmt.Sprintf("%.2f", ratio)),
					),
				)
			}
		}

		if len(recentDownloads) > 0 {
			labelStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
			activities = append(activities, labelStyle.Render("Active Downloads:"))
			activities = append(activities, recentDownloads...)
		}

		if len(recentSeeds) > 0 {
			if len(recentDownloads) > 0 {
				activities = append(activities, "")
			}
			labelStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
			activities = append(activities, labelStyle.Render("Active Seeds:"))
			activities = append(activities, recentSeeds...)
		}

		if len(activities) == 0 {
			mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
			activities = append(activities, mutedStyle.Render("No recent activity"))
		}
	} else {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		activities = append(activities, mutedStyle.Render("Loading torrents..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, activities...)

	// Use 95% of available width for cards to leave some margin
	cardWidth := int(float64(width) * 0.95)
	cardStyle := styles.CardStyle.Width(cardWidth).Height(12)
	return styles.WithBorder(cardStyle, title).Render(content)
}

func (m DashboardModel) renderSystemStatus(cache *shared.CachedData, width int) string {
	title := "💾 System Status"

	var status []string

	// Disk usage information
	if len(cache.DiskInfo) > 0 {
		labelStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		status = append(status, labelStyle.Render("Disk Usage:"))

		for path, diskInfo := range cache.DiskInfo {
			if diskInfo != nil {
				percentage := float64(diskInfo.Used) / float64(diskInfo.Total) * 100

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

				progressBar := m.createProgressBar(percentage, 20)
				healthStyle := lipgloss.NewStyle().Foreground(healthColor).Bold(true)

				status = append(status,
					fmt.Sprintf("%s: %s %.1f%% %s",
						m.truncateString(path, 15),
						progressBar,
						percentage,
						healthStyle.Render(healthText),
					),
				)
			}
		}
	}

	// Seeding service status
	if cache.SeedingInfo != nil {
		if len(status) > 0 {
			status = append(status, "")
		}

		labelStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		status = append(status, labelStyle.Render("Seeding Service:"))

		// We'll assume the service is running if we have seeding info
		successStyle := lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
		primaryStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)

		status = append(status,
			fmt.Sprintf("Status: %s", successStyle.Render("🟢 RUNNING")),
			fmt.Sprintf("Tracked Torrents: %s",
				primaryStyle.Render(fmt.Sprintf("%d", cache.SeedingInfo.TrackedTorrents))),
			fmt.Sprintf("Active Seeding: %s",
				primaryStyle.Render(fmt.Sprintf("%d", cache.SeedingInfo.ActiveSeeding))),
		)

		if cache.SeedingInfo.OverdueSeeding > 0 {
			warningStyle := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
			status = append(status,
				fmt.Sprintf("Overdue: %s",
					warningStyle.Render(fmt.Sprintf("%d", cache.SeedingInfo.OverdueSeeding))),
			)
		}
	}

	if len(status) == 0 {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		status = append(status, mutedStyle.Render("Loading system status..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, status...)

	// Use 95% of available width for cards to leave some margin
	cardWidth := int(float64(width) * 0.95)
	cardStyle := styles.CardStyle.Width(cardWidth).Height(8)
	return styles.WithBorder(cardStyle, title).Render(content)
}

// Utility functions
func (m DashboardModel) formatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}

	const unit = 1024
	if bytesPerSecond < unit {
		return fmt.Sprintf("%d B/s", bytesPerSecond)
	}

	div, exp := int64(unit), 0
	for n := bytesPerSecond / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB/s",
		float64(bytesPerSecond)/float64(div), "KMGTPE"[exp])
}

func (m DashboardModel) truncateString(s string, maxLen int) string {
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

// applyScrolling applies scrolling to content that exceeds the available height
func (m DashboardModel) applyScrolling(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)

	// Reserve space for scroll indicators and info (3 lines total)
	reservedHeight := 3
	availableHeight := height - reservedHeight

	// If content fits within available height, no scrolling needed
	if contentHeight <= availableHeight {
		return content
	}

	// Ensure scroll offset is within bounds
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	if m.scrollOffset >= contentHeight-availableHeight {
		m.scrollOffset = contentHeight - availableHeight
	}

	// Extract visible lines (respecting reserved height for indicators)
	start := m.scrollOffset
	end := start + availableHeight
	if end > contentHeight {
		end = contentHeight
	}

	visibleLines := lines[start:end]

	// Build the final output with indicators
	var finalLines []string

	// Add up arrow indicator if needed
	if m.scrollOffset > 0 {
		indicator := lipgloss.NewStyle().Foreground(styles.Primary).Render("↑ More above (Ctrl+↑/PageUp)")
		finalLines = append(finalLines, indicator)
	}

	// Add the visible content
	finalLines = append(finalLines, visibleLines...)

	// Add down arrow indicator if needed
	if m.scrollOffset < contentHeight-availableHeight {
		indicator := lipgloss.NewStyle().Foreground(styles.Primary).Render("↓ More below (Ctrl+↓/PageDown)")
		finalLines = append(finalLines, indicator)
	}

	// Add scroll position indicator (always show when scrolling)
	scrollInfo := fmt.Sprintf("Scroll: %d/%d (Ctrl+Home/End to jump)", m.scrollOffset+1, contentHeight)
	scrollInfoStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(scrollInfo)
	finalLines = append(finalLines, scrollInfoStyle)

	// Ensure we don't exceed the total height
	if len(finalLines) > height {
		// Trim from the end to fit within height
		finalLines = finalLines[:height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, finalLines...)
}

func (m DashboardModel) createProgressBar(percentage float64, width int) string {
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

func (m DashboardModel) isDownloading(state qbittorrent.TorrentState) bool {
	switch state {
	case qbittorrent.StateDownloading, qbittorrent.StateMetaDL, qbittorrent.StateStalledDL,
		qbittorrent.StateQueuedDL, qbittorrent.StateForcedDL, qbittorrent.StateCheckingDL,
		qbittorrent.StateAllocating:
		return true
	default:
		return false
	}
}

func (m DashboardModel) isSeeding(state qbittorrent.TorrentState) bool {
	switch state {
	case qbittorrent.StateUploading, qbittorrent.StateStalledUP, qbittorrent.StateQueuedUP,
		qbittorrent.StateForcedUP, qbittorrent.StateCheckingUP:
		return true
	default:
		return false
	}
}

/*
func (m DashboardModel) renderOverview_unused(cache interface{}, width int) string {
	title := styles.TableHeaderStyle.Render("📊 Overview")

	var stats []string

	if cache.Stats != nil {
		primaryStyle := lipgloss.NewStyle().Foreground(styles.Primary)
		downloadingStyle := lipgloss.NewStyle().Foreground(styles.Downloading)
		seedingStyle := lipgloss.NewStyle().Foreground(styles.Seeding)
		pausedStyle := lipgloss.NewStyle().Foreground(styles.Paused)
		errorStyle := lipgloss.NewStyle().Foreground(styles.Error)

		stats = append(stats,
			fmt.Sprintf("📋 Total Torrents: %s", primaryStyle.Render(fmt.Sprintf("%d", cache.Stats.TotalTorrents))),
			fmt.Sprintf("📥 Downloading: %s", downloadingStyle.Render(fmt.Sprintf("%d", cache.Stats.ActiveDownloads))),
			fmt.Sprintf("🌱 Seeding: %s", seedingStyle.Render(fmt.Sprintf("%d", cache.Stats.ActiveSeeds))),
			fmt.Sprintf("⏸️  Paused: %s", pausedStyle.Render(fmt.Sprintf("%d", cache.Stats.PausedTorrents))),
			fmt.Sprintf("❌ Errored: %s", errorStyle.Render(fmt.Sprintf("%d", cache.Stats.ErroredTorrents))),
		)

		// Speed information
		downSpeed := m.formatSpeed(cache.Stats.TotalDownSpeed)
		upSpeed := m.formatSpeed(cache.Stats.TotalUpSpeed)

		infoStyle := lipgloss.NewStyle().Foreground(styles.Info)
		successStyle := lipgloss.NewStyle().Foreground(styles.Success)
		mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)

		stats = append(stats, "")
		stats = append(stats,
			fmt.Sprintf("⬇️  Download Speed: %s", infoStyle.Render(downSpeed)),
			fmt.Sprintf("⬆️  Upload Speed: %s", successStyle.Render(upSpeed)),
		)

		if !cache.Stats.LastUpdate.IsZero() {
			stats = append(stats, "")
			stats = append(stats,
				fmt.Sprintf("🕒 Last Updated: %s",
					mutedStyle.Render(cache.Stats.LastUpdate.Format("15:04:05"))),
			)
		}
	} else {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		stats = append(stats, mutedStyle.Render("Loading statistics..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, stats...)

	return styles.WithBorder(
		styles.CardStyle.Width(width/2-2),
		"Overview",
	).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}

func (m DashboardModel) renderRecentActivity(cache *CachedData, width int) string {
	title := styles.TableHeaderStyle.Render("🕒 Recent Activity")

	var activities []string

	if len(cache.Torrents) > 0 {
		// Show recent downloads and uploads
		var recentDownloads []string
		var recentSeeds []string

		for _, torrent := range cache.Torrents {
			if len(recentDownloads) < 3 && (torrent.State.IsDownloading()) {
				progress := float64(torrent.Progress) * 100
				recentDownloads = append(recentDownloads,
					fmt.Sprintf("📥 %s - %s",
						m.truncateString(torrent.Name, 30),
						styles.Downloading.Render(fmt.Sprintf("%.1f%%", progress)),
					),
				)
			}

			if len(recentSeeds) < 3 && (torrent.State.IsSeeding()) {
				ratio := float64(torrent.Ratio)
				recentSeeds = append(recentSeeds,
					fmt.Sprintf("🌱 %s - %s",
						m.truncateString(torrent.Name, 30),
						styles.Seeding.Render(fmt.Sprintf("%.2f", ratio)),
					),
				)
			}
		}

		if len(recentDownloads) > 0 {
			activities = append(activities, styles.FormLabelStyle.Render("Active Downloads:"))
			activities = append(activities, recentDownloads...)
		}

		if len(recentSeeds) > 0 {
			if len(recentDownloads) > 0 {
				activities = append(activities, "")
			}
			activities = append(activities, styles.FormLabelStyle.Render("Active Seeds:"))
			activities = append(activities, recentSeeds...)
		}

		if len(activities) == 0 {
			activities = append(activities, styles.TextMuted.Render("No recent activity"))
		}
	} else {
		activities = append(activities, styles.TextMuted.Render("Loading torrents..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, activities...)

	return styles.WithBorder(
		styles.CardStyle.Width(width/2-2),
		"Recent Activity",
	).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}

func (m DashboardModel) renderSystemStatus(cache *CachedData, width int) string {
	title := styles.TableHeaderStyle.Render("💾 System Status")

	var status []string

	// Disk usage information
	if len(cache.DiskInfo) > 0 {
		status = append(status, styles.FormLabelStyle.Render("Disk Usage:"))

		for path, diskInfo := range cache.DiskInfo {
			if diskInfo != nil {
				percentage := float64(diskInfo.Used) / float64(diskInfo.Total) * 100

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

				progressBar := m.createProgressBar(percentage, 20)

				status = append(status,
					fmt.Sprintf("%s: %s %.1f%% %s",
						m.truncateString(path, 20),
						progressBar,
						percentage,
						healthColor.Render(healthText),
					),
				)
			}
		}
	}

	// Seeding service status
	if cache.SeedingInfo != nil {
		if len(status) > 0 {
			status = append(status, "")
		}

		status = append(status, styles.FormLabelStyle.Render("Seeding Service:"))

		if cache.SeedingInfo.IsRunning {
			status = append(status,
				fmt.Sprintf("Status: %s", styles.Success.Render("🟢 RUNNING")),
				fmt.Sprintf("Tracked Torrents: %s",
					styles.Primary.Render(fmt.Sprintf("%d", len(cache.SeedingInfo.TrackedTorrents)))),
			)
		} else {
			status = append(status,
				fmt.Sprintf("Status: %s", styles.Error.Render("🔴 STOPPED")),
			)
		}
	}

	if len(status) == 0 {
		status = append(status, styles.TextMuted.Render("Loading system status..."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, status...)

	return styles.WithBorder(
		styles.CardStyle.Width(width-4),
		"System Status",
	).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}

// Utility functions
func (m DashboardModel) formatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}

	const unit = 1024
	if bytesPerSecond < unit {
		return fmt.Sprintf("%d B/s", bytesPerSecond)
	}

	div, exp := int64(unit), 0
	for n := bytesPerSecond / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB/s",
		float64(bytesPerSecond)/float64(div), "KMGTPE"[exp])
}



func (m DashboardModel) createProgressBar(percentage float64, width int) string {
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

	return color.Render(bar)
}

// CachedData represents the application cache (defined here to avoid import cycles)
type CachedData struct {
	Torrents    []interface{} // Will be []qbittorrent.Torrent in actual use
	Stats       *AppStats
	DiskInfo    map[string]*DiskSpace
	SeedingInfo *SeedingInfo
	LastFetch   map[string]time.Time
}

type AppStats struct {
	TotalTorrents   int
	ActiveDownloads int
	ActiveSeeds     int
	PausedTorrents  int
	ErroredTorrents int
	TotalDownSpeed  int64
	TotalUpSpeed    int64
	LastUpdate      time.Time
}

type DiskSpace struct {
	Used  int64
	Free  int64
	Total int64
}

type SeedingInfo struct {
	IsRunning        bool
	TrackedTorrents  map[string]interface{} // Will be map[string]*SeedingTorrentStatus
}
*/
