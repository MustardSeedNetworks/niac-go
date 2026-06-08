#!/bin/sh
set -e

is_purge=0
case "$1" in
    purge|0)
        is_purge=1
        ;;
esac

if [ "$is_purge" -eq 1 ]; then
    if command -v ufw >/dev/null 2>&1; then
        ufw delete allow 8445/tcp >/dev/null 2>&1 || true
    fi
    if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --permanent --remove-port=8445/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
    fi

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
