package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderLayout() string {
	barHeight := 2
	panelHeight := a.height - barHeight - 2

	sidebarWidth := 0
	if a.showSidebar {
		sidebarWidth = max(25, a.width/5)
	}
	mainWidth := a.width - sidebarWidth
	if a.showSidebar {
		mainWidth -= 2
	}

	var mainView string
	if a.viewMode == ViewRuns {
		a.runsTable.SetSize(mainWidth-2, panelHeight-2)
		mainView = a.wrapPanel(a.runsTable.View(), a.focusArea == FocusMain)
	} else {
		a.navList.SetSize(mainWidth-2, panelHeight-2)
		mainView = a.wrapPanel(a.navList.View(), a.focusArea == FocusMain)
	}

	var topRow string
	if a.showSidebar {
		a.sidebar.SetSize(sidebarWidth-2, panelHeight-2)
		sidebarView := a.wrapPanel(a.sidebar.View(), a.focusArea == FocusSidebar)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainView)
	} else {
		topRow = mainView
	}

	a.updateStatusBar()
	statusView := a.statusBar.View()
	helpView := a.helpBar.View()

	layout := lipgloss.JoinVertical(lipgloss.Left, topRow, statusView, helpView)

	if a.toaster.HasToasts() {
		toastView := a.toaster.View()
		layout = lipgloss.JoinVertical(lipgloss.Left, toastView, layout)
	}

	if a.spinner.IsActive() {
		spinnerView := a.spinner.View()
		layout = lipgloss.JoinVertical(lipgloss.Left, spinnerView, layout)
	}

	return layout
}

func (a *App) wrapPanel(content string, active bool) string {
	style := a.theme.BorderNormal
	if active {
		style = a.theme.BorderActive
	}
	return style.Render(content)
}

func (a *App) updateStatusBar() {
	repoSource := ""
	if a.isInsideGitRepo {
		repoSource = "git"
	} else if a.config.IsMultiRepo() {
		repoSource = "selected"
	}
	a.statusBar.SetRepository(a.activeRepository, repoSource)

	groupNames := make([]string, len(a.groupPath))
	for i, g := range a.groupPath {
		groupNames[i] = g.Name
	}
	a.statusBar.SetGroupPath(groupNames)
	a.statusBar.SetWorkflow(a.selectedWorkflow)
	a.statusBar.SetRefreshStatus(a.autoRefreshEnabled, a.refreshInterval)
	a.statusBar.SetLoading(a.loading)
}

func (a *App) updateHelpBar() {
	hints := []string{"[q]uit", "[?]help", "[:]cmd", "[ctrl+f]search"}

	if a.showSidebar {
		hints = append(hints, "[tab]switch", "[1]sidebar")
	}

	if a.focusArea == FocusSidebar {
		hints = append(hints, "[enter]select", "[p]unpin", "[w]web")
	} else if a.viewMode == ViewGroups {
		hints = append(hints, "[enter]select", "[/]filter")
		if len(a.groupPath) > 0 {
			hints = append(hints, "[h]back", "[p]pin", "[w]web")
		}
	} else {
		hints = append(hints, "[j/k]nav", "[w]open", "[h]back")
	}

	a.helpBar.SetHints(hints)
}
