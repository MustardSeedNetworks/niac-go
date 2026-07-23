#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
WORKFLOW="$REPO_ROOT/.github/workflows/release.yml"
CONTENT_CONFIG="$REPO_ROOT/.goreleaser.content.yml"

job_block() {
  local job=$1
  awk -v job="$job" '
    $0 == "  " job ":" { active = 1 }
    active && $0 ~ /^  [a-z0-9_-]+:$/ && $0 != "  " job ":" { exit }
    active { print }
  ' "$WORKFLOW"
}

require_text() {
  local haystack=$1
  local needle=$2
  local message=$3
  if ! grep -Fq -- "$needle" <<<"$haystack"; then
    echo "$message"
    exit 1
  fi
}

CONTENT_JOB=$(job_block "niac-content")
GORELEASER_JOB=$(job_block "goreleaser")

if grep -Fq "needs: [goreleaser]" <<<"$CONTENT_JOB"; then
  echo "content packages must be built before the core release finalizes integrity metadata"
  exit 1
fi

require_text "$CONTENT_JOB" \
  "goreleaser release --config .goreleaser.content.yml --clean --skip=publish,sign,announce" \
  "content release builds must disable independent publishing and signing"
require_text "$CONTENT_JOB" \
  "name: niac-content-packages" \
  "content packages must cross the job boundary as a workflow artifact"
require_text "$GORELEASER_JOB" \
  "needs: [build-ui, build-windows, niac-content]" \
  "the core release must wait for the content build"
require_text "$GORELEASER_JOB" \
  "name: niac-content-packages" \
  "the core release must download the content packages"
require_text "$GORELEASER_JOB" \
  "Verify content package integrity coverage" \
  "the release must fail closed when content integrity companions are incomplete"
require_text "$GORELEASER_JOB" \
  "grep -Fq \"  \${artifact}.sbom.json\" checksums.txt" \
  "content verification must require each SBOM in the final manifest"
require_text "$GORELEASER_JOB" \
  "test -s \"\${artifact}.sbom.json.cosign.bundle\"" \
  "content verification must require each SBOM signature bundle"
require_text "$GORELEASER_JOB" \
  "artifacts+=(\"\${content_packages[@]}\")" \
  "content packages must enter the integrity-processing artifact list"
require_text "$GORELEASER_JOB" \
  "for artifact in \"\${artifacts[@]}\"; do" \
  "the processed artifact list must also drive the release upload list"
require_text "$GORELEASER_JOB" \
  "cosign sign-blob --yes --bundle \"\${artifact}.sbom.json.cosign.bundle\" \"\${artifact}.sbom.json\"" \
  "generated SBOMs must receive their own keyless signature bundles"
require_text "$GORELEASER_JOB" \
  "sha256sum \"\${artifact}.sbom.json\" >> checksums.txt" \
  "generated SBOMs must enter the final signed manifest and SLSA subject list"
require_text "$GORELEASER_JOB" \
  "uploads+=(\"\${artifact}\" \"\${artifact}.sbom.json\" \"\${artifact}.cosign.bundle\" \"\${artifact}.sbom.json.cosign.bundle\")" \
  "each processed artifact and its integrity companions must enter the upload list"

for command in "syft \"\${artifact}\"" \
  "cosign sign-blob --yes --bundle \"\${artifact}.cosign.bundle\"" \
  "sha256sum \"\${artifact}\" >> checksums.txt"; do
  require_text "$GORELEASER_JOB" "$command" \
    "the final release job is missing required content integrity command: $command"
done

VERIFY_LINE=$(grep -n "Verify content package integrity coverage" "$WORKFLOW" | cut -d: -f1)
HASH_LINE=$(grep -n "Capture artifact hashes for SLSA provenance" "$WORKFLOW" | cut -d: -f1)
if [ -z "$VERIFY_LINE" ] || [ -z "$HASH_LINE" ] || [ "$VERIFY_LINE" -ge "$HASH_LINE" ]; then
  echo "content integrity verification must complete before SLSA subjects are captured"
  exit 1
fi

require_text "$GORELEASER_JOB" \
  "path: \${{ runner.temp }}/niac-content-packages/" \
  "content packages must be downloaded outside the repository checkout"
require_text "$GORELEASER_JOB" \
  "packages=(\"\${RUNNER_TEMP}\"/niac-content-packages/*.deb \"\${RUNNER_TEMP}\"/niac-content-packages/*.rpm)" \
  "the core release must consume content packages from runner temporary storage"

if grep -Fq "mode: append" "$CONTENT_CONFIG"; then
  echo "the content configuration must not retain an append-to-release path"
  exit 1
fi

if ! awk '
  /^release:$/ { release = 1; next }
  release && /^[^ ]/ { exit }
  release && $0 == "  disable: true" { found = 1 }
  END { exit(found ? 0 : 1) }
' "$CONTENT_CONFIG"; then
  echo "the content GoReleaser configuration must disable SCM publication"
  exit 1
fi
