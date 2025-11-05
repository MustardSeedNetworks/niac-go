#!/bin/bash
set -e

VERSION="1.4.0"
BUILD_DATE=$(date -u +"%Y-%m-%d")
GIT_COMMIT=$(git rev-parse --short HEAD)
LDFLAGS="-X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE} -X main.GitCommit=${GIT_COMMIT}"

echo "Building NIAC-Go v${VERSION} releases..."
echo "Build date: ${BUILD_DATE}"
echo "Git commit: ${GIT_COMMIT}"
echo ""

# Create releases directory
mkdir -p releases/v${VERSION}

# Build niac for current platform only (requires libpcap)
echo "Building niac for native platform (darwin/arm64)..."
go build -ldflags="$LDFLAGS" -o "releases/v${VERSION}/niac-${VERSION}-darwin-arm64" ./cmd/niac
echo "  ✓ niac-${VERSION}-darwin-arm64"

echo ""
echo "Building niac-convert for all platforms (no CGO required)..."

# Build converter for each platform (no libpcap dependency)
platforms=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${platforms[@]}"; do
  IFS='/' read -r GOOS GOARCH <<< "$platform"
  converter_name="niac-convert-${VERSION}-${GOOS}-${GOARCH}"
  
  if [ "$GOOS" = "windows" ]; then
    converter_name="${converter_name}.exe"
  fi
  
  echo "Building converter for ${GOOS}/${GOARCH}..."
  GOOS=$GOOS GOARCH=$GOARCH go build -o "releases/v${VERSION}/${converter_name}" ./cmd/niac-convert
  echo "  ✓ ${converter_name}"
done

echo ""
echo "Build complete! Releases in releases/v${VERSION}/"
echo ""
echo "Note: niac binary requires libpcap and must be built on the target platform."
echo "      Use Docker or native builds for Linux/Windows releases."
echo ""
ls -lh releases/v${VERSION}/
