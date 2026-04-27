# Testing & Pre-Commit Automation

This folder contains scripts to run rigorous local checks before pushing or committing.

## Run checks manually

```bash
./testing/scripts/pre_commit_check.sh
```

What it runs:

- `gofmt` validation on tracked Go files (excluding `vendor/`)
- `go test ./...`
- `go test -race ./...`

## Install git pre-commit hook

```bash
./testing/scripts/install_pre_commit_hook.sh
```

After installation, every `git commit` automatically runs the same checks and blocks the commit on failure.

CI parity: `.github/workflows/test.yml` runs the same quality gate on push/PR.
