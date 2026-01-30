package tui

import "github.com/charmbracelet/lipgloss"

// Color palette for TUI components.
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#9b59b6") // Purple
	ColorSecondary = lipgloss.Color("#27ae60") // Green
	ColorMuted     = lipgloss.Color("#95a5a6") // Gray
	ColorWarning   = lipgloss.Color("#f39c12") // Amber
	ColorError     = lipgloss.Color("#e74c3c") // Red

	// Additional colors
	ColorInfo    = lipgloss.Color("#3498db") // Blue
	ColorSuccess = lipgloss.Color("#2ecc71") // Bright green
)

// Text styles for consistent formatting.
var (
	// TitleStyle for main headings.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// SubtitleStyle for section headings.
	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorMuted)

	// SuccessStyle for success messages.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// ErrorStyle for error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// WarningStyle for warning messages.
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// SelectedStyle for selected items in lists.
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// UnselectedStyle for unselected items in lists.
	UnselectedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// HelpStyle for help text.
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// ModelStyle for displaying model names.
	ModelStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// CostStyle for displaying costs.
	CostStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// SpinnerStyle for spinner text.
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// StageStyle for stage names.
	StageStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)
)

// Box styles for layout.
var (
	// BoxStyle for bordered containers.
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2)

	// HighlightBoxStyle for highlighted containers.
	HighlightBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(1, 2)
)
