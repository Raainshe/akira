package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
	"github.com/raainshe/akira/internal/tui/models"
	"github.com/raainshe/akira/internal/tui/shared"
	"github.com/raainshe/akira/internal/tui/styles"
)

// ViewType represents different TUI views
type ViewType int

const (
	DashboardView ViewType = iota
	TorrentsView
	SeedingView
	DiskView
	LogsView
)

// String returns the string representation of ViewType
func (v ViewType) String() string {
	switch v {
	case DashboardView:
		return "dashboard"
	case TorrentsView:
		return "torrents"
	case SeedingView:
		return "seeding"
	case DiskView:
		return "disk"
	case LogsView:
		return "logs"
	default:
		return "unknown"
	}
}

// Messages for the TUI
type (
	// Tick message for periodic updates
	tickMsg time.Time

	// Data update messages
	torrentsUpdatedMsg struct {
		torrents []qbittorrent.Torrent
		err      error
	}

	statsUpdatedMsg struct {
		stats *shared.AppStats
		err   error
	}

	diskUpdatedMsg struct {
		diskInfo map[string]*core.DiskInfo
		err      error
	}

	seedingUpdatedMsg struct {
		status *core.SeedingStatus
		err    error
	}

	// Navigation messages
	switchViewMsg ViewType

	// Control messages
	pauseUpdatesMsg  struct{}
	resumeUpdatesMsg struct{}
	forceRefreshMsg  struct{}
)

// Note: CachedData and AppStats are now defined in types.go

// AppModel is the main TUI model
type AppModel struct {
	// Context and services
	ctx            context.Context
	config         *config.Config
	torrentService *core.TorrentService
	diskService    *core.DiskService
	seedingService *core.SeedingService
	qbClient       *qbittorrent.Client

	// UI state
	currentView ViewType
	width       int
	height      int
	ready       bool

	// Data and caching
	cache         *shared.CachedData
	updatesPaused bool
	lastTick      time.Time

	// Sub-models
	dashboard models.DashboardModel
	torrents  models.TorrentsModel
	seeding   models.SeedingModel
	disk      models.DiskModel
	logs      models.LogsModel

	// Error handling
	lastError      error
	errorDisplayed time.Time
}

// NewAppModel creates a new TUI application model
func NewAppModel(ctx context.Context, config *config.Config, torrentService *core.TorrentService,
	diskService *core.DiskService, seedingService *core.SeedingService, qbClient *qbittorrent.Client) *AppModel {

	return &AppModel{
		ctx:            ctx,
		config:         config,
		torrentService: torrentService,
		diskService:    diskService,
		seedingService: seedingService,
		qbClient:       qbClient,
		currentView:    DashboardView,
		cache: &shared.CachedData{
			LastFetch: map[string]time.Time{
				"torrents": time.Time{}, // Zero time to force immediate fetch
				"stats":    time.Time{},
				"disk":     time.Time{},
				"seeding":  time.Time{},
			},
		},
		// Initialize sub-models
		dashboard: models.NewDashboardModel(),
		torrents:  models.NewTorrentsModel(),
		seeding:   models.NewSeedingModel(),
		disk:      models.NewDiskModel(),
		logs:      models.NewLogsModel(),
	}
}

// Init implements tea.Model
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		// Initial data fetch
		m.fetchTorrentsCmd(),
		m.fetchStatsCmd(),
		m.fetchDiskCmd(),
		m.fetchSeedingCmd(),
		// Start periodic updates
		m.tickCmd(),
	)
}

