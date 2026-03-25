# `cmd/` — CLI Commands

This package contains one file per DevFlow CLI command, all registered into the [cobra](https://github.com/spf13/cobra) root command defined in `root.go`.

---

## Files

### `root.go`
Defines the top-level `devflow` command and any global flags (e.g. config path). All sub-commands are added to `rootCmd` via their own `init()` functions.

---

### `build.go` — `devflow build`

Validates and registers a workflow YAML file into the local DevFlow registry.

**What it does:**
1. Reads and parses the YAML file via `services.LoadWorkflow`
2. Runs `validateWorkflow` — checks for a non-empty `workflow_name`, at least one service with a `command`, and calls `scheduler.TopoSort` to catch circular or broken dependencies
3. Checks `~/.config/devflow/storage/workflows.json` for a name conflict
4. If conflict and running in a TTY: prompts user interactively to enter a new unique name (loops until valid)
5. Saves a `WorkflowMetadata` entry (UID + name + original file path) to the registry
6. Snapshots a copy of the YAML into `~/.config/devflow/flows/<name>.yml` so it can be launched by name from anywhere

**Key flags:**
| Flag | Description |
|------|-------------|
| `-f, --file` | Path to the workflow YAML (required) |
| `-n, --name` | Override the workflow name in the registry |
| `--force` | Overwrite an existing registered workflow with the same name |

---

### `start.go` — `devflow start`

Launches a workflow — either as a background daemon (default) or in the foreground.

**Execution flow (parent process):**
1. Resolves `--file` or `--name` to an absolute file path
2. Loads the workflow to get service names
3. Calls `storage.GetAllActive()` and compares service names — if any are already running, prompts the user to rename or skip each conflict (interactive TTY) or errors (non-TTY)
4. Forks itself as a daemon child:
   - Sets `DEVFLOW_DAEMON=1` in the child's environment
   - Sets `SysProcAttr.Setsid = true` so the child is in a new session (immune to terminal close / SIGHUP)
   - Encodes any service renames as `DEVFLOW_RENAME_<original>=<new>` env vars
   - Starts the child, prints PID + log path, and exits
5. If `--no-daemon`: runs `scheduler.Start` in the foreground directly

**Execution flow (daemon child, detected via `DEVFLOW_DAEMON=1`):**
- Reads rename env vars and passes them to `scheduler.StartDaemon`

**Key flags:**
| Flag | Description |
|------|-------------|
| `-f, --file` | Path to the workflow YAML |
| `-n, --name` | Name of a registered workflow to start |
| `-D, --no-daemon` | Run in the foreground (blocking, Ctrl+C to stop) |

---

### `active.go` — `devflow active`

Lists all currently running services across all active workflows.

**What it does:**
1. Calls `storage.GetAllActive()` which glob-searches `~/.config/devflow/storage/*.state.json`
2. For each entry, checks if the PID is still alive using `syscall.Signal(0)` — dead entries are silently skipped
3. Prints a formatted table: `WORKFLOW | SERVICE | PID | TYPE | STARTED`

**TYPE** values:
- `terminal` — service opened in its own terminal window
- `detached` — service running silently in the background

---

### `stop.go` — `devflow stop`

Stops a running workflow or a single service within it.

**Syntax:**
```bash
devflow stop <workflow>             # stop all services
devflow stop <workflow>.<service>   # stop one specific service
```

**Workflow stop:**
1. Reads `~/.config/devflow/devflow.pid` and sends `SIGTERM` to the daemon
2. The daemon's context is cancelled → `scheduler.Start` runs `RunCleanup` in reverse dependency order
3. Also kills and removes any individually tracked service PIDs for that workflow

**Service stop:**
1. Finds the service entry in `GetAllActive()` matching `workflowName + serviceName`
2. Sends `SIGTERM` to that specific PID
3. Removes the entry from the state file

---

### `ls.go` — `devflow ls`

Reads `~/.config/devflow/storage/workflows.json` and prints all registered workflows with their UID and file path.

---

### `rm.go` — `devflow rm`

Removes a workflow from the registry:
1. Deletes its entry from `workflows.json`
2. Removes its snapshot from `~/.config/devflow/flows/<name>.yml`
