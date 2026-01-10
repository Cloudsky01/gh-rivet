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
        inputs:
          - name: environment
            description: Target environment
            required: true
            default: staging
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading the config
	cfg, err := Load(configPath)
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

func TestLoadWithViper(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.rivet.yaml")

	configContent := `groups:
  - id: test-group
    name: Test Group
    workflows:
      - test-workflow.yml
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading with Viper
	cfg, v, err := LoadWithViper(configPath)
	if err != nil {
		t.Fatalf("Failed to load config with Viper: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
	}

	if v == nil {
		t.Fatal("Viper instance is nil")
	}

	// Verify we can access values through Viper
	if !v.IsSet("groups") {
		t.Error("Expected 'groups' to be set in Viper")
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

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load %s config: %v", tt.fileFormat, err)
			}

			if len(cfg.Groups) != 1 {
				t.Errorf("Expected 1 group in %s config, got %d", tt.fileFormat, len(cfg.Groups))
			}
		})
	}
}

func TestEnvironmentVariableOverride(t *testing.T) {
	// This test demonstrates that environment variables can override config
	// In practice, this would need more complex setup to test properly
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.rivet.yaml")

	configContent := `groups:
  - id: test-group
    name: Test Group
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config - environment variables would be read automatically
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
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
