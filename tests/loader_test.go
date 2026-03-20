package tests

import (
	"os"
	"testing"

	"github.com/anandyadav3559/devflow/services"
)

func TestLoadWorkflow(t *testing.T) {
	// Create a temporary workflow file
	content := `
workflow_name: Test Workflow
services:
  s1:
    command: echo
    args: ["hello"]
  s2:
    command: echo
    depends_on: ["s1"]
`
	tmpFile := "test_workflow.yml"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	wf, err := services.LoadWorkflow(tmpFile)
	if err != nil {
		t.Fatalf("LoadWorkflow failed: %v", err)
	}

	if wf.WorkflowName != "Test Workflow" {
		t.Errorf("expected 'Test Workflow', got %s", wf.WorkflowName)
	}

	if len(wf.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(wf.Services))
	}
}
