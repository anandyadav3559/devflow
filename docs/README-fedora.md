# DevFlow on Fedora

[![Test](https://github.com/anandyadav3559/devflow/actions/workflows/test.yml/badge.svg)](https://github.com/anandyadav3559/devflow/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](../LICENSE)

Production-style setup guide for Fedora users.

## Prerequisites

- Fedora 40+ (tested on Fedora 43)
- Go `1.25+`
- Git
- A supported terminal emulator (`gnome-terminal`, `kgx`, `kitty`, `alacritty`, `xfce4-terminal`, `konsole`, or `xterm`)

Install dependencies:

```bash
sudo dnf install -y golang git gnome-terminal
```

## Install DevFlow

```bash
git clone https://github.com/anandyadav3559/devflow
cd devflow
go install .
```

Add Go bin to your PATH:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

Optional alias:

```bash
echo 'alias df=devflow' >> ~/.bashrc
source ~/.bashrc
```

## Quick Verification

```bash
devflow --help
devflow build -f workflows/workflow.yml
devflow start -f workflows/workflow.yml
devflow active
```

## YAML Authoring Standard

Create a workflow file like `workflows/myapp.yml`.

### Compulsory (required)

- Top level: `workflow_name`, `services`
- Per service: `command`

### Uncompulsory (optional)

- Top level: `log`, `on_close`
- Per service: `args`, `path`, `depends_on`, `port`, `detached`, `log`, `vars`, `on_close`

### Key fields (brief)

- `command`: executable to run.
- `args`: argument list for `command`.
- `path`: working directory for that command.
- `depends_on`: dependency services for startup ordering.
- `port`: readiness gate before dependents launch.
- `on_close`: cleanup commands run on stop.

## What DevFlow Creates

Primary root:

- `~/.config/devflow/`

Fallback root:

- `~/.devflow/` (if `os.UserConfigDir()` is unavailable)

Under the root it creates:

- `flows/<name>.yml`
- `storage/workflows.json`
- `storage/<workflow>.state.json`
- `<workflow>.pid`
- `<workflow>.sock`
- `logs/devflow-daemon-<workflow>-<timestamp>.log`
- `logs/<workflow>-<timestamp>/<service>.log`

## Quality Gate (Before Commit)

Run:

```bash
./testing/scripts/pre_commit_check.sh
```

Install pre-commit hook:

```bash
./testing/scripts/install_pre_commit_hook.sh
```

This enforces `gofmt`, unit tests, and race tests locally before commit.
