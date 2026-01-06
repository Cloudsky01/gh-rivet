package state

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"rivet/internal/config"
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

// DefaultStatePath returns the default state file path relative to config
func DefaultStatePath(configPath string) string {
	dir := filepath.Dir(configPath)
	return filepath.Join(dir, ".rivet.state.yaml")
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
