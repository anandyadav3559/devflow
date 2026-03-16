package services

// Workflow is the root structure parsed from a workflow YAML file.
type Workflow struct {
	WorkflowName string             `yaml:"workflow_name"`
	Services     map[string]Service `yaml:"services"`
}

// Service describes a single runnable process in the workflow.
type Service struct {
	Command   string   `yaml:"command"`
	Args      []string `yaml:"args,omitempty"`
	Path      string   `yaml:"path,omitempty"`
	Port      int      `yaml:"port,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty"`
	Detached  bool     `yaml:"detached,omitempty"`
}
