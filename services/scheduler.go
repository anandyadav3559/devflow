package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Start is the top-level entry point. It loads the workflow, resolves
// dependency order, and launches each service. Then it waits for a termination signal to run cleanup.
func Start(file string) {
	wf, err := LoadWorkflow(file)
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
			depSvc := wf.Services[dep]
			if depSvc.Port > 0 {
				fmt.Printf("  ⌛ Waiting for dependency %q (port %d) to be ready...\n", dep, depSvc.Port)
				if !WaitForPort(depSvc.Port, 15*time.Second) {
					fmt.Printf("  ⚠ Timeout waiting for %q port %d! Proceeding anyway...\n", dep, depSvc.Port)
				}
			}
		}

		if svc.Detached {
			fmt.Printf("Starting %q (detached)...\n", name)
		} else {
			fmt.Printf("Starting %q (new terminal)...\n", name)
		}

		cmd, err := RunService(name, svc)
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
				go func(serviceName string, service Service, process *exec.Cmd) {
					defer wg.Done()
					process.Wait() // Blocks until terminal is closed or background process exits
					
					// Run service-specific cleanup immediately
					if _, alreadyCleaned := cleanedUp.LoadOrStore(serviceName, true); !alreadyCleaned {
						runServiceCleanup(serviceName, service)
					}

					// Cascade: if this service dies, kill things that relied on it
					killDependents(serviceName, wf.Services, runningProcs, &procMu)
				}(name, svc, cmd)
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

// killProcessGracefully sends SIGTERM, waits up to 3 seconds, then fallbacks to SIGKILL
func killProcessGracefully(p *os.Process) {
	if p == nil {
		return
	}
	
	// Graceful shutdown via SIGTERM
	p.Signal(syscall.SIGTERM)

	// Poll up to 3 seconds for the process to exit
	for i := 0; i < 30; i++ {
		// Signal(0) returns an error if the process has died
		if err := p.Signal(syscall.Signal(0)); err != nil {
			return 
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Fallback to SIGKILL
	p.Kill()
}

// killDependents scans for services that depend on deadService, and conditionally stops their process.
func killDependents(deadService string, services map[string]Service, runningProcs map[string]*exec.Cmd, procMu *sync.Mutex) {
	if procMu == nil || runningProcs == nil {
		return
	}
	procMu.Lock()
	defer procMu.Unlock()

	for name, svc := range services {
		for _, dep := range svc.DependsOn {
			if dep == deadService {
				if p, ok := runningProcs[name]; ok {
					if p.Process != nil {
						fmt.Printf("Cascading shutdown: stopping %q because it depends on %q\n", name, deadService)
						killProcessGracefully(p.Process)
					}
					// Remove from map to avoid redundant kills
					delete(runningProcs, name)
				}
			}
		}
	}
}

// runServiceCleanup executes the cleanup commands for a single service
func runServiceCleanup(name string, svc Service) {
	if len(svc.OnClose) > 0 {
		fmt.Printf("Cleaning up service: %q\n", name)
		for _, cmd := range svc.OnClose {
			// Inherit the service's working directory if not explicitly overridden
			if cmd.Path == "" {
				cmd.Path = svc.Path
			}
			runCleanupCommand(cmd)
		}
	}
}

// runCleanupCommand executes a single cleanup command with a 15-second timeout
func runCleanupCommand(c CleanupCommand) {
	cmdStr := c.Command
	args := c.Args

	if len(args) == 0 && strings.Contains(cmdStr, " ") {
		parts := strings.Fields(cmdStr)
		cmdStr = parts[0]
		args = parts[1:]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdStr, args...)
	if c.Path != "" {
		cmd.Dir = expandPath(c.Path)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("  → Executing: %s %v", cmdStr, args)
	if cmd.Dir != "" {
		fmt.Printf(" (in %s)", cmd.Dir)
	}
	fmt.Println()

	if err := cmd.Run(); err != nil {
		fmt.Printf("  ✗ Error running cleanup command: %v\n", err)
	}
}

// RunCleanup executes any remaining service-level cleanup in reverse order, then global cleanup.
// It accepts a tracker map to avoid double-executing service cleanups run asynchronously.
func RunCleanup(wf *Workflow, order []string, cleanedUp *sync.Map, runningProcs map[string]*exec.Cmd, procMu *sync.Mutex) {
	// For testing, we might pass nil, so let's default it
	if cleanedUp == nil {
		cleanedUp = &sync.Map{}
	}

	// Reverse the order for cleanup
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		svc := wf.Services[name]
		
		if _, alreadyCleaned := cleanedUp.LoadOrStore(name, true); !alreadyCleaned {
			// Ensure the window is killed before sequential cleanup
			if runningProcs != nil && procMu != nil {
				procMu.Lock()
				if p, ok := runningProcs[name]; ok && p.Process != nil {
					killProcessGracefully(p.Process)
				}
				procMu.Unlock()
			}

			runServiceCleanup(name, svc)
		}
	}

	// Run global cleanup
	if len(wf.OnClose) > 0 {
		fmt.Println("Running global cleanup...")
		for _, cmd := range wf.OnClose {
			runCleanupCommand(cmd)
		}
	}
}



// topoSort returns service names in dependency order using Kahn's algorithm.
// Returns an error if an unknown dependency or circular dependency is found.
func TopoSort(services map[string]Service) ([]string, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dependency -> services that need it

	for name := range services {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
		for _, dep := range services[name].DependsOn {
			if _, ok := services[dep]; !ok {
				return nil, fmt.Errorf("service %q depends on unknown service %q", name, dep)
			}
			dependents[dep] = append(dependents[dep], name)
			inDegree[name]++
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var ordered []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		ordered = append(ordered, node)

		deps := dependents[node]
		sort.Strings(deps)
		for _, dep := range deps {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(ordered) != len(services) {
		return nil, fmt.Errorf("circular dependency detected among services")
	}

	return ordered, nil
}
