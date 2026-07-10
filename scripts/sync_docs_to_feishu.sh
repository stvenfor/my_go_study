#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v lark-cli >/dev/null 2>&1; then
  echo "Error: lark-cli not found. Install: https://github.com/larksuite/cli#installation" >&2
  exit 1
fi

PYTHON_BIN="${PYTHON_BIN:-python3}"
REQ_FILE="scripts/feishu_doc_sync/requirements.txt"

if ! "$PYTHON_BIN" -c "import yaml" >/dev/null 2>&1; then
  "$PYTHON_BIN" -m pip install -r "$REQ_FILE"
fi

CMD="${1:-sync}"
shift || true

case "$CMD" in
  bootstrap|sync)
    exec "$PYTHON_BIN" scripts/feishu_doc_sync/main.py "$CMD" "$@"
    ;;
  dry-run)
    exec "$PYTHON_BIN" scripts/feishu_doc_sync/main.py sync --dry-run "$@"
    ;;
  *)
    echo "Usage: $0 [bootstrap|sync|dry-run] [--file path/to/file.md]" >&2
    exit 1
    ;;
esac
