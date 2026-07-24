#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/sync-demo-catalog.sh --sync|--check

Generates and validates the Go examples/ layout from MustardSeedNetworks/niac-demo-catalog.

Environment:
  NIAC_DEMO_CATALOG_URL      Catalog git URL.
  NIAC_DEMO_CATALOG_REF      Catalog branch, tag, or commit. Defaults to main.
  NIAC_DEMO_CATALOG_DIR      Existing catalog checkout. Defaults to .catalog/niac-demo-catalog.
  NIAC_GO_EXAMPLES_DIR       Generated output path. Defaults to examples.
  NIAC_DEMO_CATALOG_OFFLINE  Set to 1 to require a clean local catalog checkout and skip git fetch.
USAGE
}

if [ "$#" -ne 1 ]; then
  usage
  exit 2
fi

case "$1" in
  --sync) MODE="sync" ;;
  --check) MODE="check" ;;
  -h|--help)
    usage
    exit 0
    ;;
  *)
    usage
    exit 2
    ;;
esac

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: missing required command: $1" >&2
    exit 1
  fi
}

require_command git
require_command go

is_git_checkout() {
  git -C "$1" rev-parse --is-inside-work-tree >/dev/null 2>&1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CATALOG_URL="${NIAC_DEMO_CATALOG_URL:-git@github.com:MustardSeedNetworks/niac-demo-catalog.git}"
CATALOG_REF="${NIAC_DEMO_CATALOG_REF:-main}"
CATALOG_DIR="${NIAC_DEMO_CATALOG_DIR:-$PROJECT_ROOT/.catalog/niac-demo-catalog}"
EXAMPLES_DIR="${NIAC_GO_EXAMPLES_DIR:-$PROJECT_ROOT/examples}"
OFFLINE="${NIAC_DEMO_CATALOG_OFFLINE:-0}"

if [ "$OFFLINE" = "1" ]; then
  if ! is_git_checkout "$CATALOG_DIR"; then
    echo "ERROR: offline catalog must be a git checkout: $CATALOG_DIR" >&2
    exit 1
  fi
else
  if is_git_checkout "$CATALOG_DIR"; then
    git -C "$CATALOG_DIR" fetch --depth 1 origin "$CATALOG_REF"
  else
    mkdir -p "$(dirname "$CATALOG_DIR")"
    git clone --filter=blob:none --no-checkout "$CATALOG_URL" "$CATALOG_DIR"
    git -C "$CATALOG_DIR" fetch --depth 1 origin "$CATALOG_REF"
  fi
  git -C "$CATALOG_DIR" checkout --detach FETCH_HEAD
fi

if [ -n "$(git -C "$CATALOG_DIR" status --porcelain --untracked-files=normal)" ]; then
  echo "ERROR: catalog checkout has uncommitted content: $CATALOG_DIR" >&2
  exit 1
fi

SOURCE_COMMIT="$(git -C "$CATALOG_DIR" rev-parse HEAD)"
SOURCE_URL="$(git -C "$CATALOG_DIR" remote get-url origin)"
(
  cd "$PROJECT_ROOT"
  go run ./cmd/niac-catalog-sync \
    -mode "$MODE" \
    -catalog-dir "$CATALOG_DIR" \
    -examples-dir "$EXAMPLES_DIR" \
    -repository "$SOURCE_URL" \
    -commit "$SOURCE_COMMIT"
)
