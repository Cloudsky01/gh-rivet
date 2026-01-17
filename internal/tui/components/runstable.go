package components

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

const (
	colStatus     = "status"
	colID         = "id"
	colTitle      = "title"
	colConclusion = "conclusion"
	colBranch     = "branch"
	colCreated    = "created"
)

// RunsTable displays workflow runs in a table
type RunsTable struct {
	table        table.Model
	runs         []models.GHRun
	width        int
	height       int
	focused      bool
	visible      bool
	workflowName string
	loading      bool
	err          error
	theme        *theme.Theme
	pageSize     int
}

// NewRunsTable creates a new runs table component
func NewRunsTable(t *theme.Theme) RunsTable {
	return RunsTable{
		theme:    t,
		visible:  false,
		pageSize: 15,
	}
}

// NewRunsTablePtr creates a new runs table component as a pointer
func NewRunsTablePtr(t *theme.Theme) *RunsTable {
	rt := NewRunsTable(t)
	return &rt
}

// SetRuns sets the workflow runs data
func (r *RunsTable) SetRuns(runs []models.GHRun, workflowName string) {
	r.runs = runs
	r.workflowName = workflowName
	r.err = nil
	r.rebuildTable()
}

// SetSize sets dimensions
func (r *RunsTable) SetSize(width, height int) {
	if r.width == width && r.height == height {
		return // No change, don't rebuild
	}
	r.width = width
	r.height = height
	r.rebuildTable()
}

// SetFocused sets focus state
func (r *RunsTable) SetFocused(focused bool) {
	r.focused = focused
	if r.table.GetHighlightedRowIndex() >= 0 {
		r.table = r.table.Focused(focused)
	}
}

// IsFocused returns focus state
func (r *RunsTable) IsFocused() bool {
	return r.focused
}

// SetVisible sets visibility
func (r *RunsTable) SetVisible(visible bool) {
	r.visible = visible
}

// IsVisible returns visibility
func (r *RunsTable) IsVisible() bool {
	return r.visible
}

// Toggle toggles visibility
func (r *RunsTable) Toggle() {
	r.visible = !r.visible
}

// SetLoading sets loading state
func (r *RunsTable) SetLoading(loading bool) {
	r.loading = loading
}

// SetError sets error state
func (r *RunsTable) SetError(err error) {
	r.err = err
}

// SelectedRunID returns the ID of the selected run
func (r *RunsTable) SelectedRunID() int {
	row := r.table.HighlightedRow()
	if row.Data == nil {
		return 0
	}
	if idStr, ok := row.Data[colID].(string); ok {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return 0
		}
		return id
	}
	return 0
}

// Runs returns the current runs
func (r *RunsTable) Runs() []models.GHRun {
	return r.runs
}

// WorkflowName returns the current workflow name
func (r *RunsTable) WorkflowName() string {
	return r.workflowName
}

func (r *RunsTable) rebuildTable() {
	if r.width == 0 || r.height == 0 || len(r.runs) == 0 {
		return
	}

	// Preserve current highlighted row index
	currentIdx := r.table.GetHighlightedRowIndex()
	if currentIdx < 0 {
		currentIdx = 0
	}

	// Calculate column widths dynamically
	idWidth := 10
	statusWidth := 12
	conclusionWidth := 12
	branchWidth := 20
	createdWidth := 19
	titleWidth := max(20, r.width-idWidth-statusWidth-conclusionWidth-branchWidth-createdWidth-10)

	columns := []table.Column{
		table.NewColumn(colID, "ID", idWidth),
		table.NewColumn(colTitle, "Title", titleWidth),
		table.NewColumn(colStatus, "Status", statusWidth),
		table.NewColumn(colConclusion, "Conclusion", conclusionWidth),
		table.NewColumn(colBranch, "Branch", branchWidth),
		table.NewColumn(colCreated, "Created", createdWidth),
	}

	rows := make([]table.Row, len(r.runs))
	for i, run := range r.runs {
		createdStr := run.CreatedAt.Format("2006-01-02 15:04:05")

		// Truncate title if needed
		title := run.DisplayTitle
		if len(title) > titleWidth-2 {
			title = title[:titleWidth-5] + "..."
		}

		rows[i] = table.NewRow(table.RowData{
			colID:         strconv.Itoa(run.DatabaseID),
			colTitle:      title,
			colStatus:     run.Status,
			colConclusion: run.Conclusion,
			colBranch:     run.HeadBranch,
			colCreated:    createdStr,
		})
	}

	// Highlighted row style
	highlightStyle := lipgloss.NewStyle().
		Background(r.theme.Colors.BgHighlight).
		Foreground(r.theme.Colors.Primary).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(r.theme.Colors.Primary)

	r.table = table.New(columns).
		WithRows(rows).
		WithPageSize(r.pageSize).
		Focused(r.focused).
		BorderRounded().
		WithBaseStyle(lipgloss.NewStyle().
			Foreground(r.theme.Colors.Text).
			BorderForeground(r.theme.Colors.Border)).
		HighlightStyle(highlightStyle).
		HeaderStyle(headerStyle).
		WithHighlightedRow(currentIdx)
}

func (r *RunsTable) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			r.table = r.table.WithHighlightedRow(r.table.GetHighlightedRowIndex() + 1)
			return nil
		case "k":
			idx := r.table.GetHighlightedRowIndex()
			if idx > 0 {
				r.table = r.table.WithHighlightedRow(idx - 1)
			}
			return nil
		case "g":
			r.table = r.table.WithHighlightedRow(0)
			return nil
		case "G":
			r.table = r.table.WithHighlightedRow(len(r.runs) - 1)
			return nil
		}
	}

	var cmd tea.Cmd
	r.table, cmd = r.table.Update(msg)
	return cmd
}

func (r *RunsTable) View() string {

	var b strings.Builder

	// Header
	titleStyle := r.theme.Title
	if r.focused {
		titleStyle = r.theme.TitleActive
	}
	title := fmt.Sprintf("📋 Runs: %s", r.workflowName)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Status info
	statusInfo := r.theme.TextMuted.Render(fmt.Sprintf("Total: %d runs", len(r.runs)))
	b.WriteString(statusInfo)
	b.WriteString("\n\n")

	// Content
	if r.loading {
		b.WriteString(r.theme.StatusInProgress.Render(r.theme.Icons.InProgress + " Loading runs..."))
	} else if r.err != nil {
		b.WriteString(r.theme.StatusError.Render(fmt.Sprintf("Error: %v", r.err)))
	} else if len(r.runs) == 0 {
		b.WriteString(r.theme.TextMuted.Render("No workflow runs found"))
	} else {
		b.WriteString(r.table.View())
	}

	b.WriteString("\n")

	// Help hints
	hints := r.theme.TextMuted.Render("[j/k] nav [w] open in browser [esc] close")
	b.WriteString(hints)

	return lipgloss.NewStyle().
		Width(r.width).
		Height(r.height).
		Render(b.String())
}
