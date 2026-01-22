#!/bin/bash
# =============================================================================
# NiAC - macOS launchd Service Uninstallation
# =============================================================================
set -e

PLIST_DST="/Library/LaunchDaemons/com.niac.plist"
INSTALL_DIR="/usr/local/niac"

# Check for root
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Unload the service
if [ -f "${PLIST_DST}" ]; then
    echo "Unloading service..."
    launchctl unload "${PLIST_DST}" 2>/dev/null || true
    rm -f "${PLIST_DST}"
    echo "Service plist removed."
else
    echo "Service plist not found at ${PLIST_DST}"
fi

# Ask before removing files
echo ""
read -p "Remove installation directory ${INSTALL_DIR}? [y/N] " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "${INSTALL_DIR}"
    echo "Installation directory removed."
else
    echo "Installation directory preserved."
fi

echo ""
echo "NiAC service uninstalled."
