# DevFlow on Linux (Non-Fedora)

[![Test](https://github.com/anandyadav3559/devflow/actions/workflows/test.yml/badge.svg)](https://github.com/anandyadav3559/devflow/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](../LICENSE)

Setup guide for Ubuntu, Debian, Arch, and other Linux distributions.

## Prerequisites

- Go `1.25+`
- Git
- One supported terminal emulator (`gnome-terminal`, `kgx`, `kitty`, `alacritty`, `xfce4-terminal`, `konsole`, or `xterm`)

## Install Dependencies

Ubuntu / Debian:

```bash
sudo apt update
sudo apt install -y golang-go git gnome-terminal
```

Arch:

```bash
sudo pacman -Sy --needed go git gnome-terminal
```

## Install DevFlow

```bash
git clone https://github.com/anandyadav3559/devflow
cd devflow
go install .
```

Add Go bin to PATH:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

Optional alias:

```bash
echo 'alias df=devflow' >> ~/.bashrc
source ~/.bashrc
```

If you use Zsh, write these to `~/.zshrc` instead.

## Quick Verification

```bash
devflow --help
devflow build -f workflows/workflow.yml
devflow start -f workflows/workflow.yml
devflow active
```

## YAML Authoring Standard

Create workflow file example: `workflows/myapp.yml`.

### Compulsory (required)

- Top level: `workflow_name`, `services`
- Per service: `command`

### Uncompulsory (optional)

- Top level: `log`, `on_close`
- Per service: `args`, `path`, `depends_on`, `port`, `detached`, `log`, `vars`, `on_close`

### Key fields (brief)

- `command`: process/binary to execute.
- `args`: argument array for process.
- `path`: working directory for process.
- `depends_on`: startup dependency graph.
- `port`: readiness check target.
- `on_close`: cleanup command(s) for controlled shutdown.

## What DevFlow Creates

Primary root:

- `~/.config/devflow/`

Fallback root:

- `~/.devflow/` (if `os.UserConfigDir()` is unavailable)

Generated files/folders include:

- `flows/<name>.yml`
- `storage/workflows.json`
- `storage/<workflow>.state.json`
- `<workflow>.pid`
- `<workflow>.sock`
- `logs/devflow-daemon-<workflow>-<timestamp>.log`
- `logs/<workflow>-<timestamp>/<service>.log`

## Quality Gate (Before Commit)

```bash
./testing/scripts/pre_commit_check.sh
```

Install automatic pre-commit checks:

```bash
./testing/scripts/install_pre_commit_hook.sh
```

This enforces formatting, unit tests, and race tests before commits.
