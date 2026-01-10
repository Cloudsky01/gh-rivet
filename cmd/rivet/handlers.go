package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/git"
	"github.com/Cloudsky01/gh-rivet/internal/wizard"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	asciiStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
)

func getASCIIArt() string {
	return `
     _           _
    (_)         | |
 _ __ ___   _____| |_
| '__| \ \ / / _ \ __|
| |  | |\ V /  __/ |_|
|_|  |_| \_/ \___|\__|
`
}

func printSuccessSummary(configPath string, cfg *config.Config) {
	divider := dividerStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println()
	fmt.Println(divider)
	fmt.Println(successStyle.Render("✅ Configuration created successfully!"))
	fmt.Println(divider)
	fmt.Println()

	fmt.Println(labelStyle.Render("📁 Config file: ") + infoStyle.Render(configPath))
	fmt.Println(labelStyle.Render("📦 Repository:  ") + infoStyle.Render(cfg.Repository))
	fmt.Println(labelStyle.Render("📊 Groups:      ") + infoStyle.Render(fmt.Sprintf("%d", len(cfg.Groups))))

	totalWorkflows := 0
	for _, group := range cfg.Groups {
		totalWorkflows += len(group.GetAllWorkflows())
	}
	fmt.Println(labelStyle.Render("⚙️  Workflows:   ") + infoStyle.Render(fmt.Sprintf("%d", totalWorkflows)))

	fmt.Println()
	fmt.Println(headerStyle.Render("🚀 Next steps:"))
	fmt.Println(infoStyle.Render("   rivet              # Launch the TUI"))
	fmt.Println(infoStyle.Render("   rivet --help       # See all options"))
	fmt.Println()
}

func handleMissingConfig(configPath string) error {
	detectedRepo, _ := git.DetectRepository()

	if detectedRepo != "" && wizard.IsTTY() {
		shouldCreate := false
		err := wizard.AskConfirm(
			"No configuration found",
			fmt.Sprintf("Would you like to create a configuration for %s?", detectedRepo),
			&shouldCreate,
		)
		if err != nil {
			return err
		}

		if shouldCreate {
			repo = detectedRepo
			return runInit(nil, nil)
		}
	}

	fmt.Println(wizard.GetWarnStyle().Render("⚠ No configuration file found at " + configPath))
	fmt.Println()
	fmt.Println(headerStyle.Render("To get started, run:"))
	fmt.Println(infoStyle.Render("  rivet init"))
	fmt.Println()

	if detectedRepo != "" {
		fmt.Println(wizard.GetInfoStyle().Render("Detected repository: " + detectedRepo))
		fmt.Println(infoStyle.Render("  rivet init --repo " + detectedRepo))
		fmt.Println()
	} else {
		fmt.Println(infoStyle.Render("Or specify a repository:"))
		fmt.Println(infoStyle.Render("  rivet init --repo owner/repo"))
		fmt.Println()
	}

	return nil
}

func handleNoLocalWorkflows() error {
	fmt.Println()
	fmt.Println(wizard.GetWarnStyle().Render("⚠ No local workflows found"))
	fmt.Println()
	fmt.Println(infoStyle.Render("This can happen if:"))
	fmt.Println(infoStyle.Render("  • You're not in a repository with GitHub Actions workflows"))
	fmt.Println(infoStyle.Render("  • The .github/workflows directory doesn't exist"))
	fmt.Println()

	detectedRepo, _ := git.DetectRepository()
	if detectedRepo != "" {
		fmt.Println(wizard.GetInfoStyle().Render("Detected repository: " + detectedRepo))
		fmt.Println()

		if wizard.IsTTY() {
			shouldFetch := false
			err := wizard.AskConfirm(
				"Fetch workflows from GitHub",
				fmt.Sprintf("Would you like to fetch workflows from %s?", detectedRepo),
				&shouldFetch,
			)
			if err != nil {
				return err
			}

			if shouldFetch {
				repo = detectedRepo
				return runInit(nil, nil)
			}
		} else {
			fmt.Println(infoStyle.Render("Run with the --repo flag to fetch workflows from GitHub:"))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  rivet init --repo %s", detectedRepo)))
		}
	} else {
		fmt.Println(infoStyle.Render("To create a config for a specific repository:"))
		fmt.Println(infoStyle.Render("  rivet init --repo owner/repo"))
	}

	fmt.Println()
	return nil
}

func handleNoWorkflows(configPath, targetRepo string, useRemoteWorkflows bool) error {
	fmt.Println()
	fmt.Println(wizard.GetWarnStyle().Render("⚠ No workflow files found"))
	fmt.Println()

	if !useRemoteWorkflows {
		fmt.Println(infoStyle.Render("This can happen if:"))
		fmt.Println(infoStyle.Render("  • The .github/workflows directory is empty"))
		fmt.Println(infoStyle.Render("  • No workflows exist in the repository"))
		fmt.Println()
		fmt.Println(infoStyle.Render("To create a config for a repository with workflows:"))
		fmt.Println(infoStyle.Render("  rivet init --repo owner/repo"))
		fmt.Println()

		if wizard.IsTTY() {
			shouldCreate := false
			err := wizard.AskConfirm(
				"Create minimal configuration",
				"Would you like to create an empty configuration anyway?",
				&shouldCreate,
			)
			if err != nil {
				return err
			}
			if !shouldCreate {
				return nil
			}
		}
	} else {
		fmt.Println(infoStyle.Render(fmt.Sprintf("The repository %s doesn't have any workflows yet.", targetRepo)))
		fmt.Println()
		fmt.Println(wizard.GetInfoStyle().Render("Creating minimal configuration..."))
	}

	if targetRepo == "" {
		detectedRepo, _ := git.DetectRepository()
		targetRepo = detectedRepo
	}

	if targetRepo == "" {
		return fmt.Errorf("cannot create configuration without a repository. Use --repo flag")
	}

	cfg := &config.Config{
		Repository: targetRepo,
		Groups: []config.Group{
			{
				ID:          "workflows",
				Name:        "Workflows",
				Description: "All workflows",
				Workflows:   []string{},
			},
		},
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	printSuccessSummary(configPath, cfg)
	return nil
}
