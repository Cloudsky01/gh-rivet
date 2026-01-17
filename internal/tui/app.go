package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/github"
	"github.com/Cloudsky01/gh-rivet/internal/state"
	"github.com/Cloudsky01/gh-rivet/internal/tui/components"
	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

type workflowRunsMsg struct {
	runs []models.GHRun
	err  error
}

type refreshTickMsg struct {
	timestamp time.Time
}

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

// MenuOptions is deprecated, use AppOptions instead
type MenuOptions struct {
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
