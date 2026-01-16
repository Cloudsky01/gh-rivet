package tui

import (
	"strings"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/sahilm/fuzzy"
)

// SearchResult represents a single search result item
type SearchResult struct {
	Type         string        // "group" or "workflow"
	Name         string        // Display name
	Description  string        // Group description or workflow filename
	GroupPath    []string      // Path to the item (breadcrumb)
	Group        *config.Group // The group this belongs to (for navigation)
	WorkflowName string        // Workflow filename (if workflow)
	Score        int           // Fuzzy match score
}

// searchableItem implements fuzzy.Source for fuzzy matching
type searchableItem struct {
	result     SearchResult
	searchText string // Combined text for searching
}

type searchableItems []searchableItem

func (s searchableItems) String(i int) string {
	return s[i].searchText
}

func (s searchableItems) Len() int {
	return len(s)
}

// collectAllSearchableItems recursively collects all groups and workflows from config
func collectAllSearchableItems(cfg *config.Config) []searchableItem {
	var items []searchableItem

	for i := range cfg.Groups {
		items = append(items, collectFromGroup(&cfg.Groups[i], []string{})...)
	}

	return items
}

// collectFromGroup recursively collects searchable items from a group
func collectFromGroup(g *config.Group, parentPath []string) []searchableItem {
	var items []searchableItem

	currentPath := append([]string{}, parentPath...)
	currentPath = append(currentPath, g.Name)

	// Add the group itself as a searchable item
	groupSearchText := g.Name
	if g.Description != "" {
		groupSearchText += " " + g.Description
	}
	items = append(items, searchableItem{
		result: SearchResult{
			Type:        "group",
			Name:        g.Name,
			Description: g.Description,
			GroupPath:   parentPath, // Path to parent (we navigate INTO this group)
			Group:       g,
		},
		searchText: groupSearchText,
	})

	// Add workflows from this group
	workflowDefs := make(map[string]*config.Workflow)
	for i := range g.WorkflowDefs {
		wf := &g.WorkflowDefs[i]
		workflowDefs[wf.File] = wf
	}

	// Collect all workflows (both from Workflows and WorkflowDefs)
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

		// Search text includes both display name and filename
		searchText := displayName + " " + wfFile
		items = append(items, searchableItem{
			result: SearchResult{
				Type:         "workflow",
				Name:         displayName,
				Description:  wfFile,
				GroupPath:    currentPath,
				Group:        g,
				WorkflowName: wfFile,
			},
			searchText: searchText,
		})
	}

	// Recurse into nested groups
	for i := range g.Groups {
		items = append(items, collectFromGroup(&g.Groups[i], currentPath)...)
	}

	return items
}

// globalSearch performs a fuzzy search across all groups and workflows
func globalSearch(cfg *config.Config, query string) []SearchResult {
	if query == "" {
		return nil
	}

	allItems := collectAllSearchableItems(cfg)
	if len(allItems) == 0 {
		return nil
	}

	// Use sahilm/fuzzy for ranked fuzzy matching
	matches := fuzzy.FindFrom(query, searchableItems(allItems))

	results := make([]SearchResult, 0, len(matches))
	for _, match := range matches {
		result := allItems[match.Index].result
		result.Score = match.Score
		results = append(results, result)
	}

	return results
}

// formatGroupPath formats a group path for display
func formatGroupPath(path []string) string {
	if len(path) == 0 {
		return "/"
	}
	return strings.Join(path, " > ")
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
