#!/usr/bin/env bash
#
# Build ./niac for the E2E suite and the phone-width gate, from the UI already
# built into internal/api/ui.
#
# Why this exists rather than `make build`: `make build` rebuilds the UI, which
# both callers have already done (the E2E job downloads the build-ui artifact;
# the phone-width gate's reusable workflow in MustardSeedNetworks/.github
# builds it itself). gopacket/pcap needs cgo and libpcap headers, so unlike
# seed's and stem's E2E binaries this one cannot be built CGO-free.
#
# The ldflags MUST mirror the Makefile's GO_LDFLAGS. Per the universal build
# contract, a binary built without them reports "unknown" from /__version.
set -euo pipefail

cd "$(dirname "$0")/.."

# The phone-width gate runs on a bare runner with no libpcap. The E2E job
# installs it first through the apt-install composite, which bounds and
# retries a flaky mirror; a reusable workflow's caller cannot add that step, so
# this fallback is a plain install and a mirror outage fails the gate by name.
if [ "$(uname -s)" = Linux ] && [ ! -e /usr/include/pcap/pcap.h ]; then
  sudo apt-get update -qq
  sudo apt-get install -y -qq --no-install-recommends libpcap-dev
fi

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION_PKG="github.com/MustardSeedNetworks/niac-go/internal/version"

if [ -z "$(find internal/api/ui -type f ! -name .gitkeep -print -quit)" ]; then
  echo "::error::internal/api/ui is empty — build the UI first; refusing to build a UI-less binary" >&2
  exit 1
fi
UI_BUILD_HASH=$(find internal/api/ui -type f -exec md5sum {} \; | sort | md5sum | cut -d' ' -f1)

echo "building e2e backend: version=${VERSION} commit=${COMMIT} uiBuildHash=${UI_BUILD_HASH}"

CGO_ENABLED=1 go build -trimpath -buildvcs=false \
  -ldflags "-s -w \
    -X ${VERSION_PKG}.Version=${VERSION} \
    -X ${VERSION_PKG}.Commit=${COMMIT} \
    -X ${VERSION_PKG}.BuildTime=${BUILD_TIME} \
    -X ${VERSION_PKG}.UIBuildHash=${UI_BUILD_HASH}" \
  -o niac ./cmd/niac
