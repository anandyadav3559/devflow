# `services/scheduler` — Orchestration Core

This package runs the lifecycle for workflow services: order, launch, monitor, IPC control, and cleanup.

## Core Components

- `scheduler.go`
  - `StartDaemon(...)`: daemon bootstrap, log redirection, PID persistence.
  - `Start(...)`: load workflow, apply renames, topo-order launch, monitor, and shutdown handling.
- `sorter.go`
  - `TopoSort(...)`: Kahn algorithm for dependency ordering with unknown/cycle detection.
- `cleanup.go`
  - graceful termination policy (`SIGTERM`, then `SIGKILL` fallback), `on_close` execution, dependent cascade.
- `ipc.go`
  - Unix socket IPC (`<workflow>.sock`) with `0600` permissions.
  - supports daemon commands like `START <service...>`.

## Runtime Flow

```text
devflow start
  -> daemon child (optional)
  -> StartDaemon
     -> Start
        -> LoadWorkflow
        -> TopoSort
        -> spawnGraph
        -> monitor goroutines
        -> RunCleanup on signal or natural completion
```

## Important Behaviors

- Service rename/skip support via `DEVFLOW_RENAME_*` env variables passed from parent process.
- Duplicate service guard checks active state within same workflow.
- Dependency port readiness wait before starting dependents.
- Process and state cleanup on both graceful stop and natural exits.
