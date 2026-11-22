#!/bin/sh
#
# Run the niac daemon the CI E2E suite and the phone-width gate test against,
# listening on 127.0.0.1:PORT. Stays in the foreground (it execs niac), so a
# caller backgrounds it and owns its lifetime.
#
#   scripts/e2e-daemon.sh PORT
#
# NIAC_E2E_DRY_RUN_SIMULATION runs simulations against the synthetic
# e2e-dry-run0 interface without binding a real capture engine, and
# --attachment-policy is what lets an authoring spec reach *start*: a binding
# with no approving policy fails preflight with attachment_policy_denied, by
# design. ui/playwright.config.ts starts the same daemon for a local run and
# must keep the same flags, or specs pass locally and fail in CI.
#
# The certificate directory is a fresh temporary one, not the default
# ./certs: the daemon refuses a directory whose ancestors belong to another
# account, and a CI container runs as a different user than owns its workspace.
set -eu

if [ "$#" -ne 1 ]; then
  printf 'usage: %s PORT\n' "$0" >&2
  exit 2
fi
repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cert_dir=$(mktemp -d)

NIAC_E2E_DRY_RUN_SIMULATION=1 \
  exec "$repo_dir/niac" daemon --listen "127.0.0.1:$1" \
  --cert-dir "$cert_dir" \
  --attachment-policy e2e-dry-run0=access:200
