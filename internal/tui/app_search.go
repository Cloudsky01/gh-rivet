package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/tui/components"
)

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
