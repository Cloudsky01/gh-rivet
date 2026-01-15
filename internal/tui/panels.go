package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PanelType represents which panel is currently focused
type PanelType int

const (
	SidebarPanel PanelType = iota
	NavigationPanel
	DetailsPanel
	RunsPanel
)

// renderSidebar renders the left sidebar panel with pinned workflows
func (m MenuModel) renderSidebar(width, height int) string {
	var content strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render("📌 Pinned Workflows")

	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", width-4))
	content.WriteString("\n")

	// Show filter input if filtering is active
	if m.sidebarFilterActive {
		filterPrompt := lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			Render(fmt.Sprintf("Filter: %s█", m.sidebarFilterInput))
		content.WriteString(filterPrompt)
		content.WriteString("\n")
		content.WriteString(strings.Repeat("─", width-4))
		content.WriteString("\n")
	} else if m.sidebarFilterInput != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("🔍 Filtered by: %q (press [esc] to clear)", m.sidebarFilterInput))
		content.WriteString(filterInfo)
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Get all items and apply custom filter
	allItems := m.pinnedList.Items()
	items := filterPinnedItems(allItems, m.sidebarFilterInput)

	if len(items) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  No pinned workflows =(")
		content.WriteString(emptyText)
	} else {
		// Calculate visible window (each item takes 3 lines)
		linesPerItem := 3
		maxVisible := max(1, (height-10)/linesPerItem)
		visibleStart := 0
		visibleEnd := len(items)

		if len(items) > maxVisible {
			cursor := m.pinnedList.Index()
			visibleStart = max(0, cursor-maxVisible/2)
			visibleEnd = min(len(items), visibleStart+maxVisible)

			// Adjust if we're at the end
			if visibleEnd == len(items) && visibleEnd-visibleStart < maxVisible {
				visibleStart = max(0, visibleEnd-maxVisible)
			}

			// Show scroll indicator
			scrollInfo := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(fmt.Sprintf("  (%d-%d of %d)", visibleStart+1, visibleEnd, len(items)))
			content.WriteString(scrollInfo)
			content.WriteString("\n")
		}

		for i := visibleStart; i < visibleEnd; i++ {
			item := items[i]
			pli, ok := item.(pinnedListItem)
			if !ok {
				continue
			}

			// Use filtered index if filtering is active
			var isSelected bool
			if m.sidebarFilterInput != "" {
				isSelected = m.activePanel == SidebarPanel && i == m.sidebarFilteredIndex
			} else {
				isSelected = m.activePanel == SidebarPanel && i == m.pinnedList.Index()
			}

			prefix := "  "
			fgColor := lipgloss.Color("white")
			if isSelected {
				prefix = "> "
				fgColor = lipgloss.Color("blue")
			}

			// Truncate long workflow names
			workflowName := pli.workflowName
			maxNameWidth := width - 8
			if len(workflowName) > maxNameWidth {
				workflowName = workflowName[:maxNameWidth-3] + "..."
			}

			workflowLine := lipgloss.NewStyle().
				Foreground(fgColor).
				Render(fmt.Sprintf("%s%s", prefix, workflowName))

			// Truncate long group names
			groupName := pli.group.Name
			maxGroupWidth := width - 8
			if len(groupName) > maxGroupWidth {
				groupName = groupName[:maxGroupWidth-3] + "..."
			}

			groupLine := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(fmt.Sprintf("   (%s)", groupName))

			content.WriteString(workflowLine)
			content.WriteString("\n")
			content.WriteString(groupLine)
			content.WriteString("\n")

			if i < visibleEnd-1 {
				content.WriteString("\n")
			}
		}
	}

	// Fill remaining space
	lines := strings.Split(content.String(), "\n")
	remainingHeight := height - len(lines) - 2
	if remainingHeight > 0 {
		content.WriteString(strings.Repeat("\n", remainingHeight))
	}

	// Help hint at bottom
	var hintText string
	if m.sidebarFilterActive {
		hintText = "[enter] apply filter | [esc] cancel"
	} else if m.sidebarFilterInput != "" {
		hintText = "[j/k] navigate | [enter] select | [esc] clear filter"
	} else {
		hintText = "[s] focus | [/] filter"
	}
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(hintText)
	content.WriteString("\n")
	content.WriteString(hint)

	return lipgloss.NewStyle().
		Width(width - 2).
		Height(height).
		Render(content.String())
}

