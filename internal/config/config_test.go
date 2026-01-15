package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.rivet.yaml")

	configContent := `groups:
  - id: test-group
    name: Test Group
    description: A test group
    workflows:
      - test-workflow.yml
    workflowDefs:
      - file: deploy.yml
        name: Deploy Workflow
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading the config
	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate the loaded config
	if len(cfg.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(cfg.Groups))
	}

	if cfg.Groups[0].ID != "test-group" {
		t.Errorf("Expected group ID 'test-group', got '%s'", cfg.Groups[0].ID)
	}

	if cfg.Groups[0].Name != "Test Group" {
		t.Errorf("Expected group name 'Test Group', got '%s'", cfg.Groups[0].Name)
	}

	if len(cfg.Groups[0].Workflows) != 1 {
		t.Errorf("Expected 1 workflow, got %d", len(cfg.Groups[0].Workflows))
	}

	if len(cfg.Groups[0].WorkflowDefs) != 1 {
		t.Errorf("Expected 1 workflow def, got %d", len(cfg.Groups[0].WorkflowDefs))
	}

	if cfg.Groups[0].WorkflowDefs[0].File != "deploy.yml" {
		t.Errorf("Expected workflow file 'deploy.yml', got '%s'", cfg.Groups[0].WorkflowDefs[0].File)
	}
}

func TestMerge(t *testing.T) {
	baseConfig := &Config{
		Repository: "base/repo",
		Preferences: &Preferences{
			Theme:           "light",
			RefreshInterval: 30,
			CustomSettings: map[string]string{
				"key1": "val1",
			},
		},
		Groups: []Group{{ID: "base", Name: "Base"}},
	}

	overrideConfig := &Config{
		Repository: "override/repo",
		Preferences: &Preferences{
			Theme: "dark",
			// RefreshInterval missing, should keep base
			CustomSettings: map[string]string{
				"key2": "val2", // should add
			},
		},
		Groups: []Group{{ID: "override", Name: "Override"}}, // Should replace
	}

	baseConfig.Merge(overrideConfig)

	if baseConfig.Repository != "override/repo" {
		t.Errorf("Expected repository 'override/repo', got '%s'", baseConfig.Repository)
	}

	if baseConfig.Preferences.Theme != "dark" {
		t.Errorf("Expected theme 'dark', got '%s'", baseConfig.Preferences.Theme)
	}

	if baseConfig.Preferences.RefreshInterval != 30 {
		t.Errorf("Expected refresh interval 30, got %d", baseConfig.Preferences.RefreshInterval)
	}

	if len(baseConfig.Groups) != 1 || baseConfig.Groups[0].ID != "override" {
		t.Error("Groups should be replaced by override config")
	}

	if val, ok := baseConfig.Preferences.CustomSettings["key1"]; !ok || val != "val1" {
		t.Error("CustomSettings['key1'] should be preserved")
	}

	if val, ok := baseConfig.Preferences.CustomSettings["key2"]; !ok || val != "val2" {
		t.Error("CustomSettings['key2'] should be added")
	}
}

func TestLoadMultipleFormats(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		filename   string
		content    string
		fileFormat string
	}{
		{
			name:       "YAML",
			filename:   "test.rivet.yaml",
			fileFormat: "yaml",
			content: `groups:
  - id: yaml-test
    name: YAML Test
    workflows:
      - test.yml
`,
		},
		{
			name:       "JSON",
			filename:   "test.rivet.json",
			fileFormat: "json",
			content: `{
  "groups": [
    {
      "id": "json-test",
      "name": "JSON Test",
      "workflows": ["test.yml"]
    }
  ]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			cfg, err := LoadFromPath(configPath)
			if err != nil {
				t.Fatalf("Failed to load %s config: %v", tt.fileFormat, err)
			}

			if len(cfg.Groups) != 1 {
				t.Errorf("Expected 1 group in %s config, got %d", tt.fileFormat, len(cfg.Groups))
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: &Config{
				Repository: "owner/repo",
				Groups: []Group{
					{
						ID:   "test",
						Name: "Test Group",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty config",
			config: &Config{
				Repository: "owner/repo",
				Groups:     []Group{},
			},
			expectError: true,
		},
		{
			name: "Missing group ID",
			config: &Config{
				Repository: "owner/repo",
				Groups: []Group{
					{
						Name: "Test Group",
					},
				},
			},
			expectError: true,
		},
		{
			name: "Missing group name",
			config: &Config{
				Repository: "owner/repo",
				Groups: []Group{
					{
						ID: "test",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