// Update implements tea.Model
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "p":
			if m.updatesPaused {
				m.updatesPaused = false
				cmds = append(cmds, m.tickCmd())
			} else {
				m.updatesPaused = true
			}

		case "r":
			if !m.updatesPaused {
				cmds = append(cmds, tea.Batch(
					m.fetchTorrentsCmd(),
					m.fetchStatsCmd(),
					m.fetchDiskCmd(),
					m.fetchSeedingCmd(),
				))
			}

		case "1":
			m.currentView = DashboardView
		case "2":
			m.currentView = TorrentsView
		case "3":
			m.currentView = SeedingView
		case "4":
			m.currentView = DiskView
		case "5":
			m.currentView = LogsView

		case "tab":
			// Cycle through views
			m.currentView = ViewType((int(m.currentView) + 1) % 5)
		}

	case tickMsg:
		if !m.updatesPaused {
			m.lastTick = time.Time(msg)

			// Determine what needs updating based on intervals
			var updateCmds []tea.Cmd

			if m.shouldUpdateTorrents() {
				updateCmds = append(updateCmds, m.fetchTorrentsCmd())
			}

			if m.shouldUpdateStats() {
				updateCmds = append(updateCmds, m.fetchStatsCmd())
			}

			if m.shouldUpdateDisk() {
				updateCmds = append(updateCmds, m.fetchDiskCmd())
			}

			if m.shouldUpdateSeeding() {
				updateCmds = append(updateCmds, m.fetchSeedingCmd())
			}

			// Schedule next tick
			updateCmds = append(updateCmds, m.tickCmd())

			cmds = append(cmds, tea.Batch(updateCmds...))
		}

	case torrentsUpdatedMsg:
		if msg.err != nil {
			m.lastError = msg.err
			m.errorDisplayed = time.Now()
		} else {
			m.cache.Torrents = msg.torrents
			m.cache.LastFetch["torrents"] = time.Now()

			// Update stats from torrents
			m.updateStatsFromTorrents()
		}

	case statsUpdatedMsg:
		if msg.err != nil {
			m.lastError = msg.err
			m.errorDisplayed = time.Now()
		} else {
			m.cache.Stats = msg.stats
			m.cache.LastFetch["stats"] = time.Now()
		}

	case diskUpdatedMsg:
		if msg.err != nil {
			m.lastError = msg.err
			m.errorDisplayed = time.Now()
		} else {
			m.cache.DiskInfo = msg.diskInfo
			m.cache.LastFetch["disk"] = time.Now()
		}

	case seedingUpdatedMsg:
		if msg.err != nil {
			m.lastError = msg.err
			m.errorDisplayed = time.Now()
		} else {
			m.cache.SeedingInfo = msg.status
			m.cache.LastFetch["seeding"] = time.Now()
		}
	}

	// Update current view model
	switch m.currentView {
	case DashboardView:
		m.dashboard, cmd = m.dashboard.Update(msg)
		cmds = append(cmds, cmd)
	case TorrentsView:
		m.torrents, cmd = m.torrents.Update(msg)
		cmds = append(cmds, cmd)

	case SeedingView:
		m.seeding, cmd = m.seeding.Update(msg)
		cmds = append(cmds, cmd)
	case DiskView:
		m.disk, cmd = m.disk.Update(msg)
		cmds = append(cmds, cmd)
	case LogsView:
		m.logs, cmd = m.logs.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m AppModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Build the main layout
	header := m.renderHeader()
	sidebar := m.renderSidebar()
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	// Combine layout
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		main,
		statusBar,
	)
}

// Render components
func (m AppModel) renderHeader() string {
	title := "🌟 Akira - Torrent Management"

	var status string
	warningStyle := lipgloss.NewStyle().Foreground(styles.Warning)
	successStyle := lipgloss.NewStyle().Foreground(styles.Success)

	if m.updatesPaused {
		status = warningStyle.Render("⏸️  PAUSED")
	} else {
		status = successStyle.Render("🔄 LIVE")
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center,
		title,
		lipgloss.NewStyle().Width(m.width-len(title)-len(status)-4).Render(""),
		status,
	)

	return styles.HeaderStyle.Width(m.width).Render(headerContent)
}

