package tui

import (
	"testing"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/charmbracelet/bubbles/list"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		target   string
		expected bool
	}{
		{
			name:     "empty query matches everything",
			query:    "",
			target:   "anything",
			expected: true,
		},
		{
			name:     "exact match",
			query:    "test",
			target:   "test",
			expected: true,
		},
		{
			name:     "case insensitive match",
			query:    "test",
			target:   "TEST",
			expected: true,
		},
		{
			name:     "fuzzy match with gaps",
			query:    "tst",
			target:   "test",
			expected: true,
		},
		{
			name:     "fuzzy match in longer string",
			query:    "wrk",
			target:   "workflow-check",
			expected: true,
		},
		{
			name:     "no match",
			query:    "xyz",
			target:   "test",
			expected: false,
		},
		{
			name:     "partial match at end",
			query:    "flow",
			target:   "workflow",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzyMatch(tt.query, tt.target)
			if result != tt.expected {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.query, tt.target, result, tt.expected)
			}
		})
	}
}

func TestFilterPinnedItems(t *testing.T) {
	// Create test items
	group := &config.Group{Name: "test-group"}
	items := []list.Item{
		pinnedListItem{
			workflowName: "deploy-prod",
			groupPath:    "production",
			group:        group,
		},
		pinnedListItem{
			workflowName: "test-ci",
			groupPath:    "development",
			group:        group,
		},
		pinnedListItem{
			workflowName: "deploy-staging",
			groupPath:    "staging",
			group:        group,
		},
	}

	tests := []struct {
		name          string
		filterInput   string
		expectedCount int
	}{
		{
			name:          "empty filter returns all",
			filterInput:   "",
			expectedCount: 3,
		},
		{
			name:          "filter by workflow name",
			filterInput:   "deploy",
			expectedCount: 2,
		},
		{
			name:          "filter by group path",
			filterInput:   "prod",
			expectedCount: 1,
		},
		{
			name:          "fuzzy match",
			filterInput:   "dpl",
			expectedCount: 2,
		},
		{
			name:          "no matches",
			filterInput:   "xyz",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterPinnedItems(items, tt.filterInput)
			if len(filtered) != tt.expectedCount {
				t.Errorf("filterPinnedItems with input %q returned %d items, want %d",
					tt.filterInput, len(filtered), tt.expectedCount)
			}
		})
	}
}

func TestFilterNavigationItems(t *testing.T) {
	// Create test items with both groups and workflows
	backendGroup := &config.Group{Name: "backend"}
	frontendGroup := &config.Group{Name: "frontend"}
	items := []list.Item{
		listItem{
			isGroup:      true,
			name:         "📂 backend",
			description:  "Backend services - 10 workflows",
			group:        backendGroup,
			workflowName: "",
		},
		listItem{
			isGroup:      true,
			name:         "📂 frontend",
			description:  "Frontend services - 5 workflows",
			group:        frontendGroup,
			workflowName: "",
		},
		listItem{
			isGroup:      false,
			name:         "📄 deploy.yml",
			description:  "Deploy workflow",
			workflowName: "deploy.yml",
		},
		listItem{
			isGroup:      false,
			name:         "📄 test.yml",
			description:  "Test workflow",
			workflowName: "test.yml",
		},
	}

	tests := []struct {
		name          string
		filterInput   string
		expectedCount int
		description   string
	}{
		{
			name:          "empty filter returns all",
			filterInput:   "",
			expectedCount: 4,
			description:   "Should return all groups and workflows",
		},
		{
			name:          "filter by workflow name",
			filterInput:   "deploy",
			expectedCount: 1,
			description:   "Should match deploy.yml workflow",
		},
		{
			name:          "filter by group name",
			filterInput:   "backend",
			expectedCount: 1,
			description:   "Should match backend group",
		},
		{
			name:          "filter groups by description",
			filterInput:   "frontend",
			expectedCount: 1,
			description:   "Should match frontend group by name",
		},
		{
			name:          "fuzzy match workflow",
			filterInput:   "tst",
			expectedCount: 1,
			description:   "Should fuzzy match test.yml",
		},
		{
			name:          "fuzzy match group",
			filterInput:   "frnt",
			expectedCount: 1,
			description:   "Should fuzzy match frontend group",
		},
		{
			name:          "no matches",
			filterInput:   "xyz",
			expectedCount: 0,
			description:   "Should return no matches",
		},
		{
			name:          "match multiple items",
			filterInput:   "service",
			expectedCount: 2,
			description:   "Should match both groups (they have 'services' in description)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterNavigationItems(items, tt.filterInput)
			if len(filtered) != tt.expectedCount {
				t.Errorf("filterNavigationItems with input %q returned %d items, want %d (%s)",
					tt.filterInput, len(filtered), tt.expectedCount, tt.description)
			}
		})
	}
}
