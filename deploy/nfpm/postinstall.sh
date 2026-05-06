#!/bin/sh
# Post-install: set raw-socket capabilities, configure firewall (best-effort),
# reload systemd, and (re)start the niac service. Runs after both fresh
# installs and upgrades.
set -e

BINARY=/usr/bin/niac

# Set capabilities for raw socket access (required for packet capture).
if command -v setcap >/dev/null 2>&1; then
    setcap 'cap_net_raw,cap_net_admin=+ep' "$BINARY" || \
        echo "warning: could not set capabilities on $BINARY"
fi

# ufw (Debian/Ubuntu) — open the WebUI port if ufw is active.
if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
    ufw allow 8080/tcp comment 'NiAC WebUI HTTP' >/dev/null 2>&1 || true
fi

# firewalld (RHEL/Fedora) — same idea.
if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
    firewall-cmd --permanent --add-port=8080/tcp >/dev/null 2>&1 || true
    firewall-cmd --reload >/dev/null 2>&1 || true
fi

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
  NiAC installed successfully
==========================================

Web interface: http://localhost:8080

Commands:
  View logs:  journalctl -u niac -f
  Restart:    sudo systemctl restart niac
  Status:     sudo systemctl status niac
  Stop:       sudo systemctl stop niac

EOF

exit 0
