package tui

import (
	"strings"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/sahilm/fuzzy"
)

type SearchResult struct {
	Type         string
	Name         string
	Description  string
	GroupPath    []string
	Group        *config.Group
	WorkflowName string
	Score        int
}

type searchableItem struct {
	result     SearchResult
	searchText string
}

type searchableItems []searchableItem

func (s searchableItems) String(i int) string { return s[i].searchText }
func (s searchableItems) Len() int            { return len(s) }

func collectAllSearchableItems(cfg *config.Config) []searchableItem {
	var items []searchableItem
	for i := range cfg.Groups {
		items = append(items, collectFromGroup(&cfg.Groups[i], []string{})...)
	}
	return items
}

func collectFromGroup(g *config.Group, parentPath []string) []searchableItem {
	var items []searchableItem

	currentPath := append([]string{}, parentPath...)
	currentPath = append(currentPath, g.Name)

	groupSearchText := g.Name
	if g.Description != "" {
		groupSearchText += " " + g.Description
	}
	items = append(items, searchableItem{
		result: SearchResult{
			Type:        "group",
			Name:        g.Name,
			Description: g.Description,
			GroupPath:   parentPath,
			Group:       g,
		},
		searchText: groupSearchText,
	})

	workflowDefs := make(map[string]*config.Workflow)
	for i := range g.WorkflowDefs {
		wf := &g.WorkflowDefs[i]
		workflowDefs[wf.File] = wf
	}

	seenWorkflows := make(map[string]bool)
	allWorkflows := append([]string{}, g.Workflows...)
	for _, wf := range g.WorkflowDefs {
		if !contains(allWorkflows, wf.File) {
			allWorkflows = append(allWorkflows, wf.File)
		}
	}

	for _, wfFile := range allWorkflows {
		if seenWorkflows[wfFile] {
			continue
		}
		seenWorkflows[wfFile] = true

		displayName := wfFile
		if wfDef, ok := workflowDefs[wfFile]; ok && wfDef.Name != "" {
			displayName = wfDef.Name
		}

		items = append(items, searchableItem{
			result: SearchResult{
				Type:         "workflow",
				Name:         displayName,
				Description:  wfFile,
				GroupPath:    currentPath,
				Group:        g,
				WorkflowName: wfFile,
			},
			searchText: displayName + " " + wfFile,
		})
	}

	for i := range g.Groups {
		items = append(items, collectFromGroup(&g.Groups[i], currentPath)...)
	}

	return items
}

func globalSearch(cfg *config.Config, query string) []SearchResult {
	if query == "" {
		return nil
	}

	allItems := collectAllSearchableItems(cfg)
	if len(allItems) == 0 {
		return nil
	}

	matches := fuzzy.FindFrom(query, searchableItems(allItems))

	results := make([]SearchResult, 0, len(matches))
	for _, match := range matches {
		result := allItems[match.Index].result
		result.Score = match.Score
		results = append(results, result)
	}

	return results
}

func formatGroupPath(path []string) string {
	if len(path) == 0 {
		return "/"
	}
	return strings.Join(path, " > ")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
