package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

// fuzzyMatch performs a simple fuzzy match check
// Returns true if all characters of query appear in target in order (case-insensitive)
func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}

	query = strings.ToLower(query)
	target = strings.ToLower(target)

	queryIdx := 0
	for _, targetChar := range target {
		if queryIdx < len(query) && rune(query[queryIdx]) == targetChar {
			queryIdx++
		}
	}

	return queryIdx == len(query)
}

// filterPinnedItems filters pinned list items based on the filter input
func filterPinnedItems(items []list.Item, filterInput string) []list.Item {
	if filterInput == "" {
		return items
	}

	var filtered []list.Item
	for _, item := range items {
		pli, ok := item.(pinnedListItem)
		if !ok {
			continue
		}

		// Match against workflow name and group path
		if fuzzyMatch(filterInput, pli.workflowName) ||
			fuzzyMatch(filterInput, pli.groupPath) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// filterNavigationItems filters navigation list items based on the filter input
// This works for both groups and workflows at any nesting level
func filterNavigationItems(items []list.Item, filterInput string) []list.Item {
	if filterInput == "" {
		return items
	}

	var filtered []list.Item
	for _, item := range items {
		li, ok := item.(listItem)
		if !ok {
			continue
		}

		// Match against name and description (works for both groups and workflows)
		// Groups have names like "📂 backend" and descriptions like "Backend services - 5 workflows"
		// Workflows have names like "📄 deploy.yml" and descriptions like "Workflow (pinned)"
		if fuzzyMatch(filterInput, li.name) ||
			fuzzyMatch(filterInput, li.description) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}
