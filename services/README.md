# `services` — Execution Runtime

This package handles workflow loading, process spawning, and runtime checks.

## Core Files

- `loader.go`: parse workflow YAML into internal types.
- `runner.go`: launch each service in detached mode or new terminal mode.
- `terminal.go`: terminal detection, shell wrapping, and cross-platform fallbacks.

## Runtime Behavior

- `RunService(...)` chooses detached or terminal path per service definition.
- `runDetached(...)` supports optional logging and port preflight checks.
- `WaitForPort(...)` polls dependency ports before dependent startup.
- `openInNewTerminal(...)` selects the first available supported terminal emulator.

## Terminal Selection Order

1. `config.Current.Terminal` (`config.yml`)
2. `$TERMINAL` / `$TERM_PROGRAM`
3. supported fallback list (`gnome-terminal`, `kgx`, `kitty`, `alacritty`, `xfce4-terminal`, `konsole`, `xterm`)

## Scheduler Subpackage

See `services/scheduler/README.md` for orchestration lifecycle, IPC, and cleanup behavior.
