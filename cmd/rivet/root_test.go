package main

import (
	"path/filepath"
	"testing"

	"github.com/Cloudsky01/gh-rivet/internal/paths"
)

func TestDetermineConfigSaveTarget_UserDefault(t *testing.T) {
	tmpDir := t.TempDir()
	p := &paths.Paths{
		UserConfigDir:         filepath.Join(tmpDir, "config"),
		RepoDefaultConfigPath: filepath.Join(tmpDir, ".github", paths.LegacyConfigFileName),
	}

	path, location, err := determineConfigSaveTarget(p, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "config", paths.ConfigFileName)
	if path != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, path)
	}

	if location != saveLocationUser {
		t.Fatalf("expected saveLocationUser, got %v", location)
	}
}

func TestDetermineConfigSaveTarget_TeamRequiresRepo(t *testing.T) {
	p := &paths.Paths{}

	if _, _, err := determineConfigSaveTarget(p, false, "", "team"); err == nil {
		t.Fatal("expected error when repo path is unavailable for team config")
	}
}

func TestDetermineConfigSaveTarget_TeamPath(t *testing.T) {
	tmpDir := t.TempDir()
	teamPath := filepath.Join(tmpDir, ".github", paths.LegacyConfigFileName)
	p := &paths.Paths{
		UserConfigDir:         filepath.Join(tmpDir, "config"),
		RepoDefaultConfigPath: teamPath,
	}

	path, location, err := determineConfigSaveTarget(p, false, "", "team")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != teamPath {
		t.Fatalf("expected team path %s, got %s", teamPath, path)
	}

	if location != saveLocationTeam {
		t.Fatalf("expected saveLocationTeam, got %v", location)
	}
}

func TestDetermineConfigSaveTarget_Explicit(t *testing.T) {
	tmpDir := t.TempDir()
	explicit := filepath.Join(tmpDir, "custom", "config.yaml")
	p := &paths.Paths{
		UserConfigDir: filepath.Join(tmpDir, "config"),
	}

	path, location, err := determineConfigSaveTarget(p, true, explicit, "team")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != explicit {
		t.Fatalf("expected explicit path %s, got %s", explicit, path)
	}

	if location != saveLocationExplicit {
		t.Fatalf("expected saveLocationExplicit, got %v", location)
	}
}
