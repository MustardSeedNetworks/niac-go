#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/sync-demo-catalog.sh --sync|--check

Generates the Go examples/ layout from MustardSeedNetworks/niac-demo-catalog.

Environment:
  NIAC_DEMO_CATALOG_URL      Catalog git URL.
  NIAC_DEMO_CATALOG_REF      Catalog branch, tag, or commit. Defaults to main.
  NIAC_DEMO_CATALOG_DIR      Existing catalog checkout. Defaults to .catalog/niac-demo-catalog.
  NIAC_GO_EXAMPLES_DIR       Generated output path. Defaults to examples.
  NIAC_DEMO_CATALOG_OFFLINE  Set to 1 to require NIAC_DEMO_CATALOG_DIR and skip git fetch.
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

copy_dir() {
  local source="$1"
  local destination="$2"

  if [ ! -d "$source" ]; then
    echo "ERROR: expected catalog directory missing: $source" >&2
    exit 1
  fi

  mkdir -p "$destination"
  rsync -a "$source"/ "$destination"/
}

require_command diff
require_command git
require_command mktemp
require_command rsync

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CATALOG_URL="${NIAC_DEMO_CATALOG_URL:-git@github.com:MustardSeedNetworks/niac-demo-catalog.git}"
CATALOG_REF="${NIAC_DEMO_CATALOG_REF:-main}"
CATALOG_DIR="${NIAC_DEMO_CATALOG_DIR:-$PROJECT_ROOT/.catalog/niac-demo-catalog}"
EXAMPLES_DIR="${NIAC_GO_EXAMPLES_DIR:-$PROJECT_ROOT/examples}"
OFFLINE="${NIAC_DEMO_CATALOG_OFFLINE:-0}"

if [ "$OFFLINE" = "1" ]; then
  if [ ! -d "$CATALOG_DIR" ]; then
    echo "ERROR: NIAC_DEMO_CATALOG_OFFLINE=1 but catalog directory does not exist: $CATALOG_DIR" >&2
    exit 1
  fi
else
  if [ -d "$CATALOG_DIR/.git" ]; then
    git -C "$CATALOG_DIR" fetch --depth 1 origin "$CATALOG_REF"
    git -C "$CATALOG_DIR" checkout --detach FETCH_HEAD
  else
    mkdir -p "$(dirname "$CATALOG_DIR")"
    git clone --depth 1 --branch "$CATALOG_REF" "$CATALOG_URL" "$CATALOG_DIR"
  fi
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

copy_dir "$CATALOG_DIR/scenarios/go-yaml" "$STAGE"
copy_dir "$CATALOG_DIR/walks/raw" "$STAGE/device_walks"
copy_dir "$CATALOG_DIR/walks/sanitized" "$STAGE/device_walks_sanitized"
copy_dir "$CATALOG_DIR/captures/shared" "$STAGE/captures"
copy_dir "$CATALOG_DIR/captures/go-extra" "$STAGE/pcaps"
copy_dir "$CATALOG_DIR/tools/walk-scripts/go" "$STAGE/walk_scripts"
if [ -f "$CATALOG_DIR/tools/walk-scripts/java/run_demo.sh" ]; then
  cp "$CATALOG_DIR/tools/walk-scripts/java/run_demo.sh" "$STAGE/walk_scripts/run_demo.sh"
fi
copy_dir "$CATALOG_DIR/docs/imported/go-examples" "$STAGE"

case "$MODE" in
  sync)
    mkdir -p "$EXAMPLES_DIR"
    rsync -a --delete "$STAGE"/ "$EXAMPLES_DIR"/
    echo "OK: generated $EXAMPLES_DIR from the shared demo catalog."
    ;;
  check)
    if ! diff -qr "$STAGE" "$EXAMPLES_DIR"; then
      echo "ERROR: $EXAMPLES_DIR does not match the shared demo catalog. Run scripts/sync-demo-catalog.sh --sync." >&2
      exit 1
    fi
    echo "OK: $EXAMPLES_DIR matches the shared demo catalog."
    ;;
esac
