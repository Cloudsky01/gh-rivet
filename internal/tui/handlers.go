package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
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
		return m, m.getRefreshTickerCmd()
	case refreshTickMsg:
		if m.selectedWorkflow != "" && !m.loading {
			m.loading = true
			cmds := []tea.Cmd{m.fetchWorkflowRunsCmd, m.getRefreshTickerCmd()}
			return m, tea.Batch(cmds...)
		}
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
	if m.globalSearchActive {
		return m.updateGlobalSearch(msg)
	}
	if m.helpModalActive {
		return m.updateHelpModal(msg)
	}

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
		if m.selectedWorkflow != "" && !m.loading {
			m.loading = true
			if m.refreshInterval > 0 && m.autoRefreshEnabled {
				m.startRefreshTicker()
			}
			return m, m.fetchWorkflowRunsCmd
		}
		return m, nil

	case "ctrl+t":
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

	case "ctrl+f":
		m.globalSearchActive = true
		m.globalSearchInput = ""
		m.globalSearchResults = nil
		m.globalSearchIndex = 0
		return m, nil

	case "?":
		m.helpModalActive = true
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
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			selectedItem.group.TogglePin(selectedItem.workflowName)
			if err := m.config.Save(m.configPath); err != nil {
				m.err = fmt.Errorf("failed to save config: %w", err)
			}
			m.pinnedList.SetItems(buildPinnedListItems(m.config))
			m.saveState()
		}
		return m, nil

	case "w":
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			if err := m.gh.OpenWorkflowInBrowser(selectedItem.workflowName); err != nil {
				m.err = err
			}
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.pinnedList, cmd = m.pinnedList.Update(msg)
		return m, cmd
	}
}

