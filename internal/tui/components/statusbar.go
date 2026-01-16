package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

// StatusBar displays the current location and status
type StatusBar struct {
	width           int
	repository      string
	groupPath       []string
	workflowName    string
	autoRefresh     bool
	refreshInterval int
	loading         bool
	theme           *theme.Theme
}

// NewStatusBar creates a new status bar
func NewStatusBar(t *theme.Theme) StatusBar {
	return StatusBar{
		theme: t,
	}
}

// SetSize sets the width
func (s *StatusBar) SetSize(width int) {
	s.width = width
}

// SetRepository sets the current repository
func (s *StatusBar) SetRepository(repo string) {
	s.repository = repo
}

// SetGroupPath sets the current group path
func (s *StatusBar) SetGroupPath(path []string) {
	s.groupPath = path
}

// SetWorkflow sets the current workflow
func (s *StatusBar) SetWorkflow(name string) {
	s.workflowName = name
}

// SetRefreshStatus sets the auto-refresh status
func (s *StatusBar) SetRefreshStatus(enabled bool, interval int) {
	s.autoRefresh = enabled
	s.refreshInterval = interval
}

// SetLoading sets the loading state
func (s *StatusBar) SetLoading(loading bool) {
	s.loading = loading
}

// View renders the status bar
func (s *StatusBar) View() string {
	// Build breadcrumb
	parts := []string{}
	if s.repository != "" {
		parts = append(parts, "📦 "+s.repository)
	}
	for _, group := range s.groupPath {
		parts = append(parts, group)
	}
	if s.workflowName != "" {
		parts = append(parts, s.workflowName)
	}

	breadcrumb := strings.Join(parts, " > ")

	// Build right side status
	var statusParts []string

	if s.loading {
		statusParts = append(statusParts,
			s.theme.StatusInProgress.Render(s.theme.Icons.InProgress+" Loading"))
	}

	if s.refreshInterval > 0 {
		refreshSymbol := s.theme.Icons.Error
		refreshStyle := s.theme.StatusError
		if s.autoRefresh {
			refreshSymbol = s.theme.Icons.Success
			refreshStyle = s.theme.StatusSuccess
		}
		statusParts = append(statusParts,
			refreshStyle.Render(fmt.Sprintf("%s Auto: %ds", refreshSymbol, s.refreshInterval)))
	}

	status := strings.Join(statusParts, " | ")

	// Calculate spacing
	leftWidth := lipgloss.Width(breadcrumb)
	rightWidth := lipgloss.Width(status)
	spacerWidth := s.width - leftWidth - rightWidth - 4

	var content string
	if spacerWidth > 0 {
		content = s.theme.Breadcrumb.Render(breadcrumb) +
			strings.Repeat(" ", spacerWidth) +
			status
	} else {
		// Truncate breadcrumb if needed
		maxBreadcrumb := s.width - rightWidth - 8
		if maxBreadcrumb > 10 && len(breadcrumb) > maxBreadcrumb {
			breadcrumb = "..." + breadcrumb[len(breadcrumb)-maxBreadcrumb+3:]
		}
		content = s.theme.Breadcrumb.Render(breadcrumb) + " " + status
	}

	return s.theme.StatusBar.
		Width(s.width).
		Render(content)
}

// HelpBar displays context-sensitive keybindings
type HelpBar struct {
	width int
	hints []string
	theme *theme.Theme
}

// NewHelpBar creates a new help bar
func NewHelpBar(t *theme.Theme) HelpBar {
	return HelpBar{
		theme: t,
	}
}

// SetSize sets the width
func (h *HelpBar) SetSize(width int) {
	h.width = width
}

// SetHints sets the keybinding hints
func (h *HelpBar) SetHints(hints []string) {
	h.hints = hints
}

// View renders the help bar
func (h *HelpBar) View() string {
	content := strings.Join(h.hints, " ")
	return h.theme.HelpBar.
		Width(h.width).
		Render(content)
}

// GlobalHints returns the global keybinding hints
func GlobalHints() []string {
	return []string{
		"[q]uit",
		"[tab] cycle",
		"[/] filter",
		"[ctrl+f] search",
		"[?] help",
	}
}

// SidebarHints returns sidebar-specific hints
func SidebarHints() []string {
	return []string{
		"[enter] select",
		"[p] unpin",
		"[w] web",
		"[1] toggle",
	}
}

// NavigationHints returns navigation panel hints
func NavigationHints(inGroup bool) []string {
	hints := []string{"[enter] select"}
	if inGroup {
		hints = append(hints, "[h/esc] back", "[p] pin", "[w] web")
	}
	return hints
}

// DetailsHints returns details panel hints
func DetailsHints(hasWorkflow bool) []string {
	if hasWorkflow {
		return []string{"[r] runs", "[w] web", "[esc] clear"}
	}
	return []string{}
}

// RunsHints returns runs panel hints
func RunsHints() []string {
	return []string{"[w] open run", "[esc] close"}
}
