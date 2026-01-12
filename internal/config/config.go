package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Cloudsky01/gh-rivet/internal/paths"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Preferences contains user-specific settings that should not be shared
type Preferences struct {
	RefreshInterval int               `yaml:"refreshInterval,omitempty"` // in seconds, 0 = disabled
	Theme           string            `yaml:"theme,omitempty"`           // Theme preference (e.g., "dark", "light")
	Keybindings     string            `yaml:"keybindings,omitempty"`     // Keybinding style (e.g., "vim", "emacs")
	CustomSettings  map[string]string `yaml:"customSettings,omitempty"`  // Extensible custom settings
}

type Config struct {
	Repository  string       `yaml:"repository"`
	Preferences *Preferences `yaml:"preferences,omitempty"` // User preferences (new, optional)
	Groups      []Group      `yaml:"groups,omitempty"`

	// Internal fields (not serialized)
	configSource paths.ConfigSource `yaml:"-"` // Track where this config came from
	configPath   string             `yaml:"-"` // Path to the config file
}

// GetRefreshInterval returns the refresh interval, checking preferences first
func (c *Config) GetRefreshInterval() int {
	if c.Preferences != nil {
		return c.Preferences.RefreshInterval
	}
	return 0
}

// SetRefreshInterval sets the refresh interval in preferences
func (c *Config) SetRefreshInterval(interval int) {
	if c.Preferences == nil {
		c.Preferences = &Preferences{}
	}
	c.Preferences.RefreshInterval = interval
}

// GetConfigSource returns the source of this config
func (c *Config) GetConfigSource() paths.ConfigSource {
	return c.configSource
}

// GetConfigPath returns the path to this config file
func (c *Config) GetConfigPath() string {
	return c.configPath
}

type Workflow struct {
	File string `yaml:"file"`
	Name string `yaml:"name,omitempty"`
}

func (w *Workflow) DisplayName() string {
	if w.Name != "" {
		return w.Name
	}
	return w.File
}

type Group struct {
	ID               string     `yaml:"id"`
	Name             string     `yaml:"name"`
	Description      string     `yaml:"description,omitempty"`
	Workflows        []string   `yaml:"workflows,omitempty"`
	WorkflowDefs     []Workflow `yaml:"workflowDefs,omitempty"`
	WorkflowPatterns []string   `yaml:"workflowPatterns,omitempty"`
	Jobs             []string   `yaml:"jobs,omitempty"`
	Groups           []Group    `yaml:"groups,omitempty"`
	PinnedWorkflows  []string   `yaml:"pinnedWorkflows,omitempty"`
}

// Load loads a config file from a single path (legacy function, kept for backward compatibility)
func Load(path string) (*Config, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".rivet")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME")
	}

	v.SetEnvPrefix("RIVET")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.configPath = v.ConfigFileUsed()

	return &config, nil
}

// LoadFromPath loads a config from a specific path with source tracking
func LoadFromPath(path string, source paths.ConfigSource) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.configSource = source
	config.configPath = path

	return &config, nil
}

