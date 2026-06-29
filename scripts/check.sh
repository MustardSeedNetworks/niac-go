#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

ruff format --check scripts
ruff check scripts
go test ./...
