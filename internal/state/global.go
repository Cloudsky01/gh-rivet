package state

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Cloudsky01/gh-rivet/internal/paths"
)

type GlobalState struct {
	ActiveRepository string `yaml:"activeRepository,omitempty"`
}

func LoadGlobal(p *paths.Paths) (*GlobalState, error) {
	path := p.GlobalStateFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalState{}, nil
		}
		return nil, fmt.Errorf("failed to read global state: %w", err)
	}

	var state GlobalState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return &GlobalState{}, nil
	}

	return &state, nil
}

func (s *GlobalState) Save(p *paths.Paths) error {
	path := p.GlobalStateFile()

	if err := p.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to ensure state directory: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal global state: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
