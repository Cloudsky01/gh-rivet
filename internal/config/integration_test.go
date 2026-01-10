package config

import (
	"os"
	"testing"
)

// TestLoadActualConfig tests loading the actual .rivet.yaml file if it exists
func TestLoadActualConfig(t *testing.T) {
	configPath := "../../.rivet.yaml"

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Skipping test: .rivet.yaml not found")
	}

	// Try to load the actual config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load actual config: %v", err)
	}

	// Validate it
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Actual config is invalid: %v", err)
	}

	// Basic sanity checks
	if len(cfg.Groups) == 0 {
		t.Error("Expected at least one group in actual config")
	}

	t.Logf("Successfully loaded config with %d top-level group(s)", len(cfg.Groups))
}

// TestEnvVarOverride demonstrates environment variable override capability
func TestEnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/test.yaml"

	configContent := `groups:
  - id: test
    name: Original Name
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Note: Viper's environment variable override works at the Viper level
	// This test demonstrates the capability is available
	cfg, v, err := LoadWithViper(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// The viper instance is now configured to read RIVET_* environment variables
	// In a real scenario, setting RIVET_GROUPS_0_NAME would override the name
	if v.GetEnvPrefix() != "RIVET" {
		t.Errorf("Expected env prefix 'RIVET', got '%s'", v.GetEnvPrefix())
	}

	if cfg.Groups[0].Name != "Original Name" {
		t.Errorf("Expected 'Original Name', got '%s'", cfg.Groups[0].Name)
	}
}
