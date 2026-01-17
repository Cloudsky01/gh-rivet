package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

type KeyBinding struct {
	Key         string
	Description string
}

type KeySection struct {
	Title    string
	Bindings []KeyBinding
}

type HelpOverlay struct {
	active   bool
	sections []KeySection
	scroll   int
	width    int
	height   int
	theme    *theme.Theme
}

func NewHelpOverlay(t *theme.Theme) HelpOverlay {
	return HelpOverlay{
		theme:    t,
		sections: DefaultKeySections(),
	}
}

func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height
}

func (h *HelpOverlay) IsActive() bool {
	return h.active
}

func (h *HelpOverlay) Toggle() {
	h.active = !h.active
	h.scroll = 0
}

func (h *HelpOverlay) Close() {
	h.active = false
	h.scroll = 0
}

func (h *HelpOverlay) Update(msg tea.Msg) tea.Cmd {
	if !h.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "?":
			h.Close()
		case "j", "down":
			h.scroll++
			// Bounding will be done in View() to avoid recalculating everything here
		case "k", "up":
			if h.scroll > 0 {
				h.scroll--
			}
		case "g":
			h.scroll = 0
		case "G":
			h.scroll = 999 // Will be bounded in View()
		}
	}
	return nil
}

func (h *HelpOverlay) View() string {
	if !h.active {
		return ""
	}

	overlayWidth := max(60, h.width*70/100)
	overlayHeight := max(20, h.height*80/100)

	keyStyle := h.theme.Selected
	descStyle := h.theme.Text
	sectionStyle := h.theme.Title

	// Build all content lines
	var lines []string
	for _, section := range h.sections {
		lines = append(lines, sectionStyle.Render(section.Title))
		lines = append(lines, "")
		for _, binding := range section.Bindings {
			keyWidth := 20
			key := keyStyle.Render(padRight(binding.Key, keyWidth))
			desc := descStyle.Render(binding.Description)
			lines = append(lines, "  "+key+desc)
		}
		lines = append(lines, "")
	}

	// Calculate visible area (account for title, footer, padding, and borders)
	maxVisible := overlayHeight - 8
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Bound scroll position
	maxScroll := max(0, len(lines)-maxVisible)
	if h.scroll > maxScroll {
		h.scroll = maxScroll
	}
	if h.scroll < 0 {
		h.scroll = 0
	}

	visibleStart := h.scroll
	visibleEnd := min(len(lines), visibleStart+maxVisible)

	// Build the content
	var b strings.Builder

	// Title
	title := h.theme.TitleActive.Render(" Keyboard Shortcuts ")
	b.WriteString(lipgloss.PlaceHorizontal(overlayWidth-4, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Visible lines
	for i := visibleStart; i < visibleEnd; i++ {
		b.WriteString(lines[i])
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	// Footer with scroll indicator and close instruction
	b.WriteString("\n\n")
	if len(lines) > maxVisible {
		scrollInfo := h.theme.TextMuted.Render(
			lipgloss.PlaceHorizontal(overlayWidth-4, lipgloss.Center, "[j/k to scroll]"))
		b.WriteString(scrollInfo)
		b.WriteString("\n")
	}
	closeInfo := h.theme.TextMuted.Render("Press ? or esc to close")
	b.WriteString(lipgloss.PlaceHorizontal(overlayWidth-4, lipgloss.Left, closeInfo))

	// Render content with proper styling
	overlayContent := lipgloss.NewStyle().
		Width(overlayWidth - 4).
		Padding(1, 2).
		Render(b.String())

	overlayBox := h.theme.BorderActive.
		Width(overlayWidth).
		Render(overlayContent)

	// Place the overlay centered on screen
	return lipgloss.Place(
		h.width,
		h.height,
		lipgloss.Center,
		lipgloss.Center,
		overlayBox,
	)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func DefaultKeySections() []KeySection {
	return []KeySection{
		{
			Title: "Global",
			Bindings: []KeyBinding{
				{Key: "q / Ctrl+c", Description: "Quit"},
				{Key: "?", Description: "Toggle help"},
				{Key: ":", Description: "Command palette"},
				{Key: "Ctrl+f", Description: "Global search"},
				{Key: "Tab", Description: "Cycle panels forward"},
				{Key: "Shift+Tab", Description: "Cycle panels backward"},
				{Key: "1", Description: "Toggle sidebar"},
			},
		},
		{
			Title: "Navigation",
			Bindings: []KeyBinding{
				{Key: "j / ↓", Description: "Move down"},
				{Key: "k / ↑", Description: "Move up"},
				{Key: "Enter / l", Description: "Select / Enter group"},
				{Key: "Esc / h", Description: "Go back / Cancel"},
				{Key: "g", Description: "Go to top of list"},
				{Key: "G", Description: "Go to bottom of list"},
			},
		},
		{
			Title: "Panel Focus",
			Bindings: []KeyBinding{
				{Key: "s", Description: "Focus sidebar"},
				{Key: "d", Description: "Focus details"},
				{Key: "r", Description: "Toggle runs panel"},
			},
		},
		{
			Title: "Actions",
			Bindings: []KeyBinding{
				{Key: "p", Description: "Pin/unpin workflow"},
				{Key: "w", Description: "Open in browser"},
				{Key: "Ctrl+r", Description: "Refresh data"},
				{Key: "Ctrl+t", Description: "Toggle auto-refresh"},
			},
		},
		{
			Title: "Filter Mode (/)",
			Bindings: []KeyBinding{
				{Key: "/", Description: "Start filtering"},
				{Key: "↑/↓ or Ctrl+p/n", Description: "Navigate while typing"},
				{Key: "Enter", Description: "Confirm and exit filter"},
				{Key: "Esc", Description: "Clear filter"},
				{Key: "n / N", Description: "Next/Previous match (after filter)"},
			},
		},
		{
			Title: "Command Palette (:)",
			Bindings: []KeyBinding{
				{Key: "Tab", Description: "Autocomplete command"},
				{Key: "Enter", Description: "Execute command"},
				{Key: "Esc", Description: "Close palette"},
			},
		},
	}
}
