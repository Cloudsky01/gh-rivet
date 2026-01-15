package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m MenuModel) View() string {
	return m.renderMultiPanelLayout()
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg), nil
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case workflowRunsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.workflowRuns = msg.runs
			m.table = buildWorkflowRunsTable(msg.runs, m.tablePageSize)
		}
		// Continue listening for refresh ticks
		return m, m.getRefreshTickerCmd()
	case refreshTickMsg:
		// Auto-refresh timer fired - fetch workflow runs if not already loading
		if m.selectedWorkflow != "" && !m.loading {
			m.loading = true
			cmds := []tea.Cmd{m.fetchWorkflowRunsCmd, m.getRefreshTickerCmd()}
			return m, tea.Batch(cmds...)
		}
		// Continue listening for refresh ticks
		return m, m.getRefreshTickerCmd()
	}

	return m.updateActiveComponent(msg)
}

func (m MenuModel) handleWindowResize(msg tea.WindowSizeMsg) MenuModel {
	m.width = msg.Width
	m.height = msg.Height

	sidebarWidth := m.width / 5
	detailsWidth := m.width / 3
	mainWidth := m.width - sidebarWidth - detailsWidth - 6
	panelHeight := m.height - 6

	if m.showRunsPanel {
		runsHeight := panelHeight / 3
		panelHeight = panelHeight - runsHeight - 3
	}

	m.list.SetSize(mainWidth-4, panelHeight-4)
	m.pinnedList.SetSize(sidebarWidth-4, panelHeight-4)

	if len(m.workflowRuns) > 0 {
		m.table = buildWorkflowRunsTable(m.workflowRuns, m.tablePageSize)
	}

	return m
}

func (m MenuModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sidebarFilterActive || m.navigationFilterActive {
		switch m.activePanel {
		case SidebarPanel:
			return m.updateSidebarPanel(msg)
		case NavigationPanel:
			return m.updateNavigationPanel(msg)
		default:
			panic("unhandled default case")
		}
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.stopRefreshTicker()
		return m, tea.Quit

	case "s":
		m.activePanel = SidebarPanel
		return m, nil
	case "g":
		m.activePanel = NavigationPanel
		return m, nil
	case "d":
		m.activePanel = DetailsPanel
		return m, nil
	case "r":
		if m.selectedWorkflow != "" {
			m.showRunsPanel = !m.showRunsPanel
			if m.showRunsPanel {
				m.activePanel = RunsPanel
			} else if m.activePanel == RunsPanel {
				m.activePanel = NavigationPanel
			}
			return m.handleWindowResize(tea.WindowSizeMsg{Width: m.width, Height: m.height}), nil
		}
		return m, nil

	case "ctrl+r":
		// Manual refresh - fetch latest runs and restart the auto-refresh timer
		if m.selectedWorkflow != "" && !m.loading {
			m.loading = true
			// Restart ticker if auto-refresh is enabled
			if m.refreshInterval > 0 && m.autoRefreshEnabled {
				m.startRefreshTicker()
			}
			return m, m.fetchWorkflowRunsCmd
		}
		return m, nil

	case "ctrl+t":
		// Toggle auto-refresh
		if m.refreshInterval > 0 {
			m.autoRefreshEnabled = !m.autoRefreshEnabled
			if m.autoRefreshEnabled {
				m.startRefreshTicker()
			} else {
				m.stopRefreshTicker()
			}
		}
		return m, nil

	case "tab":
		m.activePanel = m.getNextPanel()
		return m, nil
	case "shift+tab":
		m.activePanel = m.getPreviousPanel()
		return m, nil
	}

	switch m.activePanel {
	case SidebarPanel:
		return m.updateSidebarPanel(msg)
	case NavigationPanel:
		return m.updateNavigationPanel(msg)
	case DetailsPanel:
		return m.updateDetailsPanel(msg)
	case RunsPanel:
		return m.updateRunsPanel(msg)
	}

	return m, nil
}

