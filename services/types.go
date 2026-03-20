package services

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// CleanupCommand defines a single command to run during teardown.
type CleanupCommand struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
	Path    string   `yaml:"path,omitempty"`
}

// CleanupCommands is a slice of CleanupCommand that can handle either a single map or a list of maps in YAML.
type CleanupCommands []CleanupCommand

func (c *CleanupCommands) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		var list []CleanupCommand
		if err := value.Decode(&list); err != nil {
			return err
		}
		*c = list
		return nil
	}

	if value.Kind == yaml.MappingNode {
		var single CleanupCommand
		if err := value.Decode(&single); err != nil {
			return err
		}
		*c = CleanupCommands{single}
		return nil
	}

	return fmt.Errorf("on_close must be a map or a sequence of maps")
}

// Workflow is the root structure parsed from a workflow YAML file.
type Workflow struct {
	WorkflowName string             `yaml:"workflow_name"`
	Services     map[string]Service `yaml:"services"`
	OnClose      CleanupCommands    `yaml:"on_close,omitempty"`
}

// Service describes a single runnable process in the workflow.
type Service struct {
	Command   string             `yaml:"command"`
	Args      []string           `yaml:"args,omitempty"`
	Vars      map[string]string  `yaml:"vars,omitempty"`
	Path      string             `yaml:"path,omitempty"`
	Port      int                `yaml:"port,omitempty"`
	DependsOn []string           `yaml:"depends_on,omitempty"`
	Detached  bool               `yaml:"detached,omitempty"`
	OnClose   CleanupCommands    `yaml:"on_close,omitempty"`
}
