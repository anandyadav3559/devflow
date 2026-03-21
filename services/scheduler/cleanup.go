package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/anandyadav3559/devflow/services"
)

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
func killDependents(deadService string, servicesMap map[string]services.Service, runningProcs map[string]*exec.Cmd, procMu *sync.Mutex) {
	if procMu == nil || runningProcs == nil {
		return
	}
	procMu.Lock()
	defer procMu.Unlock()

	for name, svc := range servicesMap {
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
func runServiceCleanup(name string, svc services.Service) {
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
func runCleanupCommand(c services.CleanupCommand) {
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
		// We use expandPath from services package if it was exported.
		// Since it starts with a lowercase 'e', it's NOT exported.
		// I will have to export 'expandPath' in terminal.go or move it to a common place.
		// For now, let's assume I will export it.
		cmd.Dir = services.ExpandPath(c.Path)
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
func RunCleanup(wf *services.Workflow, order []string, cleanedUp *sync.Map, runningProcs map[string]*exec.Cmd, procMu *sync.Mutex) {
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