func (m MenuModel) updateSidebarPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sidebarFilterActive {
		switch msg.String() {
		case "enter":
			m.sidebarFilterActive = false
			m.sidebarFilteredIndex = 0
			return m, nil
		case "esc":
			m.sidebarFilterActive = false
			m.sidebarFilterInput = ""
			m.sidebarFilteredIndex = 0
			return m, nil
		case "backspace":
			if len(m.sidebarFilterInput) > 0 {
				m.sidebarFilterInput = m.sidebarFilterInput[:len(m.sidebarFilterInput)-1]
			}
			return m, nil
		default:
			key := msg.String()
			if len(key) == 1 {
				m.sidebarFilterInput += key
			}
			return m, nil
		}
	}

	if m.sidebarFilterInput != "" {
		allItems := m.pinnedList.Items()
		filteredItems := filterPinnedItems(allItems, m.sidebarFilterInput)

		switch msg.String() {
		case "esc":
			m.sidebarFilterInput = ""
			m.sidebarFilteredIndex = 0
			return m, nil
		case "j", "down":
			if m.sidebarFilteredIndex < len(filteredItems)-1 {
				m.sidebarFilteredIndex++
			}
			return m, nil
		case "k", "up":
			if m.sidebarFilteredIndex > 0 {
				m.sidebarFilteredIndex--
			}
			return m, nil
		case "enter":
			if m.sidebarFilteredIndex < len(filteredItems) {
				selectedItem, ok := filteredItems[m.sidebarFilteredIndex].(pinnedListItem)
				if ok {
					m.selectedWorkflow = selectedItem.workflowName
					m.selectedGroup = selectedItem.group
					m.loading = true
					m.activePanel = DetailsPanel
					m.startRefreshTicker()
					m.saveState()
					return m, m.fetchWorkflowRunsCmd
				}
			}
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "/":
		// Start filtering
		m.sidebarFilterActive = true
		m.sidebarFilterInput = ""
		m.sidebarFilteredIndex = 0
		return m, nil

	case "enter":
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			m.selectedWorkflow = selectedItem.workflowName
			m.selectedGroup = selectedItem.group
			m.loading = true
			m.activePanel = DetailsPanel
			m.startRefreshTicker()
			m.saveState()
			return m, m.fetchWorkflowRunsCmd
		}
		return m, nil

	case "p":
		// Unpin workflow
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			selectedItem.group.TogglePin(selectedItem.workflowName)

			if err := m.config.Save(m.configPath); err != nil {
				m.err = fmt.Errorf("failed to save config: %w", err)
			}

			// Refresh pinned list
			m.pinnedList.SetItems(buildPinnedListItems(m.config))
			m.saveState()
		}
		return m, nil

	case "w":
		// Open workflow in browser
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			if err := m.gh.OpenWorkflowInBrowser(selectedItem.workflowName); err != nil {
				m.err = err
			}
		}
		return m, nil

	default:
		// Pass through to list for navigation (j/k, up/down, etc.)
		var cmd tea.Cmd
		m.pinnedList, cmd = m.pinnedList.Update(msg)
		return m, cmd
	}
}

