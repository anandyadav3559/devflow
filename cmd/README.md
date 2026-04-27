# `cmd` — CLI Layer

This package exposes the DevFlow command surface via Cobra.

## Command Summary

- `root.go`: root `devflow` command.
- `build.go`: validate and register workflow files.
- `start.go`: launch workflows (daemon by default, foreground with `--no-daemon`).
- `active.go`: show currently active services from state files.
- `stop.go`: stop a workflow or one service.
- `ls.go`: list registered workflows.
- `rm.go`: remove a registered workflow.

## Behavior Highlights

- `build` checks schema + dependency graph before saving metadata and snapshot.
- `start` supports `--file`, `--name`, and interactive conflict handling.
- Daemon handoff uses `DEVFLOW_DAEMON=1` and `setsid` to detach from invoking shell.
- `stop` supports `<workflow>` and `<workflow>.<service>` syntax.

## Notes

- Commands now initialize storage with explicit `Bootstrap()` error handling.
- `ls` and `rm` are `RunE`-based and return errors instead of forcing `os.Exit`.