// calculateSidebarWidth calculates the actual width the sidebar will take up
func (m AppModel) calculateSidebarWidth() int {
	// Create a temporary sidebar to measure its actual width
	var items []string

	views := []struct {
		view ViewType
		icon string
		name string
		key  string
	}{
		{DashboardView, "📊", "Dashboard", "1"},
		{TorrentsView, "📋", "Torrents", "2"},
		{SeedingView, "🌱", "Seeding", "3"},
		{DiskView, "💾", "Disk Usage", "4"},
		{LogsView, "📜", "Logs", "5"},
	}

	for _, v := range views {
		item := fmt.Sprintf("[%s] %s %s", v.key, v.icon, v.name)
		if m.currentView == v.view {
			item = styles.TableRowSelectedStyle.Render(item)
		} else {
			item = styles.TableRowStyle.Render(item)
		}
		items = append(items, item)
	}

	navigation := lipgloss.JoinVertical(lipgloss.Left, items...)

	// Render the sidebar to get its actual width
	sidebar := styles.WithBorder(
		styles.SidebarStyle.Height(m.height-4),
		"Navigation",
	).Render(navigation)

	// Use lipgloss's width calculation which accounts for scaling and character width
	maxWidth := lipgloss.Width(sidebar)
	return maxWidth
}

func (m AppModel) renderSidebar() string {
	var items []string

	views := []struct {
		view ViewType
		icon string
		name string
		key  string
	}{
		{DashboardView, "📊", "Dashboard", "1"},
		{TorrentsView, "📋", "Torrents", "2"},
		{SeedingView, "🌱", "Seeding", "3"},
		{DiskView, "💾", "Disk Usage", "4"},
		{LogsView, "📜", "Logs", "5"},
	}

	for _, v := range views {
		item := fmt.Sprintf("[%s] %s %s", v.key, v.icon, v.name)
		if m.currentView == v.view {
			item = styles.TableRowSelectedStyle.Render(item)
		} else {
			item = styles.TableRowStyle.Render(item)
		}
		items = append(items, item)
	}

	navigation := lipgloss.JoinVertical(lipgloss.Left, items...)

	return styles.WithBorder(
		styles.SidebarStyle.Height(m.height-4),
		"Navigation",
	).Render(navigation)
}

func (m AppModel) renderContent() string {
	sidebarWidth := m.calculateSidebarWidth()
	contentWidth := m.width - sidebarWidth - 4 // Subtract calculated sidebar width and some padding
	contentHeight := m.height - 4              // Subtract header and status bar

	var content string

	switch m.currentView {
	case DashboardView:
		content = m.dashboard.View(m.cache, contentWidth, contentHeight)
	case TorrentsView:
		content = m.torrents.View(m.cache, contentWidth, contentHeight)

	case SeedingView:
		content = m.seeding.View(m.cache, contentWidth, contentHeight)
	case DiskView:
		content = m.disk.View(m.cache, contentWidth, contentHeight)
	case LogsView:
		content = m.logs.View(m.cache, contentWidth, contentHeight)
	default:
		content = "Unknown view"
	}

	return styles.ContentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}

func (m AppModel) renderStatusBar() string {
	var parts []string

	// Current time
	parts = append(parts, time.Now().Format("15:04:05"))

	// Update status
	if m.updatesPaused {
		parts = append(parts, "Updates: PAUSED")
	} else {
		parts = append(parts, fmt.Sprintf("Last update: %s",
			time.Since(m.lastTick).Truncate(time.Second)))
	}

	// Error display
	if m.lastError != nil && time.Since(m.errorDisplayed) < 5*time.Second {
		errorStyle := lipgloss.NewStyle().Foreground(styles.Error)
		parts = append(parts, errorStyle.Render(fmt.Sprintf("Error: %v", m.lastError)))
	}

	// Help text
	help := "Tab: Switch • P: Pause • R: Refresh • Q: Quit • Ctrl+↑/↓: Scroll"

	statusContent := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.JoinHorizontal(lipgloss.Left, parts...),
		lipgloss.NewStyle().Width(m.width-len(lipgloss.JoinHorizontal(lipgloss.Left, parts...))-len(help)-4).Render(""),
		styles.HelpStyle.Render(help),
	)

	return styles.StatusBarStyle.Width(m.width).Render(statusContent)
}

// Update interval logic
func (m AppModel) getUpdateInterval() time.Duration {
	if m.cache.Stats == nil {
		return 2 * time.Second
	}

	activeDownloads := m.cache.Stats.ActiveDownloads

	switch {
	case activeDownloads > 5:
		return 1 * time.Second // Fast updates for many active downloads
	case activeDownloads > 0:
		return 2 * time.Second // Normal updates for some downloads
	default:
		return 5 * time.Second // Slow updates when idle
	}
}