// updateNavigationPanel handles input when navigation panel is focused
func (m MenuModel) updateNavigationPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle active filtering - capture all input
	if m.navigationFilterActive {
		switch msg.String() {
		case "enter":
			// Apply filter
			m.navigationFilterActive = false
			m.navigationFilteredIndex = 0 // Reset to first filtered item
			return m, nil
		case "esc":
			// Cancel filter
			m.navigationFilterActive = false
			m.navigationFilterInput = ""
			m.navigationFilteredIndex = 0
			return m, nil
		case "backspace":
			// Remove last character
			if len(m.navigationFilterInput) > 0 {
				m.navigationFilterInput = m.navigationFilterInput[:len(m.navigationFilterInput)-1]
			}
			return m, nil
		default:
			// Add character to filter input (only printable chars)
			key := msg.String()
			if len(key) == 1 {
				m.navigationFilterInput += key
			}
			return m, nil
		}
	}

	// If filter is applied, handle navigation through filtered items
	if m.navigationFilterInput != "" {
		allItems := m.list.Items()
		filteredItems := filterNavigationItems(allItems, m.navigationFilterInput)

		switch msg.String() {
		case "p":
			if len(m.groupPath) == 0 {
				return m, nil
			}
			if m.navigationFilteredIndex < len(filteredItems) {
				selectedItem, ok := filteredItems[m.navigationFilteredIndex].(listItem)
				if ok && !selectedItem.isGroup {
					currentGroup := m.groupPath[len(m.groupPath)-1]
					currentGroup.TogglePin(selectedItem.workflowName)

					if err := m.config.Save(m.configPath); err != nil {
						m.err = fmt.Errorf("failed to save config: %w", err)
					}

					m.list.SetItems(buildListItems(m.config, m.groupPath))
					m.pinnedList.SetItems(buildPinnedListItems(m.config))
					m.saveState()
				}
			}
			return m, nil
		case "esc":
			// Clear filter
			m.navigationFilterInput = ""
			m.navigationFilteredIndex = 0
			return m, nil
		case "j", "down":
			// Move down in filtered list
			if m.navigationFilteredIndex < len(filteredItems)-1 {
				m.navigationFilteredIndex++
			}
			return m, nil
		case "k", "up":
			// Move up in filtered list
			if m.navigationFilteredIndex > 0 {
				m.navigationFilteredIndex--
			}
			return m, nil
		case "enter", "l":
			// Select item from filtered list
			if m.navigationFilteredIndex < len(filteredItems) {
				selectedItem, ok := filteredItems[m.navigationFilteredIndex].(listItem)
				if !ok {
					return m, nil
				}

				if selectedItem.isGroup {
					// Enter group
					if selectedItem.group != nil {
						m.groupPath = append(m.groupPath, selectedItem.group)
						m.list.SetItems(buildListItems(m.config, m.groupPath))
						m.list.ResetSelected()
						// Clear filter when entering a group
						m.navigationFilterInput = ""
						m.navigationFilteredIndex = 0
						m.saveState()
					}
				} else {
					m.selectedWorkflow = selectedItem.workflowName
					m.loading = true
					m.activePanel = DetailsPanel
					m.startRefreshTicker()
					m.saveState()
					return m, m.fetchWorkflowRunsCmd
				}
			}
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "/":
		// Start filtering
		m.navigationFilterActive = true
		m.navigationFilterInput = ""
		m.navigationFilteredIndex = 0
		return m, nil

	case "esc", "backspace", "h":
		// Navigate up in hierarchy (only if no filter is active)
		if len(m.groupPath) > 0 {
			m.groupPath = m.groupPath[:len(m.groupPath)-1]
			m.list.SetItems(buildListItems(m.config, m.groupPath))
			m.navigationFilterInput = ""
			m.navigationFilteredIndex = 0
			m.saveState()
		}
		return m, nil

	case "enter", "l":
		// Navigate into group or select workflow
		selectedItem, ok := m.list.SelectedItem().(listItem)
		if !ok {
			return m, nil
		}

		if selectedItem.isGroup {
			// Check if it's the "Go back" item
			if selectedItem.group == nil && len(m.groupPath) > 0 {
				// Go back to parent
				m.groupPath = m.groupPath[:len(m.groupPath)-1]
				m.list.SetItems(buildListItems(m.config, m.groupPath))
				m.list.ResetFilter()
				m.saveState()
			} else {
				// Enter group
				m.groupPath = append(m.groupPath, selectedItem.group)
				m.list.SetItems(buildListItems(m.config, m.groupPath))
				m.list.ResetSelected()
				m.saveState()
			}
		} else {
			m.selectedWorkflow = selectedItem.workflowName
			m.loading = true
			m.activePanel = DetailsPanel
			m.startRefreshTicker()
			m.saveState()
			return m, m.fetchWorkflowRunsCmd
		}
		return m, nil

	case "p":
		// Toggle pin
		if len(m.groupPath) > 0 {
			selectedItem, ok := m.list.SelectedItem().(listItem)
			if ok && !selectedItem.isGroup {
				currentGroup := m.groupPath[len(m.groupPath)-1]
				currentGroup.TogglePin(selectedItem.workflowName)

				if err := m.config.Save(m.configPath); err != nil {
					m.err = fmt.Errorf("failed to save config: %w", err)
				}

				// Refresh both lists
				m.list.SetItems(buildListItems(m.config, m.groupPath))
				m.pinnedList.SetItems(buildPinnedListItems(m.config))
				m.saveState()
			}
		}
		return m, nil

	case "w":
		// Open workflow in browser
		selectedItem, ok := m.list.SelectedItem().(listItem)
		if !ok {
			return m, nil
		}
		if selectedItem.isGroup {
			return m, nil
		}
		if err := m.gh.OpenWorkflowInBrowser(selectedItem.workflowName); err != nil {
			m.err = err
		}
		return m, nil

	default:
		// Pass through to list for navigation (j/k, up/down, etc.)
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
}

// updateDetailsPanel handles input when details panel is focused
func (m MenuModel) updateDetailsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		// Open workflow in browser
		if m.selectedWorkflow != "" {
			if err := m.gh.OpenWorkflowInBrowser(m.selectedWorkflow); err != nil {
				m.err = err
			}
		}
		return m, nil

	case "esc":
		// Clear selection
		m.selectedWorkflow = ""
		m.workflowRuns = nil
		m.activePanel = NavigationPanel
		m.stopRefreshTicker()
		return m, nil
	}

	return m, nil
}

