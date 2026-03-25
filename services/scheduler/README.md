# `services/scheduler/` — Orchestration Engine

This sub-package is the brain of DevFlow. It sorts services by dependency, launches them in order, tracks their state, and handles all forms of shutdown.

---

## Files

### `scheduler.go`

Contains the two public entry points and the main service launch loop.

#### `StartDaemon(ctx, file)`

Called by the daemon child process (`DEVFLOW_DAEMON=1`). Before delegating to `Start`, it:

1. Opens a timestamped log file at `~/.config/devflow/logs/devflow-daemon-<ts>.log`
2. Redirects the process's own `os.Stdout` and `os.Stderr` to that file — **all scheduler output goes to the log, not the terminal**
3. Writes the daemon's PID to `~/.config/devflow/devflow.pid`
4. Reads `DEVFLOW_RENAME_<name>=<new>` environment variables (set by the parent when the user resolved name conflicts) and builds a rename map
5. Calls `Start(ctx, file, renames)`
6. On `Start` return: removes the PID file

#### `Start(ctx, file, renames)`

The core orchestration loop. It:

1. **Loads** the workflow via `services.LoadWorkflow`
2. **Applies renames**: For each entry in `renames`, renames the key in `wf.Services` and patches `depends_on` references in all other services. Empty value = service is skipped entirely.
3. **Sorts** services with `TopoSort` (Kahn's algorithm) to get a deterministic launch order
4. **Launches** services one by one:
   - Checks `IsServiceNameActive(name)` — if already running globally, skips with a warning
   - Waits for dependency ports (up to 15s per dependency)
   - Calls `services.RunService` for the actual process launch
   - On success: records the PID via `storage.SaveService`
   - Spawns a **monitor goroutine** per service (see below)
5. **Blocks** in a `select` waiting for either:
   - `ctx.Done()` (SIGTERM/SIGINT from `devflow stop`) → `RunCleanup`
   - All monitor goroutines to finish (all services exited naturally) → `RunCleanup`

#### Monitor Goroutine

For each successfully started service, a goroutine calls `process.Wait()`. When the process exits:
1. Removes the service from `storage.RemovePID` (cleans `state.json`)
2. Calls the finalizer function (closes the log file)
3. Runs the service's `on_close` commands
4. Calls `killDependents` — sends SIGTERM to any service that listed this one in `depends_on`

---

### `sorter.go`

```go
func TopoSort(services map[string]internal.Service) ([]string, error)
```

Implements **Kahn's algorithm** for topological sorting:

1. Build an in-degree map: count how many unresolved dependencies each service has
2. Enqueue all services with in-degree 0 (no dependencies) into a priority queue
3. Process the queue: add the service to the result, decrement the in-degree of its dependents, enqueue any that reach 0
4. If the result length < number of services → a cycle exists → error

Returns an ordered `[]string` of service names. The scheduler launches them in this order and cleans up in **reverse** order.

**Errors returned:**
- `"circular dependency detected"` — two or more services depend on each other cyclically
- `"unknown dependency: X"` — a service lists a `depends_on` entry that doesn't exist in the workflow

---

### `cleanup.go`

Handles all forms of graceful shutdown.

#### `killProcessGracefully(p *os.Process)`

1. Sends `SIGTERM`
2. Polls every 100ms for up to 3 seconds using `Signal(0)` (existence check)
3. If the process is still alive after 3s, sends `SIGKILL`

This gives each service a chance to flush buffers, close DB connections, etc. before being force-killed.

#### `killDependents(deadService, servicesMap, runningProcs, mu)`

Called when any monitored service exits unexpectedly. Scans `servicesMap` for services that list `deadService` in their `depends_on`, then calls `killProcessGracefully` on each one. This creates a **cascade shutdown** — closing a terminal window can trigger the entire dependent chain to terminate cleanly.

#### `runServiceCleanup(name, svc)`

Runs all `on_close` commands for a single service, in the order they are defined. Each command runs with a **15-second timeout** via `context.WithTimeout`. If the service has no `on_close`, this is a no-op.

#### `RunCleanup(wf, order, cleanedUp, runningProcs, mu)`

The global cleanup sequence triggered on SIGTERM or natural exit:

1. Iterates `order` **in reverse** (dependents before dependencies)
2. For each service not already cleaned up (tracked in `cleanedUp sync.Map`):
   - Calls `killProcessGracefully` on the running process
   - Calls `runServiceCleanup`
3. After all services: runs the workflow-level `on_close` commands

The `cleanedUp` map prevents double-execution when a service was already cleaned up by its monitor goroutine before the global cleanup ran.

---

## Data Flow

```
devflow start
    └── scheduler.StartDaemon
            ├── Redirect logs
            ├── Write PID
            └── scheduler.Start
                    ├── LoadWorkflow
                    ├── Apply renames
                    ├── TopoSort → ordered []string
                    └── For each service:
                            ├── IsServiceNameActive? → skip if yes
                            ├── WaitForPort (dependency)
                            ├── RunService → *exec.Cmd
                            ├── SaveService → state.json
                            └── goroutine: Wait → RemovePID → on_close → killDependents
```
