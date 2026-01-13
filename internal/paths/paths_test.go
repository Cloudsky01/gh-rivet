package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if p.UserConfigDir == "" {
		t.Error("UserConfigDir should not be empty")
	}

	if p.UserStateDir == "" {
		t.Error("UserStateDir should not be empty")
	}

	if p.UserCacheDir == "" {
		t.Error("UserCacheDir should not be empty")
	}

	// Check that paths contain the app name
	if !strings.Contains(p.UserConfigDir, AppName) {
		t.Errorf("UserConfigDir should contain '%s', got: %s", AppName, p.UserConfigDir)
	}

	if !strings.Contains(p.UserStateDir, AppName) {
		t.Errorf("UserStateDir should contain '%s', got: %s", AppName, p.UserStateDir)
	}

	if !strings.Contains(p.UserCacheDir, AppName) {
		t.Errorf("UserCacheDir should contain '%s', got: %s", AppName, p.UserCacheDir)
	}
}

func TestNewWithProject(t *testing.T) {
	projectRoot := "/path/to/project"
	p, err := NewWithProject(projectRoot)
	if err != nil {
		t.Fatalf("NewWithProject() failed: %v", err)
	}

	if p.ProjectRoot != projectRoot {
		t.Errorf("ProjectRoot = %s, want %s", p.ProjectRoot, projectRoot)
	}

	expectedRepoDefault := filepath.Join(projectRoot, ".github", LegacyConfigFileName)
	if p.RepoDefaultConfigPath != expectedRepoDefault {
		t.Errorf("RepoDefaultConfigPath = %s, want %s", p.RepoDefaultConfigPath, expectedRepoDefault)
	}

	expectedProjectUser := filepath.Join(projectRoot, ".git", AppName, ConfigFileName)
	if p.ProjectUserConfigPath != expectedProjectUser {
		t.Errorf("ProjectUserConfigPath = %s, want %s", p.ProjectUserConfigPath, expectedProjectUser)
	}
}

func TestUserConfigFile(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	configFile := p.UserConfigFile()
	if !strings.HasSuffix(configFile, ConfigFileName) {
		t.Errorf("UserConfigFile should end with '%s', got: %s", ConfigFileName, configFile)
	}
}

func TestUserStateFile(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name      string
		owner     string
		repo      string
		wantMatch string
	}{
		{
			name:      "with owner and repo",
			owner:     "octocat",
			repo:      "hello-world",
			wantMatch: "octocat_hello-world.state.yaml",
		},
		{
			name:      "empty owner and repo",
			owner:     "",
			repo:      "",
			wantMatch: StateFileName,
		},
		{
			name:      "owner with special chars",
			owner:     "owner/with/slashes",
			repo:      "repo:name",
			wantMatch: "owner_with_slashes_repo_name.state.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateFile := p.UserStateFile(tt.owner, tt.repo)
			if !strings.HasSuffix(stateFile, tt.wantMatch) {
				t.Errorf("UserStateFile() should end with '%s', got: %s", tt.wantMatch, stateFile)
			}
		})
	}
}

