package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/anandyadav3559/devflow/internal/config"
	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/anandyadav3559/devflow/services"
	"github.com/anandyadav3559/devflow/services/scheduler"
	"github.com/spf13/cobra"
)

var workflowFile string
var startName string
var noDaemon bool

// envDaemonKey is set in the child process so we know it is the daemon.
const envDaemonKey = "DEVFLOW_DAEMON"

// envRenamePrefix is used to pass service renames from parent → daemon child.
const envRenamePrefix = "DEVFLOW_RENAME_"

var startCmd = &cobra.Command{
	Use:   "start <workflow>[.<service>] [additional_services...]",
	Short: "Start a workflow (by name or file)",
	Long: `Starts a workflow defined in the specified YAML file or by its registered name.
This command resolves dependencies among services and safely launches them
in individual terminal windows (or detached processes). By default the
orchestrator runs as a background daemon so the terminal is freed immediately.
Use --no-daemon to keep it in the foreground.`,
	Example: `  devflow start -f workflows/my-project.yml
  devflow start -n my_custom_workflow
  devflow start my_workflow.frontend backend`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// load config
		if err := config.Load(); err != nil {
			fmt.Println("Warning:", err)
		}

		// ensure dirs exist (lazy init)
		if err := storage.Bootstrap(); err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		var targetServices []string
		targetService := ""
		if len(args) > 0 {
			target := args[0]
			parts := strings.SplitN(target, ".", 2)
			if startName == "" {
				startName = parts[0]
			}
			if len(parts) == 2 {
				targetService = parts[1]
				targetServices = append(targetServices, targetService)
			}
			if len(args) > 1 {
				targetServices = append(targetServices, args[1:]...)
			}
		}

		// resolve by name if specified
		if startName != "" {
			workflowFile = filepath.Join(storage.GetFlowsPath(), startName+".yml")
		} else if workflowFile == "" {
			return fmt.Errorf("you must specify either --file (-f) or --name (-n), or provide the name as an argument")
		}

		// validate file
		if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
			return fmt.Errorf("workflow file %q not found", workflowFile)
		}

		// Pre-load workflow to get its internal name
		wf, err := services.LoadWorkflow(workflowFile)
		if err != nil {
			return fmt.Errorf("cannot load workflow: %w", err)
		}

		// Determine the primary name for this run (for PID/Log files)
		effectiveName := wf.WorkflowName
		if startName != "" {
			effectiveName = startName
		}

		// ── Daemon child path ─────────────────────────────────────────────
		if os.Getenv(envDaemonKey) == "1" {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			// Check if parent passed an override session name (the interactive alias)
			if sessionName := os.Getenv("DEVFLOW_SESSION_NAME"); sessionName != "" {
				effectiveName = sessionName
			}

			scheduler.StartDaemon(ctx, effectiveName, workflowFile, targetServices)
			return nil
		}

		// ── Pre-flight: check for name conflicts / already-running ─────────
		// Common TTY check
		isTTY := func() bool {
			fi, _ := os.Stdin.Stat()
			return (fi.Mode() & os.ModeCharDevice) != 0
		}

		// 1. Check if name conflicts with a REGISTERED (built) workflow
		// Only check if we are starting from a literal file path (-f)
		if startName == "" {
			wfs, _ := storage.LoadWorkflows()
			nameExists := func(name string) bool {
				for _, reg := range wfs {
					if reg.Name == name && reg.File != workflowFile {
						return true
					}
				}
				return false
			}

			if nameExists(wf.WorkflowName) {
				if !isTTY() {
					return fmt.Errorf("workflow name %q is already taken by a registered workflow; "+
						"change the workflow_name in your YAML or use 'devflow rm %s' first",
						wf.WorkflowName, wf.WorkflowName)
				}

				fmt.Printf("⚠  Workflow name %q is already taken by a registered workflow.\n", wf.WorkflowName)
				reader := bufio.NewReader(os.Stdin)
				for {
					fmt.Printf("   Enter a new name for this run: ")
					input, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("aborted")
					}
					input = strings.TrimSpace(input)
					if input != "" {
						if !nameExists(input) {
							effectiveName = input
							fmt.Printf("   ✓ Will run as %q for this session.\n", effectiveName)
							break
						}
						fmt.Printf("   %q also exists. Try again.\n", input)
					}
				}
			}
		}

		// ALSO: Check if effectiveName has a live daemon already!
		if pid := storage.GetWorkflowDaemonPID(effectiveName); pid > 0 {
			if len(targetServices) > 0 {
				sockPath := storage.GetDaemonSocketPath(effectiveName)
				fmt.Printf("Daemon is running (PID %d). Sending START %v command via IPC...\n", pid, targetServices)
				cmdStr := "START " + strings.Join(targetServices, " ")
				resp, err := scheduler.SendIPCCommand(sockPath, cmdStr)
				if err != nil {
					return fmt.Errorf("could not send command to daemon (is it dead?): %w", err)
				}
				fmt.Printf("Daemon response: %s\n", strings.TrimSpace(resp))
				return nil
			}

			if !isTTY() {
				return fmt.Errorf("workflow %q is already running (PID %d)", effectiveName, pid)
			}
			fmt.Printf("⚠  Workflow %q is already running (PID %d).\n", effectiveName, pid)
			fmt.Printf("   Note: starting another instance might cause port conflicts.\n")
		}

		// 2. Check for already-running services
		// We only care about conflicts WITHIN the same workflow name now.
		// (User clarified: "multiple services can be of same name [across workflows]
		// only multiple service [in same workflow] can not have same name")
		active, err := storage.GetAllActive()
		if err != nil {
			return fmt.Errorf("cannot read active state: %w", err)
		}

		runningInThisWF := make(map[string]storage.ActiveEntry)
		for _, e := range active {
			if e.WorkflowName == effectiveName {
				runningInThisWF[e.ServiceName] = e
			}
		}

		// Collect conflicts and resolve them interactively
		// renames maps original name → new name chosen by user
		renames := make(map[string]string)

		for svcName := range wf.Services {
			if existing, ok := runningInThisWF[svcName]; ok {
				if !isTTY() {
					return fmt.Errorf("service %q already running in workflow %q (PID %d); "+
						"rename it or stop it first", svcName, effectiveName, existing.PID)
				}

				fmt.Printf("⚠  Service %q is already running in workflow %q (PID %d).\n",
					svcName, effectiveName, existing.PID)

				reader := bufio.NewReader(os.Stdin)
				for {
					fmt.Printf("   Enter a new name for this service (or press Enter to skip): ")
					input, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("aborted")
					}
					input = strings.TrimSpace(input)
					if input == "" {
						fmt.Printf("   Skipping %q — it will not be started.\n", svcName)
						renames[svcName] = "" // empty = skip
						break
					}
					if _, conflict := runningInThisWF[input]; conflict {
						fmt.Printf("   %q is also already running in this workflow. Try again.\n", input)
						continue
					}
					if input == svcName {
						fmt.Printf("   That's the same name. Try again.\n")
						continue
					}
					renames[svcName] = input
					fmt.Printf("   ✓ Will start as %q instead.\n", input)
					break
				}
			}
		}

		// ── No-daemon mode ────────────────────────────────────────────────
		if noDaemon {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			scheduler.Start(ctx, effectiveName, workflowFile, renames, targetServices)
			return nil
		}

		// ── Parent: self-fork into daemon ─────────────────────────────────
		ts := time.Now().Format("20060102-150405")
		logPath := storage.GetDaemonLogPath(effectiveName, ts)
		if err := os.MkdirAll(storage.GetLogsPath(), 0755); err != nil {
			return fmt.Errorf("failed to create logs directory: %w", err)
		}

		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot resolve own executable: %w", err)
		}

		childArgs := []string{"start", "--file", workflowFile}
		if startName != "" {
			// Ensure child knows it's running a registered workflow by name
			childArgs = append(childArgs, "--name", startName)
		}
		if len(targetServices) > 0 {
			childArgs = append(childArgs, targetServices...)
		}

		childEnv := append(os.Environ(), envDaemonKey+"=1")
		// Pass the session name to the child
		childEnv = append(childEnv, "DEVFLOW_SESSION_NAME="+effectiveName)

		// Pass renames as env vars to the daemon child
		for orig, newName := range renames {
			childEnv = append(childEnv, envRenamePrefix+orig+"="+newName)
		}

		child := exec.Command(self, childArgs...)
		child.Env = childEnv
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		child.Stdin = nil
		child.Stdout = nil
		child.Stderr = nil

		if err := child.Start(); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		fmt.Printf("✓ DevFlow daemon started  PID=%d\n", child.Process.Pid)
		fmt.Printf("  Logs → %s\n", logPath)
		fmt.Printf("  Run 'devflow stop %s' to terminate.\n", effectiveName)

		return nil
	},
}

func init() {
	startCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Path to workflow file")
	startCmd.Flags().StringVarP(&startName, "name", "n", "", "Name of a registered workflow to start")
	startCmd.Flags().BoolVarP(&noDaemon, "no-daemon", "D", false, "Run in the foreground (do not daemonize)")

	rootCmd.AddCommand(startCmd)
}
