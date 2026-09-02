#!/usr/bin/env bash
# test-storybook-contract.sh — prove the Storybook suite can actually fail.
#
# A gate nobody has watched go red is indistinguishable from a gate that stopped
# running. This plants a real defect of each kind the suite is supposed to catch
# and asserts the run fails FOR THAT REASON, not merely that it fails.
#
# The defects live in src/test/storybook/StorybookGate.stories.tsx, switched on
# by VITE_STORYBOOK_INJECT_DEFECT so nothing broken is ever committed.

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root/ui"

if [ -z "$(node -p 'require("./package.json").scripts["test:storybook:run"] ?? ""')" ]; then
  echo 'Storybook runner is missing' >&2
  exit 1
fi

verify_defect() {
  local defect=$1
  local expected=$2
  local log_file
  log_file=$(mktemp)

  if VITE_STORYBOOK_INJECT_DEFECT="$defect" npm run test:storybook:run >"$log_file" 2>&1; then
    echo "Injected $defect defect did not fail the Storybook suite" >&2
    rm -f "$log_file"
    return 1
  fi

  # Failing is not enough — it has to fail for the right reason, or an unrelated
  # breakage would be read as the gate working.
  if ! grep -Eqi "$expected" "$log_file"; then
    echo "Injected $defect defect failed, but not for the expected reason" >&2
    cat "$log_file" >&2
    rm -f "$log_file"
    return 1
  fi

  echo "✓ injected $defect defect was caught"
  rm -f "$log_file"
}

verify_defect interaction 'SharedComponentInteraction|Shared Component Interaction'
verify_defect accessibility 'button-name|discernible text'

echo "✓ Storybook contract holds: both injected defects fail the suite"
