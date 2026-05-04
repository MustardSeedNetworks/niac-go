#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

ruff check scripts
if command -v black >/dev/null 2>&1; then
  black --check scripts
elif command -v uv >/dev/null 2>&1; then
  uvx black --check scripts
else
  echo "black not found; install black or uv to run Python formatting checks" >&2
  exit 1
fi
go test ./...