// LoadMultiTier loads config from multiple sources and merges them
// The order of precedence (lowest to highest):
// 1. Repository default (.github/.rivet.yaml)
// 2. User global config (~/.config/rivet/config.yaml)
// 3. Project user config (.git/.rivet/config.yaml)
// 4. Environment variables (RIVET_*)
// 5. Explicit path via CLI flag (if provided)
func LoadMultiTier(p *paths.Paths, explicitPath string) (*Config, error) {
	var baseConfig *Config
	var err error

	if explicitPath != "" {
		// CLI flag has highest priority - load only this file
		baseConfig, err = LoadFromPath(explicitPath, paths.SourceCLIFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", explicitPath, err)
		}
	} else {
		// Load and merge from multiple sources
		configPaths := p.GetConfigPaths()

		if len(configPaths) == 0 {
			// No config files found, check for legacy config
			if legacyPath, found := p.FindLegacyConfig(); found {
				return LoadFromPath(legacyPath, paths.SourceRepoDefault)
			}
			return nil, fmt.Errorf("no configuration file found")
		}

		// Start with the first (lowest priority) config
		baseConfig, err = LoadFromPath(configPaths[0], p.GetConfigSource(configPaths[0]))
		if err != nil {
			return nil, fmt.Errorf("failed to load base config from %s: %w", configPaths[0], err)
		}

		// Merge higher priority configs
		for i := 1; i < len(configPaths); i++ {
			overrideConfig, err := LoadFromPath(configPaths[i], p.GetConfigSource(configPaths[i]))
			if err != nil {
				// Log warning but continue with partial config
				fmt.Fprintf(os.Stderr, "Warning: failed to load config from %s: %v\n", configPaths[i], err)
				continue
			}

			baseConfig = MergeConfigs(baseConfig, overrideConfig)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(baseConfig)

	return baseConfig, nil
}

// MergeConfigs merges two configs, with override taking precedence
// Only non-empty values from override are applied to base
func MergeConfigs(base, override *Config) *Config {
	merged := &Config{
		Repository:   base.Repository,
		Preferences:  base.Preferences,
		Groups:       base.Groups,
		configSource: base.configSource,
		configPath:   base.configPath,
	}

	// Override repository if specified
	if override.Repository != "" {
		merged.Repository = override.Repository
		merged.configSource = override.configSource
		merged.configPath = override.configPath
	}

	// Merge preferences
	if override.Preferences != nil {
		if merged.Preferences == nil {
			merged.Preferences = &Preferences{}
		}

		if override.Preferences.RefreshInterval != 0 {
			merged.Preferences.RefreshInterval = override.Preferences.RefreshInterval
		}
		if override.Preferences.Theme != "" {
			merged.Preferences.Theme = override.Preferences.Theme
		}
		if override.Preferences.Keybindings != "" {
			merged.Preferences.Keybindings = override.Preferences.Keybindings
		}
		if len(override.Preferences.CustomSettings) > 0 {
			if merged.Preferences.CustomSettings == nil {
				merged.Preferences.CustomSettings = make(map[string]string)
			}
			for k, v := range override.Preferences.CustomSettings {
				merged.Preferences.CustomSettings[k] = v
			}
		}
	}

	// Override groups if specified (full replacement, not merge)
	if len(override.Groups) > 0 {
		merged.Groups = override.Groups
	}

	return merged
}

// applyEnvOverrides applies environment variable overrides to the config
func applyEnvOverrides(config *Config) {
	// Check for RIVET_REPOSITORY
	if repo := os.Getenv("RIVET_REPOSITORY"); repo != "" {
		config.Repository = repo
		config.configSource = paths.SourceEnvVar
	}

	// Check for RIVET_REFRESH_INTERVAL
	if interval := os.Getenv("RIVET_REFRESH_INTERVAL"); interval != "" {
		var val int
		if _, err := fmt.Sscanf(interval, "%d", &val); err == nil {
			config.SetRefreshInterval(val)
		}
	}

	// Check for RIVET_PREFERENCES_THEME
	if theme := os.Getenv("RIVET_PREFERENCES_THEME"); theme != "" {
		if config.Preferences == nil {
			config.Preferences = &Preferences{}
		}
		config.Preferences.Theme = theme
	}

	// Check for RIVET_PREFERENCES_KEYBINDINGS
	if kb := os.Getenv("RIVET_PREFERENCES_KEYBINDINGS"); kb != "" {
		if config.Preferences == nil {
			config.Preferences = &Preferences{}
		}
		config.Preferences.Keybindings = kb
	}
}

func LoadWithViper(path string) (*Config, *viper.Viper, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".rivet")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME")
	}

	v.SetEnvPrefix("RIVET")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, v, nil
}

func WatchConfig(v *viper.Viper, onConfigChange func(*Config)) {
	v.OnConfigChange(func(e fsnotify.Event) {
		var newConfig Config
		if err := v.Unmarshal(&newConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error reloading config: %v\n", err)
			return
		}
		onConfigChange(&newConfig)
	})
	v.WatchConfig()
}

func (c *Config) Save(path string) error {
	return c.SaveWithHeader(path, true)
}

// SaveToUserConfig saves the config to the user's config file
func (c *Config) SaveToUserConfig(p *paths.Paths) error {
	if err := p.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %w", err)
	}

	return c.SaveWithHeader(p.UserConfigFile(), true)
}

// SaveToRepoDefault saves the config to the repository default location
func (c *Config) SaveToRepoDefault(p *paths.Paths) error {
	if p.RepoDefaultConfigPath == "" {
		return fmt.Errorf("no repository root configured")
	}

	// Ensure .github directory exists
	githubDir := filepath.Dir(p.RepoDefaultConfigPath)
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github directory: %w", err)
	}

	return c.SaveWithHeader(p.RepoDefaultConfigPath, true)
}

