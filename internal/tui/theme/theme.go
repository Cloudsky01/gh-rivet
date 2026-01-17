// Package theme provides centralized styling for the TUI application.
// Inspired by k9s, this provides a consistent visual language across all components.
package theme

import "github.com/charmbracelet/lipgloss"

// Colors defines the color palette for the application
type Colors struct {
	// Primary colors
	Primary   lipgloss.Color // Main accent color (selection, active borders)
	Secondary lipgloss.Color // Secondary accent
	Accent    lipgloss.Color // Highlights, search, special items

	// Text colors
	Text      lipgloss.Color // Normal text
	TextDim   lipgloss.Color // Dimmed/secondary text
	TextMuted lipgloss.Color // Very dim text (hints, disabled)

	// Status colors
	Success lipgloss.Color // Success/completed
	Warning lipgloss.Color // In progress/warning
	Error   lipgloss.Color // Failed/error

	// Background colors
	BgPrimary   lipgloss.Color // Main background
	BgSecondary lipgloss.Color // Secondary background (bars, headers)
	BgHighlight lipgloss.Color // Highlighted items

	// Border colors
	Border       lipgloss.Color // Normal borders
	BorderActive lipgloss.Color // Active/focused borders
}

// Theme contains all styling for the application
type Theme struct {
	Colors Colors

	// Pre-built styles for common elements
	Title        lipgloss.Style
	TitleActive  lipgloss.Style
	Subtitle     lipgloss.Style
	Text         lipgloss.Style
	TextDim      lipgloss.Style
	TextMuted    lipgloss.Style
	Selected     lipgloss.Style
	Cursor       lipgloss.Style
	StatusBar    lipgloss.Style
	HelpBar      lipgloss.Style
	Breadcrumb   lipgloss.Style
	BorderNormal lipgloss.Style
	BorderActive lipgloss.Style
	FilterInput  lipgloss.Style
	FilterPrompt lipgloss.Style

	// Status styles
	StatusSuccess    lipgloss.Style
	StatusWarning    lipgloss.Style
	StatusError      lipgloss.Style
	StatusInProgress lipgloss.Style

	// Icons
	Icons IconSet
}

// IconSet defines the icons used throughout the app
type IconSet struct {
	Folder      string
	FolderOpen  string
	Workflow    string
	Pin         string
	Success     string
	Error       string
	InProgress  string
	Pending     string
	Search      string
	Filter      string
	Refresh     string
	RefreshAuto string
	Back        string
	Selected    string
	Unselected  string
}

// DefaultColors returns the default color palette (dark theme)
func DefaultColors() Colors {
	return Colors{
		// Primary colors - using a blue accent like the original
		Primary:   lipgloss.Color("39"),  // Bright blue
		Secondary: lipgloss.Color("33"),  // Slightly darker blue
		Accent:    lipgloss.Color("141"), // Purple accent for search/special

		// Text colors
		Text:      lipgloss.Color("252"), // Bright white
		TextDim:   lipgloss.Color("245"), // Gray
		TextMuted: lipgloss.Color("240"), // Darker gray

		// Status colors
		Success: lipgloss.Color("42"),  // Green
		Warning: lipgloss.Color("214"), // Orange/yellow
		Error:   lipgloss.Color("196"), // Red

		// Background colors
		BgPrimary:   lipgloss.Color(""),    // Terminal default
		BgSecondary: lipgloss.Color("236"), // Dark gray
		BgHighlight: lipgloss.Color("238"), // Slightly lighter

		// Border colors
		Border:       lipgloss.Color("240"), // Gray
		BorderActive: lipgloss.Color("39"),  // Blue when active
	}
}

// DefaultIcons returns the default icon set
func DefaultIcons() IconSet {
	return IconSet{
		Folder:      "📁",
		FolderOpen:  "📂",
		Workflow:    "⚙️ ",
		Pin:         "📌",
		Success:     "✓",
		Error:       "✗",
		InProgress:  "⟳",
		Pending:     "○",
		Search:      "🔍",
		Filter:      "⏵",
		Refresh:     "↻",
		RefreshAuto: "⟳",
		Back:        "←",
		Selected:    "▸",
		Unselected:  " ",
	}
}

// Default returns the default theme
func Default() *Theme {
	colors := DefaultColors()
	icons := DefaultIcons()

	return &Theme{
		Colors: colors,
		Icons:  icons,

		// Title styles
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Primary),

		TitleActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Text).
			Background(colors.Primary).
			Padding(0, 1),

		Subtitle: lipgloss.NewStyle().
			Foreground(colors.TextDim),

		// Text styles
		Text: lipgloss.NewStyle().
			Foreground(colors.Text),

		TextDim: lipgloss.NewStyle().
			Foreground(colors.TextDim),

		TextMuted: lipgloss.NewStyle().
			Foreground(colors.TextMuted),

		// Selection styles
		Selected: lipgloss.NewStyle().
			Foreground(colors.Primary).
			Bold(true),

		Cursor: lipgloss.NewStyle().
			Foreground(colors.Primary).
			Bold(true),

		// Bar styles
		StatusBar: lipgloss.NewStyle().
			Background(colors.BgSecondary).
			Foreground(colors.TextDim).
			Padding(0, 1),

		HelpBar: lipgloss.NewStyle().
			Background(colors.BgSecondary).
			Foreground(colors.TextMuted).
			Padding(0, 1),

		Breadcrumb: lipgloss.NewStyle().
			Background(colors.BgSecondary).
			Foreground(colors.Accent).
			Padding(0, 1),

		// Border styles
		BorderNormal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.Border),

		BorderActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BorderActive),

		// Filter styles
		FilterInput: lipgloss.NewStyle().
			Foreground(colors.Text).
			Background(colors.BgHighlight).
			Padding(0, 1),

		FilterPrompt: lipgloss.NewStyle().
			Foreground(colors.Accent).
			Bold(true),

		// Status styles
		StatusSuccess: lipgloss.NewStyle().
			Foreground(colors.Success),

		StatusWarning: lipgloss.NewStyle().
			Foreground(colors.Warning),

		StatusError: lipgloss.NewStyle().
			Foreground(colors.Error),

		StatusInProgress: lipgloss.NewStyle().
			Foreground(colors.Warning),
	}
}

// StatusIcon returns the appropriate icon for a workflow status
func (t *Theme) StatusIcon(status, conclusion string) (string, lipgloss.Style) {
	switch status {
	case "completed":
		if conclusion == "success" {
			return t.Icons.Success, t.StatusSuccess
		}
		return t.Icons.Error, t.StatusError
	case "in_progress":
		return t.Icons.InProgress, t.StatusInProgress
	case "queued", "waiting", "pending":
		return t.Icons.Pending, t.TextDim
	default:
		return t.Icons.Pending, t.TextDim
	}
}

// ItemPrefix returns the cursor prefix for an item
func (t *Theme) ItemPrefix(selected bool) string {
	if selected {
		return t.Icons.Selected + " "
	}
	return t.Icons.Unselected + " "
}

// Divider returns a horizontal divider line
func (t *Theme) Divider(width int) string {
	return t.TextMuted.Render(lipgloss.NewStyle().
		Width(width).
		Render(repeatChar("─", width)))
}

// repeatChar repeats a character n times
func repeatChar(char string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += char
	}
	return result
}
