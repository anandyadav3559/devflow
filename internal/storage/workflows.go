package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func GenerateUID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a time-derived value if crypto entropy is unavailable.
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

func getWorkflowsFile() string {
	return filepath.Join(GetStoragePath(), "workflows.json")
}

func LoadWorkflows() ([]WorkflowMetadata, error) {
	file := getWorkflowsFile()
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorkflowMetadata{}, nil
		}
		return nil, err
	}

	var wfs []WorkflowMetadata
	if err := json.Unmarshal(data, &wfs); err != nil {
		return nil, err
	}
	return wfs, nil
}

func SaveWorkflow(newWf WorkflowMetadata) error {
	wfs, err := LoadWorkflows()
	if err != nil {
		return err
	}

	// Check for duplicates by name
	for i, wf := range wfs {
		if wf.Name == newWf.Name {
			if wf.File == newWf.File {
				// If it's the same name and same file, just update it (e.g. new UID or refresh)
				wfs[i] = newWf
				return writeWorkflows(wfs)
			}
			return fmt.Errorf("a workflow with the name %q already exists (at %s)", newWf.Name, wf.File)
		}
	}

	wfs = append(wfs, newWf)
	return writeWorkflows(wfs)
}

func DeleteWorkflow(name string) error {
	wfs, err := LoadWorkflows()
	if err != nil {
		return err
	}

	foundIndex := -1
	for i, wf := range wfs {
		if wf.Name == name {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return fmt.Errorf("workflow %q not found", name)
	}

	// Delete the local copy snapshot
	flowPath := filepath.Join(GetFlowsPath(), name+".yml")
	if err := os.Remove(flowPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed removing workflow snapshot %q: %w", flowPath, err)
	}

	// Remove from list
	wfs = append(wfs[:foundIndex], wfs[foundIndex+1:]...)
	return writeWorkflows(wfs)
}

func writeWorkflows(wfs []WorkflowMetadata) error {
	data, err := json.MarshalIndent(wfs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getWorkflowsFile(), data, 0644)
}
