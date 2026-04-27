package storage

import (
	"os"
	"path/filepath"
)

func Bootstrap() error {
	dirs := []string{
		GetBasePath(),
		GetStoragePath(),
		GetLogsPath(),
		GetFlowsPath(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// create workflows.json
	file := filepath.Join(GetStoragePath(), "workflows.json")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err := os.WriteFile(file, []byte("[]"), 0644); err != nil {
			return err
		}
	}
	return nil
}
