#!/bin/sh
# Post-remove: clean up firewall rules + the service user on a *complete*
# uninstall (not on upgrade). Detection is per-format:
#   deb -> $1 == "purge"
#   rpm -> $1 == "0"   (number of remaining instances)
set -e

is_purge=0
case "$1" in
    purge)
        is_purge=1
        ;;
    0)
        is_purge=1
        ;;
esac

if [ "$is_purge" -eq 1 ]; then
    if command -v ufw >/dev/null 2>&1; then
        ufw delete allow 8080/tcp >/dev/null 2>&1 || true
    fi
    if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --permanent --remove-port=8080/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
    fi

    if getent passwd niac >/dev/null 2>&1; then
        userdel niac >/dev/null 2>&1 || true
    fi
    if getent group niac >/dev/null 2>&1; then
        groupdel niac >/dev/null 2>&1 || true
    fi

    rm -rf /etc/niac /var/lib/niac /var/log/niac
fi

exit 0
