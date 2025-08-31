package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	Primary   = lipgloss.Color("#00D4AA")
	Secondary = lipgloss.Color("#7C3AED")
	Accent    = lipgloss.Color("#F59E0B")

	// Status colors
	Success = lipgloss.Color("#10B981")
	Warning = lipgloss.Color("#FCD34D") // More vibrant yellow
	Error   = lipgloss.Color("#EF4444")
	Info    = lipgloss.Color("#3B82F6")

	// State colors
	Downloading = lipgloss.Color("#3B82F6")
	Seeding     = lipgloss.Color("#10B981")
	Paused      = lipgloss.Color("#6B7280")
	Completed   = lipgloss.Color("#8B5CF6")

	// UI colors
	Background = lipgloss.Color("#0F172A")
	Surface    = lipgloss.Color("#1E293B")
	Border     = lipgloss.Color("#334155")
	Text       = lipgloss.Color("#F1F5F9")
	TextMuted  = lipgloss.Color("#94A3B8")
	TextDim    = lipgloss.Color("#64748B")
)

// Base styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Background)

	// Layout styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(Surface).
			Padding(0, 2).
			Bold(true)

	SidebarStyle = lipgloss.NewStyle().
			Background(Surface).
			Border(lipgloss.NormalBorder()).
			BorderForeground(Border).
			Padding(1, 2).
			Width(20)

	ContentStyle = lipgloss.NewStyle().
			Background(Background).
			Padding(1, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Background(Surface).
			Padding(0, 1)

	// Component styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				Padding(0, 1)

	TableRowStyle = lipgloss.NewStyle().
			Foreground(Text).
			Padding(0, 1)

	TableRowSelectedStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Primary).
				Padding(0, 1)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(Primary)

	ProgressBarCompleteStyle = lipgloss.NewStyle().
					Foreground(Success)

	// Form styles
	FormLabelStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	FormInputStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Surface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1)

	FormInputFocusedStyle = lipgloss.NewStyle().
				Foreground(Text).
				Background(Surface).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(Background).
			Background(Primary).
			Padding(0, 2).
			Bold(true)

	ButtonSelectedStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Accent).
				Padding(0, 2).
				Bold(true)

	// Card styles
	CardStyle = lipgloss.NewStyle().
			Background(Surface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2).
			Margin(0, 1)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Italic(true)

	KeyStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)
)

// Utility functions
func GetStateColor(state string) lipgloss.Color {
	switch state {
	case "downloading", "metaDL", "stalledDL", "queuedDL", "forcedDL", "checkingDL", "allocating":
		return Downloading
	case "uploading", "stalledUP", "queuedUP", "forcedUP", "checkingUP":
		return Seeding
	case "pausedDL", "pausedUP":
		return Paused
	case "error", "missingFiles", "checkingResumeData":
		return Error
	default:
		return Text
	}
}

func GetStateStyle(state string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(GetStateColor(state))
}

// Layout helpers
func WithBorder(style lipgloss.Style, title string) lipgloss.Style {
	return style.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border)
}

func WithPadding(style lipgloss.Style, vertical, horizontal int) lipgloss.Style {
	return style.Padding(vertical, horizontal)
}

func WithMargin(style lipgloss.Style, vertical, horizontal int) lipgloss.Style {
	return style.Margin(vertical, horizontal)
}
