// Package tui provides the terminal user interface for rivet.
// This file contains the main application model using a component-based architecture.
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

// ViewMode represents the current view/panel focus
type ViewMode int

const (
	ViewSidebar ViewMode = iota
	ViewNavigation
	ViewDetails
	ViewRuns
)

// App is the main application model
type App struct {
	// Configuration
	config     *config.Config
	configPath string
	statePath  string
	gh         *github.Client

	// Theming
	theme *theme.Theme

	// Components
	sidebar   components.Sidebar
	navList   components.List
	details   components.Details
	runsTable components.RunsTable
	search    components.Search
	statusBar components.StatusBar
	helpBar   components.HelpBar

	// Navigation state
	groupPath        []*config.Group
	selectedWorkflow string
	selectedGroup    *config.Group

	// UI state
	activeView  ViewMode
	showSidebar bool
	showRuns    bool
	width       int
	height      int

	// Data state
	workflowRuns []models.GHRun
	loading      bool
	err          error

	// Refresh timer
	refreshInterval    int
	refreshTicker      *time.Ticker
	autoRefreshEnabled bool
}

// AppOptions contains options for creating a new App
type AppOptions struct {
	StartWithPinned bool
	StatePath       string
	NoRestoreState  bool
	RefreshInterval int
}

// NewApp creates a new App with the given configuration
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
		details:            components.NewDetails(t),
		runsTable:          components.NewRunsTable(t),
		search:             components.NewSearch(t),
		statusBar:          components.NewStatusBar(t),
		helpBar:            components.NewHelpBar(t),
		groupPath:          []*config.Group{},
		activeView:         ViewNavigation,
		showSidebar:        true,
		showRuns:           false,
		refreshInterval:    opts.RefreshInterval,
		autoRefreshEnabled: opts.RefreshInterval > 0,
	}

	// Set up search function
	app.search.SetSearchFunc(func(query string) []components.SearchResult {
		return app.performGlobalSearch(query)
	})

	// Initialize components
	app.refreshNavList()
	app.refreshPinnedList()
	app.updateStatusBar()

	// Restore state if requested
	if !opts.NoRestoreState {
		app.restoreState()
	}

	// Start with pinned view if requested
	if opts.StartWithPinned && len(cfg.GetAllPinnedWorkflows()) > 0 {
		app.activeView = ViewSidebar
	}

	app.updateFocus()

	return app
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleResize(msg)

	case tea.KeyMsg:
		return a.handleKey(msg)

	case workflowRunsMsg:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err
			a.details.SetError(msg.err)
			a.runsTable.SetError(msg.err)
		} else {
			a.workflowRuns = msg.runs
			a.details.SetRuns(msg.runs)
			a.runsTable.SetRuns(msg.runs, a.selectedWorkflow)
		}
		a.details.SetLoading(false)
		a.runsTable.SetLoading(false)
		return a, a.getRefreshTickerCmd()

	case refreshTickMsg:
		if a.selectedWorkflow != "" && !a.loading {
			a.loading = true
			a.details.SetLoading(true)
			return a, tea.Batch(a.fetchWorkflowRunsCmd, a.getRefreshTickerCmd())
		}
		return a, a.getRefreshTickerCmd()
	}

	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}

	// Show search overlay if active
	if a.search.IsActive() {
		return a.search.View()
	}

	return a.renderLayout()
}

// handleResize handles window resize events
func (a *App) handleResize(msg tea.WindowSizeMsg) (*App, tea.Cmd) {
	a.width = msg.Width
	a.height = msg.Height

	// Calculate panel dimensions
	sidebarWidth := 0
	if a.showSidebar {
		sidebarWidth = max(25, a.width/5)
	}
	mainWidth := a.width - sidebarWidth - 4 // Account for borders

	// Split main area between nav and details
	navWidth := mainWidth * 2 / 3
	detailsWidth := mainWidth - navWidth

	// Calculate heights
	barHeight := 2 // status bar + help bar
	panelHeight := a.height - barHeight - 2

	runsHeight := 0
	if a.showRuns {
		runsHeight = max(10, panelHeight/3)
		panelHeight = panelHeight - runsHeight - 2
	}

	// Update component sizes
	a.sidebar.SetSize(sidebarWidth-2, panelHeight)
	a.navList.SetSize(navWidth-2, panelHeight)
	a.details.SetSize(detailsWidth-2, panelHeight)
	a.runsTable.SetSize(a.width-4, runsHeight)
	a.search.SetSize(a.width, a.height)
	a.statusBar.SetSize(a.width)
	a.helpBar.SetSize(a.width)

	return a, nil
}