func TestEnsureDirs(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Override paths to use temp directory
	p := &Paths{
		UserConfigDir:  filepath.Join(tmpDir, "config", AppName),
		UserStateDir:   filepath.Join(tmpDir, "state", AppName),
		UserCacheDir:   filepath.Join(tmpDir, "cache", AppName),
		ProjectRoot:    filepath.Join(tmpDir, "project"),
		usingFallbacks: make(map[string]bool),
	}

	err := p.EnsureDirs()
	if err != nil {
		t.Fatalf("EnsureDirs() failed: %v", err)
	}

	// Check that directories were created
	dirs := []string{
		p.UserConfigDir,
		p.UserStateDir,
		p.UserCacheDir,
		filepath.Join(p.ProjectRoot, ".git", AppName),
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory %s was not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestFindLegacyConfig(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	githubDir := filepath.Join(projectRoot, ".github")

	// Create directories
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create a legacy config file
	legacyConfigPath := filepath.Join(githubDir, LegacyConfigFileName)
	if err := os.WriteFile(legacyConfigPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	p := &Paths{
		ProjectRoot: projectRoot,
	}

	path, found := p.FindLegacyConfig()
	if !found {
		t.Error("FindLegacyConfig() should find the legacy config")
	}

	if path != legacyConfigPath {
		t.Errorf("FindLegacyConfig() = %s, want %s", path, legacyConfigPath)
	}
}

func TestGetConfigPaths(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")

	// Create test config files
	userConfigDir := filepath.Join(tmpDir, "config")
	userConfigPath := filepath.Join(userConfigDir, ConfigFileName)
	repoDefaultPath := filepath.Join(projectRoot, ".github", LegacyConfigFileName)
	projectUserPath := filepath.Join(projectRoot, ".git", AppName, ConfigFileName)

	// Create directories
	for _, dir := range []string{userConfigDir, filepath.Join(projectRoot, ".github"), filepath.Join(projectRoot, ".git", AppName)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create config files
	for _, path := range []string{userConfigPath, repoDefaultPath, projectUserPath} {
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create config file %s: %v", path, err)
		}
	}

	p := &Paths{
		UserConfigDir:          userConfigDir,
		ProjectRoot:            projectRoot,
		RepoDefaultConfigPath:  repoDefaultPath,
		ProjectUserConfigPath:  projectUserPath,
	}

	paths := p.GetConfigPaths()

	// Should return all three configs in order of precedence
	if len(paths) != 3 {
		t.Errorf("GetConfigPaths() returned %d paths, want 3", len(paths))
	}

	// Check order: repo default, user config, project user config
	expectedOrder := []string{repoDefaultPath, userConfigPath, projectUserPath}
	for i, expected := range expectedOrder {
		if i >= len(paths) {
			t.Errorf("Missing path at index %d", i)
			continue
		}
		if paths[i] != expected {
			t.Errorf("GetConfigPaths()[%d] = %s, want %s", i, paths[i], expected)
		}
	}
}

func TestGetConfigSource(t *testing.T) {
	p := &Paths{
		UserConfigDir:         "/home/user/.config/rivet",
		ProjectRoot:           "/project",
		RepoDefaultConfigPath: "/project/.github/.rivet.yaml",
		ProjectUserConfigPath: "/project/.git/rivet/config.yaml",
	}

	tests := []struct {
		name string
		path string
		want ConfigSource
	}{
		{
			name: "user config",
			path: "/home/user/.config/rivet/config.yaml",
			want: SourceUserConfig,
		},
		{
			name: "project user config",
			path: "/project/.git/rivet/config.yaml",
			want: SourceProjectConfig,
		},
		{
			name: "repo default",
			path: "/project/.github/.rivet.yaml",
			want: SourceRepoDefault,
		},
		{
			name: "unknown",
			path: "/some/random/path",
			want: SourceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GetConfigSource(tt.path)
			if got != tt.want {
				t.Errorf("GetConfigSource(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestConfigSourceString(t *testing.T) {
	tests := []struct {
		source ConfigSource
		want   string
	}{
		{SourceUserConfig, "user config"},
		{SourceProjectConfig, "project config"},
		{SourceRepoDefault, "repository default"},
		{SourceEnvVar, "environment variable"},
		{SourceCLIFlag, "CLI flag"},
		{SourceUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.source.String()
			if got != tt.want {
				t.Errorf("ConfigSource.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with/slash", "with_slash"},
		{"with\\backslash", "with_backslash"},
		{"with:colon", "with_colon"},
		{"with*asterisk", "with_asterisk"},
		{"with?question", "with_question"},
		{"with\"quote", "with_quote"},
		{"with<less", "with_less"},
		{"with>greater", "with_greater"},
		{"with|pipe", "with_pipe"},
		{"owner/repo:name", "owner_repo_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeForFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeForFilename(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestXDGCompliance(t *testing.T) {
	// This test verifies XDG Base Directory specification compliance
	p, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// On Unix-like systems, check for proper XDG paths
	if runtime.GOOS != "windows" {
		// UserConfigDir should use XDG_CONFIG_HOME or ~/.config
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			homeDir, _ := os.UserHomeDir()
			configHome = filepath.Join(homeDir, ".config")
		}
		expectedConfigDir := filepath.Join(configHome, AppName)

		if p.UserConfigDir != expectedConfigDir {
			t.Logf("Note: UserConfigDir = %s, XDG standard suggests: %s", p.UserConfigDir, expectedConfigDir)
			t.Logf("This may be acceptable depending on os.UserConfigDir() implementation")
		}

		// UserStateDir should use XDG_STATE_HOME or ~/.local/state
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			homeDir, _ := os.UserHomeDir()
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		expectedStateDir := filepath.Join(stateHome, AppName)

		if p.UserStateDir != expectedStateDir {
			t.Errorf("UserStateDir = %s, want %s", p.UserStateDir, expectedStateDir)
		}
	}
}
