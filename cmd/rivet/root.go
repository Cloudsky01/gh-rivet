package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Cloudsky01/gh-rivet/internal/ascii"
	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/git"
	"github.com/Cloudsky01/gh-rivet/internal/github"
	"github.com/Cloudsky01/gh-rivet/internal/paths"
	"github.com/Cloudsky01/gh-rivet/internal/tui"
	"github.com/Cloudsky01/gh-rivet/internal/wizard"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	configPath      string
	repo            string
	force           bool
	reset           bool
	statePath       string
	noState         bool
	timeoutSeconds  int
	refreshInterval int

	rootCmd = &cobra.Command{
		Use:   "rivet",
		Short: "Interactive TUI for GitHub Actions workflows",
		Long: `Browse and manage GitHub Actions workflows organized by configurable groups.
Wraps the GitHub CLI (gh) to provide an interactive terminal interface.

Requirements: GitHub CLI (gh) installed and authenticated
Get started:  rivet init`,
		RunE:    runView,
		Version: version,
	}

	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Create a new configuration file",
		Long:  `Create a .rivet.yaml config by scanning .github/workflows or fetching from GitHub.`,
		RunE:  runInit,
	}

	updateRepoCmd = &cobra.Command{
		Use:   "update-repo [owner/repo]",
		Short: "Update repository in config",
		Long:  `Update the repository setting in .rivet.yaml. Auto-detects from .git/config if not specified.`,
		RunE:  runUpdateRepo,
		Args:  cobra.MaximumNArgs(1),
	}
)

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (default: auto-detect)")
	rootCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository (owner/repo format)")
	rootCmd.Flags().StringVar(&statePath, "state", "", "Path to state file")
	rootCmd.Flags().BoolVar(&noState, "no-state", false, "Disable state persistence")
	rootCmd.Flags().IntVar(&timeoutSeconds, "timeout", 30, "GitHub API timeout in seconds")
	rootCmd.Flags().IntVar(&refreshInterval, "refresh-interval", 0, "Auto-refresh interval in seconds (0 = disabled, min 5)")

	// Store original help functions before overriding
	originalRootHelpFunc := rootCmd.HelpFunc()
	originalInitHelpFunc := initCmd.HelpFunc()

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println(asciiStyle.Render(ascii.GetASCIIArt()))
		fmt.Println()
		originalRootHelpFunc(cmd, args)
	})

	initCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println(asciiStyle.Render(ascii.GetASCIIArt()))
		fmt.Println()
		originalInitHelpFunc(cmd, args)
	})

	initCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to save configuration file (default: user config)")
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing config")
	initCmd.Flags().BoolVar(&reset, "reset", false, "Delete existing config and create new one")
	initCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository (owner/repo) to fetch workflows from")

	updateRepoCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (default: auto-detect)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(updateRepoCmd)
	rootCmd.SetVersionTemplate(`{{printf "rivet %s\n" .Version}}`)
}

func checkGitHubCLI() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI not installed\nInstall: https://cli.github.com/ or 'brew install gh'")
	}

	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub CLI not authenticated\nRun: gh auth login")
	}

	return nil
}

