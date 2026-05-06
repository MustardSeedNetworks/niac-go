#!/bin/sh
# Pre-install: ensure the niac service user and group exist before files are
# unpacked, so the directory ownership entries in the package can resolve.
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
        --comment "NiAC Network Device Simulator" \
        niac
fi

exit 0
