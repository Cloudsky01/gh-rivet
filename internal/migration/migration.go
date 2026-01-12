package migration

import (
	"fmt"
	"os"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/paths"
)

// MigrationChoice represents the user's choice for config migration
type MigrationChoice int

const (
	MigrateToUser MigrationChoice = iota // Migrate to user config
	KeepAsTeam                            // Keep as team default and create user config
	Skip                                  // Skip migration for now
)

// NeedsMigration checks if there's a legacy config that should be migrated
func NeedsMigration(p *paths.Paths) (bool, string) {
	// Check if user config already exists
	if _, err := os.Stat(p.UserConfigFile()); err == nil {
		return false, "" // Already migrated
	}

	// Check for legacy config
	if legacyPath, found := p.FindLegacyConfig(); found {
		// Check if it's in a repository location (.github/)
		return true, legacyPath
	}

	return false, ""
}

// Migrate performs the config migration based on user choice
func Migrate(p *paths.Paths, legacyPath string, choice MigrationChoice) error {
	// Load the legacy config
	legacyConfig, err := config.LoadFromPath(legacyPath, paths.SourceRepoDefault)
	if err != nil {
		return fmt.Errorf("failed to load legacy config: %w", err)
	}

	switch choice {
	case MigrateToUser:
		// Move config to user config location
		if err := legacyConfig.SaveToUserConfig(p); err != nil {
			return fmt.Errorf("failed to save user config: %w", err)
		}

		fmt.Printf("✓ Configuration migrated to: %s\n", p.UserConfigFile())
		fmt.Printf("  Legacy config at %s can now be removed or kept as team default.\n", legacyPath)

	case KeepAsTeam:
		// Create a user config based on legacy config
		// In this mode, we keep the repo config as-is and create a user override
		userConfig := &config.Config{
			Repository: legacyConfig.Repository,
			Preferences: &config.Preferences{
				RefreshInterval: legacyConfig.GetRefreshInterval(),
			},
		}

		if err := userConfig.SaveToUserConfig(p); err != nil {
			return fmt.Errorf("failed to save user config: %w", err)
		}

		fmt.Printf("✓ User configuration created at: %s\n", p.UserConfigFile())
		fmt.Printf("  Team defaults remain at: %s\n", legacyPath)
		fmt.Println("  You can now customize your user config without affecting team settings.")

	case Skip:
		fmt.Println("Migration skipped. Using legacy config for now.")
		fmt.Println("Run 'rivet config migrate' later to migrate your configuration.")

	default:
		return fmt.Errorf("invalid migration choice")
	}

	return nil
}

// AutoMigrate attempts to automatically migrate config if it's clearly a user-specific config
func AutoMigrate(p *paths.Paths) (bool, error) {
	needsMigration, legacyPath := NeedsMigration(p)
	if !needsMigration {
		return false, nil
	}

	// Auto-migrate if the legacy config is in home directory (clearly user-specific)
	homeDir, err := os.UserHomeDir()
	if err == nil && legacyPath == fmt.Sprintf("%s/.rivet.yaml", homeDir) {
		if err := Migrate(p, legacyPath, MigrateToUser); err != nil {
			return false, err
		}
		return true, nil
	}

	// Don't auto-migrate if it's in a repository location - let user decide
	return false, nil
}

// GetMigrationPrompt returns a user-friendly prompt for migration
func GetMigrationPrompt(legacyPath string) string {
	return fmt.Sprintf(`
Configuration Migration Required
═══════════════════════════════════════════════════════════════

Rivet has a new configuration system that separates user-specific
settings from team-shared defaults.

Found legacy config at: %s

Migration Options:
  1. Move to User Config (Recommended for personal configs)
     → Moves your config to ~/.config/rivet/config.yaml
     → Your personal settings, not shared with team

  2. Keep as Team Default
     → Keeps existing config as team default (.github/.rivet.yaml)
     → Creates a new user config for your personal overrides
     → Good for shared repositories

  3. Skip for Now
     → Continue using legacy config (will prompt again later)

What would you like to do? (1/2/3): `, legacyPath)
}

// ShowMigrationSuccess shows a success message after migration
func ShowMigrationSuccess(p *paths.Paths) {
	fmt.Println("\n✓ Configuration migration complete!")
	fmt.Println("\nYour config locations:")
	fmt.Printf("  User config:  %s\n", p.UserConfigFile())
	if p.ProjectRoot != "" {
		fmt.Printf("  Repo default: %s (optional)\n", p.RepoDefaultConfigPath)
		fmt.Printf("  Project user: %s (optional)\n", p.ProjectUserConfigPath)
	}
	fmt.Println("\nCommands:")
	fmt.Println("  rivet config path  - Show config file locations")
	fmt.Println("  rivet config show  - Display merged configuration")
	fmt.Println("  rivet config edit  - Edit your user config")
	fmt.Println()
}

// MigrateStateFiles migrates state files to the new location
func MigrateStateFiles(p *paths.Paths, repository string) error {
	// This is handled by state.LoadWithPaths which includes migration logic
	// This function is a placeholder for future batch migration if needed
	return nil
}
