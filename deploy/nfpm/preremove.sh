#!/bin/sh
# Pre-remove: stop and disable the systemd service. Runs both on uninstall and
# before an upgrade swaps the binary, which is fine — postinstall restarts.
set -e

if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet niac.service 2>/dev/null; then
        systemctl stop niac.service || true
    fi
    if systemctl is-enabled --quiet niac.service 2>/dev/null; then
        systemctl disable niac.service || true
    fi
fi

exit 0
