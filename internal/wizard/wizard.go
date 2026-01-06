package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"rivet/internal/config"
)

// GroupBuilder holds the state for building a group
type GroupBuilder struct {
	ID          string
	Name        string
	Description string
	Workflows   []string
}

// Wizard handles the interactive configuration creation
type Wizard struct {
	availableWorkflows []string
	groups             []GroupBuilder
	configPath         string
}

// New creates a new wizard with the discovered workflows
func New(workflows []string, configPath string) *Wizard {
	return &Wizard{
		availableWorkflows: workflows,
		groups:             []GroupBuilder{},
		configPath:         configPath,
	}
}

// Run executes the interactive wizard
func (w *Wizard) Run() (*config.Config, error) {
	// Welcome message
	fmt.Println()
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("blue"))
	fmt.Println(titleStyle.Render("Rivet Configuration Wizard"))
	fmt.Printf("Found %d workflow(s) in .github/workflows\n\n", len(w.availableWorkflows))

	// Main loop - keep creating groups until the user is done
	for {
		addMore := true
		if len(w.groups) > 0 {
			// Ask if a user wants to add another group
			if err := w.promptAddMoreGroups(&addMore); err != nil {
				return nil, err
			}
		}

		if !addMore {
			break
		}

		// Create a new group
		group, err := w.createGroup()
		if err != nil {
			return nil, err
		}

		if group != nil {
			w.groups = append(w.groups, *group)
			fmt.Printf("\nGroup '%s' created with %d workflow(s)\n\n",
				group.Name, len(group.Workflows))
		}
	}

	// Handle case where no groups were created
	if len(w.groups) == 0 {
		return w.createDefaultConfig(), nil
	}

	return w.buildConfig(), nil
}

// promptAddMoreGroups asks if the user wants to add another group
func (w *Wizard) promptAddMoreGroups(addMore *bool) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add another group?").
				Description(fmt.Sprintf("You have %d group(s) configured", len(w.groups))).
				Affirmative("Yes").
				Negative("No, finish setup").
				Value(addMore),
		),
	)

	return form.Run()
}

// createGroup walks through the group creation process
func (w *Wizard) createGroup() (*GroupBuilder, error) {
	group := &GroupBuilder{}

	// Step 1: Get group name and description
	if err := w.promptGroupDetails(group); err != nil {
		return nil, err
	}

	// Generate ID from name
	group.ID = w.generateID(group.Name)

	// Step 2: Select workflows for this group
	if err := w.promptWorkflowSelection(group); err != nil {
		return nil, err
	}

	return group, nil
}

// promptGroupDetails gets the group name and description
func (w *Wizard) promptGroupDetails(group *GroupBuilder) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Group Name").
				Description("A display name for this group (e.g., 'CI/CD', 'Frontend')").
				Placeholder("My Group").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}).
				Value(&group.Name),

			huh.NewInput().
				Title("Description (optional)").
				Description("A brief description of what this group contains").
				Placeholder("Workflows for...").
				Value(&group.Description),
		),
	)

	return form.Run()
}

// promptWorkflowSelection lets user select workflows for the group
func (w *Wizard) promptWorkflowSelection(group *GroupBuilder) error {
	// Get remaining workflows (not yet assigned to any group)
	available := w.getRemainingWorkflows()

	if len(available) == 0 {
		fmt.Println("\nNo workflows remaining to assign.")
		return nil
	}

	// Build options for multi-select
	options := make([]huh.Option[string], len(available))
	for i, wf := range available {
		options[i] = huh.NewOption(wf, wf)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(fmt.Sprintf("Select workflows for '%s'", group.Name)).
				Description("Use space to select, enter to confirm. Type to filter.").
				Options(options...).
				Filterable(true).
				Limit(len(available)). // Allow selecting all
				Value(&group.Workflows),
		),
	)

	return form.Run()
}

// getRemainingWorkflows returns workflows not yet assigned to any group
func (w *Wizard) getRemainingWorkflows() []string {
	assigned := make(map[string]bool)
	for _, group := range w.groups {
		for _, wf := range group.Workflows {
			assigned[wf] = true
		}
	}

	var remaining []string
	for _, wf := range w.availableWorkflows {
		if !assigned[wf] {
			remaining = append(remaining, wf)
		}
	}
	return remaining
}

// generateID creates a URL-safe ID from a name
func (w *Wizard) generateID(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	id := strings.ToLower(strings.TrimSpace(name))
	id = strings.ReplaceAll(id, " ", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	id = result.String()

	// Ensure uniqueness
	baseID := id
	counter := 1
	for w.idExists(id) {
		id = fmt.Sprintf("%s-%d", baseID, counter)
		counter++
	}

	return id
}

// idExists checks if an ID is already used
func (w *Wizard) idExists(id string) bool {
	for _, group := range w.groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

// buildConfig converts the wizard state to a Config
func (w *Wizard) buildConfig() *config.Config {
	groups := make([]config.Group, len(w.groups))

	for i, gb := range w.groups {
		groups[i] = config.Group{
			ID:          gb.ID,
			Name:        gb.Name,
			Description: gb.Description,
			Workflows:   gb.Workflows,
		}
	}

	return &config.Config{
		Groups: groups,
	}
}

// createDefaultConfig creates a config with all workflows in one group
func (w *Wizard) createDefaultConfig() *config.Config {
	return &config.Config{
		Groups: []config.Group{
			{
				ID:          "workflows",
				Name:        "Workflows",
				Description: "All discovered workflows",
				Workflows:   w.availableWorkflows,
			},
		},
	}
}
