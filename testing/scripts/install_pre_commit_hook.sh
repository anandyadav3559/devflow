#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOOK_PATH="${ROOT_DIR}/.git/hooks/pre-commit"

if [[ ! -d "${ROOT_DIR}/.git" ]]; then
  echo "Not a git repository: ${ROOT_DIR}"
  exit 1
fi

cat > "${HOOK_PATH}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
"${ROOT_DIR}/testing/scripts/pre_commit_check.sh"
EOF

chmod +x "${HOOK_PATH}"
echo "Installed pre-commit hook at ${HOOK_PATH}"
