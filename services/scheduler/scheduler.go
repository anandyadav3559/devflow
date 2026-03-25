package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	internal "github.com/anandyadav3559/devflow/internal/storage"
	"github.com/anandyadav3559/devflow/services"
)

// renamePrefix matches DEVFLOW_RENAME_ env vars set by the parent.
const renamePrefix = "DEVFLOW_RENAME_"

// StartDaemon is called by the daemon child process. It redirects the process's
// own stdout/stderr to a log file, records the PID, applies any service renames
// passed from the parent, then calls Start normally.
func StartDaemon(ctx context.Context, workflowName string, file string) {
	ts := time.Now().Format("20060102-150405")
	logPath := internal.GetDaemonLogPath(workflowName, ts)

	// Ensure logs dir exists
	os.MkdirAll(internal.GetLogsPath(), 0755)

	logF, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "devflow daemon: cannot open log file: %v\n", err)
	} else {
		os.Stdout = logF
		os.Stderr = logF
	}

	// Write PID file
	pidPath := internal.GetDaemonPidPath(workflowName)
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)

	// Collect renames passed from the parent via env vars
	renames := map[string]string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, renamePrefix) {
			parts := strings.SplitN(env[len(renamePrefix):], "=", 2)
			if len(parts) == 2 {
				renames[parts[0]] = parts[1]
			}
		}
	}

	// Run the normal scheduler; output goes to logF
	Start(ctx, workflowName, file, renames)

	// Cleanup PID on exit
	os.Remove(pidPath)
}




// Start is the top-level entry point. It loads the workflow, resolves
// dependency order, and launches each service. Then it waits for a termination signal to run cleanup.
// renames maps original service name → new name (empty string = skip that service).
func Start(ctx context.Context, workflowName string, file string, renames map[string]string) {
	wf, err := services.LoadWorkflow(file)
	if err != nil {
		fmt.Println("Error loading workflow:", err)
		return
	}

	// Override workflow name with the effective name (e.g. the alias)
	wf.WorkflowName = workflowName


	// Apply renames: rename keys in wf.Services, skip empty-value entries
	if len(renames) > 0 {
		for orig, newName := range renames {
			svc, exists := wf.Services[orig]
			if !exists {
				continue
			}
			delete(wf.Services, orig)
			if newName != "" {
				// Also fix up depends_on references in other services
				for name, other := range wf.Services {
					for i, dep := range other.DependsOn {
						if dep == orig {
							wf.Services[name].DependsOn[i] = newName
						}
					}
				}
				wf.Services[newName] = svc
				fmt.Printf("  ↳ Service %q renamed to %q for this run.\n", orig, newName)
			} else {
				fmt.Printf("  ↳ Service %q skipped.\n", orig)
			}
		}
	}

	fmt.Println("Workflow Name:", wf.WorkflowName)

	order, err := TopoSort(wf.Services)
	if err != nil {
		fmt.Println("Dependency error:", err)
		return
	}

	var wg sync.WaitGroup
	var cleanedUp sync.Map
	var failedServices sync.Map // Track services that failed to launch

	runningProcs := make(map[string]*exec.Cmd)
	var procMu sync.Mutex

	var logDir string
	for _, svc := range wf.Services {
		if svc.Log || wf.Log {
			logDir = filepath.Join(internal.GetLogsPath(), fmt.Sprintf("%s-%s", wf.WorkflowName, time.Now().Format("20060102-150405")))
			os.MkdirAll(logDir, 0755)
			break
		}
	}

	for _, name := range order {
		svc := wf.Services[name]

		// 1. Fail Fast: check if any dependency failed
		shouldFail := false
		for _, dep := range svc.DependsOn {
			if _, failed := failedServices.Load(dep); failed {
				fmt.Printf("  ⚠ Skipping %q because dependency %q failed to start.\n", name, dep)
				failedServices.Store(name, true)
				shouldFail = true
				break
			}
		}
		if shouldFail {
			continue
		}

		// 2. Duplicate service name check - ONLY within the same workflow now
		if alive, existing := internal.IsServiceNameActiveInWorkflow(wf.WorkflowName, name); alive {
			fmt.Printf("  ✗ Skipping %q: already running in this workflow (PID %d). Stop it first.\n",
				name, existing.PID)
			failedServices.Store(name, true)
			continue
		}


		// 3. Readiness Polling: Wait for dependencies to be ready!
		for _, dep := range svc.DependsOn {
			if depSvc, ok := wf.Services[dep]; ok && depSvc.Port > 0 {
				fmt.Printf("  ⌛ Waiting for dependency %q (port %d) to be ready...\n", dep, depSvc.Port)
				if !services.WaitForPort(depSvc.Port, 15*time.Second) {
					fmt.Printf("  ⚠ Timeout waiting for %q port %d! Proceeding anyway...\n", dep, depSvc.Port)
				}
			}
		}

		if svc.Detached {
			fmt.Printf("Starting %q (detached)...\n", name)
		} else {
			fmt.Printf("Starting %q (new terminal)...\n", name)
		}

		cmd, finalizer, err := services.RunService(ctx, wf.WorkflowName, name, svc, wf.Log, logDir)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
			failedServices.Store(name, true)
		} else {
			fmt.Printf("  ✓ OK\n")
			if cmd != nil {
				procMu.Lock()
				runningProcs[name] = cmd
				procMu.Unlock()

				// Track service in active state (both detached and terminal)
				if cmd.Process != nil {
					_ = internal.SaveService(wf.WorkflowName, internal.ActiveEntry{
						WorkflowName: wf.WorkflowName,
						WorkflowUID:  wf.WorkflowName,
						ServiceName:  name,
						PID:          cmd.Process.Pid,
						Detached:     svc.Detached,
						StartedAt:    time.Now().Format(time.RFC3339),
					})
				}

				wg.Add(1)
				go func(serviceName string, service internal.Service, process *exec.Cmd, fin func()) {
					defer wg.Done()
					process.Wait() // Blocks until terminal is closed or background process exits

					// Remove from active state tracking
					_ = internal.RemovePID(wf.WorkflowName, serviceName)

					if fin != nil {
						fin()
					}

					// Run service-specific cleanup immediately
					if _, alreadyCleaned := cleanedUp.LoadOrStore(serviceName, true); !alreadyCleaned {
						runServiceCleanup(serviceName, service)
					}

					// Cascade: if this service dies, kill things that relied on it
					killDependents(serviceName, wf.Services, runningProcs, &procMu)
				}(name, svc, cmd, finalizer)
			}
		}
	}

	// Wait for termination signal or all child processes to gracefully close
	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	fmt.Println("\nAll services started. Press Ctrl+C to terminate and run cleanup.")

	select {
	case <-ctx.Done():
		fmt.Println("\nReceived termination signal. Running global cleanup sequence...")
		RunCleanup(wf, order, &cleanedUp, runningProcs, &procMu)
		wg.Wait() // Ensure all goroutines wrap up cleanly after we kill them
	case <-doneChan:
		fmt.Println("\nAll child windows closed naturally. Running global cleanup sequence...")
		RunCleanup(wf, order, &cleanedUp, runningProcs, &procMu)
	}

	fmt.Println("Cleanup complete. Exiting.")
}