// SaveWithHeader saves the config with an optional header
func (c *Config) SaveWithHeader(path string, includeHeader bool) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var fullContent string
	if includeHeader {
		header := `# Rivet Configuration
# Learn more: https://github.com/Cloudsky01/gh-rivet
#
# Configuration structure:
# - repository: GitHub repository in owner/repo format
# - preferences: User-specific settings (optional)
#   - refreshInterval: Auto-refresh interval in seconds (0 = disabled)
#   - theme: Color theme preference
#   - keybindings: Keybinding style (vim, emacs, etc.)
# - groups: Organize your workflows into groups
#   - id: Unique identifier (auto-generated from name)
#   - name: Display name shown in the TUI
#   - description: Optional description
#   - workflows: List of workflow filenames
#   - pinnedWorkflows: Workflows to pin to the top
#   - groups: Nested groups for hierarchical organization
#
# Configuration locations:
#   User config: ~/.config/rivet/config.yaml (user-specific settings)
#   Repo default: .github/.rivet.yaml (team-shared defaults, optional)
#   Project user: .git/.rivet/config.yaml (per-project user overrides)
#
# Run 'rivet config --help' for more information

`
		fullContent = header + string(data)
	} else {
		fullContent = string(data)
	}

	if err := os.WriteFile(path, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	c.configPath = path

	return nil
}

func (c *Config) Validate() error {
	if c.Repository == "" {
		return fmt.Errorf("configuration must specify a repository (owner/repo)")
	}

	if len(c.Groups) == 0 {
		return fmt.Errorf("configuration must have at least one group")
	}

	for _, group := range c.Groups {
		if err := c.validateGroup(&group, ""); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) validateGroup(group *Group, path string) error {
	currentPath := path
	if currentPath != "" {
		currentPath += "/"
	}
	currentPath += group.ID

	if group.ID == "" {
		return fmt.Errorf("group at path %s missing id", currentPath)
	}
	if group.Name == "" {
		return fmt.Errorf("group %s missing name", currentPath)
	}

	for _, pattern := range group.Jobs {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex pattern in group %s: %s (%w)", currentPath, pattern, err)
		}
	}

	for i := range group.Groups {
		if err := c.validateGroup(&group.Groups[i], currentPath); err != nil {
			return err
		}
	}

	return nil
}

func (g *Group) GetAllWorkflows() []string {
	workflows := make([]string, 0)

	workflows = append(workflows, g.Workflows...)

	for _, wf := range g.WorkflowDefs {
		workflows = append(workflows, wf.File)
	}

	for i := range g.Groups {
		workflows = append(workflows, g.Groups[i].GetAllWorkflows()...)
	}

	return workflows
}

func (g *Group) GetWorkflowDef(filename string) *Workflow {
	for i := range g.WorkflowDefs {
		if g.WorkflowDefs[i].File == filename {
			return &g.WorkflowDefs[i]
		}
	}
	return nil
}

func (g *Group) HasWorkflows() bool {
	if len(g.Workflows) > 0 || len(g.WorkflowDefs) > 0 {
		return true
	}

	for i := range g.Groups {
		if g.Groups[i].HasWorkflows() {
			return true
		}
	}

	return false
}

func (g *Group) IsLeaf() bool {
	return len(g.Groups) == 0
}

func (g *Group) IsPinned(workflowName string) bool {
	return slices.Contains(g.PinnedWorkflows, workflowName)
}

func (g *Group) TogglePin(workflowName string) {
	if g.IsPinned(workflowName) {
		for i, wf := range g.PinnedWorkflows {
			if wf == workflowName {
				g.PinnedWorkflows = append(g.PinnedWorkflows[:i], g.PinnedWorkflows[i+1:]...)
				break
			}
		}
	} else {
		g.PinnedWorkflows = append(g.PinnedWorkflows, workflowName)
	}
}

type PinnedWorkflow struct {
	WorkflowName string
	GroupPath    []string
	Group        *Group
}

func (c *Config) GetAllPinnedWorkflows() []PinnedWorkflow {
	var pinned []PinnedWorkflow

	for i := range c.Groups {
		pinned = append(pinned, collectPinnedFromGroup(&c.Groups[i], []string{})...)
	}

	return pinned
}

func collectPinnedFromGroup(g *Group, parentPath []string) []PinnedWorkflow {
	var pinned []PinnedWorkflow

	currentPath := append([]string{}, parentPath...)
	currentPath = append(currentPath, g.Name)

	for _, wf := range g.PinnedWorkflows {
		pinned = append(pinned, PinnedWorkflow{
			WorkflowName: wf,
			GroupPath:    currentPath,
			Group:        g,
		})
	}

	for i := range g.Groups {
		pinned = append(pinned, collectPinnedFromGroup(&g.Groups[i], currentPath)...)
	}

	return pinned
}