func runView(cmd *cobra.Command, _ []string) error {
	if err := checkGitHubCLI(); err != nil {
		return err
	}

	// If a config path was explicitly provided via CLI flag, use it directly
	if cmd.Flags().Changed("config") {
		cfg, err := config.LoadMerged([]string{configPath})
		if err != nil {
			return fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
		return runViewWithConfig(cfg, configPath)
	}

	// Otherwise, use the proper precedence system
	projectRoot, _ := git.GetGitRepositoryRoot()
	var p *paths.Paths
	var err error
	if projectRoot != "" {
		p, err = paths.NewWithProject(projectRoot)
	} else {
		p, err = paths.New()
	}
	if err != nil {
		return fmt.Errorf("failed to initialize paths: %w", err)
	}

	// Load and merge all available configs
	configPaths := p.GetConfigPaths()
	if len(configPaths) > 0 {
		cfg, err := config.LoadMerged(configPaths)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
		// Use the last loaded path as the primary config path for display/save purposes
		primaryPath := configPaths[len(configPaths)-1]
		return runViewWithConfig(cfg, primaryPath)
	}

	// No config found, trigger init flow
	return handleMissingConfig()
}

func runViewWithConfig(cfg *config.Config, configPath string) error {
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

	if !git.RepositoryFormatRegex.MatchString(repo) {
		return fmt.Errorf("invalid repository format '%s'. Expected format: OWNER/REPO (e.g., github/cli)", repo)
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	gh := github.NewClientWithTimeout(repo, timeout)

	// Use CLI flag if provided, otherwise use config value
	interval := refreshInterval
	if interval == 0 && cfg.GetRefreshInterval() > 0 {
		interval = cfg.GetRefreshInterval()
	}

	opts := tui.MenuOptions{
		StatePath:       statePath,
		NoRestoreState:  noState,
		RefreshInterval: interval,
	}

	model := tui.NewMenuModel(cfg, configPath, gh, opts)
	if err := tui.RunMenu(model); err != nil {
		return err
	}

	return nil
}

func runInit(cmd *cobra.Command, _ []string) error {
	// Initialize paths first - we'll need this throughout
	projectRoot, _ := git.GetGitRepositoryRoot()
	var p *paths.Paths
	var err error
	if projectRoot != "" {
		p, err = paths.NewWithProject(projectRoot)
	} else {
		p, err = paths.New()
	}
	if err != nil {
		return fmt.Errorf("failed to initialize paths: %w", err)
	}

	// Determine if user explicitly provided a config path via CLI
	explicitConfigPath := cmd != nil && cmd.Flags().Changed("config")

	// Use current best guess for save path - the wizard may change this later.
	savePathHint := p.UserConfigFile()
	if explicitConfigPath && configPath != "" {
		savePathHint = configPath
	}

	var workflows []string
	var useRemoteWorkflows bool

	// If --repo flag is provided, fetch workflows from GitHub
	if repo != "" {
		if err := git.ValidateRepositoryFormat(repo); err != nil {
			return err
		}

		timeout := time.Duration(timeoutSeconds) * time.Second
		ghClient := github.NewClientWithTimeout("", timeout)
		ctx := context.Background()

		// Validate repository exists with spinner
		_, err := wizard.RunWithSpinner(ctx, fmt.Sprintf("Validating repository %s", repo), func() (any, error) {
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
		result, err := wizard.RunWithSpinner(ctx, fmt.Sprintf("Fetching workflows from %s", repo), func() (interface{}, error) {
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
		return handleNoWorkflows(p, repo, useRemoteWorkflows)
	}

	w := wizard.New(workflows, savePathHint)

	// Set repository if using remote workflows
	if useRemoteWorkflows {
		w.SetRepository(repo)
	}

	cfg, err := w.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	targetPath, location, err := determineConfigSaveTarget(p, explicitConfigPath, configPath, w.GetConfigType())
	if err != nil {
		return err
	}

	// Handle --reset flag for the chosen target
	if reset {
		if err := os.Remove(targetPath); err == nil {
			fmt.Println(infoStyle.Render("Removed existing config: " + targetPath))
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing config at %s: %w", targetPath, err)
		}
	} else if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("configuration file %s already exists. Use --force to overwrite or --reset to start fresh", targetPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check existing config at %s: %w", targetPath, err)
		}
	}

	// Ensure directories exist and save to the desired location
	switch location {
	case saveLocationTeam:
		if err := cfg.SaveToRepoDefault(p); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	case saveLocationExplicit:
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := cfg.Save(targetPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	default:
		if err := p.EnsureDirs(); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := cfg.Save(targetPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	}

	printSuccessSummary(targetPath, cfg)
	return nil
}

func runUpdateRepo(cmd *cobra.Command, args []string) error {
	// Auto-detect config path if not explicitly provided
	var actualConfigPath string
	if cmd.Flags().Changed("config") {
		actualConfigPath = configPath
	} else {
		projectRoot, _ := git.GetGitRepositoryRoot()
		var p *paths.Paths
		var err error
		if projectRoot != "" {
			p, err = paths.NewWithProject(projectRoot)
		} else {
			p, err = paths.New()
		}
		if err != nil {
			return fmt.Errorf("failed to initialize paths: %w", err)
		}

		configPaths := p.GetConfigPaths()
		if len(configPaths) == 0 {
			return fmt.Errorf("no configuration found. Run 'rivet init' first")
		}
		actualConfigPath = configPaths[len(configPaths)-1]
	}

	cfg, err := config.LoadFromPath(actualConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config from %s: %w", actualConfigPath, err)
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

	timeout := time.Duration(timeoutSeconds) * time.Second
	ghClient := github.NewClientWithTimeout("", timeout)
	ctx := context.Background()
	exists, err := ghClient.RepositoryExists(ctx, newRepo)
	if err != nil || !exists {
		return fmt.Errorf("failed to validate repository %s: %v", newRepo, err)
	}

	// Update config
	cfg.Repository = newRepo

	// Save updated config
	if err := cfg.Save(actualConfigPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println(successStyle.Render("✓ Repository updated to: " + newRepo))
	fmt.Println(infoStyle.Render("Configuration saved to: " + actualConfigPath))

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type configSaveLocation int

const (
	saveLocationUser configSaveLocation = iota
	saveLocationTeam
	saveLocationExplicit
)

func determineConfigSaveTarget(p *paths.Paths, explicit bool, explicitPath string, configType string) (string, configSaveLocation, error) {
	if explicit && explicitPath != "" {
		return explicitPath, saveLocationExplicit, nil
	}

	switch strings.ToLower(configType) {
	case "team":
		if p.RepoDefaultConfigPath == "" {
			return "", saveLocationUser, fmt.Errorf("team configuration requires running inside a git repository or specifying --config to choose a path explicitly")
		}
		return p.RepoDefaultConfigPath, saveLocationTeam, nil
	default:
		return p.UserConfigFile(), saveLocationUser, nil
	}
}