// updateRunsPanel handles input when runs panel is focused
func (m MenuModel) updateRunsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		// Open run in browser
		runID := m.getSelectedRunID()
		if runID > 0 {
			if err := m.gh.OpenRunInBrowser(runID); err != nil {
				m.err = err
			}
		}
		return m, nil

	case "esc":
		// Close runs panel
		m.showRunsPanel = false
		m.activePanel = DetailsPanel
		return m.handleWindowResize(tea.WindowSizeMsg{Width: m.width, Height: m.height}), nil

	default:
		// Pass navigation keys to table (j/k, arrows, etc.)
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

// updateActiveComponent passes messages to the active component
func (m MenuModel) updateActiveComponent(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.activePanel {
	case SidebarPanel:
		m.pinnedList, cmd = m.pinnedList.Update(msg)
	case NavigationPanel:
		m.list, cmd = m.list.Update(msg)
	case RunsPanel:
		m.table, cmd = m.table.Update(msg)
	default:
		panic("unhandled default case")
	}

	return m, cmd
}

// getNextPanel returns the next panel in the cycle
func (m MenuModel) getNextPanel() PanelType {
	switch m.activePanel {
	case SidebarPanel:
		return NavigationPanel
	case NavigationPanel:
		return DetailsPanel
	case DetailsPanel:
		if m.showRunsPanel {
			return RunsPanel
		}
		return SidebarPanel
	case RunsPanel:
		return SidebarPanel
	default:
		return NavigationPanel
	}
}

// getPreviousPanel returns the previous panel in the cycle
func (m MenuModel) getPreviousPanel() PanelType {
	switch m.activePanel {
	case SidebarPanel:
		if m.showRunsPanel {
			return RunsPanel
		}
		return DetailsPanel
	case NavigationPanel:
		return SidebarPanel
	case DetailsPanel:
		return NavigationPanel
	case RunsPanel:
		return DetailsPanel
	default:
		return NavigationPanel
	}
}
