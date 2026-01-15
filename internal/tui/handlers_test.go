package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
)

func TestNavigationFilterAllowsPin(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &config.Config{
		Repository: "owner/repo",
		Groups: []config.Group{
			{
				ID:        "services",
				Name:      "Services",
				Workflows: []string{"build.yml", "deploy.yml"},
			},
		},
	}

	m := NewMenuModel(cfg, configPath, nil, MenuOptions{})

	// Enter the first group so workflows are visible
	m.groupPath = []*config.Group{&cfg.Groups[0]}
	m.list.SetItems(buildListItems(cfg, m.groupPath))
	m.navigationFilterInput = "build"
	m.navigationFilteredIndex = 0
	m.activePanel = NavigationPanel

	if cfg.Groups[0].IsPinned("build.yml") {
		t.Fatal("expected workflow to start unpinned")
	}

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := m.updateNavigationPanel(key)
	updated := model.(MenuModel)

	if !cfg.Groups[0].IsPinned("build.yml") {
		t.Fatal("workflow should be pinned while filter is active")
	}

	if len(updated.pinnedList.Items()) == 0 {
		t.Fatal("pinned list should refresh after pinning while filtered")
	}
}
