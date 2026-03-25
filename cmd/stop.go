package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <workflow> | <workflow>.<service>",
	Short: "Stop a running workflow or a single service within one",
	Long: `Stop all services in a workflow or a specific service within it.

  devflow stop myworkflow            – terminates the entire workflow (and its daemon)
  devflow stop myworkflow.backend    – kills only the "backend" service in "myworkflow"`,
	Example: `  devflow stop my_project
  devflow stop my_project.redis`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Split on first '.' to detect workflow.service syntax
		parts := strings.SplitN(target, ".", 2)
		workflowName := parts[0]
		serviceName := ""
		if len(parts) == 2 {
			serviceName = parts[1]
		}

		if serviceName != "" {
			// ── Stop a single service ──────────────────────────────────────
			return stopSingleService(workflowName, serviceName)
		}

		// ── Stop entire workflow ───────────────────────────────────────────
		return stopWorkflow(workflowName)
	},
}

// stopWorkflow sends SIGTERM to the daemon PID (which triggers its cleanup
// cascade for all services), then kills any remaining tracked services.
func stopWorkflow(workflowName string) error {
	killed := 0

	// 1. Try to stop via daemon PID file
	daemonPID := storage.GetWorkflowDaemonPID(workflowName)
	if daemonPID > 0 {
		if err := signalPID(daemonPID, syscall.SIGTERM); err != nil {
			fmt.Printf("  ⚠ Could not signal daemon (PID %d): %v\n", daemonPID, err)
		} else {
			fmt.Printf("  ✓ Sent SIGTERM to daemon (PID %d)\n", daemonPID)
			killed++
		}
	}

	// 2. Also kill any individually tracked services for this workflow
	entries, err := storage.GetAllActive()
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.WorkflowName != workflowName {
			continue
		}
		if err := signalPID(e.PID, syscall.SIGTERM); err != nil {
			fmt.Printf("  ⚠ Could not signal %q (PID %d): %v\n", e.ServiceName, e.PID, err)
		} else {
			fmt.Printf("  ✓ Stopped %q (PID %d)\n", e.ServiceName, e.PID)
			killed++
		}
		_ = storage.RemovePID(workflowName, e.ServiceName)
	}

	if killed == 0 {
		return fmt.Errorf("no running services found for workflow %q", workflowName)
	}
	return nil
}

// stopSingleService kills one specific service by name within a workflow.
func stopSingleService(workflowName, serviceName string) error {
	entries, err := storage.GetAllActive()
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.WorkflowName == workflowName && e.ServiceName == serviceName {
			if err := signalPID(e.PID, syscall.SIGTERM); err != nil {
				return fmt.Errorf("could not signal %q (PID %d): %w", serviceName, e.PID, err)
			}
			_ = storage.RemovePID(workflowName, serviceName)
			fmt.Printf("  ✓ Stopped %q (PID %d)\n", serviceName, e.PID)
			return nil
		}
	}

	return fmt.Errorf("service %q in workflow %q is not running", serviceName, workflowName)
}

// signalPID sends sig to the process with the given PID.
func signalPID(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

// pidFromFile reads an integer PID from a file (helper for daemon PID file).
func pidFromFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