// renderNavigation renders the center navigation panel with group hierarchy
func (m MenuModel) renderNavigation(width, height int) string {
	var content strings.Builder

	// Title with current path
	var titleText string
	if len(m.groupPath) == 0 {
		titleText = "📁 Browse Groups"
	} else {
		parts := []string{}
		for _, group := range m.groupPath {
			parts = append(parts, group.Name)
		}
		titleText = fmt.Sprintf("📁 %s", strings.Join(parts, " > "))
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render(titleText)

	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", width-4))
	content.WriteString("\n")

	// Show filter input if filtering is active
	if m.navigationFilterActive {
		filterPrompt := lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			Render(fmt.Sprintf("Filter: %s█", m.navigationFilterInput))
		content.WriteString(filterPrompt)
		content.WriteString("\n")
		content.WriteString(strings.Repeat("─", width-4))
		content.WriteString("\n")
	} else if m.navigationFilterInput != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("🔍 Filtered by: %q (press [esc] to clear)", m.navigationFilterInput))
		content.WriteString(filterInfo)
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Get all items and apply custom filter
	allItems := m.list.Items()
	items := filterNavigationItems(allItems, m.navigationFilterInput)
	visibleStart := 0
	visibleEnd := len(items)

	// Calculate visible window
	// Each item takes 2 lines (title + description)
	linesPerItem := 2
	maxVisible := max(1, (height-8)/linesPerItem) // At least show 1 item

	if len(items) > maxVisible {
		cursor := m.list.Index()
		// Center the cursor in the viewport
		visibleStart = max(0, cursor-maxVisible/2)
		visibleEnd = min(len(items), visibleStart+maxVisible)

		// Adjust if we're at the end
		if visibleEnd == len(items) && visibleEnd-visibleStart < maxVisible {
			visibleStart = max(0, visibleEnd-maxVisible)
		}
	}

	// Show scroll indicator if needed
	if len(items) > maxVisible {
		scrollInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("  (%d-%d of %d)", visibleStart+1, visibleEnd, len(items)))
		content.WriteString(scrollInfo)
		content.WriteString("\n")
	}

	for i := visibleStart; i < visibleEnd; i++ {
		item := items[i].(listItem)

		// Use filtered index if filtering is active
		var isSelected bool
		if m.navigationFilterInput != "" {
			isSelected = m.activePanel == NavigationPanel && i == m.navigationFilteredIndex
		} else {
			isSelected = m.activePanel == NavigationPanel && i == m.list.Index()
		}

		prefix := "  "
		fgColor := lipgloss.Color("white")
		if isSelected {
			prefix = "> "
			fgColor = lipgloss.Color("blue")
		}

		// Items already have icons from buildListItems, so just render them
		// Truncate long titles to fit width
		title := item.Title()
		maxTitleWidth := width - 10
		if len(title) > maxTitleWidth {
			title = title[:maxTitleWidth-3] + "..."
		}

		itemLine := lipgloss.NewStyle().
			Foreground(fgColor).
			Render(fmt.Sprintf("%s%s", prefix, title))

		content.WriteString(itemLine)
		content.WriteString("\n")

		// Show description in gray, also truncated
		if item.Description() != "" {
			desc := item.Description()
			maxDescWidth := width - 10
			if len(desc) > maxDescWidth {
				desc = desc[:maxDescWidth-3] + "..."
			}

			descLine := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(fmt.Sprintf("     %s", desc))
			content.WriteString(descLine)
			content.WriteString("\n")
		}
	}

	// Help hint
	var hintText string
	if m.navigationFilterActive {
		hintText = "[enter] apply | [esc] cancel filter"
	} else if m.navigationFilterInput != "" {
		hintText = "[j/k] navigate | [enter] select | [esc] clear filter"
	} else {
		hintText = "[enter] select"
		if len(m.groupPath) > 0 {
			hintText += " | [esc] back | [/] filter groups & workflows"
		} else {
			hintText += " | [/] filter groups"
		}
		hintText += " | [g] focus"
	}
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(hintText)
	content.WriteString("\n")
	content.WriteString(hint)

	return lipgloss.NewStyle().
		Width(width - 2).
		Height(height).
		Render(content.String())
}

// renderDetails renders the right details panel with workflow info
func (m MenuModel) renderDetails(width, height int) string {
	var content strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render("📊 Workflow Details")

	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", width-4))
	content.WriteString("\n\n")

	if m.selectedWorkflow == "" {
		// No workflow selected
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  Select a workflow to\n  view details")
		content.WriteString(emptyText)
	} else {
		// Show workflow name
		nameLabel := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Workflow:")
		workflowName := lipgloss.NewStyle().
			Foreground(lipgloss.Color("blue")).
			Bold(true).
			Render(m.selectedWorkflow)

		content.WriteString(fmt.Sprintf("%s\n%s\n\n", nameLabel, workflowName))

		if m.loading {
			loadingText := lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Render("⟳ Loading workflow runs...")
			content.WriteString(loadingText)
		} else if len(m.workflowRuns) > 0 {
			recentLabel := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("Recent Runs:")
			content.WriteString(recentLabel)
			content.WriteString("\n\n")

			// Show up to 3 most recent runs
			displayCount := min(3, len(m.workflowRuns))
			for i := range displayCount {
				run := m.workflowRuns[i]

				// Status icon
				statusIcon := "●"
				statusColor := lipgloss.Color("240")
				switch run.Status {
				case "completed":
					if run.Conclusion == "success" {
						statusIcon = "✓"
						statusColor = lipgloss.Color("green")
					} else {
						statusIcon = "✗"
						statusColor = lipgloss.Color("red")
					}
				case "in_progress":
					statusIcon = "⟳"
					statusColor = lipgloss.Color("yellow")
				}

				statusText := lipgloss.NewStyle().
					Foreground(statusColor).
					Render(statusIcon)

				runInfo := lipgloss.NewStyle().
					Foreground(lipgloss.Color("white")).
					Render(fmt.Sprintf(" #%d %s", run.DatabaseID, run.HeadBranch))

				content.WriteString(fmt.Sprintf("%s%s\n", statusText, runInfo))
			}

			content.WriteString("\n")

			// Show hint to view full table
			hintText := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("[r] View all runs")
			content.WriteString(hintText)
		} else if m.err != nil {
			errText := lipgloss.NewStyle().
				Foreground(lipgloss.Color("red")).
				Render(fmt.Sprintf("Error: %v", m.err))
			content.WriteString(errText)
		} else {
			noRunsText := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("No runs found")
			content.WriteString(noRunsText)
		}
	}

	// Help hint
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("[d] Focus details")

	// Fill remaining space and add hint at bottom
	lines := strings.Split(content.String(), "\n")
	remainingHeight := height - len(lines) - 2
	if remainingHeight > 0 {
		content.WriteString(strings.Repeat("\n", remainingHeight))
	}
	content.WriteString("\n")
	content.WriteString(hint)

	return lipgloss.NewStyle().
		Width(width - 2).
		Height(height).
		Render(content.String())
}

