package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
)

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.helpOverlay.IsActive() {
		a.helpOverlay.Update(msg)
		return a, nil
	}

	if a.repoSwitcher.IsActive() {
		selectedRepo, cmd := a.repoSwitcher.Update(msg)
		if selectedRepo != nil {
			return a.switchRepository(selectedRepo.Repository)
		}
		return a, cmd
	}

	if a.cmdPalette.IsActive() {
		cmd, teaCmd := a.cmdPalette.Update(msg)
		if cmd != nil {
			return a.executeCommand(cmd)
		}
		return a, teaCmd
	}

	if a.search.IsActive() {
		result, cmd := a.search.Update(msg)
		if result != nil {
			return a.navigateToSearchResult(result)
		}
		return a, cmd
	}

	if a.isFiltering() {
		return a.handleFilterKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		a.stopRefreshTicker()
		a.saveState()
		return a, tea.Quit

	case "?":
		a.helpOverlay.Toggle()
		return a, nil

	case ":":
		a.cmdPalette.Open()
		return a, nil

	case "1":
		a.showSidebar = !a.showSidebar
		if !a.showSidebar && a.focusArea == FocusSidebar {
			a.focusArea = FocusMain
		}
		a.updateFocus()
		return a.handleResize(tea.WindowSizeMsg{Width: a.width, Height: a.height})

	case "tab":
		return a.handleTabKey()

	case "shift+tab":
		return a.handleShiftTabKey()

	case "s":
		if a.showSidebar {
			a.focusArea = FocusSidebar
			a.updateFocus()
		}
		return a, nil

	case "ctrl+f":
		a.search.Open()
		return a, nil

	case "ctrl+r":
		return a.handleRefreshKey()

	case "ctrl+t":
		return a.handleToggleAutoRefresh()
	}

	if a.focusArea == FocusSidebar {
		return a.handleSidebarKey(msg)
	}

	switch a.viewMode {
	case ViewGroups:
		return a.handleGroupsKey(msg)
	case ViewRuns:
		return a.handleRunsKey(msg)
	}

	return a, nil
}

func (a *App) handleTabKey() (tea.Model, tea.Cmd) {
	if a.showSidebar {
		if a.focusArea == FocusSidebar {
			a.focusArea = FocusMain
		} else {
			a.focusArea = FocusSidebar
		}
		a.updateFocus()
	}
	return a, nil
}

func (a *App) handleShiftTabKey() (tea.Model, tea.Cmd) {
	if a.showSidebar {
		if a.focusArea == FocusMain {
			a.focusArea = FocusSidebar
		} else {
			a.focusArea = FocusMain
		}
		a.updateFocus()
	}
	return a, nil
}

func (a *App) handleRefreshKey() (tea.Model, tea.Cmd) {
	if a.selectedWorkflow != "" && !a.loading {
		a.loading = true
		a.runsTable.SetLoading(true)
		cmds := []tea.Cmd{a.spinner.Start("Refreshing..."), a.fetchWorkflowRunsCmd}
		if a.refreshInterval > 0 && a.autoRefreshEnabled {
			a.startRefreshTicker()
		}
		return a, tea.Batch(cmds...)
	}
	return a, nil
}

func (a *App) handleToggleAutoRefresh() (tea.Model, tea.Cmd) {
	if a.refreshInterval > 0 {
		a.autoRefreshEnabled = !a.autoRefreshEnabled
		if a.autoRefreshEnabled {
			a.startRefreshTicker()
		} else {
			a.stopRefreshTicker()
		}
		a.updateStatusBar()
	}
	return a, nil
}

