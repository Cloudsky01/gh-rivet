package config

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Repository string  `yaml:"repository"`
	Groups     []Group `yaml:"groups,omitempty"`
}

type Workflow struct {
	File string `yaml:"file"`
	Name string `yaml:"name,omitempty"`
}

func (w *Workflow) DisplayName() string {
	if w.Name != "" {
		return w.Name
	}
	return w.File
}

type Group struct {
	ID               string     `yaml:"id"`
	Name             string     `yaml:"name"`
	Description      string     `yaml:"description,omitempty"`
	Workflows        []string   `yaml:"workflows,omitempty"`
	WorkflowDefs     []Workflow `yaml:"workflowDefs,omitempty"`
	WorkflowPatterns []string   `yaml:"workflowPatterns,omitempty"`
	Jobs             []string   `yaml:"jobs,omitempty"`
	Groups           []Group    `yaml:"groups,omitempty"`
	PinnedWorkflows  []string   `yaml:"pinnedWorkflows,omitempty"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".rivet")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME")
	}

	v.SetEnvPrefix("RIVET")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func LoadWithViper(path string) (*Config, *viper.Viper, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".rivet")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME")
	}

	v.SetEnvPrefix("RIVET")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, v, nil
}

func WatchConfig(v *viper.Viper, onConfigChange func(*Config)) {
	v.OnConfigChange(func(e fsnotify.Event) {
		var newConfig Config
		if err := v.Unmarshal(&newConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error reloading config: %v\n", err)
			return
		}
		onConfigChange(&newConfig)
	})
	v.WatchConfig()
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add helpful header comment
	header := `# Rivet Configuration
# Learn more: https://github.com/Cloudsky01/gh-rivet
#
# Configuration structure:
# - repository: GitHub repository in owner/repo format
# - groups: Organize your workflows into groups
#   - id: Unique identifier (auto-generated from name)
#   - name: Display name shown in the TUI
#   - description: Optional description
#   - workflows: List of workflow filenames
#   - pinnedWorkflows: Workflows to pin to the top
#   - groups: Nested groups for hierarchical organization
#
# Run 'rivet --help' for more information

`
	fullContent := header + string(data)

	if err := os.WriteFile(path, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Repository == "" {
		return fmt.Errorf("configuration must specify a repository (owner/repo)")
	}

	if len(c.Groups) == 0 {
		return fmt.Errorf("configuration must have at least one group")
	}

	for _, group := range c.Groups {
		if err := c.validateGroup(&group, ""); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) validateGroup(group *Group, path string) error {
	currentPath := path
	if currentPath != "" {
		currentPath += "/"
	}
	currentPath += group.ID

	if group.ID == "" {
		return fmt.Errorf("group at path %s missing id", currentPath)
	}
	if group.Name == "" {
		return fmt.Errorf("group %s missing name", currentPath)
	}

	for _, pattern := range group.Jobs {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex pattern in group %s: %s (%w)", currentPath, pattern, err)
		}
	}

	for i := range group.Groups {
		if err := c.validateGroup(&group.Groups[i], currentPath); err != nil {
			return err
		}
	}

	return nil
}

func (g *Group) GetAllWorkflows() []string {
	workflows := make([]string, 0)

	workflows = append(workflows, g.Workflows...)

	for _, wf := range g.WorkflowDefs {
		workflows = append(workflows, wf.File)
	}

	for i := range g.Groups {
		workflows = append(workflows, g.Groups[i].GetAllWorkflows()...)
	}

	return workflows
}

func (g *Group) GetWorkflowDef(filename string) *Workflow {
	for i := range g.WorkflowDefs {
		if g.WorkflowDefs[i].File == filename {
			return &g.WorkflowDefs[i]
		}
	}
	return nil
}

func (g *Group) HasWorkflows() bool {
	if len(g.Workflows) > 0 || len(g.WorkflowDefs) > 0 {
		return true
	}

	for i := range g.Groups {
		if g.Groups[i].HasWorkflows() {
			return true
		}
	}

	return false
}

func (g *Group) IsLeaf() bool {
	return len(g.Groups) == 0
}

func (g *Group) IsPinned(workflowName string) bool {
	return slices.Contains(g.PinnedWorkflows, workflowName)
}

func (g *Group) TogglePin(workflowName string) {
	if g.IsPinned(workflowName) {
		for i, wf := range g.PinnedWorkflows {
			if wf == workflowName {
				g.PinnedWorkflows = append(g.PinnedWorkflows[:i], g.PinnedWorkflows[i+1:]...)
				break
			}
		}
	} else {
		g.PinnedWorkflows = append(g.PinnedWorkflows, workflowName)
	}
}

type PinnedWorkflow struct {
	WorkflowName string
	GroupPath    []string
	Group        *Group
}

func (c *Config) GetAllPinnedWorkflows() []PinnedWorkflow {
	var pinned []PinnedWorkflow

	for i := range c.Groups {
		pinned = append(pinned, collectPinnedFromGroup(&c.Groups[i], []string{})...)
	}

	return pinned
}

func collectPinnedFromGroup(g *Group, parentPath []string) []PinnedWorkflow {
	var pinned []PinnedWorkflow

	currentPath := append([]string{}, parentPath...)
	currentPath = append(currentPath, g.Name)

	for _, wf := range g.PinnedWorkflows {
		pinned = append(pinned, PinnedWorkflow{
			WorkflowName: wf,
			GroupPath:    currentPath,
			Group:        g,
		})
	}

	for i := range g.Groups {
		pinned = append(pinned, collectPinnedFromGroup(&g.Groups[i], currentPath)...)
	}

	return pinned
}
