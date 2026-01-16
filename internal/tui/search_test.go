package tui

import (
	"testing"

	"github.com/Cloudsky01/gh-rivet/internal/config"
)

func TestGlobalSearch(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:          "backend",
				Name:        "Backend Services",
				Description: "Server-side workflows",
				Workflows:   []string{"deploy.yml", "test.yml"},
				Groups: []config.Group{
					{
						ID:        "api",
						Name:      "API",
						Workflows: []string{"api-deploy.yml"},
					},
				},
			},
			{
				ID:          "frontend",
				Name:        "Frontend",
				Description: "Client-side workflows",
				Workflows:   []string{"build-ui.yml"},
			},
		},
	}

	tests := []struct {
		name          string
		query         string
		wantResults   bool
		wantMinCount  int
		wantFirstType string
	}{
		{
			name:        "empty query returns nil",
			query:       "",
			wantResults: false,
		},
		{
			name:          "search by group name",
			query:         "backend",
			wantResults:   true,
			wantMinCount:  1,
			wantFirstType: "group",
		},
		{
			name:          "search by workflow name",
			query:         "deploy",
			wantResults:   true,
			wantMinCount:  2, // deploy.yml and api-deploy.yml
			wantFirstType: "workflow",
		},
		{
			name:         "search by description",
			query:        "server",
			wantResults:  true,
			wantMinCount: 1,
		},
		{
			name:         "fuzzy search",
			query:        "bknd",
			wantResults:  true,
			wantMinCount: 1,
		},
		{
			name:        "no matches",
			query:       "zzzznotfound",
			wantResults: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := globalSearch(cfg, tt.query)

			if tt.wantResults && len(results) == 0 {
				t.Errorf("expected results for query %q, got none", tt.query)
			}

			if !tt.wantResults && len(results) > 0 {
				t.Errorf("expected no results for query %q, got %d", tt.query, len(results))
			}

			if tt.wantMinCount > 0 && len(results) < tt.wantMinCount {
				t.Errorf("expected at least %d results for query %q, got %d", tt.wantMinCount, tt.query, len(results))
			}

			if tt.wantFirstType != "" && len(results) > 0 && results[0].Type != tt.wantFirstType {
				t.Errorf("expected first result type %q, got %q", tt.wantFirstType, results[0].Type)
			}
		})
	}
}

func TestCollectAllSearchableItems(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "group1",
				Name:      "Group One",
				Workflows: []string{"workflow1.yml"},
				WorkflowDefs: []config.Workflow{
					{File: "workflow2.yml", Name: "Custom Name"},
				},
				Groups: []config.Group{
					{
						ID:        "nested",
						Name:      "Nested Group",
						Workflows: []string{"nested-workflow.yml"},
					},
				},
			},
		},
	}

	items := collectAllSearchableItems(cfg)

	// Should have: Group One, workflow1.yml, workflow2.yml, Nested Group, nested-workflow.yml
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}

	// Verify group paths are correct
	for _, item := range items {
		if item.result.Type == "workflow" && item.result.Name == "Custom Name" {
			if item.result.Description != "workflow2.yml" {
				t.Errorf("workflow description should be filename, got %q", item.result.Description)
			}
		}
	}
}

func TestFormatGroupPath(t *testing.T) {
	tests := []struct {
		path []string
		want string
	}{
		{[]string{}, "/"},
		{[]string{"Group"}, "Group"},
		{[]string{"Parent", "Child"}, "Parent > Child"},
		{[]string{"A", "B", "C"}, "A > B > C"},
	}

	for _, tt := range tests {
		got := formatGroupPath(tt.path)
		if got != tt.want {
			t.Errorf("formatGroupPath(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
