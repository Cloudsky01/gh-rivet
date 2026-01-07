package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

func buildWorkflowRunsTable(runs []models.GHRun) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 12},
		{Title: "Title", Width: 50},
		{Title: "Status", Width: 12},
		{Title: "Conclusion", Width: 15},
		{Title: "Branch", Width: 25},
		{Title: "Created", Width: 19},
	}

	var rows []table.Row
	for _, run := range runs {
		createdStr := run.CreatedAt.Format("2006-01-02 15:04:05")

		statusStyled := lipgloss.NewStyle().
			Foreground(getStatusColor(run.Status)).
			Render(run.Status)
		conclusionStyled := lipgloss.NewStyle().
			Foreground(getConclusionColor(run.Conclusion)).
			Render(run.Conclusion)

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", run.DatabaseID),
			run.DisplayTitle,
			statusStyled,
			conclusionStyled,
			run.HeadBranch,
			createdStr,
		})
	}

	tableHeight := min(len(rows), 10)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("blue"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

func getStatusColor(status string) lipgloss.Color {
	switch status {
	case "completed":
		return "green"
	case "in_progress", "queued", "waiting", "requested", "pending":
		return "yellow"
	case "failed":
		return "red"
	default:
		return "white"
	}
}

func getConclusionColor(conclusion string) lipgloss.Color {
	switch conclusion {
	case "success":
		return "green"
	case "failure", "cancelled", "timed_out":
		return "red"
	case "neutral", "skipped", "stale":
		return "yellow"
	case "action_required":
		return "cyan"
	default:
		return "240"
	}
}
