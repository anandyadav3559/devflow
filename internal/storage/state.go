package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// ActiveEntry represents a single running service instance.
type ActiveEntry struct {
	WorkflowName string `json:"workflow_name"`
	WorkflowUID  string `json:"workflow_uid"`
	ServiceName  string `json:"service_name"`
	PID          int    `json:"pid"`
	Detached     bool   `json:"detached"`
	StartedAt    string `json:"started_at"` // RFC3339
}

// State represents the runtime state of a workflow.
type State struct {
	// DetachedPIDs kept for backward compat but no longer the primary store.
	DetachedPIDs map[string]int `json:"detached_pids"`
	// Services tracks ALL running services (detached + terminal).
	Services map[string]ActiveEntry `json:"services"`
}

var stateMu sync.Mutex

func getStateFile(workflowUID string) string {
	return filepath.Join(GetStoragePath(), fmt.Sprintf("%s.state.json", workflowUID))
}

// LoadState loads the runtime state for a specific workflow UID.
func LoadState(uid string) (State, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return loadStateUnlocked(uid)
}

func loadStateUnlocked(uid string) (State, error) {
	file := getStateFile(uid)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyState(), nil
		}
		return State{}, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	if s.DetachedPIDs == nil {
		s.DetachedPIDs = make(map[string]int)
	}
	if s.Services == nil {
		s.Services = make(map[string]ActiveEntry)
	}
	return s, nil
}

func emptyState() State {
	return State{
		DetachedPIDs: make(map[string]int),
		Services:     make(map[string]ActiveEntry),
	}
}

// SavePID records a service's PID in both the legacy map and the Services map.
func SavePID(uid string, serviceName string, pid int) error {
	return SaveService(uid, ActiveEntry{
		WorkflowUID: uid,
		ServiceName: serviceName,
		PID:         pid,
	})
}

// SaveService records a full ActiveEntry for a running service.
func SaveService(uid string, entry ActiveEntry) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	s, err := loadStateUnlocked(uid)
	if err != nil {
		return err
	}

	s.Services[entry.ServiceName] = entry
	// Maintain legacy map too
	s.DetachedPIDs[entry.ServiceName] = entry.PID

	return writeState(uid, s)
}

// RemovePID removes a service from tracking.
func RemovePID(uid string, serviceName string) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	s, err := loadStateUnlocked(uid)
	if err != nil {
		return err
	}
	delete(s.DetachedPIDs, serviceName)
	delete(s.Services, serviceName)

	if len(s.Services) == 0 {
		// Clean up the state file entirely when no services are left
		if err := os.Remove(getStateFile(uid)); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return writeState(uid, s)
}

// GetAllActive reads all state files and returns every service that still has
// a running process. Dead PIDs are automatically filtered out.
func GetAllActive() ([]ActiveEntry, error) {
	stateMu.Lock()
	defer stateMu.Unlock()

	entries, err := filepath.Glob(filepath.Join(GetStoragePath(), "*.state.json"))
	if err != nil {
		return nil, err
	}

	var active []ActiveEntry
	for _, f := range entries {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var s State
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		for _, e := range s.Services {
			if isProcessAlive(e.PID) {
				active = append(active, e)
			}
		}
	}
	return active, nil
}

// IsServiceNameActiveInWorkflow returns true if a specific service is running within a specific workflow.
func IsServiceNameActiveInWorkflow(workflowName, serviceName string) (bool, ActiveEntry) {
	entries, _ := GetAllActive()
	for _, e := range entries {
		if e.WorkflowName == workflowName && e.ServiceName == serviceName {
			return true, e
		}
	}
	return false, ActiveEntry{}
}

// IsServiceNameActive returns true if a service with the given name is already
// running in any workflow.
func IsServiceNameActive(name string) (bool, ActiveEntry) {
	entries, _ := GetAllActive()
	for _, e := range entries {
		if e.ServiceName == name {
			return true, e
		}
	}
	return false, ActiveEntry{}
}

// GetWorkflowDaemonPID returns the daemon PID for a running workflow by name.
// Returns 0 if not found or not running.
func GetWorkflowDaemonPID(workflowName string) int {
	data, err := os.ReadFile(GetDaemonPidPath(workflowName))
	if err != nil {
		return 0
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0
	}
	if isProcessAlive(pid) {
		return pid
	}
	return 0
}

// isProcessAlive checks whether a PID is still alive by sending signal 0.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0: no signal sent — just checks process existence
	return proc.Signal(syscall.Signal(0)) == nil
}

func writeState(uid string, s State) error {
	if err := os.MkdirAll(GetStoragePath(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getStateFile(uid), data, 0644)
}
