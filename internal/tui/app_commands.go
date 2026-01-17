package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/tui/components"
)

func (a *App) setupCommands() {
	cmds := []components.Command{
		{Name: "quit", Aliases: []string{"q", "exit"}, Description: "Exit the application"},
		{Name: "refresh", Aliases: []string{"r"}, Description: "Refresh current view"},
		{Name: "search", Aliases: []string{"s", "find"}, Description: "Open global search"},
		{Name: "help", Aliases: []string{"h", "?"}, Description: "Show help"},
		{Name: "pin", Aliases: []string{"p"}, Description: "Pin/unpin selected workflow"},
		{Name: "open", Aliases: []string{"o", "web", "browser"}, Description: "Open in browser"},
		{Name: "sidebar", Aliases: []string{"1"}, Description: "Toggle sidebar"},
		{Name: "back", Aliases: []string{"b"}, Description: "Go back"},
	}
	a.cmdPalette.SetCommands(cmds)
}

func (a *App) executeCommand(cmd *components.Command) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch cmd.Name {
	case "quit":
		a.stopRefreshTicker()
		a.saveState()
		return a, tea.Quit

	case "refresh":
		if a.selectedWorkflow != "" && !a.loading {
			a.loading = true
			a.runsTable.SetLoading(true)
			cmds = append(cmds, a.spinner.Start("Refreshing..."))
			cmds = append(cmds, a.fetchWorkflowRunsCmd)
		}

	case "search":
		a.search.Open()

	case "help":
		a.helpOverlay.Toggle()

	case "pin":
		return a.handlePinAction()

	case "open":
		return a.handleOpenAction()

	case "sidebar":
		a.showSidebar = !a.showSidebar
		if !a.showSidebar && a.focusArea == FocusSidebar {
			a.focusArea = FocusMain
		}
		a.updateFocus()
		return a.handleResize(tea.WindowSizeMsg{Width: a.width, Height: a.height})

	case "back":
		if a.viewMode == ViewRuns {
			a.viewMode = ViewGroups
			a.selectedWorkflow = ""
			a.selectedGroup = nil
			a.stopRefreshTicker()
			a.updateFocus()
			a.updateStatusBar()
		} else if len(a.groupPath) > 0 {
			a.groupPath = a.groupPath[:len(a.groupPath)-1]
			a.refreshNavList()
			a.saveState()
		}
	}

	if len(cmds) > 0 {
		return a, tea.Batch(cmds...)
	}
	return a, nil
}

func (a *App) handlePinAction() (tea.Model, tea.Cmd) {
	if a.focusArea == FocusSidebar {
		if item := a.sidebar.SelectedItem(); item != nil {
			if group, ok := item.Data.(*config.Group); ok {
				group.TogglePin(item.WorkflowName)
				if err := a.config.Save(a.configPath); err != nil {
					return a, a.toaster.Error("Failed to save")
				}
				a.refreshPinnedList()
				a.refreshNavList()
				a.saveState()
				return a, a.toaster.Success("Unpinned workflow")
			}
		}
	} else if a.viewMode == ViewGroups && len(a.groupPath) > 0 {
		if item := a.navList.SelectedItem(); item != nil {
			if navItem, ok := item.Data.(*navItemData); ok && !navItem.isGroup {
				currentGroup := a.groupPath[len(a.groupPath)-1]
				wasPinned := currentGroup.IsPinned(navItem.workflowName)
				currentGroup.TogglePin(navItem.workflowName)
				if err := a.config.Save(a.configPath); err != nil {
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

func (a *App) handleOpenAction() (tea.Model, tea.Cmd) {
	var err error
	if a.focusArea == FocusSidebar {
		if item := a.sidebar.SelectedItem(); item != nil {
			err = a.gh.OpenWorkflowInBrowser(item.WorkflowName)
		}
	} else if a.viewMode == ViewGroups {
		if item := a.navList.SelectedItem(); item != nil {
			if navItem, ok := item.Data.(*navItemData); ok && !navItem.isGroup {
				err = a.gh.OpenWorkflowInBrowser(navItem.workflowName)
			}
		}
	} else if a.viewMode == ViewRuns {
		runID := a.runsTable.SelectedRunID()
		if runID > 0 {
			err = a.gh.OpenRunInBrowser(runID)
		}
	}

	if err != nil {
		return a, a.toaster.Error("Failed to open browser")
	}
	return a, a.toaster.Info("Opening in browser...")
}
