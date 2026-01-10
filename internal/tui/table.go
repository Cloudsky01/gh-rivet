package tui

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

const (
	columnKeyID         = "id"
	columnKeyTitle      = "title"
	columnKeyStatus     = "status"
	columnKeyConclusion = "conclusion"
	columnKeyBranch     = "branch"
	columnKeyCreated    = "created"
)

func buildWorkflowRunsTable(runs []models.GHRun, pageSize int) table.Model {
	// Build rows from runs data
	var rows []table.Row
	for _, run := range runs {
		createdStr := run.CreatedAt.Format("2006-01-02 15:04:05")

		// Don't apply styling in the data - bubble-table handles it differently
		rows = append(rows, table.NewRow(table.RowData{
			columnKeyID:         strconv.Itoa(run.DatabaseID),
			columnKeyTitle:      run.DisplayTitle,
			columnKeyStatus:     run.Status,
			columnKeyConclusion: run.Conclusion,
			columnKeyBranch:     run.HeadBranch,
			columnKeyCreated:    createdStr,
		}))
	}

	// Define columns with appropriate widths
	columns := []table.Column{
		table.NewColumn(columnKeyID, "ID", 10),
		table.NewColumn(columnKeyTitle, "Title", 40),
		table.NewColumn(columnKeyStatus, "Status", 12),
		table.NewColumn(columnKeyConclusion, "Conclusion", 12),
		table.NewColumn(columnKeyBranch, "Branch", 20),
		table.NewColumn(columnKeyCreated, "Created", 19),
	}

	// Create table
	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue"))

	t := table.New(columns).
		WithRows(rows).
		Focused(true).
		WithPageSize(pageSize).
		BorderRounded().
		WithBaseStyle(lipgloss.NewStyle().BorderForeground(lipgloss.Color("240"))).
		HighlightStyle(highlightStyle).
		HeaderStyle(headerStyle)

	return t
}

func getStatusColor(status string) lipgloss.Color {
	switch status {
	case "completed":
		return lipgloss.Color("green")
	case "in_progress", "queued", "waiting", "requested", "pending":
		return lipgloss.Color("yellow")
	case "failed":
		return lipgloss.Color("red")
	default:
		return lipgloss.Color("white")
	}
}

func getConclusionColor(conclusion string) lipgloss.Color {
	switch conclusion {
	case "success":
		return lipgloss.Color("green")
	case "failure", "cancelled", "timed_out":
		return lipgloss.Color("red")
	case "neutral", "skipped", "stale":
		return lipgloss.Color("yellow")
	case "action_required":
		return lipgloss.Color("cyan")
	default:
		return lipgloss.Color("240")
	}
}

// getSelectedRunID gets the database ID of the currently selected run
func (m MenuModel) getSelectedRunID() int {
	if len(m.workflowRuns) == 0 {
		return 0
	}

	// Get the highlighted row from the table
	highlightedRow := m.table.HighlightedRow()
	if highlightedRow.Data == nil {
		return 0
	}

	idStr, ok := highlightedRow.Data[columnKeyID].(string)
	if !ok {
		return 0
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0
	}

	return id
}
