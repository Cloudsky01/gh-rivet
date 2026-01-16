package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/github"
	"github.com/Cloudsky01/gh-rivet/internal/state"
	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

type workflowRunsMsg struct {
	runs []models.GHRun
	err  error
}

type refreshTickMsg struct {
	timestamp time.Time
}

type viewState int

const (
	browsingGroups viewState = iota
	viewingPinnedWorkflows
	viewingWorkflowOutput
)

type MenuModel struct {
	config           *config.Config
	configPath       string
	statePath        string
	gh               *github.Client
	viewState        viewState
	groupPath        []*config.Group
	list             list.Model
	pinnedList       list.Model
	table            table.Model
	selectedWorkflow string
	selectedGroup    *config.Group
	workflowRuns     []models.GHRun
	err              error
	width, height    int
	loading          bool

	activePanel   PanelType
	showRunsPanel bool

	tablePageSize int

	sidebarFilterActive     bool
	sidebarFilterInput      string
	sidebarFilteredIndex    int
	navigationFilterActive  bool
	navigationFilterInput   string
	navigationFilteredIndex int

	// Refresh timer fields
	refreshInterval    int
	refreshTicker      *time.Ticker
	autoRefreshEnabled bool

	// Global search fields
	globalSearchActive  bool
	globalSearchInput   string
	globalSearchResults []SearchResult
	globalSearchIndex   int

	// Help modal
	helpModalActive bool
}

type MenuOptions struct {
	StartWithPinned bool
	StatePath       string
	NoRestoreState  bool
	RefreshInterval int
}

func NewMenuModel(cfg *config.Config, configPath string, gh *github.Client, opts MenuOptions) MenuModel {
	items := buildListItems(cfg, []*config.Group{})

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("blue")).
		BorderForeground(lipgloss.Color("blue"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")).
		BorderForeground(lipgloss.Color("blue"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Browse Groups"
	l.SetShowStatusBar(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Padding(0, 0, 1, 0)

	statePath := opts.StatePath
	if statePath == "" {
		statePath = state.DefaultStatePath(configPath)
	}

	// Initialize pinned list
	pinnedList := createPinnedList(cfg)

	m := MenuModel{
		config:             cfg,
		configPath:         configPath,
		statePath:          statePath,
		gh:                 gh,
		viewState:          browsingGroups,
		groupPath:          []*config.Group{},
		list:               l,
		pinnedList:         pinnedList,
		activePanel:        NavigationPanel, // Default to navigation panel
		showRunsPanel:      false,
		tablePageSize:      10,
		refreshInterval:    opts.RefreshInterval,
		autoRefreshEnabled: opts.RefreshInterval > 0,
	}

	if !opts.NoRestoreState {
		if savedState, err := state.Load(m.statePath); err == nil {
			m.restoreState(savedState)
		}
	}

	if opts.StartWithPinned && len(cfg.GetAllPinnedWorkflows()) > 0 {
		m.pinnedList = createPinnedList(cfg)
		m.viewState = viewingPinnedWorkflows
	}

	return m
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) fetchWorkflowRuns() ([]models.GHRun, error) {
	return m.gh.GetWorkflowRuns(m.selectedWorkflow, 20)
}

func (m MenuModel) fetchWorkflowRunsCmd() tea.Msg {
	runs, err := m.gh.GetWorkflowRuns(m.selectedWorkflow, 20)
	return workflowRunsMsg{runs: runs, err: err}
}

func RunMenu(m MenuModel) error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func (m MenuModel) restoreState(s *state.NavigationState) {
	if len(s.GroupPath) > 0 {
		resolvedPath, ok := state.ResolveGroupPath(m.config, s.GroupPath)
		if ok && len(resolvedPath) > 0 {
			m.groupPath = resolvedPath
			m.list.SetItems(buildListItems(m.config, m.groupPath))

			breadcrumb := ""
			for i, group := range m.groupPath {
				if i > 0 {
					breadcrumb += " > "
				}
				breadcrumb += group.Name
			}
			m.list.Title = breadcrumb
		}
	}

	if s.ListIndex > 0 {
		items := m.list.Items()
		if s.ListIndex < len(items) {
			m.list.Select(s.ListIndex)
		}
	}

	switch s.ViewState {
	case state.ViewPinnedWorkflows:
		if len(m.config.GetAllPinnedWorkflows()) > 0 {
			m.pinnedList = createPinnedList(m.config)
			if s.PinnedListIndex > 0 && s.PinnedListIndex < len(m.pinnedList.Items()) {
				m.pinnedList.Select(s.PinnedListIndex)
			}
			m.viewState = viewingPinnedWorkflows
		}

	case state.ViewWorkflowOutput:
		m.selectedWorkflow = s.SelectedWorkflow
		if s.FromPinnedView {
			for _, pw := range m.config.GetAllPinnedWorkflows() {
				if pw.WorkflowName == s.SelectedWorkflow {
					m.selectedGroup = pw.Group
					break
				}
			}
		}
		if m.selectedWorkflow != "" {
			runs, err := m.fetchWorkflowRuns()
			if err != nil {
				m.err = err
			} else {
				m.workflowRuns = runs
				m.table = buildWorkflowRunsTable(runs, m.tablePageSize)
			}
			m.viewState = viewingWorkflowOutput
		}
	}
}

func (m MenuModel) saveState() {
	s := &state.NavigationState{
		GroupPath:       state.ExtractGroupIDs(m.groupPath),
		ListIndex:       m.list.Index(),
		PinnedListIndex: m.pinnedList.Index(),
	}

	switch m.viewState {
	case browsingGroups:
		s.ViewState = state.ViewBrowsingGroups
	case viewingPinnedWorkflows:
		s.ViewState = state.ViewPinnedWorkflows
	case viewingWorkflowOutput:
		s.ViewState = state.ViewWorkflowOutput
		s.SelectedWorkflow = m.selectedWorkflow
		s.FromPinnedView = m.selectedGroup != nil
	}

	if err := s.Save(m.statePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}
}

// startRefreshTicker starts the auto-refresh timer if enabled and interval is valid
func (m *MenuModel) startRefreshTicker() {
	if m.refreshInterval <= 0 || !m.autoRefreshEnabled {
		return
	}

	m.stopRefreshTicker() // Stop any existing ticker
	m.refreshTicker = time.NewTicker(time.Duration(m.refreshInterval) * time.Second)
}

// stopRefreshTicker stops the auto-refresh timer
func (m *MenuModel) stopRefreshTicker() {
	if m.refreshTicker != nil {
		m.refreshTicker.Stop()
		m.refreshTicker = nil
	}
}

// getRefreshTickerCmd returns a command that waits for the next refresh tick
func (m *MenuModel) getRefreshTickerCmd() tea.Cmd {
	if m.refreshTicker == nil {
		return nil
	}

	return func() tea.Msg {
		return refreshTickMsg{timestamp: <-m.refreshTicker.C}
	}
}
