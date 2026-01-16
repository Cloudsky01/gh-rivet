package tui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/github"
	"github.com/Cloudsky01/gh-rivet/internal/state"
	"github.com/Cloudsky01/gh-rivet/internal/tui/components"
	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

type ViewMode int

const (
	ViewGroups ViewMode = iota
	ViewRuns
)

type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusMain
)

type App struct {
	config     *config.Config
	configPath string
	statePath  string
	gh         *github.Client

	theme *theme.Theme

	sidebar     components.Sidebar
	navList     components.List
	runsTable   *components.RunsTable
	search      components.Search
	cmdPalette  components.CmdPalette
	helpOverlay components.HelpOverlay
	toaster     components.Toaster
	spinner     components.Spinner
	statusBar   components.StatusBar
	helpBar     components.HelpBar

	groupPath        []*config.Group
	selectedWorkflow string
	selectedGroup    *config.Group

	viewMode    ViewMode
	focusArea   FocusArea
	showSidebar bool
	width       int
	height      int

	workflowRuns []models.GHRun
	loading      bool
	err          error

	refreshInterval    int
	refreshTicker      *time.Ticker
	autoRefreshEnabled bool
}

type AppOptions struct {
	StartWithPinned bool
	StatePath       string
	NoRestoreState  bool
	RefreshInterval int
}

func NewApp(cfg *config.Config, configPath string, gh *github.Client, opts AppOptions) *App {
	t := theme.Default()

	statePath := opts.StatePath
	if statePath == "" {
		statePath = state.DefaultStatePath(configPath)
	}

	app := &App{
		config:             cfg,
		configPath:         configPath,
		statePath:          statePath,
		gh:                 gh,
		theme:              t,
		sidebar:            components.NewSidebar(t),
		navList:            components.NewList(t, "📁 Groups"),
		runsTable:          components.NewRunsTablePtr(t),
		search:             components.NewSearch(t),
		cmdPalette:         components.NewCmdPalette(t),
		helpOverlay:        components.NewHelpOverlay(t),
		toaster:            components.NewToaster(t),
		spinner:            components.NewSpinner(t),
		statusBar:          components.NewStatusBar(t),
		helpBar:            components.NewHelpBar(t),
		groupPath:          []*config.Group{},
		viewMode:           ViewGroups,
		focusArea:          FocusMain,
		showSidebar:        true,
		refreshInterval:    opts.RefreshInterval,
		autoRefreshEnabled: opts.RefreshInterval > 0,
	}

	app.search.SetSearchFunc(func(query string) []components.SearchResult {
		return app.performGlobalSearch(query)
	})

	app.setupCommands()
	app.refreshNavList()
	app.refreshPinnedList()
	app.updateStatusBar()

	if !opts.NoRestoreState {
		app.restoreState()
	}

	if opts.StartWithPinned && len(cfg.GetAllPinnedWorkflows()) > 0 {
		app.focusArea = FocusSidebar
	}

	app.updateFocus()

	return app
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleResize(msg)

	case tea.KeyMsg:
		return a.handleKey(msg)

	case workflowRunsMsg:
		a.loading = false
		a.spinner.Stop()
		if msg.err != nil {
			a.err = msg.err
			a.runsTable.SetError(msg.err)
			cmds = append(cmds, a.toaster.Error("Failed to load runs"))
		} else {
			a.workflowRuns = msg.runs
			a.runsTable.SetRuns(msg.runs, a.selectedWorkflow)
		}
		a.runsTable.SetLoading(false)
		cmds = append(cmds, a.getRefreshTickerCmd())
		return a, tea.Batch(cmds...)

	case refreshTickMsg:
		if a.selectedWorkflow != "" && !a.loading {
			a.loading = true
			a.runsTable.SetLoading(true)
			return a, tea.Batch(a.fetchWorkflowRunsCmd, a.getRefreshTickerCmd())
		}
		return a, a.getRefreshTickerCmd()

	case components.ToastExpiredMsg:
		a.toaster.Update(msg)
		return a, nil

	default:
		if a.spinner.IsActive() {
			cmd := a.spinner.Update(msg)
			if cmd != nil {
				return a, cmd
			}
		}
	}

	return a, nil
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}

	if a.helpOverlay.IsActive() {
		return a.helpOverlay.View()
	}

	if a.cmdPalette.IsActive() {
		return a.cmdPalette.View()
	}

	if a.search.IsActive() {
		return a.search.View()
	}

	return a.renderLayout()
}

