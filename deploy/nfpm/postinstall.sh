#!/bin/sh
set -e

BINARY=/usr/bin/niac

if command -v setcap >/dev/null 2>&1; then
    setcap 'cap_net_raw,cap_net_admin=+ep' "$BINARY" || \
        echo "warning: could not set capabilities on $BINARY"
else
    echo "warning: setcap not found; install libcap/libcap2-bin for non-root operation"
fi

PORT=8445
PACKAGE_STATE_DIR=/var/lib/niac-package
UFW_MARKER="$PACKAGE_STATE_DIR/firewall-ufw-owned"
FIREWALLD_MARKER="$PACKAGE_STATE_DIR/firewall-firewalld-owned"

record_firewall_ownership() {
    mkdir -p "$PACKAGE_STATE_DIR"
    chmod 0700 "$PACKAGE_STATE_DIR"
    touch "$1"
}

open_firewall() {
    opened=""
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
        if ! ufw status 2>/dev/null | grep -Eq "^${PORT}/tcp[[:space:]]+ALLOW" && \
            ufw allow ${PORT}/tcp comment 'NIAC WebUI HTTPS' >/dev/null 2>&1; then
            record_firewall_ownership "$UFW_MARKER"
            opened="ufw"
        fi
    fi
    if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        if ! firewall-cmd --permanent --query-port=${PORT}/tcp >/dev/null 2>&1; then
            if firewall-cmd --permanent --add-port=${PORT}/tcp >/dev/null 2>&1; then
                firewall-cmd --reload >/dev/null 2>&1 || true
                record_firewall_ownership "$FIREWALLD_MARKER"
                opened="${opened:+$opened, }firewalld"
            fi
        fi
    fi
    if [ -n "$opened" ]; then
        echo "Opened TCP ${PORT} on: $opened (NIAC_OPEN_FIREWALL set)"
    fi
}

firewall_hint() {
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
        echo "  sudo ufw allow ${PORT}/tcp"
    elif command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        echo "  sudo firewall-cmd --permanent --add-port=${PORT}/tcp && sudo firewall-cmd --reload"
    fi
}

# Network exposure is the operator's choice. We do NOT open the firewall by
# default; set NIAC_OPEN_FIREWALL=1 to opt in (e.g. automated provisioning).
case "${NIAC_OPEN_FIREWALL:-0}" in
    1 | true | yes | TRUE | YES) open_firewall ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
    systemctl enable niac.service >/dev/null 2>&1 || true
    if systemctl is-active --quiet niac.service 2>/dev/null; then
        systemctl restart niac.service || true
    else
        systemctl start niac.service || true
    fi
fi

cat <<'EOF'

==========================================
  NIAC installed successfully
==========================================

Web interface: https://localhost:8445

Commands:
  View logs:  journalctl -u niac -f
  Restart:    sudo systemctl restart niac
  Status:     sudo systemctl status niac
  Stop:       sudo systemctl stop niac

EOF

# When we did not open the firewall, tell the operator how (only if one is active).
case "${NIAC_OPEN_FIREWALL:-0}" in
    1 | true | yes | TRUE | YES) : ;;
    *)
        hint=$(firewall_hint)
        if [ -n "$hint" ]; then
            echo "An active firewall was detected; TCP ${PORT} is NOT open to the network."
            echo "To allow remote access to the web interface, run:"
            echo "$hint"
            echo "(or re-install with NIAC_OPEN_FIREWALL=1 to open it automatically)"
            echo ""
        fi
        ;;
esac

exit 0