// renderRunsPanel renders the bottom expandable panel with full workflow runs table
func (m MenuModel) renderRunsPanel(width, height int) string {
	if !m.showRunsPanel {
		return ""
	}

	var content strings.Builder

	// Title
	titleText := fmt.Sprintf("📋 Workflow Runs: %s", m.selectedWorkflow)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render(titleText)

	content.WriteString(title)
	content.WriteString("\n")

	// Show total runs count
	runsInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("Total: %d runs", len(m.workflowRuns)))
	content.WriteString(runsInfo)
	content.WriteString("\n\n")

	if m.err != nil {
		errText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("red")).
			Render(fmt.Sprintf("Error: %v", m.err))
		content.WriteString(errText)
	} else if len(m.workflowRuns) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No workflow runs found")
		content.WriteString(emptyText)
	} else {
		content.WriteString(m.table.View())
	}

	content.WriteString("\n")

	// Keybindings hint
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("[w] open run | [esc] close | Use j/k or ↑/↓ to navigate")
	content.WriteString(hint)

	return lipgloss.NewStyle().
		Width(width - 2).
		Height(height).
		Render(content.String())
}

// renderBreadcrumb renders the breadcrumb bar showing current location
func (m MenuModel) renderBreadcrumb() string {
	parts := []string{fmt.Sprintf("📦 %s", m.config.Repository)}

	for _, group := range m.groupPath {
		parts = append(parts, ">", group.Name)
	}

	if m.selectedWorkflow != "" {
		parts = append(parts, ">", m.selectedWorkflow)
	}

	breadcrumbText := strings.Join(parts, " ")

	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("69")).
		Padding(0, 1).
		Render(breadcrumbText)
}

// renderStatusBar renders the refresh status bar
func (m MenuModel) renderStatusBar() string {
	if m.refreshInterval <= 0 {
		// No status bar if refresh is not configured
		return ""
	}

	// Determine refresh status and color
	autoRefreshSymbol := "✗"
	refreshColor := lipgloss.Color("red")
	if m.autoRefreshEnabled {
		autoRefreshSymbol = "✓"
		refreshColor = lipgloss.Color("green")
	}

	statusContent := lipgloss.NewStyle().
		Foreground(refreshColor).
		Render(fmt.Sprintf("  %s Auto-refresh: %ds", autoRefreshSymbol, m.refreshInterval))

	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("69")).
		Padding(0, 1).
		Render(statusContent)
}

// renderHelpBar renders the help bar with available keybindings
func (m MenuModel) renderHelpBar() string {
	var keys []string

	// Global keys
	keys = append(keys, "[q]uit", "[tab] cycle panels")

	// Panel-specific keys
	switch m.activePanel {
	case SidebarPanel:
		keys = append(keys, "[enter]select", "[p]unpin", "[w]web")
	case NavigationPanel:
		if len(m.groupPath) > 0 {
			keys = append(keys, "[enter]select", "[esc]back", "[p]pin", "[w]web")
		} else {
			keys = append(keys, "[enter]enter group")
		}
	case DetailsPanel:
		if m.selectedWorkflow != "" {
			keys = append(keys, "[r]view runs", "[w]web")
		}
	case RunsPanel:
		keys = append(keys, "[w]open run", "[esc]close")
	}

	// Add refresh shortcuts if a workflow is selected and refresh is configured
	if m.selectedWorkflow != "" && m.refreshInterval > 0 {
		keys = append(keys, "[Ctrl+R]refresh", "[Ctrl+T]toggle")
	}

	helpText := strings.Join(keys, " ")

	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(helpText)
}

// panelBorder wraps content with a border, highlighting if active
func (m MenuModel) panelBorder(content string, panelType PanelType) string {
	isActive := m.activePanel == panelType

	borderColor := lipgloss.Color("240") // Dim gray
	if isActive {
		borderColor = lipgloss.Color("blue") // Bright blue
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(content)
}