func (a *App) handleResize(msg tea.WindowSizeMsg) (*App, tea.Cmd) {
	a.width = msg.Width
	a.height = msg.Height

	barHeight := 2
	panelHeight := a.height - barHeight - 2

	sidebarWidth := 0
	if a.showSidebar {
		sidebarWidth = max(25, a.width/5)
	}
	mainWidth := a.width - sidebarWidth - 2

	a.sidebar.SetSize(sidebarWidth-2, panelHeight)
	a.navList.SetSize(mainWidth-2, panelHeight)
	a.runsTable.SetSize(mainWidth-2, panelHeight)
	a.search.SetSize(a.width, a.height)
	a.cmdPalette.SetSize(a.width, a.height)
	a.helpOverlay.SetSize(a.width, a.height)
	a.toaster.SetWidth(a.width)
	a.statusBar.SetSize(a.width)
	a.helpBar.SetSize(a.width)

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.helpOverlay.IsActive() {
		a.helpOverlay.Update(msg)
		return a, nil
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
		if a.showSidebar {
			if a.focusArea == FocusSidebar {
				a.focusArea = FocusMain
			} else {
				a.focusArea = FocusSidebar
			}
			a.updateFocus()
		}
		return a, nil

	case "shift+tab":
		if a.showSidebar {
			if a.focusArea == FocusMain {
				a.focusArea = FocusSidebar
			} else {
				a.focusArea = FocusMain
			}
			a.updateFocus()
		}
		return a, nil

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

	case "ctrl+t":
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

	case "w":
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

	default:
		a.navList.Update(msg)
		return a, nil
	}
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

func (a *App) selectNavItem(item *components.ListItem) (*App, tea.Cmd) {
	navItem, ok := item.Data.(*navItemData)
	if !ok {
		return a, nil
	}

	if navItem.isGroup {
		if navItem.group != nil {
			a.groupPath = append(a.groupPath, navItem.group)
			a.navList.ClearFilter()
			a.refreshNavList()
			a.saveState()
		}
		return a, nil
	}

	return a.selectWorkflow(navItem.workflowName, nil)
}

func (a *App) selectWorkflowFromSidebar(item *components.PinnedItem) (*App, tea.Cmd) {
	group, _ := item.Data.(*config.Group)
	return a.selectWorkflow(item.WorkflowName, group)
}

func (a *App) selectWorkflow(name string, group *config.Group) (*App, tea.Cmd) {
	a.selectedWorkflow = name
	a.selectedGroup = group
	a.loading = true
	a.viewMode = ViewRuns
	a.runsTable.SetVisible(true)
	a.runsTable.SetLoading(true)
	a.focusArea = FocusMain
	a.updateFocus()
	a.startRefreshTicker()
	a.updateStatusBar()
	a.saveState()
	return a, tea.Batch(a.spinner.Start("Loading runs..."), a.fetchWorkflowRunsCmd)
}

func (a *App) navigateToSearchResult(result *components.SearchResult) (*App, tea.Cmd) {
	if result.Type == "group" {
		a.groupPath = a.resolveGroupPath(result.GroupPath)
		if result.Data != nil {
			if group, ok := result.Data.(*config.Group); ok {
				a.groupPath = append(a.groupPath, group)
			}
		}
		a.refreshNavList()
		a.viewMode = ViewGroups
		a.focusArea = FocusMain
		a.updateFocus()
		a.saveState()
		return a, nil
	}

	a.groupPath = a.resolveGroupPath(result.GroupPath)
	a.refreshNavList()

	var group *config.Group
	if len(a.groupPath) > 0 {
		group = a.groupPath[len(a.groupPath)-1]
	}

	return a.selectWorkflow(result.WorkflowName, group)
}

type navItemData struct {
	isGroup      bool
	group        *config.Group
	workflowName string
	isPinned     bool
}

func (a *App) refreshNavList() {
	items := a.buildNavItems()
	a.navList.SetItems(items)

	if len(a.groupPath) == 0 {
		a.navList.SetTitle("📁 Groups")
	} else {
		current := a.groupPath[len(a.groupPath)-1]
		a.navList.SetTitle("📁 " + current.Name)
	}
}

func (a *App) buildNavItems() []components.ListItem {
	var items []components.ListItem

	var groups []config.Group
	if len(a.groupPath) == 0 {
		groups = a.config.Groups
	} else {
		currentGroup := a.groupPath[len(a.groupPath)-1]
		groups = currentGroup.Groups

		workflows := make([]string, 0)
		workflowDefs := make(map[string]*config.Workflow)

		workflows = append(workflows, currentGroup.Workflows...)

		for i := range currentGroup.WorkflowDefs {
			wf := &currentGroup.WorkflowDefs[i]
			workflowDefs[wf.File] = wf
			found := false
			for _, w := range workflows {
				if w == wf.File {
					found = true
					break
				}
			}
			if !found {
				workflows = append(workflows, wf.File)
			}
		}

		var pinnedWorkflows []string
		var unpinnedWorkflows []string
		for _, wf := range workflows {
			if currentGroup.IsPinned(wf) {
				pinnedWorkflows = append(pinnedWorkflows, wf)
			} else {
				unpinnedWorkflows = append(unpinnedWorkflows, wf)
			}
		}

		getDisplayName := func(filename string) string {
			if wfDef, ok := workflowDefs[filename]; ok && wfDef.Name != "" {
				return wfDef.Name
			}
			return filename
		}

		for _, wf := range pinnedWorkflows {
			items = append(items, components.ListItem{
				ID:          wf,
				Title:       getDisplayName(wf),
				Description: wf,
				Icon:        a.theme.Icons.Pin,
				Data: &navItemData{
					isGroup:      false,
					workflowName: wf,
					isPinned:     true,
				},
			})
		}

		for _, wf := range unpinnedWorkflows {
			items = append(items, components.ListItem{
				ID:          wf,
				Title:       getDisplayName(wf),
				Description: wf,
				Icon:        a.theme.Icons.Workflow,
				Data: &navItemData{
					isGroup:      false,
					workflowName: wf,
					isPinned:     false,
				},
			})
		}
	}

	for i := range groups {
		group := &groups[i]
		workflowCount := a.countWorkflows(group)

		items = append(items, components.ListItem{
			ID:          group.ID,
			Title:       group.Name,
			Description: fmt.Sprintf("%d workflows", workflowCount),
			Icon:        a.theme.Icons.Folder,
			Data: &navItemData{
				isGroup: true,
				group:   group,
			},
		})
	}

	return items
}

func (a *App) countWorkflows(group *config.Group) int {
	count := len(group.Workflows) + len(group.WorkflowDefs)
	for i := range group.Groups {
		count += a.countWorkflows(&group.Groups[i])
	}
	return count
}

func (a *App) refreshPinnedList() {
	pinnedWorkflows := a.config.GetAllPinnedWorkflows()
	items := make([]components.PinnedItem, len(pinnedWorkflows))

	for i, pw := range pinnedWorkflows {
		items[i] = components.PinnedItem{
			WorkflowName: pw.WorkflowName,
			GroupName:    pw.Group.Name,
			GroupID:      pw.Group.ID,
			Data:         pw.Group,
		}
	}

	a.sidebar.SetItems(items)
}

func (a *App) updateStatusBar() {
	a.statusBar.SetRepository(a.config.Repository)

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

func (a *App) performGlobalSearch(query string) []components.SearchResult {
	var results []components.SearchResult

	var searchGroup func(group *config.Group, path []string)
	searchGroup = func(group *config.Group, path []string) {
		results = append(results, components.SearchResult{
			Type:      "group",
			Name:      group.Name,
			GroupPath: path,
			Data:      group,
		})

		currentPath := append(path, group.Name)

		workflowDefs := make(map[string]string)
		for i := range group.WorkflowDefs {
			wf := &group.WorkflowDefs[i]
			if wf.Name != "" {
				workflowDefs[wf.File] = wf.Name
			}
		}

		allWorkflows := make([]string, 0)
		allWorkflows = append(allWorkflows, group.Workflows...)
		for i := range group.WorkflowDefs {
			wf := &group.WorkflowDefs[i]
			found := false
			for _, w := range allWorkflows {
				if w == wf.File {
					found = true
					break
				}
			}
			if !found {
				allWorkflows = append(allWorkflows, wf.File)
			}
		}

		for _, wfFile := range allWorkflows {
			displayName := wfFile
			if name, ok := workflowDefs[wfFile]; ok {
				displayName = name
			}
			results = append(results, components.SearchResult{
				Type:         "workflow",
				Name:         displayName,
				Description:  wfFile,
				GroupPath:    currentPath,
				WorkflowName: wfFile,
				Data:         group,
			})
		}

		for i := range group.Groups {
			searchGroup(&group.Groups[i], currentPath)
		}
	}

	for i := range a.config.Groups {
		searchGroup(&a.config.Groups[i], []string{})
	}

	return components.FuzzySearchItems(results, query)
}

func (a *App) resolveGroupPath(names []string) []*config.Group {
	if len(names) == 0 {
		return []*config.Group{}
	}

	path := make([]*config.Group, 0, len(names))
	currentGroups := a.config.Groups

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

func RunApp(app *App) error {
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func NewAppFromMenuOptions(cfg *config.Config, configPath string, gh *github.Client, opts MenuOptions) *App {
	return NewApp(cfg, configPath, gh, AppOptions{
		StartWithPinned: opts.StartWithPinned,
		StatePath:       opts.StatePath,
		NoRestoreState:  opts.NoRestoreState,
		RefreshInterval: opts.RefreshInterval,
	})
}
