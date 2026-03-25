package storage

import (
	"fmt"
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

// GetDaemonPidPath returns the path to the devflow daemon PID file for a specific workflow.
func GetDaemonPidPath(workflowName string) string {
	return filepath.Join(GetBasePath(), workflowName+".pid")
}

// GetDaemonLogPath returns a timestamped log file path for the daemon of a specific workflow.
// ts should be a compact timestamp string, e.g. "20060102-150405".
func GetDaemonLogPath(workflowName string, ts string) string {
	return filepath.Join(GetLogsPath(), fmt.Sprintf("devflow-daemon-%s-%s.log", workflowName, ts))
}
