package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("blue"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
)

func (m MenuModel) View() string {
	switch m.viewState {
	case browsingGroups:
		return m.renderBrowsingView()
	case viewingPinnedWorkflows:
		return m.renderPinnedView()
	case viewingWorkflowOutput:
		return m.renderWorkflowView()
	}
	return ""
}

func (m MenuModel) renderBrowsingView() string {
	view := m.list.View()

	if len(m.groupPath) > 0 {
		hint := hintStyle.Render("\nenter select | w open in web | p toggle pin | esc back | tab pinned | q quit")
		return view + hint
	}

	hint := hintStyle.Render("\nenter select | tab pinned workflows | q quit")
	return view + hint
}

func (m MenuModel) renderPinnedView() string {
	view := m.pinnedList.View()
	hint := hintStyle.Render("\nenter select | p unpin | w open in browser | tab/esc groups | q quit")
	return view + hint
}

func (m MenuModel) renderWorkflowView() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render(fmt.Sprintf("Workflow: %s", m.selectedWorkflow)))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	} else if len(m.workflowRuns) == 0 {
		s.WriteString("No workflow runs found.")
	} else {
		s.WriteString(m.table.View())
		s.WriteString("\n\n")
	}

	s.WriteString(hintStyle.Render("\nw open run in browser | esc back | q quit"))
	return s.String()
}
