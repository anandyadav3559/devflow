package services

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LoadWorkflow reads a YAML workflow file and returns the parsed Workflow.
func LoadWorkflow(file string) (*Workflow, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}

	return &wf, nil
}
