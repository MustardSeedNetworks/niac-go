#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
WORKFLOW="$REPO_ROOT/.github/workflows/release.yml"
NPCAP_CACHE_WORKFLOW="$REPO_ROOT/.github/workflows/cache-npcap-sdk.yml"
CONTENT_CONFIG="$REPO_ROOT/.goreleaser.content.yml"
CORE_CONFIG="$REPO_ROOT/.goreleaser.yml"
RELEASE_PLEASE_CONFIG="$REPO_ROOT/.github/release-please-config.json"
POSTINSTALL="$REPO_ROOT/deploy/nfpm/postinstall.sh"
POSTREMOVE="$REPO_ROOT/deploy/nfpm/postremove.sh"
WINDOWS_INSTALLER="$REPO_ROOT/deploy/windows/install.ps1"

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
NPCAP_JOB=$(job_block "prepare-npcap-sdk")
WINDOWS_JOB=$(job_block "build-windows")
NPCAP_CACHE_WORKFLOW_CONTENT=$(<"$NPCAP_CACHE_WORKFLOW")

require_text "$NPCAP_JOB" \
  'key: npcap-sdk-${{ env.NPCAP_SDK_VERSION }}-${{ env.NPCAP_SDK_SHA256 }}' \
  "the Npcap SDK cache key must include both the pinned version and checksum"
require_text "$NPCAP_JOB" \
  "if: steps.npcap-cache.outputs.cache-hit != 'true'" \
  "the official Npcap origin must only be contacted on a cache miss"
require_text "$NPCAP_JOB" \
  "Npcap SDK cache miss and the official origin could not be reached" \
  "an Npcap SDK cache miss must fail with a clear origin diagnostic"
require_text "$NPCAP_JOB" \
  'echo "${NPCAP_SDK_SHA256}  ${archive}" | sha256sum -c -' \
  "the prepared Npcap SDK must be verified before caching"
require_text "$NPCAP_JOB" \
  "uses: actions/cache/save@" \
  "the verified Npcap SDK must be saved to the internal build cache"
require_text "$NPCAP_JOB" \
  "enableCrossOsArchive: true" \
  "the verified Npcap SDK cache must be readable by Windows builders"
require_text "$NPCAP_JOB" \
  'path: npcap-cache/npcap-sdk-${{ env.NPCAP_SDK_VERSION }}.zip' \
  "the Npcap SDK cache must use an OS-neutral relative path"
require_text "$WINDOWS_JOB" \
  "needs: [build-ui, prepare-npcap-sdk]" \
  "both Windows builders must wait for the verified Npcap SDK cache"
require_text "$WINDOWS_JOB" \
  "uses: actions/cache/restore@" \
  "Windows builders must restore the verified Npcap SDK cache"
require_text "$WINDOWS_JOB" \
  "Verified Npcap SDK cache was not available to the Windows builder" \
  "Windows builders must fail clearly when the verified cache is unavailable"
require_text "$WINDOWS_JOB" \
  'echo "${NPCAP_SDK_SHA256}  ${archive}" | sha256sum -c -' \
  "Windows builders must independently verify the Npcap SDK cache"
if grep -Fq "curl " <<<"$WINDOWS_JOB"; then
  echo "Windows builders must not contact the Npcap SDK origin directly"
  exit 1
fi
if grep -Fq 'runner.temp }}/npcap-sdk' <<<"$NPCAP_JOB$WINDOWS_JOB"; then
  echo "cross-OS cache paths must not depend on the runner-specific temporary directory"
  exit 1
fi
if grep -Fq "actions/upload-artifact@" <<<"$NPCAP_JOB"; then
  echo "the Npcap SDK must not be published as a downloadable workflow artifact"
  exit 1
fi
require_text "$NPCAP_CACHE_WORKFLOW_CONTENT" \
  'branches:' \
  "the persistent Npcap SDK cache must be seeded from the default branch"
require_text "$NPCAP_CACHE_WORKFLOW_CONTENT" \
  '      - main' \
  "the persistent Npcap SDK cache must be accessible to tagged releases"
require_text "$NPCAP_CACHE_WORKFLOW_CONTENT" \
  '      - ".github/workflows/release.yml"' \
  "changing the release workflow must refresh the default-branch Npcap SDK cache"
require_text "$NPCAP_CACHE_WORKFLOW_CONTENT" \
  'echo "${NPCAP_SDK_SHA256}  ${archive}" | sha256sum -c -' \
  "the default-branch Npcap SDK cache must be checksum verified"
if grep -Fq "actions/upload-artifact@" <<<"$NPCAP_CACHE_WORKFLOW_CONTENT"; then
  echo "the default-branch cache workflow must not publish the Npcap SDK as an artifact"
  exit 1
fi

RELEASE_NPCAP_VERSION=$(sed -n 's/^  NPCAP_SDK_VERSION: "\(.*\)"/\1/p' "$WORKFLOW")
CACHE_NPCAP_VERSION=$(sed -n 's/^  NPCAP_SDK_VERSION: "\(.*\)"/\1/p' "$NPCAP_CACHE_WORKFLOW")
RELEASE_NPCAP_SHA=$(sed -n 's/^  NPCAP_SDK_SHA256: "\(.*\)"/\1/p' "$WORKFLOW")
CACHE_NPCAP_SHA=$(sed -n 's/^  NPCAP_SDK_SHA256: "\(.*\)"/\1/p' "$NPCAP_CACHE_WORKFLOW")
if [ "$RELEASE_NPCAP_VERSION" != "$CACHE_NPCAP_VERSION" ] ||
  [ "$RELEASE_NPCAP_SHA" != "$CACHE_NPCAP_SHA" ]; then
  echo "release and default-branch cache workflows must use the same Npcap SDK pin"
  exit 1
