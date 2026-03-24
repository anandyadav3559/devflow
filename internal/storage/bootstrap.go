package storage

import (
	"os"
	"path/filepath"
)

func Bootstrap() {
	dirs := []string{
		GetBasePath(),
		GetStoragePath(),
		GetLogsPath(),
		GetFlowsPath(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(err)
		}
	}

	// create workflows.json
	file := filepath.Join(GetStoragePath(), "workflows.json")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		os.WriteFile(file, []byte("[]"), 0644)
	}
}
