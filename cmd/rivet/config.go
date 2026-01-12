package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/git"
	"github.com/Cloudsky01/gh-rivet/internal/migration"
	"github.com/Cloudsky01/gh-rivet/internal/paths"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage rivet configuration",
		Long: `Manage rivet configuration files.

Configuration Locations:
  User config:     ~/.config/rivet/config.yaml (user-specific settings)
  Repo default:    .github/.rivet.yaml (team-shared defaults, optional)
  Project user:    .git/.rivet/config.yaml (per-project user overrides)

Configuration Precedence (lowest to highest):
  1. Repository default
  2. User global config
  3. Project user config
  4. Environment variables (RIVET_*)
  5. CLI flags`,
	}

	configPathCmd = &cobra.Command{
		Use:   "path",
		Short: "Show configuration file locations",
		Long:  `Display the paths to all configuration files and their existence status.`,
		RunE:  runConfigPath,
	}

	configShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Display merged configuration",
		Long:  `Show the effective configuration after merging all sources.`,
		RunE:  runConfigShow,
	}

	configEditCmd = &cobra.Command{
		Use:   "edit",
		Short: "Edit user configuration file",
		Long:  `Open the user configuration file in $EDITOR (or vim/nano if not set).`,
		RunE:  runConfigEdit,
	}

	configResetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset user configuration",
		Long:  `Remove the user configuration file to reset to defaults.`,
		RunE:  runConfigReset,
	}

	configMigrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrate legacy configuration",
		Long:  `Migrate from legacy .rivet.yaml to the new multi-tier configuration system.`,
		RunE:  runConfigMigrate,
	}
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configMigrateCmd)
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	// Detect project root
	projectRoot, _ := git.GetGitRepositoryRoot()

	// Create paths
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

	fmt.Println("Configuration File Locations")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	// User config
	userConfigPath := p.UserConfigFile()
	userExists := fileExists(userConfigPath)
	fmt.Printf("User Config:        %s %s\n", userConfigPath, existsIndicator(userExists))

	// Project-specific paths
	if projectRoot != "" {
		fmt.Println()
		fmt.Printf("Project Root:       %s\n", projectRoot)

		repoExists := fileExists(p.RepoDefaultConfigPath)
		fmt.Printf("Repo Default:       %s %s\n", p.RepoDefaultConfigPath, existsIndicator(repoExists))

		projectUserExists := fileExists(p.ProjectUserConfigPath)
		fmt.Printf("Project User:       %s %s\n", p.ProjectUserConfigPath, existsIndicator(projectUserExists))
	}

	// State directory
	fmt.Println()
	fmt.Printf("State Directory:    %s\n", p.UserStateDir)
	fmt.Printf("Cache Directory:    %s\n", p.UserCacheDir)

	// Check for legacy config
	fmt.Println()
	if legacyPath, found := p.FindLegacyConfig(); found {
		fmt.Printf("⚠ Legacy Config:    %s (consider migrating)\n", legacyPath)
		fmt.Println("  Run 'rivet config migrate' to migrate to the new system")
	}

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Detect project root
	projectRoot, _ := git.GetGitRepositoryRoot()

	// Create paths
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

	// Load merged config
	cfg, err := config.LoadMultiTier(p, configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Println("Merged Configuration")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Printf("Source: %s\n", cfg.GetConfigSource())
	fmt.Printf("Path:   %s\n", cfg.GetConfigPath())
	fmt.Println()

	// Marshal and display config
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Println(string(data))

	// Show which configs contributed
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("Active Configuration Files:")
	for _, path := range p.GetConfigPaths() {
		fmt.Printf("  • %s (%s)\n", path, p.GetConfigSource(path))
	}

	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	// Create paths
	p, err := paths.New()
	if err != nil {
		return fmt.Errorf("failed to initialize paths: %w", err)
	}

	userConfigPath := p.UserConfigFile()

	// Ensure config directory exists
	if err := p.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to ensure config directory: %w", err)
	}

	// Create default config if it doesn't exist
	if !fileExists(userConfigPath) {
		fmt.Printf("Creating new user config at: %s\n", userConfigPath)

		// Detect repository from git
		repository := ""
		if _, err := git.GetGitRepositoryRoot(); err == nil {
			if repo, err := git.DetectRepository(); err == nil {
				repository = repo
			}
		}

		defaultConfig := &config.Config{
			Repository: repository,
			Preferences: &config.Preferences{
				RefreshInterval: 30,
			},
		}

		if err := defaultConfig.SaveToUserConfig(p); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found. Set $EDITOR or $VISUAL environment variable")
	}

	// Open editor
	editorCmd := exec.Command(editor, userConfigPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	fmt.Printf("✓ Configuration saved to: %s\n", userConfigPath)
	return nil
}

func runConfigReset(cmd *cobra.Command, args []string) error {
	// Create paths
	p, err := paths.New()
	if err != nil {
		return fmt.Errorf("failed to initialize paths: %w", err)
	}

	userConfigPath := p.UserConfigFile()

	if !fileExists(userConfigPath) {
		fmt.Println("No user configuration file found.")
		return nil
	}

	// Confirm deletion
	fmt.Printf("This will delete your user configuration at:\n  %s\n\n", userConfigPath)
	fmt.Print("Are you sure? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Reset cancelled.")
		return nil
	}

	// Delete config file
	if err := os.Remove(userConfigPath); err != nil {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	fmt.Printf("✓ User configuration removed: %s\n", userConfigPath)
	return nil
}

func runConfigMigrate(cmd *cobra.Command, args []string) error {
	// Detect project root
	projectRoot, _ := git.GetGitRepositoryRoot()

	// Create paths
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

	// Check if migration is needed
	needsMigration, legacyPath := migration.NeedsMigration(p)
	if !needsMigration {
		fmt.Println("✓ Configuration is already up to date.")
		fmt.Println("  No legacy config found or migration already complete.")
		return nil
	}

	// Show migration prompt
	fmt.Print(migration.GetMigrationPrompt(legacyPath))

	// Read user choice
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(response)

	// Parse choice
	var choice migration.MigrationChoice
	switch response {
	case "1":
		choice = migration.MigrateToUser
	case "2":
		choice = migration.KeepAsTeam
	case "3":
		choice = migration.Skip
		return nil
	default:
		fmt.Println("Invalid choice. Migration cancelled.")
		return nil
	}

	// Perform migration
	if err := migration.Migrate(p, legacyPath, choice); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Show success message
	migration.ShowMigrationSuccess(p)

	return nil
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func existsIndicator(exists bool) string {
	if exists {
		return "✓"
	}
	return "✗"
}
