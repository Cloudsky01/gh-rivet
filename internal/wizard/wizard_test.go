package wizard

import (
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Test Group",
			expected: "test-group",
		},
		{
			name:     "with special characters",
			input:    "CI/CD Pipeline",
			expected: "cicd-pipeline",
		},
		{
			name:     "with numbers",
			input:    "Test 123",
			expected: "test-123",
		},
		{
			name:     "with extra spaces",
			input:    "  Test   Group  ",
			expected: "test---group",
		},
		{
			name:     "with underscores",
			input:    "test_group_name",
			expected: "testgroupname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wizard{}
			result := w.generateID(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetRemainingWorkflows(t *testing.T) {
	w := &Wizard{
		availableWorkflows: []string{"a.yml", "b.yml", "c.yml", "d.yml"},
		groups: []GroupBuilder{
			{
				Name:      "Group 1",
				Workflows: []string{"a.yml", "b.yml"},
			},
		},
	}

	remaining := w.getRemainingWorkflows()
	expected := []string{"c.yml", "d.yml"}

	if len(remaining) != len(expected) {
		t.Errorf("expected %d remaining workflows, got %d", len(expected), len(remaining))
		return
	}

	for i, wf := range remaining {
		if wf != expected[i] {
			t.Errorf("expected workflow %d to be %s, got %s", i, expected[i], wf)
		}
	}
}

func TestIDExists(t *testing.T) {
	w := &Wizard{
		groups: []GroupBuilder{
			{ID: "test-1"},
			{ID: "test-2"},
		},
	}

	if !w.idExists("test-1") {
		t.Error("expected test-1 to exist")
	}

	if !w.idExists("test-2") {
		t.Error("expected test-2 to exist")
	}

	if w.idExists("test-3") {
		t.Error("expected test-3 to not exist")
	}
}

func TestIsTTY(t *testing.T) {
	result := isTTY()
	t.Logf("isTTY returned: %v", result)
}
