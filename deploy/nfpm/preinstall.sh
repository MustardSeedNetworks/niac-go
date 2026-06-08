#!/bin/sh
set -e

if ! getent group niac >/dev/null 2>&1; then
    groupadd --system niac
fi

if ! getent passwd niac >/dev/null 2>&1; then
    useradd --system \
        --gid niac \
        --home-dir /var/lib/niac \
        --no-create-home \
        --shell /usr/sbin/nologin \
        --comment "NIAC - Network In A Can" \
        niac
fi

exit 0