fi

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
  'cp deploy/windows/install.ps1 "${stage}/install.ps1"' \
  "Windows archives must contain the dedicated installer"
if grep -Fq 'cp deploy/windows/build.ps1 "${stage}/install.ps1"' <<<"$GORELEASER_JOB"; then
  echo "Windows archives must not rename the distribution builder as install.ps1"
  exit 1
fi
require_text "$(<"$WINDOWS_INSTALLER")" \
  '$SourceBinary = Join-Path $ScriptDir $BinaryName' \
  "the Windows installer must resolve niac.exe relative to its own archive location"
require_text "$(<"$WINDOWS_INSTALLER")" \
  '[Security.Principal.WindowsBuiltInRole]::Administrator' \
  "the Windows installer must require an elevated PowerShell session"
require_text "$(<"$WINDOWS_INSTALLER")" \
  'service install' \
  "the Windows installer must install the NIAC service"
require_text "$(<"$WINDOWS_INSTALLER")" \
  'if ($LASTEXITCODE -ne 0)' \
  "the Windows installer must fail when native service commands fail"
require_text "$(<"$WINDOWS_INSTALLER")" \
  'Get-Service -Name $ServiceName' \
  "the Windows installer must detect an existing service before replacing the binary"
require_text "$(<"$WINDOWS_INSTALLER")" \
  'Stop-Service -Name $ServiceName' \
  "the Windows installer must stop an existing service before replacing the binary"
require_text "$(<"$WINDOWS_INSTALLER")" \
  '& sc.exe config $ServiceName "binPath=" "`"$InstalledBinary`" service run"' \
  "the Windows installer must rebind an existing service to the installed binary"
require_text "$(<"$WINDOWS_INSTALLER")" \
  'Get-NetFirewallRule -DisplayName "NiAC Simulator"' \
  "the Windows installer must preserve a single firewall rule across upgrades"
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

require_text "$(<"$CORE_CONFIG")" \
  "draft: true" \
  "GoReleaser must keep releases draft until every integrity stage succeeds"
require_text "$(<"$CORE_CONFIG")" \
  "use_existing_draft: true" \
  "GoReleaser must upload into the draft created by Release Please"
require_text "$(<"$RELEASE_PLEASE_CONFIG")" \
  '"draft": true' \
  "Release Please must create an unpublished release"
require_text "$(<"$RELEASE_PLEASE_CONFIG")" \
  '"force-tag-creation": true' \
  "Release Please must create the tag that triggers artifact builds"

PROVENANCE_JOB=$(job_block "provenance")
require_text "$PROVENANCE_JOB" \
  "Publish completed release" \
  "provenance must publish the release only after attaching its bundle"
require_text "$PROVENANCE_JOB" \
  'gh release edit "$TARGET_TAG" --draft=false --repo "${GITHUB_REPOSITORY}"' \
  "the final release step must publish the completed draft"
ATTACH_LINE=$(grep -n "Attach attestation bundle to release" "$WORKFLOW" | cut -d: -f1)
PUBLISH_LINE=$(grep -n "Publish completed release" "$WORKFLOW" | cut -d: -f1)
if [ -z "$ATTACH_LINE" ] || [ -z "$PUBLISH_LINE" ] || [ "$ATTACH_LINE" -ge "$PUBLISH_LINE" ]; then
  echo "the release must remain draft until provenance attachment completes"
  exit 1
fi

for marker in firewall-ufw-owned firewall-firewalld-owned; do
  require_text "$(<"$POSTINSTALL")" "$marker" \
    "postinstall must record firewall rule ownership with $marker"
  require_text "$(<"$POSTREMOVE")" "$marker" \
    "postremove must require firewall rule ownership marker $marker"
done
require_text "$(<"$POSTINSTALL")" \
  'PACKAGE_STATE_DIR=/var/lib/niac-package' \
  "firewall ownership state must remain outside daemon-writable storage"
require_text "$(<"$POSTINSTALL")" \
  'firewall-cmd --permanent --query-port=${PORT}/tcp' \
  "postinstall must not claim a pre-existing firewalld rule"
require_text "$(<"$POSTINSTALL")" \
  'grep -Eq "^${PORT}/tcp[[:space:]]+ALLOW"' \
  "postinstall must not claim a pre-existing ufw rule"
require_text "$(<"$POSTREMOVE")" \
  '[ -f "$UFW_MARKER" ]' \
  "postremove must gate ufw deletion on package ownership"
require_text "$(<"$POSTREMOVE")" \
  '[ -f "$FIREWALLD_MARKER" ]' \
  "postremove must gate firewalld deletion on package ownership"
require_text "$(<"$POSTREMOVE")" \
  'firewall-offline-cmd --remove-port=8445/tcp' \
  "postremove must delete package-owned permanent rules while firewalld is stopped"

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
