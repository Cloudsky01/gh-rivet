package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) startRefreshTicker() {
	if a.refreshInterval <= 0 || !a.autoRefreshEnabled {
		return
	}
	a.stopRefreshTicker()
	a.refreshTicker = time.NewTicker(time.Duration(a.refreshInterval) * time.Second)
}

func (a *App) stopRefreshTicker() {
	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
		a.refreshTicker = nil
	}
}

func (a *App) getRefreshTickerCmd() tea.Cmd {
	if a.refreshTicker == nil {
		return nil
	}
	return func() tea.Msg {
		return refreshTickMsg{timestamp: <-a.refreshTicker.C}
	}
}

func (a *App) fetchWorkflowRunsCmd() tea.Msg {
	runs, err := a.gh.GetWorkflowRuns(a.selectedWorkflow, 20)
	return workflowRunsMsg{runs: runs, err: err}
}
