package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Cloudsky01/gh-rivet/internal/config"
)

func TestLoadNonExistent(t *testing.T) {
	state, err := Load("/nonexistent/path/state.yaml")
	if err != nil {
		t.Fatalf("Load should not return error for non-existent file: %v", err)
	}

	if state.ViewState != ViewBrowsingGroups {
		t.Errorf("Expected ViewBrowsingGroups, got %s", state.ViewState)
	}

	if len(state.GroupPath) != 0 {
		t.Errorf("Expected empty group path, got %v", state.GroupPath)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".rivet.state.yaml")

	original := &NavigationState{
		ViewState:        ViewWorkflowOutput,
		GroupPath:        []string{"services", "backend"},
		SelectedWorkflow: "deploy.yml",
		FromPinnedView:   true,
		ListIndex:        5,
		PinnedListIndex:  2,
	}

	// Save
	if err := original.Save(statePath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := Load(statePath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify
	if loaded.ViewState != original.ViewState {
		t.Errorf("ViewState: got %s, want %s", loaded.ViewState, original.ViewState)
	}

	if len(loaded.GroupPath) != len(original.GroupPath) {
		t.Errorf("GroupPath length: got %d, want %d", len(loaded.GroupPath), len(original.GroupPath))
	}

	for i, id := range original.GroupPath {
		if loaded.GroupPath[i] != id {
			t.Errorf("GroupPath[%d]: got %s, want %s", i, loaded.GroupPath[i], id)
		}
	}

	if loaded.SelectedWorkflow != original.SelectedWorkflow {
		t.Errorf("SelectedWorkflow: got %s, want %s", loaded.SelectedWorkflow, original.SelectedWorkflow)
	}

	if loaded.FromPinnedView != original.FromPinnedView {
		t.Errorf("FromPinnedView: got %v, want %v", loaded.FromPinnedView, original.FromPinnedView)
	}

	if loaded.ListIndex != original.ListIndex {
		t.Errorf("ListIndex: got %d, want %d", loaded.ListIndex, original.ListIndex)
	}

	if loaded.PinnedListIndex != original.PinnedListIndex {
		t.Errorf("PinnedListIndex: got %d, want %d", loaded.PinnedListIndex, original.PinnedListIndex)
	}
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".rivet.state.yaml")

	// Create a state file
	state := &NavigationState{ViewState: ViewBrowsingGroups}
	if err := state.Save(statePath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("State file should exist after save")
	}

	// Clear it
	if err := Clear(statePath); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("State file should not exist after clear")
	}

	// Clearing non-existent file should not error
	if err := Clear(statePath); err != nil {
		t.Errorf("Clear of non-existent file should not error: %v", err)
	}
}

func TestDefaultStatePath(t *testing.T) {
	tests := []struct {
		configPath string
		expected   string
	}{
		{".rivet.yaml", ".rivet.state.yaml"},
		{"/home/user/.rivet.yaml", "/home/user/.rivet.state.yaml"},
		{"/project/config/rivet.yaml", "/project/config/.rivet.state.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.configPath, func(t *testing.T) {
			result := DefaultStatePath(tt.configPath)
			if result != tt.expected {
				t.Errorf("DefaultStatePath(%q) = %q, want %q", tt.configPath, result, tt.expected)
			}
		})
	}
}

func TestResolveGroupPath(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{
			{
				ID:   "services",
				Name: "Services",
				Groups: []config.Group{
					{
						ID:        "backend",
						Name:      "Backend",
						Workflows: []string{"deploy.yml"},
					},
					{
						ID:        "frontend",
						Name:      "Frontend",
						Workflows: []string{"build.yml"},
					},
				},
			},
			{
				ID:        "infra",
				Name:      "Infrastructure",
				Workflows: []string{"terraform.yml"},
			},
		},
	}

	tests := []struct {
		name     string
		groupIDs []string
		wantLen  int
		wantOK   bool
	}{
		{
			name:     "empty path",
			groupIDs: []string{},
			wantLen:  0,
			wantOK:   true,
		},
		{
			name:     "single level",
			groupIDs: []string{"services"},
			wantLen:  1,
			wantOK:   true,
		},
		{
			name:     "nested path",
			groupIDs: []string{"services", "backend"},
			wantLen:  2,
			wantOK:   true,
		},
		{
			name:     "invalid first level",
			groupIDs: []string{"nonexistent"},
			wantLen:  0,
			wantOK:   false,
		},
		{
			name:     "invalid nested level",
			groupIDs: []string{"services", "nonexistent"},
			wantLen:  1,
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, ok := ResolveGroupPath(cfg, tt.groupIDs)
			if ok != tt.wantOK {
				t.Errorf("ResolveGroupPath() ok = %v, want %v", ok, tt.wantOK)
			}
			if len(groups) != tt.wantLen {
				t.Errorf("ResolveGroupPath() len = %d, want %d", len(groups), tt.wantLen)
			}
		})
	}
}

func TestExtractGroupIDs(t *testing.T) {
	groups := []*config.Group{
		{ID: "services", Name: "Services"},
		{ID: "backend", Name: "Backend"},
	}

	ids := ExtractGroupIDs(groups)

	if len(ids) != 2 {
		t.Fatalf("Expected 2 IDs, got %d", len(ids))
	}

	if ids[0] != "services" {
		t.Errorf("Expected 'services', got %s", ids[0])
	}

	if ids[1] != "backend" {
		t.Errorf("Expected 'backend', got %s", ids[1])
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".rivet.state.yaml")

	// Write corrupted YAML
	if err := os.WriteFile(statePath, []byte("{{{{not valid yaml"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// Load should return default state, not error
	state, err := Load(statePath)
	if err != nil {
		t.Fatalf("Load should not error on corrupted file: %v", err)
	}

	if state.ViewState != ViewBrowsingGroups {
		t.Errorf("Expected default ViewState, got %s", state.ViewState)
	}
}
