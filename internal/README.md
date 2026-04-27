# `internal/storage` — Persistent State

This package owns DevFlow persistence under `~/.config/devflow/` (fallback: `~/.devflow/`).

## On-Disk Layout

```text
~/.config/devflow/
├── flows/
│   └── <workflow>.yml
├── logs/
│   ├── devflow-daemon-<workflow>-<ts>.log
│   └── <workflow>-<ts>/<service>.log
├── <workflow>.pid
├── <workflow>.sock
└── storage/
    ├── workflows.json
    └── <workflow>.state.json
```

## Core Files

- `paths.go`: canonical path builders (`GetBasePath`, `GetStoragePath`, `GetDaemonPidPath`, `GetDaemonSocketPath`, ...).
- `types.go`: workflow/service schema types, including flexible `on_close` parsing.
- `workflows.go`: workflow registry CRUD and UID generation.
- `state.go`: runtime active-service state (`SaveService`, `RemovePID`, `GetAllActive`) with PID liveness filtering.
- `bootstrap.go`: creates required directories and initializes `workflows.json`.

## Notes

- `Bootstrap()` now returns an `error` (no panic path).
- State files use `services` as the source of truth; `detached_pids` remains for compatibility.
- `GetWorkflowDaemonPID(name)` checks both PID-file parsing and process liveness.
