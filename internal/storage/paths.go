package storage

import (
	"os"
	"path/filepath"
)

func GetBasePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home dir if UserConfigDir fails
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".devflow")
	}
	return filepath.Join(configDir, "devflow")
}

func GetStoragePath() string {
	return filepath.Join(GetBasePath(), "storage")
}

func GetLogsPath() string {
	return filepath.Join(GetBasePath(), "logs")
}

func GetFlowsPath() string {
	return filepath.Join(GetBasePath(), "flows")
}
