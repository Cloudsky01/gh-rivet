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
	cfg, err := LoadFromPath(configPath)
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
