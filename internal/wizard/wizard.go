package wizard

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/git"
	"github.com/Cloudsky01/gh-rivet/internal/github"
)

type GroupBuilder struct {
	ID          string
	Name        string
	Description string
	Workflows   []string
}

type Wizard struct {
	availableWorkflows []string
	groups             []GroupBuilder
	configPath         string
	repository         string
}

func New(workflows []string, configPath string) *Wizard {
	return &Wizard{
		availableWorkflows: workflows,
		groups:             []GroupBuilder{},
		configPath:         configPath,
	}
}

func (w *Wizard) SetRepository(repo string) {
	w.repository = repo
}

func (w *Wizard) Run() (*config.Config, error) {
	if !isTTY() {
		return w.runNonInteractive()
	}

	w.printWelcome()

	if w.repository == "" {
		if err := w.promptRepository(); err != nil {
			return nil, err
		}
	}

	fmt.Println(GetInfoStyle().Render(fmt.Sprintf("✓ Repository: %s", w.repository)))
	fmt.Println()

	organizationChoice := ""
	if err := w.promptOrganization(&organizationChoice); err != nil {
		return nil, err
	}

	switch organizationChoice {
	case "single":
		return w.createDefaultConfig(), nil
	case "custom":
		return w.createCustomGroups()
	default:
		return w.createDefaultConfig(), nil
	}
}

func (w *Wizard) printWelcome() {
	fmt.Println()
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	fmt.Println(titleStyle.Render("🚀 Rivet Configuration Wizard"))
	fmt.Println(subtitleStyle.Render("Let's set up your GitHub Actions workflow viewer"))
	fmt.Println()
	fmt.Println(GetInfoStyle().Render(fmt.Sprintf("Found %d workflow(s)", len(w.availableWorkflows))))
	fmt.Println()
}

func (w *Wizard) createCustomGroups() (*config.Config, error) {
	successStyle := GetInfoStyle()

	for {
		addMore := true
		if len(w.groups) > 0 {
			if err := w.promptAddMoreGroups(&addMore); err != nil {
				return nil, err
			}
		}

		if !addMore {
			break
		}

		group, err := w.createGroup()
		if err != nil {
			return nil, err
		}

		if group != nil {
			w.groups = append(w.groups, *group)
			fmt.Println(successStyle.Render(fmt.Sprintf("✓ Group '%s' created with %d workflow(s)", group.Name, len(group.Workflows))))
			fmt.Println()
		}
	}

	if len(w.groups) == 0 {
		return w.createDefaultConfig(), nil
	}

	return w.buildConfig(), nil
}

func (w *Wizard) promptOrganization(choice *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How would you like to organize your workflows?").
				Description("Groups help you categorize and navigate workflows").
				Options(
					huh.NewOption("Single group (all workflows together)", "single"),
					huh.NewOption("Custom groups (organize by category)", "custom"),
				).
				Value(choice),
		),
	).Run()
}

func (w *Wizard) promptAddMoreGroups(addMore *bool) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add another group?").
				Description(fmt.Sprintf("You have %d group(s) configured", len(w.groups))).
				Affirmative("Yes").
				Negative("No, finish setup").
				Value(addMore),
		),
	).Run()
}

func (w *Wizard) createGroup() (*GroupBuilder, error) {
	group := &GroupBuilder{}

	if err := w.promptGroupDetails(group); err != nil {
		return nil, err
	}

	group.ID = w.generateID(group.Name)

	if err := w.promptWorkflowSelection(group); err != nil {
		return nil, err
	}

	return group, nil
}

func (w *Wizard) promptGroupDetails(group *GroupBuilder) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Group Name").
				Description("A display name for this group").
				Placeholder("e.g., 'CI/CD', 'Backend', 'Frontend', 'Deployment'").
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
				Placeholder("e.g., 'Build and test workflows', 'Production deployments'").
				Value(&group.Description),
		),
	).Run()
}

