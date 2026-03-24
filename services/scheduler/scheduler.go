package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	internal "github.com/anandyadav3559/devflow/internal/storage"
	"github.com/anandyadav3559/devflow/services"
)

// Start is the top-level entry point. It loads the workflow, resolves
// dependency order, and launches each service. Then it waits for a termination signal to run cleanup.
func Start(file string) {
	wf, err := services.LoadWorkflow(file)
	if err != nil {
		fmt.Println("Error loading workflow:", err)
		return
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

		// 2. Readiness Polling: Wait for dependencies to be ready!
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

		cmd, finalizer, err := services.RunService(name, svc, wf.Log, logDir)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
			failedServices.Store(name, true)
		} else {
			fmt.Printf("  ✓ OK\n")
			if cmd != nil {
				procMu.Lock()
				runningProcs[name] = cmd
				procMu.Unlock()

				wg.Add(1)
				go func(serviceName string, service internal.Service, process *exec.Cmd, fin func()) {
					defer wg.Done()
					process.Wait() // Blocks until terminal is closed or background process exits

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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	fmt.Println("\nAll services started. Press Ctrl+C to terminate and run cleanup.")

	select {
	case <-sigChan:
		fmt.Println("\nReceived termination signal. Running global cleanup sequence...")
		RunCleanup(wf, order, &cleanedUp, runningProcs, &procMu)
		wg.Wait() // Ensure all goroutines wrap up cleanly after we kill them
	case <-doneChan:
		fmt.Println("\nAll child windows closed naturally. Running global cleanup sequence...")
		RunCleanup(wf, order, &cleanedUp, runningProcs, &procMu)
	}

	fmt.Println("Cleanup complete. Exiting.")
}
