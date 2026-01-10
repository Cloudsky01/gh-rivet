package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/git"
	"github.com/Cloudsky01/gh-rivet/internal/github"
	"github.com/Cloudsky01/gh-rivet/internal/tui"
	"github.com/Cloudsky01/gh-rivet/internal/wizard"
)

var repoFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	configPath      string
	repo            string
	runID           int
	limit           int
	debug           bool
	force           bool
	startWithPinned bool
	statePath       string
	noState         bool

	rootCmd = &cobra.Command{
		Use:   "rivet",
		Short: "Interactive TUI for browsing GitHub Actions workflows",
		Long: `Rivet is a TUI tool that helps you browse and manage GitHub Actions 
workflows organized by configurable groups. It wraps the GitHub CLI (gh)
to provide an interactive interface for viewing workflow runs.

Requirements:
  - GitHub CLI (gh) must be installed and authenticated
  - A .github/.rivet.yaml configuration file

Get started:
  rivet init              # Create a configuration file
  rivet -r owner/repo     # Browse workflows for a repository`,
		RunE:    runView,
		Version: version,
	}

	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long: `Initialize a new .rivet.yaml configuration file by scanning your
repository's .github/workflows directory. If the file already exists,
use --force to overwrite it.`,
		RunE: runInit,
	}

	updateRepoCmd = &cobra.Command{
		Use:   "update-repo [owner/repo]",
		Short: "Update the repository in your configuration",
		Long: `Update the repository setting in your .rivet.yaml configuration file.
You can provide the repository as an argument or it will be detected from .git/config.`,
		RunE: runUpdateRepo,
		Args: cobra.MaximumNArgs(1),
	}
)

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", ".github/.rivet.yaml", "Path to configuration file")
	rootCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository in OWNER/REPO format (defaults to current directory)")
	rootCmd.Flags().IntVar(&runID, "run", 0, "Specific run ID to view (defaults to latest)")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of recent workflow runs to fetch and aggregate")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode to show job matching details")
	rootCmd.Flags().BoolVarP(&startWithPinned, "pinned", "p", false, "Start with pinned workflows view")
	rootCmd.Flags().StringVar(&statePath, "state", "", "Path to state file (default: .github/.rivet.state.yaml)")
	rootCmd.Flags().BoolVar(&noState, "no-state", false, "Disable state persistence (don't save or restore navigation state)")

	initCmd.Flags().StringVarP(&configPath, "config", "c", ".github/.rivet.yaml", "Path to configuration file")
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration file")
	initCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository in OWNER/REPO format (if no local workflows exist)")

	updateRepoCmd.Flags().StringVarP(&configPath, "config", "c", ".github/.rivet.yaml", "Path to configuration file")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(updateRepoCmd)
	rootCmd.SetVersionTemplate(`{{printf "rivet %s\n" .Version}}`)
}

func checkGitHubCLI() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed.\n\nInstall it from: https://cli.github.com/\n\nOn macOS:  brew install gh\nOn Linux:  See https://github.com/cli/cli/blob/trunk/docs/install_linux.md")
	}

	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub CLI is not authenticated.\n\nRun: gh auth login")
	}

	return nil
}

func runView(cmd *cobra.Command, args []string) error {
	if err := checkGitHubCLI(); err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		// Config file doesn't exist or couldn't be parsed
		return handleMissingConfig(configPath)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Use --repo flag if provided, otherwise use config repository
	if repo == "" {
		repo = cfg.Repository
		if repo != "" {
			fmt.Println(infoStyle.Render("Using repository from config: " + repo))
		}
	}

	if repo == "" {
		return fmt.Errorf("repository must be specified with --repo flag (e.g., --repo owner/repo)")
	}

	if !repoFormatRegex.MatchString(repo) {
		return fmt.Errorf("invalid repository format '%s'. Expected format: OWNER/REPO (e.g., github/cli)", repo)
	}

	gh := github.NewClient(repo)

	opts := tui.MenuOptions{
		StartWithPinned: startWithPinned,
		StatePath:       statePath,
		NoRestoreState:  noState,
	}

	model := tui.NewMenuModel(cfg, configPath, gh, opts)
	if err := tui.RunMenu(model); err != nil {
		return err
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("configuration file %s already exists. Use --force to overwrite", configPath)
	}

	var workflows []string
	var useRemoteWorkflows bool

	// If --repo flag is provided, fetch workflows from GitHub
	if repo != "" {
		if err := git.ValidateRepositoryFormat(repo); err != nil {
			return err
		}

		ghClient := github.NewClient("")
		ctx := context.Background()

		// Validate repository exists with spinner
		_, err := wizard.RunWithSpinner(fmt.Sprintf("Validating repository %s", repo), func() (interface{}, error) {
			exists, err := ghClient.RepositoryExists(ctx, repo)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, fmt.Errorf("repository not found or not accessible")
			}
			return nil, nil
		})
		if err != nil {
			return err
		}

		// Fetch workflows from GitHub with spinner
		result, err := wizard.RunWithSpinner(fmt.Sprintf("Fetching workflows from %s", repo), func() (interface{}, error) {
			return ghClient.GetWorkflows(ctx, repo)
		})
		if err != nil {
			return fmt.Errorf("failed to fetch workflows: %w", err)
		}

		workflows = result.([]string)
		useRemoteWorkflows = true
	} else {
		// Try to discover local workflows
		workflowDir := ".github/workflows"
		localWorkflows, err := wizard.DiscoverWorkflows(workflowDir)
		if err != nil {
			return handleNoLocalWorkflows()
		}
		workflows = localWorkflows
	}

	if len(workflows) == 0 {
		return handleNoWorkflows(configPath, repo, useRemoteWorkflows)
	}

	w := wizard.New(workflows, configPath)

	// Set repository if using remote workflows
	if useRemoteWorkflows {
		w.SetRepository(repo)
	}

	cfg, err := w.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	printSuccessSummary(configPath, cfg)
	return nil
}

func runUpdateRepo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var newRepo string

	// If repository is provided as argument, use it
	if len(args) > 0 {
		newRepo = strings.TrimSpace(args[0])
	} else {
		// Try to detect from .git/config
		detectedRepo, err := git.DetectRepository()
		if err != nil || detectedRepo == "" {
			return fmt.Errorf("could not detect repository from .git/config and no repository provided\nUsage: rivet update-repo owner/repo")
		}
		newRepo = detectedRepo
		fmt.Println(infoStyle.Render("Detected repository: " + newRepo))
	}

	// Validate format
	if err := git.ValidateRepositoryFormat(newRepo); err != nil {
		return err
	}

	// Validate repository exists on GitHub
	ghClient := github.NewClient("")
	ctx := context.Background()
	exists, err := ghClient.RepositoryExists(ctx, newRepo)
	if err != nil || !exists {
		return fmt.Errorf("failed to validate repository %s: %v", newRepo, err)
	}

	// Update config
	cfg.Repository = newRepo

	// Save updated config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println(successStyle.Render("✓ Repository updated to: " + newRepo))
	fmt.Println(infoStyle.Render("Configuration saved to: " + configPath))

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