func (m AppModel) shouldUpdateTorrents() bool {
	lastFetch := m.cache.LastFetch["torrents"]
	if lastFetch.IsZero() {
		return true // Always update if never fetched
	}
	return time.Since(lastFetch) > m.getUpdateInterval()
}

func (m AppModel) shouldUpdateStats() bool {
	lastFetch := m.cache.LastFetch["stats"]
	if lastFetch.IsZero() {
		return true
	}
	return time.Since(lastFetch) > (m.getUpdateInterval() + time.Second)
}

func (m AppModel) shouldUpdateDisk() bool {
	lastFetch := m.cache.LastFetch["disk"]
	if lastFetch.IsZero() {
		return true
	}
	return time.Since(lastFetch) > 15*time.Second
}

func (m AppModel) shouldUpdateSeeding() bool {
	lastFetch := m.cache.LastFetch["seeding"]
	if lastFetch.IsZero() {
		return true
	}
	return time.Since(lastFetch) > 5*time.Second
}

// Command generators
func (m AppModel) tickCmd() tea.Cmd {
	return tea.Tick(m.getUpdateInterval(), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m AppModel) fetchTorrentsCmd() tea.Cmd {
	return func() tea.Msg {
		torrents, err := m.torrentService.GetTorrents(m.ctx, &core.TorrentFilter{})
		return torrentsUpdatedMsg{torrents: torrents, err: err}
	}
}

func (m AppModel) fetchStatsCmd() tea.Cmd {
	return func() tea.Msg {
		// This will be calculated from torrents for now
		return statsUpdatedMsg{stats: m.cache.Stats, err: nil}
	}
}

func (m AppModel) fetchDiskCmd() tea.Cmd {
	return func() tea.Msg {
		diskInfo := make(map[string]*core.DiskInfo)

		// Get disk space for all configured paths
		paths := []string{
			m.config.QBittorrent.SavePaths.Default,
			m.config.QBittorrent.SavePaths.Series,
			m.config.QBittorrent.SavePaths.Movies,
			m.config.QBittorrent.SavePaths.Anime,
		}

		for _, path := range paths {
			if path != "" {
				space, err := m.diskService.GetDiskSpace(m.ctx, path)
				if err == nil {
					diskInfo[path] = space
				}
			}
		}

		return diskUpdatedMsg{diskInfo: diskInfo, err: nil}
	}
}

func (m AppModel) fetchSeedingCmd() tea.Cmd {
	return func() tea.Msg {
		status, err := m.seedingService.GetSeedingStatus(m.ctx)
		return seedingUpdatedMsg{status: status, err: err}
	}
}

// updateStatsFromTorrents calculates stats from torrent data
func (m *AppModel) updateStatsFromTorrents() {
	if len(m.cache.Torrents) == 0 {
		return
	}

	stats := &shared.AppStats{
		LastUpdate: time.Now(),
	}

	for _, torrent := range m.cache.Torrents {
		stats.TotalTorrents++
		stats.TotalDownSpeed += torrent.Dlspeed
		stats.TotalUpSpeed += torrent.Upspeed

		switch torrent.State {
		case qbittorrent.StateDownloading, qbittorrent.StateMetaDL, qbittorrent.StateStalledDL,
			qbittorrent.StateQueuedDL, qbittorrent.StateForcedDL, qbittorrent.StateCheckingDL,
			qbittorrent.StateAllocating:
			stats.ActiveDownloads++
		case qbittorrent.StateUploading, qbittorrent.StateStalledUP, qbittorrent.StateQueuedUP,
			qbittorrent.StateForcedUP, qbittorrent.StateCheckingUP:
			stats.ActiveSeeds++
		case qbittorrent.StatePausedDL, qbittorrent.StatePausedUP:
			stats.PausedTorrents++
		case qbittorrent.StateError, qbittorrent.StateMissingFiles, qbittorrent.StateCheckingResumeData:
			stats.ErroredTorrents++
		}
	}

	m.cache.Stats = stats
}