// handleKey handles key press events
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode first
	if a.search.IsActive() {
		result, cmd := a.search.Update(msg)
		if result != nil {
			return a.navigateToSearchResult(result)
		}
		return a, cmd
	}

	// Handle component-specific filtering
	if a.isFiltering() {
		return a.handleFilterKey(msg)
	}

	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		a.stopRefreshTicker()
		a.saveState()
		return a, tea.Quit

	case "1":
		a.showSidebar = !a.showSidebar
		return a.handleResize(tea.WindowSizeMsg{Width: a.width, Height: a.height})

	case "tab":
		a.cycleView(1)
		a.updateFocus()
		return a, nil

	case "shift+tab":
		a.cycleView(-1)
		a.updateFocus()
		return a, nil

	case "s":
		if a.showSidebar {
			a.activeView = ViewSidebar
			a.updateFocus()
		}
		return a, nil

	case "g":
		a.activeView = ViewNavigation
		a.updateFocus()
		return a, nil

	case "d":
		a.activeView = ViewDetails
		a.updateFocus()
		return a, nil

	case "r":
		if a.selectedWorkflow != "" {
			a.showRuns = !a.showRuns
			a.runsTable.SetVisible(a.showRuns)
			if a.showRuns {
				a.activeView = ViewRuns
			} else if a.activeView == ViewRuns {
				a.activeView = ViewDetails
			}
			a.updateFocus()
			return a.handleResize(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		}
		return a, nil

	case "ctrl+f":
		a.search.Open()
		return a, nil

	case "ctrl+r":
		if a.selectedWorkflow != "" && !a.loading {
			a.loading = true
			a.details.SetLoading(true)
			if a.refreshInterval > 0 && a.autoRefreshEnabled {
				a.startRefreshTicker()
			}
			return a, a.fetchWorkflowRunsCmd
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

	// Delegate to active view
	return a.handleViewKey(msg)
}

// handleViewKey handles keys for the active view
func (a *App) handleViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.activeView {
	case ViewSidebar:
		return a.handleSidebarKey(msg)
	case ViewNavigation:
		return a.handleNavigationKey(msg)
	case ViewDetails:
		return a.handleDetailsKey(msg)
	case ViewRuns:
		return a.handleRunsKey(msg)
	}
	return a, nil
}

// handleSidebarKey handles keys when sidebar is focused
func (a *App) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if item := a.sidebar.SelectedItem(); item != nil {
			a.selectWorkflow(item.WorkflowName, item.Data.(*config.Group))
			return a, a.fetchWorkflowRunsCmd
		}
		return a, nil

	case "p":
		// Unpin workflow
		if item := a.sidebar.SelectedItem(); item != nil {
			if group, ok := item.Data.(*config.Group); ok {
				group.TogglePin(item.WorkflowName)
				if err := a.config.Save(a.configPath); err != nil {
					a.err = fmt.Errorf("failed to save config: %w", err)
				}
				a.refreshPinnedList()
				a.refreshNavList()
				a.saveState()
			}
		}
		return a, nil

	case "w":
		// Open in browser
		if item := a.sidebar.SelectedItem(); item != nil {
			if err := a.gh.OpenWorkflowInBrowser(item.WorkflowName); err != nil {
				a.err = err
			}
		}
		return a, nil

	default:
		a.sidebar.Update(msg)
		return a, nil
	}
}

// handleNavigationKey handles keys when navigation is focused
func (a *App) handleNavigationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		// Toggle pin
		if len(a.groupPath) > 0 {
			if item := a.navList.SelectedItem(); item != nil {
				if navItem, ok := item.Data.(*navItemData); ok && !navItem.isGroup {
					currentGroup := a.groupPath[len(a.groupPath)-1]
					currentGroup.TogglePin(navItem.workflowName)
					if err := a.config.Save(a.configPath); err != nil {
						a.err = fmt.Errorf("failed to save config: %w", err)
					}
					a.refreshNavList()
					a.refreshPinnedList()
					a.saveState()
				}
			}
		}
		return a, nil

	case "w":
		// Open in browser
		if item := a.navList.SelectedItem(); item != nil {
			if navItem, ok := item.Data.(*navItemData); ok && !navItem.isGroup {
				if err := a.gh.OpenWorkflowInBrowser(navItem.workflowName); err != nil {
					a.err = err
				}
			}
		}
		return a, nil

	default:
		a.navList.Update(msg)
		return a, nil
	}
}

