package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderMultiPanelLayout renders the new Superfile-inspired layout
func (m MenuModel) renderMultiPanelLayout() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Calculate panel dimensions with minimum widths
	minSidebarWidth := 25
	minDetailsWidth := 30
	minMainWidth := 40

	sidebarWidth := max(minSidebarWidth, m.width/5)        // 20% but at least 25 chars
	detailsWidth := max(minDetailsWidth, m.width/3)        // 33% but at least 30 chars
	mainWidth := m.width - sidebarWidth - detailsWidth - 6 // Rest minus borders
	if mainWidth < minMainWidth {
		// Terminal too small, adjust proportionally
		sidebarWidth = m.width / 4
		detailsWidth = m.width / 4
		mainWidth = m.width - sidebarWidth - detailsWidth - 6
	}

	panelHeight := m.height - 6 // Leave room for breadcrumb and help
	if panelHeight < 10 {
		panelHeight = 10 // Minimum height
	}

	// Adjust if runs panel is visible
	runsHeight := 0
	if m.showRunsPanel {
		runsHeight = max(10, panelHeight/3)
		panelHeight = panelHeight - runsHeight - 3
		if panelHeight < 10 {
			panelHeight = 10
			runsHeight = m.height - 6 - panelHeight - 3
		}
	}

	// Render each panel
	sidebar := m.renderSidebar(sidebarWidth, panelHeight)
	navigation := m.renderNavigation(mainWidth, panelHeight)
	details := m.renderDetails(detailsWidth, panelHeight)

	// Join panels horizontally with borders
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.panelBorder(sidebar, SidebarPanel),
		m.panelBorder(navigation, NavigationPanel),
		m.panelBorder(details, DetailsPanel),
	)

	// Add bottom runs panel if visible
	if m.showRunsPanel {
		runsPanel := m.renderRunsPanel(m.width-2, runsHeight)
		bordered := m.panelBorder(runsPanel, RunsPanel)
		topRow = lipgloss.JoinVertical(lipgloss.Left, topRow, bordered)
	}

	// Add breadcrumb and help bar
	breadcrumb := m.renderBreadcrumb()
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		breadcrumb,
		helpBar,
	)
}
