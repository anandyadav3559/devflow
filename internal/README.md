# `internal/storage/` — Persistent State & Registry

This package manages everything that DevFlow persists to disk. All files live under `~/.config/devflow/` (or `~/.devflow/` as a fallback if `os.UserConfigDir()` fails).

---

## Directory Layout

```
~/.config/devflow/
├── config.toml                  # User configuration (terminal preference, etc.)
├── devflow.pid                  # PID of the currently running daemon
├── flows/
│   └── <name>.yml               # Snapshotted workflow files (from devflow build)
├── logs/
│   ├── devflow-daemon-<ts>.log  # Daemon orchestrator output
│   └── <workflow>-<ts>/
│       └── <service>.log        # Per-service log files
└── storage/
    ├── workflows.json           # Registered workflow registry
    └── <workflow-name>.state.json  # Per-run active service tracking
```

---

## Files

### `paths.go`

All path helpers. **All other packages should use these functions** — never hardcode paths.

| Function | Returns |
|----------|---------|
| `GetBasePath()` | `~/.config/devflow` |
| `GetStoragePath()` | `~/.config/devflow/storage` |
| `GetLogsPath()` | `~/.config/devflow/logs` |
| `GetFlowsPath()` | `~/.config/devflow/flows` |
| `GetDaemonPidPath()` | `~/.config/devflow/devflow.pid` |
| `GetDaemonLogPath(ts)` | `~/.config/devflow/logs/devflow-daemon-<ts>.log` |

---

### `types.go`

Defines the shared data types used across the entire codebase:

#### `Workflow`
```go
type Workflow struct {
    WorkflowName string             // workflow_name in YAML
    Services     map[string]Service // keyed by service name
    OnClose      CleanupCommands    // global on_close
    Log          bool               // global log enable
}
```

#### `Service`
```go
type Service struct {
    Command   string
    Args      []string
    Vars      map[string]string
    Path      string
    Port      int
    DependsOn []string
    Detached  bool
    Log       bool
    OnClose   CleanupCommands
}
```

#### `CleanupCommands`
A custom YAML-unmarshallable type that accepts **both** a single map and a list of maps in YAML:
```yaml
# Single form:
on_close:
  command: docker
  args: ["compose", "down"]

# List form:
on_close:
  - command: echo
    args: ["done"]
  - command: rm
    args: ["-f", "tmp.lock"]
```

#### `WorkflowMetadata`
Registry entry stored in `workflows.json`:
```go
type WorkflowMetadata struct {
    UID  string   // random 8-byte hex ID
    Name string   // human name
    File string   // absolute path to original YAML
}
```

---

### `workflows.go`

Manages the workflow registry (`workflows.json`).

| Function | Description |
|----------|-------------|
| `LoadWorkflows()` | Read and parse all registered workflows |
| `SaveWorkflow(meta)` | Add or update a workflow. Returns error on name conflict with a different file |
| `DeleteWorkflow(name)` | Remove from registry and delete the flows snapshot |
| `GenerateUID()` | Generate a random 8-byte hex ID |

---

### `state.go`

Tracks which services are currently running across all active workflow sessions.

#### `ActiveEntry`

The core runtime record for a single running service:

```go
type ActiveEntry struct {
    WorkflowName string   // which workflow this service belongs to
    WorkflowUID  string   // same as WorkflowName (used as state file key)
    ServiceName  string   // the service's name within the workflow
    PID          int      // OS process ID
    Detached     bool     // true if detached background process
    StartedAt    string   // RFC3339 timestamp
}
```

#### Key Functions

| Function | Description |
|----------|-------------|
| `SaveService(uid, entry)` | Write/update an ActiveEntry to `<uid>.state.json` |
| `SavePID(uid, name, pid)` | Backward-compat wrapper around SaveService |
| `RemovePID(uid, name)` | Remove a service from state; deletes file if empty |
| `GetAllActive()` | Scan all `*.state.json` files, return entries with live PIDs |
| `IsServiceNameActive(name)` | Return `(true, entry)` if any workflow has a live service with that name |
| `GetWorkflowDaemonPID(name)` | Read `devflow.pid` and return the PID if the daemon is alive |

#### State File Format (`<workflow>.state.json`)

```json
{
  "detached_pids": { "redis": 18488 },
  "services": {
    "redis": {
      "workflow_name": "myapp",
      "workflow_uid": "myapp",
      "service_name": "redis",
      "pid": 18488,
      "detached": true,
      "started_at": "2026-03-25T14:10:00+05:30"
    }
  }
}
```

`detached_pids` is kept for backward compatibility. `services` is the primary map.

#### Process Liveness Check

```go
func isProcessAlive(pid int) bool {
    proc, _ := os.FindProcess(pid)
    return proc.Signal(syscall.Signal(0)) == nil
}
```

`syscall.Signal(0)` sends no actual signal — the kernel just checks if the process exists and the caller has permission to signal it. This is the standard POSIX "is this PID alive?" idiom.

---

### `bootstrap.go`

```go
func Bootstrap()
```

Creates all required directories on first run:
- `GetStoragePath()`
- `GetLogsPath()`
- `GetFlowsPath()`

Called at the start of every command that needs disk access.
