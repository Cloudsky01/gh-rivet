package tui

import (
	"fmt"
	"os"

	"github.com/Cloudsky01/gh-rivet/internal/state"
)

func (a *App) saveState() {
	s := &state.NavigationState{
		GroupPath: state.ExtractGroupIDs(a.groupPath),
		ListIndex: a.navList.Cursor(),
	}

	if a.viewMode == ViewRuns && a.selectedWorkflow != "" {
		s.ViewState = state.ViewWorkflowOutput
		s.SelectedWorkflow = a.selectedWorkflow
		s.FromPinnedView = a.selectedGroup != nil
	} else if a.focusArea == FocusSidebar {
		s.ViewState = state.ViewPinnedWorkflows
		s.PinnedListIndex = a.sidebar.Cursor()
	} else {
		s.ViewState = state.ViewBrowsingGroups
	}

	if err := s.Save(a.statePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}
}

func (a *App) restoreState() {
	savedState, err := state.Load(a.statePath)
	if err != nil {
		return
	}

	if len(savedState.GroupPath) > 0 {
		resolvedPath, ok := state.ResolveGroupPath(a.config, savedState.GroupPath)
		if ok && len(resolvedPath) > 0 {
			a.groupPath = resolvedPath
			a.refreshNavList()
		}
	}

	if savedState.ListIndex > 0 {
		a.navList.SetCursor(savedState.ListIndex)
	}

	switch savedState.ViewState {
	case state.ViewPinnedWorkflows:
		if len(a.config.GetAllPinnedWorkflows()) > 0 {
			a.focusArea = FocusSidebar
		}
	case state.ViewWorkflowOutput:
		if savedState.SelectedWorkflow != "" {
			a.selectedWorkflow = savedState.SelectedWorkflow
			a.viewMode = ViewRuns
			a.runsTable.SetVisible(true)
			runs, err := a.gh.GetWorkflowRuns(savedState.SelectedWorkflow, 20)
			if err != nil {
				a.err = err
			} else {
				a.workflowRuns = runs
				a.runsTable.SetRuns(runs, savedState.SelectedWorkflow)
			}
		}
	}

	a.updateFocus()
	a.updateStatusBar()
}
