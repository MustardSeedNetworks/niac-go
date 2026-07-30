#!/bin/sh
set -e

PACKAGE_STATE_DIR=/var/lib/niac-package
UFW_MARKER="$PACKAGE_STATE_DIR/firewall-ufw-owned"
FIREWALLD_MARKER="$PACKAGE_STATE_DIR/firewall-firewalld-owned"

is_purge=0
case "$1" in
    purge|0)
        is_purge=1
        ;;
esac

if [ "$is_purge" -eq 1 ]; then
    if [ -f "$UFW_MARKER" ] && command -v ufw >/dev/null 2>&1; then
        if ufw delete allow 8445/tcp >/dev/null 2>&1; then
            rm -f "$UFW_MARKER"
        fi
    fi
    if [ -f "$FIREWALLD_MARKER" ]; then
        if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
            if firewall-cmd --permanent --remove-port=8445/tcp >/dev/null 2>&1; then
                firewall-cmd --reload >/dev/null 2>&1 || true
                rm -f "$FIREWALLD_MARKER"
            fi
        elif command -v firewall-offline-cmd >/dev/null 2>&1 && \
            firewall-offline-cmd --remove-port=8445/tcp >/dev/null 2>&1; then
            rm -f "$FIREWALLD_MARKER"
        fi
    fi
    rmdir "$PACKAGE_STATE_DIR" 2>/dev/null || true

    if getent passwd niac >/dev/null 2>&1; then
        userdel niac >/dev/null 2>&1 || true
    fi
    if getent group niac >/dev/null 2>&1; then
        groupdel niac >/dev/null 2>&1 || true
    fi

    rm -rf /etc/niac /var/lib/niac /var/log/niac
else
    echo "NIAC removed. Data preserved in /var/lib/niac"
fi
