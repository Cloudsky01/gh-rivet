package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/spf13/cobra"

	"rivet/internal/config"
	"rivet/internal/github"
	"rivet/internal/tui"
	"rivet/internal/wizard"
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
  - A .rivet.yaml configuration file

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
)

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", ".rivet.yaml", "Path to configuration file")
	rootCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository in OWNER/REPO format (defaults to current directory)")
	rootCmd.Flags().IntVar(&runID, "run", 0, "Specific run ID to view (defaults to latest)")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of recent workflow runs to fetch and aggregate")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode to show job matching details")
	rootCmd.Flags().BoolVarP(&startWithPinned, "pinned", "p", false, "Start with pinned workflows view")
	rootCmd.Flags().StringVar(&statePath, "state", "", "Path to state file (default: .rivet.state.yaml)")
	rootCmd.Flags().BoolVar(&noState, "no-state", false, "Disable state persistence (don't save or restore navigation state)")

	initCmd.Flags().StringVarP(&configPath, "config", "c", ".rivet.yaml", "Path to configuration file")
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration file")

	rootCmd.AddCommand(initCmd)
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
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
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

	workflowDir := ".github/workflows"
	workflows, err := wizard.DiscoverWorkflows(workflowDir)
	if err != nil {
		return fmt.Errorf("failed to read workflows directory: %w\n\nMake sure you're in a repository with GitHub Actions workflows", err)
	}

	if len(workflows) == 0 {
		return fmt.Errorf("no workflow files found in %s", workflowDir)
	}

	w := wizard.New(workflows, configPath)
	cfg, err := w.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\nConfiguration saved to: %s\n", configPath)
	fmt.Println("Run: rivet -r owner/repo")

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
