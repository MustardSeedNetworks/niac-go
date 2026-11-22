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

# cp and pkgbuild otherwise carry extended attributes into the payload as
# AppleDouble ._ files, which the install then drops into /usr/local/niac
# (#2652).
export COPYFILE_DISABLE=1

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
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
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

# Architecture of the binary, not of the build host: the release builds the
# arm64 package from the darwin archive on whatever runner it gets.
ARCH=$(lipo -archs "$BINARY_PATH")
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

# Sign the daemon. Notarization rejects a package containing an unsigned
# executable, and the three things it checks are all set here: a Developer ID
# signature, the hardened runtime, and a secure timestamp. Skipped only when no
# identity is present, so a local dev build still works; such a package cannot
# be notarized, which is stated rather than discovered at submission.
SIGN_IDENTITY="${NIAC_SIGN_IDENTITY:-Developer ID Application}"
if security find-identity -v -p codesigning | grep -q "$SIGN_IDENTITY"; then
    codesign --force --timestamp --options runtime \
        --sign "$SIGN_IDENTITY" \
        "$BUILD_DIR/payload$INSTALL_LOCATION/$PKG_NAME"
    codesign --verify --strict "$BUILD_DIR/payload$INSTALL_LOCATION/$PKG_NAME"
else
    log_warn "No '$SIGN_IDENTITY' identity: shipping an unsigned daemon; this package cannot be notarized."
fi

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

# The operator's attachment policy file: created once, root-owned, and never
# touched again so an upgrade keeps what the operator approved. The daemon
# ignores it unless only root can write it.
POLICY_FILE=/usr/local/niac/attachment-policy.conf
if [ ! -e "$POLICY_FILE" ]; then
    (
        umask 022
        cat > "$POLICY_FILE" << 'POLICY'
# Physical attachments the NIAC daemon approves, one INTERFACE=MODE per line
# (direct, access:VLAN or trunk:VLAN,...). Manage it with: sudo niac attachment-policy
POLICY
    )
    chown root:wheel "$POLICY_FILE"
    chmod 644 "$POLICY_FILE"
fi

# Load the launchd service
launchctl load -w /Library/LaunchDaemons/com.niac.plist 2>/dev/null || true

echo "NiAC installed successfully."
echo "Web UI: https://localhost:8445"
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
PKG_OUTPUT="$REPO_ROOT/dist/niac-${VERSION}-darwin-${ARCH}.pkg"
mkdir -p "$(dirname "$PKG_OUTPUT")"

# A Developer ID *Installer* certificate is a different certificate from the
# Application one used on the binary. Signing here, rather than with a later
# productsign, keeps the package signed from the moment it exists.
SIGN_PKG=()
if [[ -n "${NIAC_INSTALLER_IDENTITY:-}" ]]; then
    SIGN_PKG=(--sign "${NIAC_INSTALLER_IDENTITY}" --timestamp)
fi

pkgbuild \
    --root "$BUILD_DIR/payload" \
    --identifier "$PKG_ID.pkg" \
    --version "$VERSION" \
    --scripts "$BUILD_DIR/scripts" \
    --install-location "/" \
    "${SIGN_PKG[@]}" \
    "$PKG_OUTPUT"

# Clean up intermediate files
rm -rf "$BUILD_DIR"

if pkgutil --payload-files "$PKG_OUTPUT" | grep '/\._'; then
    log_error "Payload contains AppleDouble ._ files (listed above)"
    exit 1
fi

echo
log_info "Package built successfully!"
echo "  Output: $PKG_OUTPUT"
echo ""
echo "  To install, double-click the .pkg file in Finder, or:"
echo "    sudo installer -pkg $PKG_OUTPUT -target /"
echo