// handleDetailsKey handles keys when details is focused
func (a *App) handleDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		if a.selectedWorkflow != "" {
			if err := a.gh.OpenWorkflowInBrowser(a.selectedWorkflow); err != nil {
				a.err = err
			}
		}
		return a, nil

	case "esc":
		a.clearSelection()
		a.activeView = ViewNavigation
		a.updateFocus()
		return a, nil
	}
	return a, nil
}

// handleRunsKey handles keys when runs table is focused
func (a *App) handleRunsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "w":
		// Open run in browser
		runID := a.runsTable.SelectedRunID()
		if runID > 0 {
			if err := a.gh.OpenRunInBrowser(runID); err != nil {
				a.err = err
			}
		}
		return a, nil

	case "esc":
		a.showRuns = false
		a.runsTable.SetVisible(false)
		a.activeView = ViewDetails
		a.updateFocus()
		return a.handleResize(tea.WindowSizeMsg{Width: a.width, Height: a.height})

	default:
		a.runsTable.Update(msg)
		return a, nil
	}
}

// handleFilterKey handles keys when a component is filtering
func (a *App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.activeView {
	case ViewSidebar:
		a.sidebar.Update(msg)
	case ViewNavigation:
		a.navList.Update(msg)
	}
	return a, nil
}

// isFiltering returns true if any component is in filter mode
func (a *App) isFiltering() bool {
	return a.sidebar.IsFiltering() || a.navList.IsFiltering()
}

// cycleView cycles through views
func (a *App) cycleView(direction int) {
	views := []ViewMode{ViewNavigation, ViewDetails}
	if a.showSidebar {
		views = append([]ViewMode{ViewSidebar}, views...)
	}
	if a.showRuns {
		views = append(views, ViewRuns)
	}

	current := 0
	for i, v := range views {
		if v == a.activeView {
			current = i
			break
		}
	}

	next := (current + direction + len(views)) % len(views)
	a.activeView = views[next]
}

// updateFocus updates component focus states
func (a *App) updateFocus() {
	a.sidebar.SetFocused(a.activeView == ViewSidebar)
	a.navList.SetFocused(a.activeView == ViewNavigation)
	a.details.SetFocused(a.activeView == ViewDetails)
	a.runsTable.SetFocused(a.activeView == ViewRuns)
	a.updateHelpBar()
}

// selectNavItem handles selection of a navigation item
func (a *App) selectNavItem(item *components.ListItem) (*App, tea.Cmd) {
	navItem, ok := item.Data.(*navItemData)
	if !ok {
		return a, nil
	}

	if navItem.isGroup {
		// Enter group
		if navItem.group != nil {
			a.groupPath = append(a.groupPath, navItem.group)
			a.navList.ClearFilter()
			a.refreshNavList()
			a.saveState()
		}
		return a, nil
	}

	// Select workflow
	a.selectWorkflow(navItem.workflowName, nil)
	return a, a.fetchWorkflowRunsCmd
}

// selectWorkflow selects a workflow and prepares to fetch runs
func (a *App) selectWorkflow(name string, group *config.Group) {
	a.selectedWorkflow = name
	a.selectedGroup = group
	a.loading = true
	a.details.SetWorkflow(name)
	a.details.SetLoading(true)
	a.runsTable.SetLoading(true)
	a.activeView = ViewDetails
	a.updateFocus()
	a.startRefreshTicker()
	a.updateStatusBar()
	a.saveState()
}

// clearSelection clears the current workflow selection
func (a *App) clearSelection() {
	a.selectedWorkflow = ""
	a.selectedGroup = nil
	a.workflowRuns = nil
	a.details.Clear()
	a.stopRefreshTicker()
	a.updateStatusBar()
}

// navigateToSearchResult navigates to a search result
func (a *App) navigateToSearchResult(result *components.SearchResult) (*App, tea.Cmd) {
	if result.Type == "group" {
		// Navigate to group
		a.groupPath = a.resolveGroupPath(result.GroupPath)
		if result.Data != nil {
			if group, ok := result.Data.(*config.Group); ok {
				a.groupPath = append(a.groupPath, group)
			}
		}
		a.refreshNavList()
		a.activeView = ViewNavigation
		a.updateFocus()
		a.saveState()
		return a, nil
	}

	// Navigate to workflow
	a.groupPath = a.resolveGroupPath(result.GroupPath)
	a.refreshNavList()

	// Get the group reference
	var group *config.Group
	if len(a.groupPath) > 0 {
		group = a.groupPath[len(a.groupPath)-1]
	}

	a.selectWorkflow(result.WorkflowName, group)
	return a, a.fetchWorkflowRunsCmd
}

