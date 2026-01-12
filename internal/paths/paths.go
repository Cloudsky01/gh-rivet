package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// AppName is the application name used in config paths
	AppName = "rivet"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"

	// StateFileName is the name of the state file
	StateFileName = "state.yaml"

	// LegacyConfigFileName is the old config file name
	LegacyConfigFileName = ".rivet.yaml"

	// LegacyStateFileName is the old state file name
	LegacyStateFileName = ".rivet.state.yaml"
)

// ConfigSource indicates where a config file came from
type ConfigSource int

const (
	SourceUnknown ConfigSource = iota
	SourceUserConfig
	SourceProjectConfig
	SourceRepoDefault
	SourceEnvVar
	SourceCLIFlag
)

func (s ConfigSource) String() string {
	switch s {
	case SourceUserConfig:
		return "user config"
	case SourceProjectConfig:
		return "project config"
	case SourceRepoDefault:
		return "repository default"
	case SourceEnvVar:
		return "environment variable"
	case SourceCLIFlag:
		return "CLI flag"
	default:
		return "unknown"
	}
}

// Paths provides access to all application paths following XDG Base Directory specification
type Paths struct {
	// UserConfigDir is the user's config directory (~/.config/rivet)
	UserConfigDir string

	// UserStateDir is the user's state directory (~/.local/state/rivet)
	UserStateDir string

	// UserCacheDir is the user's cache directory (~/.cache/rivet)
	UserCacheDir string

	// ProjectRoot is the root of the current git repository (if any)
	ProjectRoot string

	// RepoDefaultConfigPath is the path to the repository's default config (.github/.rivet.yaml)
	RepoDefaultConfigPath string

	// ProjectUserConfigPath is the path to the user's project-specific config (.git/.rivet/config.yaml)
	ProjectUserConfigPath string
}

// New creates a new Paths instance with XDG-compliant directories
func New() (*Paths, error) {
	p := &Paths{}

	// Get user config directory (XDG_CONFIG_HOME or ~/.config on Unix, %AppData% on Windows)
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config directory: %w", err)
	}
	p.UserConfigDir = filepath.Join(configDir, AppName)

	// Get user state directory (XDG_STATE_HOME or ~/.local/state on Unix)
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".local", "state")
	}
	p.UserStateDir = filepath.Join(stateDir, AppName)

	// Get user cache directory (XDG_CACHE_HOME or ~/.cache on Unix, LocalAppData on Windows)
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user cache directory: %w", err)
	}
	p.UserCacheDir = filepath.Join(cacheDir, AppName)

	return p, nil
}

// NewWithProject creates a new Paths instance with project-specific paths
func NewWithProject(projectRoot string) (*Paths, error) {
	p, err := New()
	if err != nil {
		return nil, err
	}

	p.ProjectRoot = projectRoot
	p.RepoDefaultConfigPath = filepath.Join(projectRoot, ".github", LegacyConfigFileName)
	p.ProjectUserConfigPath = filepath.Join(projectRoot, ".git", AppName, ConfigFileName)

	return p, nil
}

// UserConfigFile returns the path to the user's main config file
func (p *Paths) UserConfigFile() string {
	return filepath.Join(p.UserConfigDir, ConfigFileName)
}

// UserStateFile returns the path to the user's state file for a specific repository
func (p *Paths) UserStateFile(repoOwner, repoName string) string {
	if repoOwner == "" || repoName == "" {
		return filepath.Join(p.UserStateDir, StateFileName)
	}
	// Use owner_repo format for per-repository state
	filename := fmt.Sprintf("%s_%s.%s", sanitizeForFilename(repoOwner), sanitizeForFilename(repoName), StateFileName)
	return filepath.Join(p.UserStateDir, filename)
}

// EnsureDirs creates all necessary directories if they don't exist
func (p *Paths) EnsureDirs() error {
	dirs := []string{
		p.UserConfigDir,
		p.UserStateDir,
		p.UserCacheDir,
	}

	if p.ProjectRoot != "" {
		dirs = append(dirs, filepath.Join(p.ProjectRoot, ".git", AppName))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// FindLegacyConfig searches for legacy config files in order of precedence
// Returns the path and whether it exists
func (p *Paths) FindLegacyConfig() (string, bool) {
	// Check legacy locations in order
	legacyPaths := []string{}

	// Project-specific legacy config
	if p.ProjectRoot != "" {
		legacyPaths = append(legacyPaths,
			filepath.Join(p.ProjectRoot, ".github", LegacyConfigFileName),
			filepath.Join(p.ProjectRoot, LegacyConfigFileName),
		)
	}

	// Home directory legacy config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		legacyPaths = append(legacyPaths, filepath.Join(homeDir, LegacyConfigFileName))
	}

	// Current directory legacy config
	cwd, err := os.Getwd()
	if err == nil {
		legacyPaths = append(legacyPaths, filepath.Join(cwd, LegacyConfigFileName))
	}

	for _, path := range legacyPaths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

// FindLegacyState searches for legacy state files
func (p *Paths) FindLegacyState() (string, bool) {
	legacyPaths := []string{}

	// Project-specific legacy state
	if p.ProjectRoot != "" {
		legacyPaths = append(legacyPaths,
			filepath.Join(p.ProjectRoot, ".github", LegacyStateFileName),
			filepath.Join(p.ProjectRoot, LegacyStateFileName),
		)
	}

	// Current directory legacy state
	cwd, err := os.Getwd()
	if err == nil {
		legacyPaths = append(legacyPaths, filepath.Join(cwd, LegacyStateFileName))
	}

	for _, path := range legacyPaths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

// GetConfigPaths returns all config paths in order of precedence (lowest to highest)
// The last path in the list has the highest priority
func (p *Paths) GetConfigPaths() []string {
	paths := []string{}

	// 1. Repository default (lowest priority)
	if p.RepoDefaultConfigPath != "" {
		if _, err := os.Stat(p.RepoDefaultConfigPath); err == nil {
			paths = append(paths, p.RepoDefaultConfigPath)
		}
	}

	// 2. User global config
	userConfig := p.UserConfigFile()
	if _, err := os.Stat(userConfig); err == nil {
		paths = append(paths, userConfig)
	}

	// 3. Project user config (highest priority)
	if p.ProjectUserConfigPath != "" {
		if _, err := os.Stat(p.ProjectUserConfigPath); err == nil {
			paths = append(paths, p.ProjectUserConfigPath)
		}
	}

	return paths
}

// GetConfigSource determines which source a config path corresponds to
func (p *Paths) GetConfigSource(path string) ConfigSource {
	switch path {
	case p.UserConfigFile():
		return SourceUserConfig
	case p.ProjectUserConfigPath:
		return SourceProjectConfig
	case p.RepoDefaultConfigPath:
		return SourceRepoDefault
	default:
		return SourceUnknown
	}
}

// sanitizeForFilename removes/replaces characters that are invalid in filenames
func sanitizeForFilename(s string) string {
	// Replace slashes and other problematic characters
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	return s
}
