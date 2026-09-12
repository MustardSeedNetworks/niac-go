#!/bin/sh
set -e

# Stop the service only when the package is actually going away.
#
# RPM runs the OLD package's %preun *after* the NEW package's %post, so an
# unconditional stop here undid the install: postinstall enabled and started
# niac, this ran moments later and stopped and disabled it, and the upgrade
# finished with the service down and disabled. dpkg runs prerm before the new
# postinst, which is why the .deb was unaffected and only Fedora broke.
#
# The argument says which case this is. RPM passes the number of versions that
# will remain: 0 on removal, 1 or more during an upgrade. dpkg passes a word.
# postremove.sh already keys its own cleanup off the same convention.
case "${1:-}" in
    0 | remove | purge) ;;
    *) exit 0 ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet niac.service 2>/dev/null; then
        systemctl stop niac.service || true
    fi
    if systemctl is-enabled --quiet niac.service 2>/dev/null; then
        systemctl disable niac.service || true
    fi
fi

exit 0
