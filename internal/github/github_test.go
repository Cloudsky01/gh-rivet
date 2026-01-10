package github

import (
	"testing"
)

func TestParseWorkflowPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "single workflow",
			input: `.github/workflows/test.yml
`,
			expected: []string{"test.yml"},
		},
		{
			name: "multiple workflows",
			input: `.github/workflows/build.yml
.github/workflows/test.yml
.github/workflows/deploy.yaml
`,
			expected: []string{"build.yml", "deploy.yaml", "test.yml"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name: "with empty lines",
			input: `.github/workflows/build.yml

.github/workflows/test.yml

`,
			expected: []string{"build.yml", "test.yml"},
		},
		{
			name: "with spaces",
			input: `  .github/workflows/build.yml  
  .github/workflows/test.yml  
`,
			expected: []string{"build.yml", "test.yml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorkflowPaths(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d workflows, got %d", len(tt.expected), len(result))
				return
			}
			for i, wf := range result {
				if wf != tt.expected[i] {
					t.Errorf("expected workflow %d to be %s, got %s", i, tt.expected[i], wf)
				}
			}
		})
	}
}
