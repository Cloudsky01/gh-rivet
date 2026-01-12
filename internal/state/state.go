package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/paths"
)

// ViewState represents the current view in the TUI
type ViewState string

const (
	ViewBrowsingGroups  ViewState = "browsingGroups"
	ViewPinnedWorkflows ViewState = "viewingPinnedWorkflows"
	ViewWorkflowOutput  ViewState = "viewingWorkflowOutput"
)

// NavigationState represents the persisted navigation state
type NavigationState struct {
	// View identification
	ViewState ViewState `yaml:"viewState"`

	// For browsingGroups: path through the group hierarchy (group IDs)
	GroupPath []string `yaml:"groupPath,omitempty"`

	// For viewingWorkflowOutput: which workflow was selected
	SelectedWorkflow string `yaml:"selectedWorkflow,omitempty"`

	// Was workflow accessed via pinned view? (determines back navigation)
	FromPinnedView bool `yaml:"fromPinnedView,omitempty"`

	// List selection indices for better UX
	ListIndex       int `yaml:"listIndex,omitempty"`
	PinnedListIndex int `yaml:"pinnedListIndex,omitempty"`
}

// DefaultStatePath returns the default state file path relative to config (legacy)
func DefaultStatePath(configPath string) string {
	dir := filepath.Dir(configPath)
	return filepath.Join(dir, ".rivet.state.yaml")
}

// GetStatePath returns the state file path for a repository using the new paths system
func GetStatePath(p *paths.Paths, repository string) (string, error) {
	// Parse repository to get owner and name
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repository format, expected owner/repo: %s", repository)
	}

	// Ensure state directory exists
	if err := p.EnsureDirs(); err != nil {
		return "", fmt.Errorf("failed to ensure state directory: %w", err)
	}

	return p.UserStateFile(parts[0], parts[1]), nil
}

// LoadWithPaths loads state using the new paths system with migration support
func LoadWithPaths(p *paths.Paths, repository string) (*NavigationState, error) {
	// Get the new state path
	statePath, err := GetStatePath(p, repository)
	if err != nil {
		return defaultState(), nil
	}

	// Try loading from new location
	state, err := Load(statePath)
	if err == nil {
		return state, nil
	}

	// If new location doesn't exist, check for legacy state file
	if legacyPath, found := p.FindLegacyState(); found {
		state, err := Load(legacyPath)
		if err == nil {
			// Migrate to new location
			if err := state.Save(statePath); err == nil {
				// Migration successful, optionally clean up legacy file
				// (commented out to avoid data loss)
				// os.Remove(legacyPath)
			}
			return state, nil
		}
	}

	// Return default state if nothing found
	return defaultState(), nil
}

// defaultState returns a new default navigation state
func defaultState() *NavigationState {
	return &NavigationState{
		ViewState: ViewBrowsingGroups,
		GroupPath: []string{},
	}
}

// Load reads the navigation state from a file
func Load(path string) (*NavigationState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state if file doesn't exist
			return &NavigationState{
				ViewState: ViewBrowsingGroups,
				GroupPath: []string{},
			}, nil
		}
		return nil, err
	}

	var state NavigationState
	if err := yaml.Unmarshal(data, &state); err != nil {
		// If the state file is corrupted, return default state
		return &NavigationState{
			ViewState: ViewBrowsingGroups,
			GroupPath: []string{},
		}, nil
	}

	return &state, nil
}

// Save writes the navigation state to a file
func (s *NavigationState) Save(path string) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Clear removes the state file
func Clear(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ResolveGroupPath converts a slice of group IDs to group pointers
// Returns the resolved groups and true if all groups were found
func ResolveGroupPath(cfg *config.Config, groupIDs []string) ([]*config.Group, bool) {
	if len(groupIDs) == 0 {
		return []*config.Group{}, true
	}

	result := make([]*config.Group, 0, len(groupIDs))
	currentGroups := cfg.Groups

	for _, id := range groupIDs {
		found := false
		for i := range currentGroups {
			if currentGroups[i].ID == id {
				result = append(result, &currentGroups[i])
				currentGroups = currentGroups[i].Groups
				found = true
				break
			}
		}
		if !found {
			// Group not found, return partial path
			return result, false
		}
	}

	return result, true
}

// ExtractGroupIDs converts a slice of group pointers to their IDs
func ExtractGroupIDs(groups []*config.Group) []string {
	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}
	return ids
}