func (a *App) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if item := a.sidebar.SelectedItem(); item != nil {
			return a.selectWorkflowFromSidebar(item)
		}
		return a, nil

	case "p":
		if item := a.sidebar.SelectedItem(); item != nil {
			if group, ok := item.Data.(*config.Group); ok {
				group.TogglePin(item.WorkflowName)
				if err := a.config.Save(a.configPath); err != nil {
					a.err = fmt.Errorf("failed to save config: %w", err)
					return a, a.toaster.Error("Failed to save")
				}
				a.refreshPinnedList()
				a.refreshNavList()
				a.saveState()
				return a, a.toaster.Success("Unpinned workflow")
			}
		}
		return a, nil

	case "w":
		if item := a.sidebar.SelectedItem(); item != nil {
			if err := a.gh.OpenWorkflowInBrowser(item.WorkflowName); err != nil {
				a.err = err
				return a, a.toaster.Error("Failed to open browser")
			}
			return a, a.toaster.Info("Opening in browser...")
		}
		return a, nil

	case "l", "right":
		a.focusArea = FocusMain
		a.updateFocus()
		return a, nil

	default:
		a.sidebar.Update(msg)
		return a, nil
	}
}

func (a *App) handleGroupsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "l":
		if item := a.navList.SelectedItem(); item != nil {
			return a.selectNavItem(item)
		}
		return a, nil

	case "esc", "backspace", "h":
		if a.navList.HasFilter() {
			a.navList.ClearFilter()
			return a, nil
		}
		if len(a.groupPath) > 0 {
			a.groupPath = a.groupPath[:len(a.groupPath)-1]
			a.refreshNavList()
			a.saveState()
		}
		return a, nil

	case "p":
		return a.handlePinInGroups()

	case "w":
		return a.handleOpenInGroups()

	default:
		a.navList.Update(msg)
		return a, nil
	}
}

func (a *App) handlePinInGroups() (tea.Model, tea.Cmd) {
	if len(a.groupPath) > 0 {
		if item := a.navList.SelectedItem(); item != nil {
			if navItem, ok := item.Data.(*navItemData); ok && !navItem.isGroup {
				currentGroup := a.groupPath[len(a.groupPath)-1]
				wasPinned := currentGroup.IsPinned(navItem.workflowName)
				currentGroup.TogglePin(navItem.workflowName)
				if err := a.config.Save(a.configPath); err != nil {
					a.err = fmt.Errorf("failed to save config: %w", err)
					return a, a.toaster.Error("Failed to save")
				}
				a.refreshNavList()
				a.refreshPinnedList()
				a.saveState()
				if wasPinned {
					return a, a.toaster.Success("Unpinned workflow")
				}
				return a, a.toaster.Success("Pinned workflow")
			}
		}
	}
	return a, nil
}

func (a *App) handleOpenInGroups() (tea.Model, tea.Cmd) {
	if item := a.navList.SelectedItem(); item != nil {
		if navItem, ok := item.Data.(*navItemData); ok && !navItem.isGroup {
			if err := a.gh.OpenWorkflowInBrowser(navItem.workflowName); err != nil {
				a.err = err
				return a, a.toaster.Error("Failed to open browser")
			}
			return a, a.toaster.Info("Opening in browser...")
		}
	}
	return a, nil
}

func (a *App) handleRunsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		runID := a.runsTable.SelectedRunID()
		if runID > 0 {
			if err := a.gh.OpenRunInBrowser(runID); err != nil {
				a.err = err
				return a, a.toaster.Error("Failed to open browser")
			}
			return a, a.toaster.Info("Opening run in browser...")
		}
		return a, nil

	case "esc", "h", "backspace":
		a.viewMode = ViewGroups
		a.selectedWorkflow = ""
		a.selectedGroup = nil
		a.stopRefreshTicker()
		a.updateFocus()
		a.updateStatusBar()
		return a, nil

	default:
		a.runsTable.Update(msg)
		return a, nil
	}
}

func (a *App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.focusArea == FocusSidebar {
		a.sidebar.Update(msg)
	} else {
		a.navList.Update(msg)
	}
	return a, nil
}

func (a *App) isFiltering() bool {
	if a.focusArea == FocusSidebar {
		return a.sidebar.IsFiltering()
	}
	return a.navList.IsFiltering()
}

func (a *App) updateFocus() {
	a.sidebar.SetFocused(a.focusArea == FocusSidebar)
	a.navList.SetFocused(a.focusArea == FocusMain && a.viewMode == ViewGroups)
	a.runsTable.SetFocused(a.focusArea == FocusMain && a.viewMode == ViewRuns)
	a.updateHelpBar()
}
