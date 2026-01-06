package wizard

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DiscoverWorkflows(workflowDir string) ([]string, error) {
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil, err
	}

	var workflows []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".yml" || ext == ".yaml" {
			workflows = append(workflows, entry.Name())
		}
	}

	sort.Strings(workflows)
	return workflows, nil
}
