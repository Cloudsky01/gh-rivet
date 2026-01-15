package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/github"
)

func TestAutoRefreshToggle(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}
	m := NewMenuModel(cfg, "", gh, MenuOptions{
		RefreshInterval: 30,
	})

	// Initially auto-refresh should be enabled
	if !m.autoRefreshEnabled {
		t.Error("Expected autoRefreshEnabled to be true initially when RefreshInterval > 0")
	}

	key := tea.KeyMsg{Type: tea.KeyCtrlT}
	updatedModel, _ := m.handleKeyPress(key)
	updated := updatedModel.(MenuModel)

	if updated.autoRefreshEnabled {
		t.Error("Expected autoRefreshEnabled to be false after toggle")
	}

	updatedModel, _ = updated.handleKeyPress(key)
	updated = updatedModel.(MenuModel)

	if !updated.autoRefreshEnabled {
		t.Error("Expected autoRefreshEnabled to be true after second toggle")
	}
}

func TestStartRefreshTicker(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}

	tests := []struct {
		name               string
		refreshInterval    int
		autoRefreshEnabled bool
		expectTicker       bool
	}{
		{
			name:               "Ticker should start when enabled with interval",
			refreshInterval:    30,
			autoRefreshEnabled: true,
			expectTicker:       true,
		},
		{
			name:               "Ticker should not start when disabled",
			refreshInterval:    30,
			autoRefreshEnabled: false,
			expectTicker:       false,
		},
		{
			name:               "Ticker should not start when interval is 0",
			refreshInterval:    0,
			autoRefreshEnabled: true,
			expectTicker:       false,
		},
		{
			name:               "Ticker should not start when interval is negative",
			refreshInterval:    -1,
			autoRefreshEnabled: true,
			expectTicker:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMenuModel(cfg, "", gh, MenuOptions{
				RefreshInterval: tt.refreshInterval,
			})
			m.autoRefreshEnabled = tt.autoRefreshEnabled

			m.startRefreshTicker()

			if tt.expectTicker {
				if m.refreshTicker == nil {
					t.Error("Expected refreshTicker to be non-nil")
				} else {
					m.stopRefreshTicker()
				}
			} else {
				if m.refreshTicker != nil {
					t.Error("Expected refreshTicker to be nil")
					m.stopRefreshTicker()
				}
			}
		})
	}
}

func TestStopRefreshTicker(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}
	m := NewMenuModel(cfg, "", gh, MenuOptions{
		RefreshInterval: 30,
	})

	// Start ticker
	m.startRefreshTicker()
	if m.refreshTicker == nil {
		t.Fatal("Expected refreshTicker to be non-nil after start")
	}

	// Stop ticker
	m.stopRefreshTicker()
	if m.refreshTicker != nil {
		t.Error("Expected refreshTicker to be nil after stop")
	}

	// Stopping again should not panic
	m.stopRefreshTicker()
	if m.refreshTicker != nil {
		t.Error("Expected refreshTicker to be nil after second stop")
	}
}

func TestRenderStatusBarRefreshIndicator(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}

	tests := []struct {
		name               string
		refreshInterval    int
		autoRefreshEnabled bool
		expectInOutput     []string
		notExpectInOutput  []string
	}{
		{
			name:               "Enabled refresh shows green checkmark",
			refreshInterval:    30,
			autoRefreshEnabled: true,
			expectInOutput:     []string{"✓", "Auto-refresh", "30s"},
			notExpectInOutput:  []string{"✗"},
		},
		{
			name:               "Disabled refresh shows red X",
			refreshInterval:    30,
			autoRefreshEnabled: false,
			expectInOutput:     []string{"✗", "Auto-refresh", "30s"},
			notExpectInOutput:  []string{"✓"},
		},
		{
			name:               "No refresh interval shows nothing",
			refreshInterval:    0,
			autoRefreshEnabled: false,
			expectInOutput:     []string{},
			notExpectInOutput:  []string{"Auto-refresh", "✓", "✗"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMenuModel(cfg, "", gh, MenuOptions{
				RefreshInterval: tt.refreshInterval,
			})
			m.autoRefreshEnabled = tt.autoRefreshEnabled
			m.width = 100
			m.height = 30

			statusBar := m.renderStatusBar()

			// Strip ANSI codes for easier testing
			statusBarText := stripAnsiCodes(statusBar)

			for _, expected := range tt.expectInOutput {
				if !strings.Contains(statusBarText, expected) {
					t.Errorf("Expected status bar to contain %q, got: %q", expected, statusBarText)
				}
			}

			for _, notExpected := range tt.notExpectInOutput {
				if strings.Contains(statusBarText, notExpected) {
					t.Errorf("Expected status bar NOT to contain %q, got: %q", notExpected, statusBarText)
				}
			}
		})
	}
}

func TestGetRefreshTickerCmd(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}

	t.Run("Returns nil when ticker is not started", func(t *testing.T) {
		m := NewMenuModel(cfg, "", gh, MenuOptions{
			RefreshInterval: 0,
		})

		cmd := m.getRefreshTickerCmd()
		if cmd != nil {
			t.Error("Expected getRefreshTickerCmd to return nil when ticker is not started")
		}
	})

	t.Run("Returns command when ticker is started", func(t *testing.T) {
		m := NewMenuModel(cfg, "", gh, MenuOptions{
			RefreshInterval: 1, // 1 second for quick test
		})
		m.startRefreshTicker()
		defer m.stopRefreshTicker()

		cmd := m.getRefreshTickerCmd()
		if cmd == nil {
			t.Error("Expected getRefreshTickerCmd to return non-nil command when ticker is started")
		}
	})
}

func TestRefreshTickMsgHandling(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}
	m := NewMenuModel(cfg, "", gh, MenuOptions{
		RefreshInterval: 30,
	})
	m.selectedWorkflow = "test.yml"
	m.loading = false

	// Send refreshTickMsg
	msg := refreshTickMsg{timestamp: time.Now()}
	updatedModel, cmd := m.Update(msg)

	updatedMenuModel := updatedModel.(MenuModel)

	// Should trigger loading
	if !updatedMenuModel.loading {
		t.Error("Expected loading to be true after refreshTickMsg")
	}

	// Should return a command
	if cmd == nil {
		t.Error("Expected Update to return non-nil command after refreshTickMsg")
	}
}

func TestManualRefreshRestartsTimer(t *testing.T) {
	cfg := &config.Config{
		Repository: "test/repo",
		Groups: []config.Group{
			{
				ID:        "test",
				Name:      "Test Group",
				Workflows: []string{"test.yml"},
			},
		},
	}

	gh := &github.Client{}
	m := NewMenuModel(cfg, "", gh, MenuOptions{
		RefreshInterval: 30,
	})
	m.selectedWorkflow = "test.yml"
	m.startRefreshTicker()

	oldTicker := m.refreshTicker

	// Simulate Ctrl+R (manual refresh) - we'll test the logic directly
	m.loading = false
	if m.selectedWorkflow != "" && !m.loading && m.refreshInterval > 0 && m.autoRefreshEnabled {
		m.loading = true
		m.startRefreshTicker()
	}

	// The ticker should be restarted (new instance)
	if m.refreshTicker == oldTicker {
		t.Error("Expected ticker to be restarted (new instance)")
	}

	m.stopRefreshTicker()
}

// Helper function to strip ANSI color codes from strings
func stripAnsiCodes(s string) string {
	// Simple ANSI code stripper - removes escape sequences
	inEscape := false
	result := strings.Builder{}

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}
