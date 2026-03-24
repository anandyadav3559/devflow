package storage

import "path/filepath"

func GetBasePath() string {
	return "./.devflow" // dev mode
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
