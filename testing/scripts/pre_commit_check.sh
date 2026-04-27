#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

echo "[devflow] running gofmt check..."
GO_FILES="$(git ls-files '*.go' ':!:vendor/**')"
UNFORMATTED="$(gofmt -l ${GO_FILES})"
if [[ -n "${UNFORMATTED}" ]]; then
  echo "The following files are not gofmt-formatted:"
  echo "${UNFORMATTED}"
  echo "Run: gofmt -w ."
  exit 1
fi

echo "[devflow] running unit tests..."
go test ./...

echo "[devflow] running race tests..."
go test -race ./...

echo "[devflow] pre-commit checks passed."