func (w *Wizard) promptWorkflowSelection(group *GroupBuilder) error {
	available := w.getRemainingWorkflows()
	if len(available) == 0 {
		fmt.Println(GetWarnStyle().Render("\n⚠ No workflows remaining to assign."))
		fmt.Println()
		return nil
	}

	options := make([]huh.Option[string], len(available))
	for i, wf := range available {
		options[i] = huh.NewOption(wf, wf)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(fmt.Sprintf("Select workflows for '%s'", group.Name)).
				Description(fmt.Sprintf("Available: %d of %d workflows (Use space to select, / to filter)",
					len(available), len(w.availableWorkflows))).
				Options(options...).
				Filterable(true).
				Limit(10).
				Value(&group.Workflows),
		),
	).Run()

	if err != nil {
		return err
	}

	if len(group.Workflows) == 0 {
		fmt.Println(GetWarnStyle().Render("\n⚠ No workflows selected for this group. The group will be empty."))
		fmt.Println()
	}

	return nil
}

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

func (w *Wizard) generateID(name string) string {
	id := strings.ToLower(strings.TrimSpace(name))
	id = strings.ReplaceAll(id, " ", "-")

	var result strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	id = result.String()
	baseID := id
	counter := 1
	for w.idExists(id) {
		id = fmt.Sprintf("%s-%d", baseID, counter)
		counter++
	}

	return id
}

func (w *Wizard) idExists(id string) bool {
	for _, group := range w.groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

func (w *Wizard) promptRepository() error {
	detectedRepo, _ := git.DetectRepository()
	var repo string

	if detectedRepo != "" {
		confirmed := false
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("📦 Repository Detection").
					Description(fmt.Sprintf("Detected: %s", detectedRepo)).
					Affirmative("Yes, use this").
					Negative("No, enter different").
					Value(&confirmed),
			),
		).Run()

		if err != nil {
			return err
		}

		if confirmed {
			repo = detectedRepo
		}
	}

	if repo == "" {
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("📦 GitHub Repository").
					Description("Format: owner/repo").
					Placeholder("e.g., maintainx/maintainx").
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("repository is required")
						}
						return git.ValidateRepositoryFormat(strings.TrimSpace(s))
					}).
					Value(&repo),
			),
		).Run()

		if err != nil {
			return err
		}
	}

	repo = strings.TrimSpace(repo)

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Validating repository..."))

	client := github.NewClient("")
	ctx := context.Background()
	exists, err := client.RepositoryExists(ctx, repo)

	if err != nil || !exists {
		fmt.Println(GetErrorStyle().Render(fmt.Sprintf("\n✗ Failed to validate repository: %s", repo)))
		if err != nil {
			fmt.Println(GetErrorStyle().Render(fmt.Sprintf("  %v", err)))
		}
		fmt.Println()
		return w.promptRepository()
	}

	w.repository = repo
	return nil
}

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
		Repository: w.repository,
		Groups:     groups,
	}
}

func (w *Wizard) createDefaultConfig() *config.Config {
	fmt.Println(GetInfoStyle().Render("✓ Creating single group with all workflows"))
	fmt.Println()

	return &config.Config{
		Repository: w.repository,
		Groups: []config.Group{
			{
				ID:          "workflows",
				Name:        "Workflows",
				Description: "All workflows",
				Workflows:   w.availableWorkflows,
			},
		},
	}
}

func isTTY() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func IsTTY() bool {
	return isTTY()
}

func AskConfirm(title, description string, result *bool) error {
	if !isTTY() {
		return fmt.Errorf("cannot ask for confirmation in non-interactive mode")
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(result),
		),
	).Run()
}

func GetInfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
}

func GetWarnStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
}

func GetErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
}

func (w *Wizard) runNonInteractive() (*config.Config, error) {
	fmt.Println()
	fmt.Println("Running in non-interactive mode (no TTY detected)")
	fmt.Printf("Creating default configuration with %d workflow(s)\n", len(w.availableWorkflows))
	fmt.Println()

	if w.repository == "" {
		detectedRepo, _ := git.DetectRepository()
		if detectedRepo != "" {
			w.repository = detectedRepo
			fmt.Printf("Detected repository: %s\n", detectedRepo)
		} else {
			return nil, fmt.Errorf("no repository specified and could not detect from .git/config")
		}
	}

	return w.createDefaultConfig(), nil
}
