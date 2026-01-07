package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/config"
)

type listItem struct {
	isGroup      bool
	name         string
	description  string
	group        *config.Group
	workflowName string
	isPinned     bool
}

func (i listItem) Title() string       { return i.name }
func (i listItem) Description() string { return i.description }
func (i listItem) FilterValue() string { return i.name }

type pinnedListItem struct {
	workflowName string
	groupPath    string // Display a path like "Group > Subgroup"
	group        *config.Group
}

func (i pinnedListItem) Title() string       { return i.workflowName }
func (i pinnedListItem) Description() string { return "pinned " + i.groupPath }
func (i pinnedListItem) FilterValue() string { return i.workflowName + " " + i.groupPath }

func buildListItems(cfg *config.Config, groupPath []*config.Group) []list.Item {
	var items []list.Item

	var groups []config.Group
	var workflows []string
	var workflowDefs map[string]*config.Workflow
	var currentGroup *config.Group

	if len(groupPath) == 0 {
		groups = cfg.Groups
	} else {
		currentGroup = groupPath[len(groupPath)-1]
		groups = currentGroup.Groups
		workflows = append([]string{}, currentGroup.Workflows...)

		workflowDefs = make(map[string]*config.Workflow)
		for i := range currentGroup.WorkflowDefs {
			wf := &currentGroup.WorkflowDefs[i]
			workflowDefs[wf.File] = wf
			if !slices.Contains(workflows, wf.File) {
				workflows = append(workflows, wf.File)
			}
		}

		for _, pinned := range currentGroup.PinnedWorkflows {
			if !slices.Contains(workflows, pinned) {
				workflows = append(workflows, pinned)
			}
		}
	}

	for i := range groups {
		workflowCount := len(groups[i].GetAllWorkflows())
		description := fmt.Sprintf("%d workflows", workflowCount)
		if groups[i].Description != "" {
			description = fmt.Sprintf("%s - %d workflows", groups[i].Description, workflowCount)
		}

		items = append(items, listItem{
			isGroup:     true,
			name:        groups[i].Name,
			description: description,
			group:       &groups[i],
		})
	}

	var pinnedWorkflows []string
	var unpinnedWorkflows []string

	for _, wf := range workflows {
		if currentGroup != nil && currentGroup.IsPinned(wf) {
			pinnedWorkflows = append(pinnedWorkflows, wf)
		} else {
			unpinnedWorkflows = append(unpinnedWorkflows, wf)
		}
	}

	getWorkflowDisplay := func(filename string, pinned bool) (name, desc string) {
		name = filename
		if wfDef, ok := workflowDefs[filename]; ok && wfDef.Name != "" {
			name = wfDef.Name
		}

		if pinned {
			desc = "Workflow (pinned)"
		} else {
			desc = "Workflow"
		}
		return
	}

	for _, wf := range pinnedWorkflows {
		name, desc := getWorkflowDisplay(wf, true)
		items = append(items, listItem{
			isGroup:      false,
			name:         name,
			description:  desc,
			workflowName: wf,
			isPinned:     true,
		})
	}

	for _, wf := range unpinnedWorkflows {
		name, desc := getWorkflowDisplay(wf, false)
		items = append(items, listItem{
			isGroup:      false,
			name:         name,
			description:  desc,
			workflowName: wf,
			isPinned:     false,
		})
	}

	return items
}

func buildPinnedListItems(cfg *config.Config) []list.Item {
	var items []list.Item

	pinnedWorkflows := cfg.GetAllPinnedWorkflows()
	for _, pw := range pinnedWorkflows {
		groupPath := strings.Join(pw.GroupPath, " > ")
		items = append(items, pinnedListItem{
			workflowName: pw.WorkflowName,
			groupPath:    groupPath,
			group:        pw.Group,
		})
	}

	return items
}

func createPinnedList(cfg *config.Config) list.Model {
	items := buildPinnedListItems(cfg)

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("blue")).
		BorderForeground(lipgloss.Color("blue"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")).
		BorderForeground(lipgloss.Color("blue"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Pinned Workflows"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Padding(0, 0, 1, 0)

	return l
}
