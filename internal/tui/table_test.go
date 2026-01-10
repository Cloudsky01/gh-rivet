package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		status   string
		expected lipgloss.Color
	}{
		{"completed", "green"},
		{"in_progress", "yellow"},
		{"queued", "yellow"},
		{"waiting", "yellow"},
		{"requested", "yellow"},
		{"pending", "yellow"},
		{"failed", "red"},
		{"unknown", "white"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusColor(tt.status)
			if result != tt.expected {
				t.Errorf("getStatusColor(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestGetConclusionColor(t *testing.T) {
	tests := []struct {
		conclusion string
		expected   lipgloss.Color
	}{
		{"success", "green"},
		{"failure", "red"},
		{"cancelled", "red"},
		{"timed_out", "red"},
		{"neutral", "yellow"},
		{"skipped", "yellow"},
		{"stale", "yellow"},
		{"action_required", "cyan"},
		{"", "240"},
		{"unknown", "240"},
	}

	for _, tt := range tests {
		t.Run(tt.conclusion, func(t *testing.T) {
			result := getConclusionColor(tt.conclusion)
			if result != tt.expected {
				t.Errorf("getConclusionColor(%q) = %q, want %q", tt.conclusion, result, tt.expected)
			}
		})
	}
}
