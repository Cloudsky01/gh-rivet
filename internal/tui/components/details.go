package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
	"github.com/Cloudsky01/gh-rivet/pkg/models"
)

// Details displays workflow details and recent runs preview
type Details struct {
	workflowName string
	runs         []models.GHRun
	width        int
	height       int
	focused      bool
	loading      bool
	err          error
	theme        *theme.Theme
}

// NewDetails creates a new details component
func NewDetails(t *theme.Theme) Details {
	return Details{
		theme: t,
	}
}

// SetWorkflow sets the current workflow
func (d *Details) SetWorkflow(name string) {
	d.workflowName = name
}

// SetRuns sets the workflow runs
func (d *Details) SetRuns(runs []models.GHRun) {
	d.runs = runs
}

// SetSize sets dimensions
func (d *Details) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetFocused sets focus state
func (d *Details) SetFocused(focused bool) {
	d.focused = focused
}

// IsFocused returns focus state
func (d *Details) IsFocused() bool {
	return d.focused
}

// SetLoading sets loading state
func (d *Details) SetLoading(loading bool) {
	d.loading = loading
}

// SetError sets error state
func (d *Details) SetError(err error) {
	d.err = err
}

// WorkflowName returns current workflow
func (d *Details) WorkflowName() string {
	return d.workflowName
}

// Clear clears the details panel
func (d *Details) Clear() {
	d.workflowName = ""
	d.runs = nil
	d.loading = false
	d.err = nil
}

// View renders the details panel
func (d *Details) View() string {
	var b strings.Builder

	// Header
	titleStyle := d.theme.Title
	if d.focused {
		titleStyle = d.theme.TitleActive
	}
	b.WriteString(titleStyle.Render("📊 Details"))
	b.WriteString("\n")
	b.WriteString(d.theme.Divider(d.width - 2))
	b.WriteString("\n\n")

	if d.workflowName == "" {
		// No workflow selected
		emptyText := d.theme.TextMuted.Render("  Select a workflow to\n  view details")
		b.WriteString(emptyText)
	} else {
		// Workflow name
		nameLabel := d.theme.TextDim.Render("Workflow:")
		workflowName := d.theme.Selected.Render(d.workflowName)
		b.WriteString(fmt.Sprintf("%s\n%s\n\n", nameLabel, workflowName))

		if d.loading {
			loadingText := d.theme.StatusInProgress.Render(
				d.theme.Icons.InProgress + " Loading runs...")
			b.WriteString(loadingText)
		} else if d.err != nil {
			errText := d.theme.StatusError.Render(
				fmt.Sprintf("Error: %v", d.err))
			b.WriteString(errText)
		} else if len(d.runs) > 0 {
			recentLabel := d.theme.TextDim.Render("Recent Runs:")
			b.WriteString(recentLabel)
			b.WriteString("\n\n")

			// Show up to 5 most recent runs
			displayCount := min(5, len(d.runs))
			for i := 0; i < displayCount; i++ {
				run := d.runs[i]

				// Status icon and color
				statusIcon, statusStyle := d.theme.StatusIcon(run.Status, run.Conclusion)

				statusText := statusStyle.Render(statusIcon)
				runInfo := d.theme.Text.Render(
					fmt.Sprintf(" #%d %s", run.DatabaseID, run.HeadBranch))

				b.WriteString(fmt.Sprintf("%s%s\n", statusText, runInfo))
			}

			if len(d.runs) > displayCount {
				moreText := d.theme.TextMuted.Render(
					fmt.Sprintf("\n  +%d more runs", len(d.runs)-displayCount))
				b.WriteString(moreText)
			}

			b.WriteString("\n")

			// Hint to view full table
			hintText := d.theme.TextMuted.Render("[r] View all runs")
			b.WriteString(hintText)
		} else {
			noRunsText := d.theme.TextMuted.Render("No runs found")
			b.WriteString(noRunsText)
		}
	}

	// Pad remaining space
	contentLines := strings.Count(b.String(), "\n")
	remaining := d.height - contentLines - 2
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	// Help hint
	var hints string
	if d.workflowName != "" {
		hints = "[w] open in browser [esc] clear"
	} else {
		hints = "[d] focus"
	}
	b.WriteString("\n")
	b.WriteString(d.theme.TextMuted.Render(hints))

	return lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Render(b.String())
}
