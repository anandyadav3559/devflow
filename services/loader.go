package services

import (
	"os"

	internal "github.com/anandyadav3559/devflow/internal/storage"
	"gopkg.in/yaml.v3"
)

// LoadWorkflow reads a YAML workflow file and returns the parsed Workflow.
func LoadWorkflow(file string) (*internal.Workflow, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var wf internal.Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}

	return &wf, nil
}
