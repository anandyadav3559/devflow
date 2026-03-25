# `services/` — Service Execution Layer

This package is responsible for everything that happens **after** a workflow is parsed: detecting terminals, launching processes, and waiting for ports. It is split into the core files and the `scheduler/` sub-package.

---

## Files

### `loader.go`

Parses a YAML workflow file into the `internal/storage.Workflow` struct.

```go
func LoadWorkflow(file string) (*internal.Workflow, error)
```

Uses `gopkg.in/yaml.v3`. The `Workflow` and `Service` types are defined in `internal/storage/types.go` so they can be shared between this package and the storage layer without circular imports.

---

### `runner.go`

The single entry point for launching a service process.

```go
func RunService(ctx context.Context, wfUID string, name string, service internal.Service, globalLog bool, runLogDir string) (*exec.Cmd, func(), error)
```

**Decision logic:**
- If `service.Detached == true` → calls `runDetached` (silent background process)
- Otherwise → calls `openInNewTerminal` (new terminal window)

#### `runDetached`

Runs the service as a background process attached to the current process group (the daemon).

- **With logging:** stdout/stderr go **only** to the log file (not the main terminal)
- **Without logging:** stdout/stderr go to `/dev/null` — the process is completely silent
- Checks `service.Port` first — if the port is already in use, returns an error to prevent silently connecting to a zombie process
- Injects `service.Vars` as environment variables
- Calls `storage.SavePID` to record the PID for `devflow active`

#### `WaitForPort`

```go
func WaitForPort(port int, timeout time.Duration) bool
```

Polls a TCP port every 200ms until it becomes reachable or the timeout expires. Used by the scheduler before starting dependents.

---

### `terminal.go`

Handles terminal emulator detection and launching service processes in new windows.

#### Terminal Detection

`detectTerminal()` searches for a terminal emulator in this order:
1. The value of `config.Current.Terminal` (from `~/.config/devflow/config.toml`)
2. The `$TERMINAL` or `$TERM_PROGRAM` environment variable
3. Walks a priority list: `gnome-terminal → kgx → kitty → alacritty → xfce4-terminal → konsole → xterm`

#### `openInNewTerminal`

```go
func openInNewTerminal(ctx context.Context, title string, cmd []string, dir string, vars map[string]string, logFile string) (*exec.Cmd, func(), error)
```

Wraps the service command in a shell script via `keepOpenShell`, then launches it using the detected terminal emulator's CLI flags.

Has native fallbacks for:
- **Windows:** `cmd.exe /c start ...`
- **macOS:** `osascript` + `Terminal.app`

#### `keepOpenShell`

Builds a `sh -c` script that:
1. Exports the `vars` as environment variables
2. Runs the service command
3. If logging: pipes output through `tee` to the log file
4. After exit: prints `--- process exited (press Enter to close) ---` and waits for Enter

This keeps the terminal window open after the service exits so you can read any error output.

#### `ExpandPath`

```go
func ExpandPath(path string) string
```

Expands a leading `~` to the user's home directory. Used when resolving `service.path`.

---

## `scheduler/` Sub-package

The orchestration engine. See [`scheduler/README.md`](scheduler/README.md) for full details.

| File | Responsibility |
|------|---------------|
| `scheduler.go` | `Start` and `StartDaemon` — the main service launch loop with state tracking |
| `sorter.go` | Kahn's topological sort for dependency ordering |
| `cleanup.go` | Graceful shutdown, cascade kills, `on_close` execution |
