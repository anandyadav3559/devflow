package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// State represents the runtime state of a workflow (like detached PIDs)
type State struct {
	DetachedPIDs map[string]int `json:"detached_pids"` // mapped by service name -> pid
}

var stateMu sync.Mutex

func getStateFile(workflowUID string) string {
	return filepath.Join(GetStoragePath(), fmt.Sprintf("%s.state.json", workflowUID))
}

// LoadState loads the runtime state for a specific workflow UID.
func LoadState(uid string) (State, error) {
	stateMu.Lock()
	defer stateMu.Unlock()

	file := getStateFile(uid)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return State{DetachedPIDs: make(map[string]int)}, nil
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
	return s, nil
}

// SavePID saves a PID for a detached service into the workflow's state file.
func SavePID(uid string, serviceName string, pid int) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	// Direct file read inside the lock avoiding deadlock from LoadState
	file := getStateFile(uid)
	var s State
	data, err := os.ReadFile(file)
	if err == nil {
		json.Unmarshal(data, &s)
	}
	if s.DetachedPIDs == nil {
		s.DetachedPIDs = make(map[string]int)
	}

	s.DetachedPIDs[serviceName] = pid

	newData, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, newData, 0644)
}

// RemovePID removes a PID from the workflow's state track.
func RemovePID(uid string, serviceName string) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	file := getStateFile(uid)
	var s State
	data, err := os.ReadFile(file)
	if err == nil {
		json.Unmarshal(data, &s)
	}
	if s.DetachedPIDs == nil || s.DetachedPIDs[serviceName] == 0 {
		return nil
	}

	delete(s.DetachedPIDs, serviceName)

	newData, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, newData, 0644)
}