func (m MenuModel) updateNavigationPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.navigationFilterActive {
		switch msg.String() {
		case "enter":
			m.navigationFilterActive = false
			m.navigationFilteredIndex = 0
			return m, nil
		case "esc":
			m.navigationFilterActive = false
			m.navigationFilterInput = ""
			m.navigationFilteredIndex = 0
			return m, nil
		case "backspace":
			if len(m.navigationFilterInput) > 0 {
				m.navigationFilterInput = m.navigationFilterInput[:len(m.navigationFilterInput)-1]
			}
			return m, nil
		default:
			key := msg.String()
			if len(key) == 1 {
				m.navigationFilterInput += key
			}
			return m, nil
		}
	}

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
			m.navigationFilterInput = ""
			m.navigationFilteredIndex = 0
			return m, nil
		case "j", "down":
			if m.navigationFilteredIndex < len(filteredItems)-1 {
				m.navigationFilteredIndex++
			}
			return m, nil
		case "k", "up":
			if m.navigationFilteredIndex > 0 {
				m.navigationFilteredIndex--
			}
			return m, nil
		case "enter", "l":
			if m.navigationFilteredIndex < len(filteredItems) {
				selectedItem, ok := filteredItems[m.navigationFilteredIndex].(listItem)
				if !ok {
					return m, nil
				}

				if selectedItem.isGroup {
					if selectedItem.group != nil {
						m.groupPath = append(m.groupPath, selectedItem.group)
						m.list.SetItems(buildListItems(m.config, m.groupPath))
						m.list.ResetSelected()
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
		m.navigationFilterActive = true
		m.navigationFilterInput = ""
		m.navigationFilteredIndex = 0
		return m, nil

	case "esc", "backspace", "h":
		if len(m.groupPath) > 0 {
			m.groupPath = m.groupPath[:len(m.groupPath)-1]
			m.list.SetItems(buildListItems(m.config, m.groupPath))
			m.navigationFilterInput = ""
			m.navigationFilteredIndex = 0
			m.saveState()
		}
		return m, nil

	case "enter", "l":
		selectedItem, ok := m.list.SelectedItem().(listItem)
		if !ok {
			return m, nil
		}

		if selectedItem.isGroup {
			if selectedItem.group == nil && len(m.groupPath) > 0 {
				m.groupPath = m.groupPath[:len(m.groupPath)-1]
				m.list.SetItems(buildListItems(m.config, m.groupPath))
				m.list.ResetFilter()
				m.saveState()
			} else {
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
		if len(m.groupPath) > 0 {
			selectedItem, ok := m.list.SelectedItem().(listItem)
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

	case "w":
		selectedItem, ok := m.list.SelectedItem().(listItem)
		if !ok || selectedItem.isGroup {
			return m, nil
		}
		if err := m.gh.OpenWorkflowInBrowser(selectedItem.workflowName); err != nil {
			m.err = err
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
}

func (m MenuModel) updateDetailsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		if m.selectedWorkflow != "" {
			if err := m.gh.OpenWorkflowInBrowser(m.selectedWorkflow); err != nil {
				m.err = err
			}
		}
		return m, nil

	case "esc":
		m.selectedWorkflow = ""
		m.workflowRuns = nil
		m.activePanel = NavigationPanel
		m.stopRefreshTicker()
		return m, nil
	}

	return m, nil
}

func (m MenuModel) updateRunsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		runID := m.getSelectedRunID()
		if runID > 0 {
			if err := m.gh.OpenRunInBrowser(runID); err != nil {
				m.err = err
			}
		}
		return m, nil

	case "esc":
		m.showRunsPanel = false
		m.activePanel = DetailsPanel
		return m.handleWindowResize(tea.WindowSizeMsg{Width: m.width, Height: m.height}), nil

	default:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

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

func (m MenuModel) updateHelpModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "q":
		m.helpModalActive = false
		return m, nil
	}
	return m, nil
}

func (m MenuModel) updateGlobalSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.globalSearchActive = false
		m.globalSearchInput = ""
		m.globalSearchResults = nil
		m.globalSearchIndex = 0
		return m, nil

	case "enter":
		if len(m.globalSearchResults) > 0 && m.globalSearchIndex < len(m.globalSearchResults) {
			result := m.globalSearchResults[m.globalSearchIndex]
			m.globalSearchActive = false
			m.globalSearchInput = ""
			m.globalSearchResults = nil
			m.globalSearchIndex = 0
			return m.navigateToSearchResult(result)
		}
		return m, nil

	case "up", "ctrl+p":
		if m.globalSearchIndex > 0 {
			m.globalSearchIndex--
		}
		return m, nil

	case "down", "ctrl+n":
		if m.globalSearchIndex < len(m.globalSearchResults)-1 {
			m.globalSearchIndex++
		}
		return m, nil

	case "backspace":
		if len(m.globalSearchInput) > 0 {
			m.globalSearchInput = m.globalSearchInput[:len(m.globalSearchInput)-1]
			m.globalSearchResults = globalSearch(m.config, m.globalSearchInput)
			m.globalSearchIndex = 0
		}
		return m, nil

	default:
		key := msg.String()
		if len(key) == 1 {
			m.globalSearchInput += key
			m.globalSearchResults = globalSearch(m.config, m.globalSearchInput)
			m.globalSearchIndex = 0
		}
		return m, nil
	}
}

func (m MenuModel) navigateToSearchResult(result SearchResult) (tea.Model, tea.Cmd) {
	if result.Type == "group" {
		m.groupPath = resolveGroupPathFromNames(m.config, result.GroupPath)
		if result.Group != nil {
			m.groupPath = append(m.groupPath, result.Group)
		}
		m.list.SetItems(buildListItems(m.config, m.groupPath))
		m.list.ResetSelected()
		m.activePanel = NavigationPanel
		m.saveState()
		return m, nil
	}

	m.groupPath = resolveGroupPathFromNames(m.config, result.GroupPath)
	m.list.SetItems(buildListItems(m.config, m.groupPath))
	m.list.ResetSelected()
	m.selectedWorkflow = result.WorkflowName
	m.selectedGroup = result.Group
	m.loading = true
	m.activePanel = DetailsPanel
	m.startRefreshTicker()
	m.saveState()
	return m, m.fetchWorkflowRunsCmd
}

func resolveGroupPathFromNames(cfg *config.Config, names []string) []*config.Group {
	if len(names) == 0 {
		return []*config.Group{}
	}

	path := make([]*config.Group, 0, len(names))
	currentGroups := cfg.Groups

	for _, name := range names {
		found := false
		for i := range currentGroups {
			if currentGroups[i].Name == name {
				path = append(path, &currentGroups[i])
				currentGroups = currentGroups[i].Groups
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	return path
}
