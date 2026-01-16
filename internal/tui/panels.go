package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type PanelType int

const (
	SidebarPanel PanelType = iota
	NavigationPanel
	DetailsPanel
	RunsPanel
)

func (m MenuModel) renderSidebar(width, height int) string {
	var content strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render("📌 Pinned Workflows")

	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", width-4))
	content.WriteString("\n")

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

	allItems := m.pinnedList.Items()
	items := filterPinnedItems(allItems, m.sidebarFilterInput)

	if len(items) == 0 {
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  No pinned workflows =(")
		content.WriteString(emptyText)
	} else {
		linesPerItem := 3
		maxVisible := max(1, (height-10)/linesPerItem)
		visibleStart := 0
		visibleEnd := len(items)

		if len(items) > maxVisible {
			cursor := m.pinnedList.Index()
			visibleStart = max(0, cursor-maxVisible/2)
			visibleEnd = min(len(items), visibleStart+maxVisible)

			if visibleEnd == len(items) && visibleEnd-visibleStart < maxVisible {
				visibleStart = max(0, visibleEnd-maxVisible)
			}

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

			workflowName := pli.workflowName
			maxNameWidth := width - 8
			if len(workflowName) > maxNameWidth {
				workflowName = workflowName[:maxNameWidth-3] + "..."
			}

			workflowLine := lipgloss.NewStyle().
				Foreground(fgColor).
				Render(fmt.Sprintf("%s%s", prefix, workflowName))

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

	lines := strings.Split(content.String(), "\n")
	remainingHeight := height - len(lines) - 2
	if remainingHeight > 0 {
		content.WriteString(strings.Repeat("\n", remainingHeight))
	}

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

func (m MenuModel) renderNavigation(width, height int) string {
	var content strings.Builder

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

	allItems := m.list.Items()
	items := filterNavigationItems(allItems, m.navigationFilterInput)
	visibleStart := 0
	visibleEnd := len(items)

	linesPerItem := 2
	maxVisible := max(1, (height-8)/linesPerItem)

	if len(items) > maxVisible {
		cursor := m.list.Index()
		visibleStart = max(0, cursor-maxVisible/2)
		visibleEnd = min(len(items), visibleStart+maxVisible)

		if visibleEnd == len(items) && visibleEnd-visibleStart < maxVisible {
			visibleStart = max(0, visibleEnd-maxVisible)
		}
	}

	if len(items) > maxVisible {
		scrollInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("  (%d-%d of %d)", visibleStart+1, visibleEnd, len(items)))
		content.WriteString(scrollInfo)
		content.WriteString("\n")
	}

	for i := visibleStart; i < visibleEnd; i++ {
		item := items[i].(listItem)

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
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  Select a workflow to\n  view details")
		content.WriteString(emptyText)
	} else {
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

			displayCount := min(3, len(m.workflowRuns))
			for i := range displayCount {
				run := m.workflowRuns[i]

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

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("[d] Focus details")

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

func (m MenuModel) renderRunsPanel(width, height int) string {
	if !m.showRunsPanel {
		return ""
	}

	var content strings.Builder

	titleText := fmt.Sprintf("📋 Workflow Runs: %s", m.selectedWorkflow)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render(titleText)

	content.WriteString(title)
	content.WriteString("\n")

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

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("[w] open run | [esc] close | Use j/k or ↑/↓ to navigate")
	content.WriteString(hint)

	return lipgloss.NewStyle().
		Width(width - 2).
		Height(height).
		Render(content.String())
}

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

func (m MenuModel) renderStatusBar() string {
	if m.refreshInterval <= 0 {
		return ""
	}

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

func (m MenuModel) renderHelpBar() string {
	var keys []string

	keys = append(keys, "[q]uit", "[?] help", "[Ctrl+f] search")

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

	if m.selectedWorkflow != "" && m.refreshInterval > 0 {
		keys = append(keys, "[Ctrl+R]refresh", "[Ctrl+Shift+R]toggle")
	}

	helpText := strings.Join(keys, " ")

	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(helpText)
}

func (m MenuModel) panelBorder(content string, panelType PanelType) string {
	isActive := m.activePanel == panelType

	borderColor := lipgloss.Color("240")
	if isActive {
		borderColor = lipgloss.Color("blue")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(content)
}

func (m MenuModel) renderSearchOverlay() string {
	overlayWidth := max(50, m.width*60/100)
	overlayHeight := max(15, m.height*70/100)

	bgColor := lipgloss.Color("236")
	dimColor := lipgloss.Color("245")
	accentColor := lipgloss.Color("blue")
	textColor := lipgloss.Color("252")

	var content strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Render("Global Search")

	separator := lipgloss.NewStyle().
		Foreground(dimColor).
		Render(strings.Repeat("─", overlayWidth-6))

	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(separator)
	content.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Background(lipgloss.Color("238")).
		Padding(0, 1)

	cursor := lipgloss.NewStyle().Foreground(accentColor).Render("█")
	var inputText string
	if m.globalSearchInput == "" {
		inputText = lipgloss.NewStyle().
			Foreground(dimColor).
			Render("Type to search...") + cursor
	} else {
		inputText = m.globalSearchInput + cursor
	}

	searchIcon := lipgloss.NewStyle().
		Foreground(accentColor).
		Render("  ")

	content.WriteString(searchIcon)
	content.WriteString(inputStyle.Render(inputText))
	content.WriteString("\n\n")

	if m.globalSearchInput == "" {
		hintText := lipgloss.NewStyle().
			Foreground(dimColor).
			Render("  Search groups, workflows, and descriptions")
		content.WriteString(hintText)
	} else if len(m.globalSearchResults) == 0 {
		noResultsText := lipgloss.NewStyle().
			Foreground(dimColor).
			Render("  No results found")
		content.WriteString(noResultsText)
	} else {
		countText := lipgloss.NewStyle().
			Foreground(dimColor).
			Render(fmt.Sprintf("  %d results", len(m.globalSearchResults)))
		content.WriteString(countText)
		content.WriteString("\n\n")

		linesPerResult := 2
		maxResults := max(1, (overlayHeight-12)/linesPerResult)

		visibleStart := 0
		visibleEnd := min(len(m.globalSearchResults), maxResults)

		if m.globalSearchIndex >= visibleEnd {
			visibleStart = m.globalSearchIndex - maxResults + 1
			visibleEnd = m.globalSearchIndex + 1
		}

		for i := visibleStart; i < visibleEnd; i++ {
			result := m.globalSearchResults[i]
			isSelected := i == m.globalSearchIndex

			prefix := "  "
			nameStyle := lipgloss.NewStyle().Foreground(textColor)
			if isSelected {
				prefix = "> "
				nameStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true)
			}

			icon := "📁"
			if result.Type == "workflow" {
				icon = "📄"
			}

			name := result.Name
			maxNameWidth := overlayWidth - 15
			if len(name) > maxNameWidth {
				name = name[:maxNameWidth-3] + "..."
			}

			nameLine := nameStyle.Render(fmt.Sprintf("%s%s %s", prefix, icon, name))
			content.WriteString(nameLine)
			content.WriteString("\n")

			pathText := formatGroupPath(result.GroupPath)
			if result.Type == "workflow" && result.Description != result.Name {
				pathText = pathText + " / " + result.Description
			}
			maxPathWidth := overlayWidth - 10
			if len(pathText) > maxPathWidth {
				pathText = pathText[:maxPathWidth-3] + "..."
			}

			pathLine := lipgloss.NewStyle().Foreground(dimColor).
				Render(fmt.Sprintf("     %s", pathText))
			content.WriteString(pathLine)
			content.WriteString("\n")
		}

		if len(m.globalSearchResults) > maxResults {
			scrollInfo := lipgloss.NewStyle().
				Foreground(dimColor).
				Render(fmt.Sprintf("\n  (%d-%d of %d)", visibleStart+1, visibleEnd, len(m.globalSearchResults)))
			content.WriteString(scrollInfo)
		}
	}

	content.WriteString("\n\n")
	hint := lipgloss.NewStyle().
		Foreground(dimColor).
		Render("↑/↓ navigate  enter select  esc close")
	content.WriteString(hint)

	overlayContent := lipgloss.NewStyle().
		Width(overlayWidth-4).
		Height(overlayHeight-2).
		Padding(1, 2).
		Background(bgColor).
		Render(content.String())

	overlayBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Background(bgColor).
		Render(overlayContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		overlayBox,
	)
}

func (m MenuModel) renderHelpModal() string {
	overlayWidth := max(55, m.width*50/100)
	overlayHeight := max(20, m.height*60/100)

	bgColor := lipgloss.Color("236")
	dimColor := lipgloss.Color("245")
	accentColor := lipgloss.Color("blue")
	textColor := lipgloss.Color("252")
	keyColor := lipgloss.Color("214")

	var content strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Render("Keyboard Shortcuts")

	separator := lipgloss.NewStyle().
		Foreground(dimColor).
		Render(strings.Repeat("─", overlayWidth-6))

	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(separator)
	content.WriteString("\n\n")

	renderKey := func(key, desc string) string {
		keyStyle := lipgloss.NewStyle().
			Foreground(keyColor).
			Width(12)
		descStyle := lipgloss.NewStyle().
			Foreground(textColor)
		return keyStyle.Render(key) + descStyle.Render(desc) + "\n"
	}

	renderSection := func(title string) string {
		return lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Render(title) + "\n"
	}

	content.WriteString(renderSection("Global"))
	content.WriteString(renderKey("q", "Quit"))
	content.WriteString(renderKey("?", "Show this help"))
	content.WriteString(renderKey("Ctrl+f", "Global search"))
	content.WriteString(renderKey("Tab", "Next panel"))
	content.WriteString(renderKey("Shift+Tab", "Previous panel"))
	content.WriteString("\n")

	content.WriteString(renderSection("Panel Focus"))
	content.WriteString(renderKey("s", "Focus sidebar (pinned)"))
	content.WriteString(renderKey("g", "Focus navigation (groups)"))
	content.WriteString(renderKey("d", "Focus details"))
	content.WriteString(renderKey("r", "Toggle runs panel"))
	content.WriteString("\n")

	content.WriteString(renderSection("Navigation"))
	content.WriteString(renderKey("j / ↓", "Move down"))
	content.WriteString(renderKey("k / ↑", "Move up"))
	content.WriteString(renderKey("Enter / l", "Select / Enter group"))
	content.WriteString(renderKey("Esc / h", "Go back / Close"))
	content.WriteString(renderKey("/", "Filter current panel"))
	content.WriteString("\n")

	content.WriteString(renderSection("Actions"))
	content.WriteString(renderKey("p", "Pin/Unpin workflow"))
	content.WriteString(renderKey("w", "Open in browser"))
	content.WriteString(renderKey("Ctrl+r", "Refresh runs"))
	content.WriteString(renderKey("Ctrl+t", "Toggle auto-refresh"))

	content.WriteString("\n")
	hint := lipgloss.NewStyle().
		Foreground(dimColor).
		Render("Press ? or Esc to close")
	content.WriteString(hint)

	overlayContent := lipgloss.NewStyle().
		Width(overlayWidth-4).
		Height(overlayHeight-2).
		Padding(1, 2).
		Background(bgColor).
		Render(content.String())

	overlayBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Background(bgColor).
		Render(overlayContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		overlayBox,
	)
}
