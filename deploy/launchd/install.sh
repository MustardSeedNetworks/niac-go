#!/bin/bash
# =============================================================================
# NiAC - macOS launchd Service Installation
# =============================================================================
set -e

PLIST_SRC="$(cd "$(dirname "$0")" && pwd)/com.niac.plist"
PLIST_DST="/Library/LaunchDaemons/com.niac.plist"
INSTALL_DIR="/usr/local/niac"
LOG_DIR="${INSTALL_DIR}/logs"

# Check for root
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Create installation directory
echo "Creating installation directory..."
mkdir -p "${INSTALL_DIR}"
mkdir -p "${LOG_DIR}"

# Copy binary if provided as argument
if [ -n "$1" ] && [ -f "$1" ]; then
    echo "Installing binary..."
    cp "$1" "${INSTALL_DIR}/niac"
    chmod 755 "${INSTALL_DIR}/niac"
fi

# The operator's attachment policy file: created once, never overwritten, so a
# reinstall keeps what the operator approved. The daemon ignores it unless only
# root can write it.
POLICY_FILE="${INSTALL_DIR}/attachment-policy.conf"
if [ ! -e "${POLICY_FILE}" ]; then
    (
        umask 022
        cat > "${POLICY_FILE}" << 'POLICY'
# Physical attachments the NIAC daemon approves, one INTERFACE=MODE per line
# (direct, access:VLAN or trunk:VLAN,...). Manage it with: sudo niac attachment-policy
POLICY
    )
    chown root:wheel "${POLICY_FILE}"
    chmod 644 "${POLICY_FILE}"
fi

# Install plist
echo "Installing launchd plist..."
cp "${PLIST_SRC}" "${PLIST_DST}"
chmod 644 "${PLIST_DST}"
chown root:wheel "${PLIST_DST}"

# Load the service
echo "Loading service..."
launchctl load -w "${PLIST_DST}"

echo ""
echo "NiAC service installed successfully."
echo ""
echo "Commands:"
echo "  Status:   sudo launchctl list | grep niac"
echo "  Stop:     sudo launchctl unload ${PLIST_DST}"
echo "  Start:    sudo launchctl load -w ${PLIST_DST}"
echo "  Logs:     tail -f ${LOG_DIR}/niac.log"
echo ""
