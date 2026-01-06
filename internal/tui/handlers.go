package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		switch m.viewState {
		case browsingGroups:
			m.list.SetSize(msg.Width, msg.Height-4)
		case viewingPinnedWorkflows:
			m.pinnedList.SetSize(msg.Width, msg.Height-4)
		case viewingWorkflowOutput:
			m.table.SetWidth(msg.Width)
		}
		return m, nil

	case tea.KeyMsg:
		switch m.viewState {
		case viewingWorkflowOutput:
			return m.updateWorkflowView(msg)
		case viewingPinnedWorkflows:
			return m.updatePinnedView(msg)
		default:
			return m.updateBrowsingView(msg)
		}
	}

	switch m.viewState {
	case browsingGroups:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case viewingPinnedWorkflows:
		var cmd tea.Cmd
		m.pinnedList, cmd = m.pinnedList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m MenuModel) updateBrowsingView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab", "*":
		if len(m.config.GetAllPinnedWorkflows()) > 0 {
			m.pinnedList = createPinnedList(m.config)
			m.pinnedList.SetSize(m.width, m.height-4)
			m.viewState = viewingPinnedWorkflows
			m.saveState()
		}
		return m, nil

	case "esc", "backspace":
		if len(m.groupPath) > 0 {
			m.groupPath = m.groupPath[:len(m.groupPath)-1]
			m.list.SetItems(buildListItems(m.config, m.groupPath))
			m.list.ResetFilter()

			if len(m.groupPath) == 0 {
				m.list.Title = "Browse Groups"
			} else {
				breadcrumb := ""
				for i, group := range m.groupPath {
					if i > 0 {
						breadcrumb += " > "
					}
					breadcrumb += group.Name
				}
				m.list.Title = breadcrumb
			}
			m.saveState()
		}
		return m, nil

	case "ctrl+p", "p":
		if len(m.groupPath) > 0 {
			selectedItem, ok := m.list.SelectedItem().(listItem)
			if ok && !selectedItem.isGroup {
				currentGroup := m.groupPath[len(m.groupPath)-1]
				currentGroup.TogglePin(selectedItem.workflowName)

				if err := m.config.Save(m.configPath); err != nil {
					m.err = fmt.Errorf("failed to save config: %w", err)
				}

				m.list.SetItems(buildListItems(m.config, m.groupPath))
			}
		}
		return m, nil

	case "enter":
		selectedItem, ok := m.list.SelectedItem().(listItem)
		if !ok {
			return m, nil
		}

		if selectedItem.isGroup {
			m.groupPath = append(m.groupPath, selectedItem.group)
			m.list.SetItems(buildListItems(m.config, m.groupPath))

			breadcrumb := ""
			for i, group := range m.groupPath {
				if i > 0 {
					breadcrumb += " > "
				}
				breadcrumb += group.Name
			}
			m.list.Title = breadcrumb
			m.list.ResetSelected()
			m.saveState()
		} else {
			m.selectedWorkflow = selectedItem.workflowName
			runs, err := m.fetchWorkflowRuns()
			if err != nil {
				m.err = err
			} else {
				m.workflowRuns = runs
				m.table = buildWorkflowRunsTable(runs)
			}
			m.viewState = viewingWorkflowOutput
			m.saveState()
		}
		return m, nil

	case "w":
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
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MenuModel) updateWorkflowView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		if m.selectedGroup != nil {
			m.viewState = viewingPinnedWorkflows
			m.selectedGroup = nil
		} else {
			m.viewState = browsingGroups
		}
		m.saveState()
		return m, nil

	case "w":
		runID := m.getSelectedRunID()
		if runID > 0 {
			if err := m.gh.OpenRunInBrowser(runID); err != nil {
				m.err = err
			}
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

func (m MenuModel) updatePinnedView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pinnedList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.pinnedList, cmd = m.pinnedList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab":
		m.list.ResetFilter()
		m.viewState = browsingGroups
		m.saveState()
		return m, nil

	case "esc", "backspace":
		m.list.ResetFilter()
		m.viewState = browsingGroups
		m.saveState()
		return m, nil

	case "p", "ctrl+p":
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			selectedItem.group.TogglePin(selectedItem.workflowName)

			if err := m.config.Save(m.configPath); err != nil {
				m.err = fmt.Errorf("failed to save config: %w", err)
			}

			m.pinnedList.SetItems(buildPinnedListItems(m.config))

			if len(m.config.GetAllPinnedWorkflows()) == 0 {
				m.viewState = browsingGroups
			}
			m.saveState()
		}
		return m, nil

	case "enter":
		selectedItem, ok := m.pinnedList.SelectedItem().(pinnedListItem)
		if ok {
			m.selectedWorkflow = selectedItem.workflowName
			m.selectedGroup = selectedItem.group
			runs, err := m.fetchWorkflowRuns()
			if err != nil {
				m.err = err
			} else {
				m.workflowRuns = runs
				m.table = buildWorkflowRunsTable(runs)
			}
			m.viewState = viewingWorkflowOutput
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