// Helper types and methods

type navItemData struct {
	isGroup      bool
	group        *config.Group
	workflowName string
	isPinned     bool
}

// refreshNavList refreshes the navigation list items
func (a *App) refreshNavList() {
	items := a.buildNavItems()
	a.navList.SetItems(items)

	// Update title based on path
	if len(a.groupPath) == 0 {
		a.navList.SetTitle("📁 Groups")
	} else {
		parts := make([]string, len(a.groupPath))
		for i, g := range a.groupPath {
			parts[i] = g.Name
		}
		a.navList.SetTitle("📁 " + parts[len(parts)-1])
	}
}

// buildNavItems builds the navigation list items for the current group path
func (a *App) buildNavItems() []components.ListItem {
	var items []components.ListItem

	var groups []config.Group
	if len(a.groupPath) == 0 {
		groups = a.config.Groups
	} else {
		currentGroup := a.groupPath[len(a.groupPath)-1]
		groups = currentGroup.Groups

		// Build workflow list from all sources
		workflows := make([]string, 0)
		workflowDefs := make(map[string]*config.Workflow)

		// Add workflows from Workflows array
		workflows = append(workflows, currentGroup.Workflows...)

		// Add workflows from WorkflowDefs
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

		// Separate pinned and unpinned
		var pinnedWorkflows []string
		var unpinnedWorkflows []string
		for _, wf := range workflows {
			if currentGroup.IsPinned(wf) {
				pinnedWorkflows = append(pinnedWorkflows, wf)
			} else {
				unpinnedWorkflows = append(unpinnedWorkflows, wf)
			}
		}

		// Helper to get display name
		getDisplayName := func(filename string) string {
			if wfDef, ok := workflowDefs[filename]; ok && wfDef.Name != "" {
				return wfDef.Name
			}
			return filename
		}

		// Add pinned workflows first
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

		// Add unpinned workflows
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

	// Add subgroups
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

// countWorkflows counts total workflows in a group recursively
func (a *App) countWorkflows(group *config.Group) int {
	count := len(group.GetAllWorkflows())
	for i := range group.Groups {
		count += a.countWorkflows(&group.Groups[i])
	}
	return count
}

// refreshPinnedList refreshes the sidebar pinned items
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

// updateStatusBar updates the status bar content
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

// updateHelpBar updates the help bar hints
func (a *App) updateHelpBar() {
	hints := components.GlobalHints()

	switch a.activeView {
	case ViewSidebar:
		hints = append(hints, components.SidebarHints()...)
	case ViewNavigation:
		hints = append(hints, components.NavigationHints(len(a.groupPath) > 0)...)
	case ViewDetails:
		hints = append(hints, components.DetailsHints(a.selectedWorkflow != "")...)
	case ViewRuns:
		hints = append(hints, components.RunsHints()...)
	}

	a.helpBar.SetHints(hints)
}

// performGlobalSearch searches all groups and workflows
func (a *App) performGlobalSearch(query string) []components.SearchResult {
	var results []components.SearchResult

	var searchGroup func(group *config.Group, path []string)
	searchGroup = func(group *config.Group, path []string) {
		// Add group as result
		results = append(results, components.SearchResult{
			Type:      "group",
			Name:      group.Name,
			GroupPath: path,
			Data:      group,
		})

		// Add workflows - collect from both Workflows and WorkflowDefs
		currentPath := append(path, group.Name)

		// Build workflow display names map
		workflowDefs := make(map[string]string)
		for i := range group.WorkflowDefs {
			wf := &group.WorkflowDefs[i]
			if wf.Name != "" {
				workflowDefs[wf.File] = wf.Name
			}
		}

		// Add all workflows from this group (not recursive - GetAllWorkflows is recursive)
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

		// Recurse into subgroups
		for i := range group.Groups {
			searchGroup(&group.Groups[i], currentPath)
		}
	}

	// Search all top-level groups
	for i := range a.config.Groups {
		searchGroup(&a.config.Groups[i], []string{})
	}

	// Filter results using fuzzy search
	return components.FuzzySearchItems(results, query)
}

// resolveGroupPath converts group names to group pointers
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

// Refresh ticker methods

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

// State persistence

func (a *App) saveState() {
	s := &state.NavigationState{
		GroupPath: state.ExtractGroupIDs(a.groupPath),
		ListIndex: a.navList.Cursor(),
	}

	if a.selectedWorkflow != "" {
		s.ViewState = state.ViewWorkflowOutput
		s.SelectedWorkflow = a.selectedWorkflow
		s.FromPinnedView = a.selectedGroup != nil
	} else if a.activeView == ViewSidebar {
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

	// Restore group path
	if len(savedState.GroupPath) > 0 {
		resolvedPath, ok := state.ResolveGroupPath(a.config, savedState.GroupPath)
		if ok && len(resolvedPath) > 0 {
			a.groupPath = resolvedPath
			a.refreshNavList()
		}
	}

	// Restore list selection
	if savedState.ListIndex > 0 {
		a.navList.SetCursor(savedState.ListIndex)
	}

	// Restore view state
	switch savedState.ViewState {
	case state.ViewPinnedWorkflows:
		if len(a.config.GetAllPinnedWorkflows()) > 0 {
			a.activeView = ViewSidebar
		}
	case state.ViewWorkflowOutput:
		if savedState.SelectedWorkflow != "" {
			a.selectedWorkflow = savedState.SelectedWorkflow
			a.details.SetWorkflow(savedState.SelectedWorkflow)
			// Fetch runs
			runs, err := a.gh.GetWorkflowRuns(savedState.SelectedWorkflow, 20)
			if err != nil {
				a.err = err
			} else {
				a.workflowRuns = runs
				a.details.SetRuns(runs)
				a.runsTable.SetRuns(runs, savedState.SelectedWorkflow)
			}
			a.activeView = ViewDetails
		}
	}

	a.updateFocus()
	a.updateStatusBar()
}

// renderLayout renders the main layout
func (a *App) renderLayout() string {
	// Calculate panel dimensions
	sidebarWidth := 0
	if a.showSidebar {
		sidebarWidth = max(25, a.width/5)
	}
	mainWidth := a.width - sidebarWidth
	if a.showSidebar {
		mainWidth -= 2 // Border spacing
	}

	// Split main area
	navWidth := mainWidth * 2 / 3
	detailsWidth := mainWidth - navWidth

	// Calculate heights
	panelHeight := a.height - 2 // Status + help bars
	runsHeight := 0
	if a.showRuns {
		runsHeight = max(10, panelHeight/3)
		panelHeight = panelHeight - runsHeight - 2
	}

	// Build top panels
	var panels []string

	if a.showSidebar {
		a.sidebar.SetSize(sidebarWidth-2, panelHeight-2)
		sidebarView := a.wrapPanel(a.sidebar.View(), a.activeView == ViewSidebar)
		panels = append(panels, sidebarView)
	}

	a.navList.SetSize(navWidth-2, panelHeight-2)
	navView := a.wrapPanel(a.navList.View(), a.activeView == ViewNavigation)
	panels = append(panels, navView)

	a.details.SetSize(detailsWidth-2, panelHeight-2)
	detailsView := a.wrapPanel(a.details.View(), a.activeView == ViewDetails)
	panels = append(panels, detailsView)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, panels...)

	// Add runs panel if visible
	if a.showRuns {
		a.runsTable.SetSize(a.width-4, runsHeight-2)
		runsView := a.wrapPanel(a.runsTable.View(), a.activeView == ViewRuns)
		topRow = lipgloss.JoinVertical(lipgloss.Left, topRow, runsView)
	}

	// Add status and help bars
	a.updateStatusBar()
	statusView := a.statusBar.View()
	helpView := a.helpBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, topRow, statusView, helpView)
}

// wrapPanel wraps a panel with a border
func (a *App) wrapPanel(content string, active bool) string {
	style := a.theme.BorderNormal
	if active {
		style = a.theme.BorderActive
	}
	return style.Render(content)
}

// RunApp runs the application
func RunApp(app *App) error {
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// NewAppFromMenuOptions creates an App from the legacy MenuOptions for backwards compatibility
func NewAppFromMenuOptions(cfg *config.Config, configPath string, gh *github.Client, opts MenuOptions) *App {
	return NewApp(cfg, configPath, gh, AppOptions{
		StartWithPinned: opts.StartWithPinned,
		StatePath:       opts.StatePath,
		NoRestoreState:  opts.NoRestoreState,
		RefreshInterval: opts.RefreshInterval,
	})
}
