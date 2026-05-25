#!/bin/bash
#
# Build macOS .pkg installer for NiAC
#
# Usage:
#   ./build-pkg.sh [BINARY_PATH] [VERSION]
#
# Examples:
#   ./build-pkg.sh                                    # Use ./niac and auto-detect version
#   ./build-pkg.sh ./niac-darwin-arm64                # Use specified binary
#   ./build-pkg.sh ./niac-darwin-arm64 1.0.0          # Use specified binary and version
#
# Requirements:
#   - Xcode command line tools (pkgbuild, productbuild)
#   - The NiAC binary built for macOS
#

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PKG_ID="com.niac"
PKG_NAME="niac"
INSTALL_LOCATION="/usr/local/niac"
BUILD_DIR="$REPO_ROOT/dist/macos-pkg"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
# shellcheck disable=SC2034  # reserved for future warnings
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

# Find binary
BINARY_PATH=""
if [[ -n "$1" && -f "$1" ]]; then
    BINARY_PATH="$1"
elif [[ -f "$REPO_ROOT/niac-darwin-$(uname -m)" ]]; then
    BINARY_PATH="$REPO_ROOT/niac-darwin-$(uname -m)"
elif [[ -f "$REPO_ROOT/niac" ]]; then
    BINARY_PATH="$REPO_ROOT/niac"
else
    log_error "Cannot find niac binary. Please build first with: make build-darwin"
    exit 1
fi

# Get version
VERSION="${2:-}"
if [[ -z "$VERSION" ]]; then
    VERSION=$("$BINARY_PATH" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "0.0.0")
    VERSION="${VERSION#v}"
fi

# Architecture
ARCH=$(uname -m)
[[ "$ARCH" == "x86_64" ]] && ARCH="amd64"

log_info "Building macOS .pkg installer for NiAC"
echo "  Binary:       $BINARY_PATH"
echo "  Version:      $VERSION"
echo "  Architecture: $ARCH"
echo

# Clean and create build directory
log_step "1/5 Preparing build directory..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION/logs"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION/data"
mkdir -p "$BUILD_DIR/payload/Library/LaunchDaemons"
mkdir -p "$BUILD_DIR/scripts"

# Copy binary
log_step "2/5 Copying binary..."
cp "$BINARY_PATH" "$BUILD_DIR/payload$INSTALL_LOCATION/$PKG_NAME"
chmod 755 "$BUILD_DIR/payload$INSTALL_LOCATION/$PKG_NAME"

# Copy launchd plist to standard location
log_step "3/5 Copying launchd configuration..."
cp "$REPO_ROOT/deploy/launchd/com.niac.plist" "$BUILD_DIR/payload/Library/LaunchDaemons/"

# Create install scripts
log_step "4/5 Preparing installation scripts..."
cat > "$BUILD_DIR/scripts/postinstall" << 'SCRIPT'
#!/bin/bash
set -e

# Create directories
mkdir -p /usr/local/niac/logs
mkdir -p /usr/local/niac/data

# Load the launchd service
launchctl load -w /Library/LaunchDaemons/com.niac.plist 2>/dev/null || true

echo "NiAC installed successfully."
echo "Web UI: http://localhost:8080"
SCRIPT
chmod 755 "$BUILD_DIR/scripts/postinstall"

cat > "$BUILD_DIR/scripts/preinstall" << 'SCRIPT'
#!/bin/bash
# Stop existing service if running
launchctl unload /Library/LaunchDaemons/com.niac.plist 2>/dev/null || true
exit 0
SCRIPT
chmod 755 "$BUILD_DIR/scripts/preinstall"

# Build package
log_step "5/5 Building package..."
PKG_OUTPUT="$REPO_ROOT/dist/niac-${VERSION}-${ARCH}.pkg"
mkdir -p "$(dirname "$PKG_OUTPUT")"

pkgbuild \
    --root "$BUILD_DIR/payload" \
    --identifier "$PKG_ID.pkg" \
    --version "$VERSION" \
    --scripts "$BUILD_DIR/scripts" \
    --install-location "/" \
    "$PKG_OUTPUT"

# Clean up intermediate files
rm -rf "$BUILD_DIR"

echo
log_info "Package built successfully!"
echo "  Output: $PKG_OUTPUT"
echo ""
echo "  To install:"
echo "    sudo installer -pkg $PKG_OUTPUT -target /"
echo ""
echo "  Or double-click the .pkg file in Finder"
echo
