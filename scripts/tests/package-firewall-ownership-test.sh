#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "$TEST_ROOT"' EXIT

mkdir -p "$TEST_ROOT/bin" "$TEST_ROOT/state" "$TEST_ROOT/package-state" "$TEST_ROOT/etc" "$TEST_ROOT/log"
touch "$TEST_ROOT/niac"

sed \
  -e "s|BINARY=/usr/bin/niac|BINARY=$TEST_ROOT/niac|" \
  -e "s|PACKAGE_STATE_DIR=/var/lib/niac-package|PACKAGE_STATE_DIR=$TEST_ROOT/package-state|" \
  "$REPO_ROOT/deploy/nfpm/postinstall.sh" >"$TEST_ROOT/postinstall.sh"
sed \
  -e "s|PACKAGE_STATE_DIR=/var/lib/niac-package|PACKAGE_STATE_DIR=$TEST_ROOT/package-state|" \
  -e "s|/etc/niac /var/lib/niac /var/log/niac|$TEST_ROOT/etc $TEST_ROOT/state $TEST_ROOT/log|" \
  "$REPO_ROOT/deploy/nfpm/postremove.sh" >"$TEST_ROOT/postremove.sh"

cat >"$TEST_ROOT/bin/firewall-stub" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

command_name=$(basename "$0")
case "$command_name" in
  setcap | userdel | groupdel)
    exit 0
    ;;
  getent)
    exit 1
    ;;
  systemctl)
    if [ "${1:-}" = "is-active" ]; then
      [ "${TEST_MANAGER:-none}" = "firewalld" ]
    fi
    exit 0
    ;;
  ufw)
    case "${1:-}" in
      status)
        if [ "${TEST_MANAGER:-none}" = "ufw" ]; then
          echo "Status: active"
          [ ! -f "$TEST_ROOT/ufw-rule" ] || echo "8445/tcp ALLOW Anywhere"
        else
          echo "Status: inactive"
        fi
        ;;
      allow)
        touch "$TEST_ROOT/ufw-rule"
        ;;
      delete)
        [ "${TEST_DELETE_FAIL:-0}" != "1" ] || exit 1
        rm -f "$TEST_ROOT/ufw-rule"
        ;;
    esac
    ;;
  firewall-cmd)
    case "$*" in
      *--query-port=8445/tcp*) [ -f "$TEST_ROOT/firewalld-rule" ] ;;
      *--add-port=8445/tcp*) touch "$TEST_ROOT/firewalld-rule" ;;
      *--remove-port=8445/tcp*) rm -f "$TEST_ROOT/firewalld-rule" ;;
    esac
    ;;
  firewall-offline-cmd)
    case "$*" in
      *--remove-port=8445/tcp*) rm -f "$TEST_ROOT/firewalld-rule" ;;
    esac
    ;;
esac
EOF

for command_name in setcap systemctl ufw firewall-cmd firewall-offline-cmd getent userdel groupdel; do
  cp "$TEST_ROOT/bin/firewall-stub" "$TEST_ROOT/bin/$command_name"
done
chmod +x "$TEST_ROOT/bin/"* "$TEST_ROOT/postinstall.sh" "$TEST_ROOT/postremove.sh"

run_script() {
  env TEST_ROOT="$TEST_ROOT" TEST_MANAGER="$1" TEST_DELETE_FAIL="${TEST_DELETE_FAIL:-0}" \
    NIAC_OPEN_FIREWALL="${2:-0}" \
    PATH="$TEST_ROOT/bin:$PATH" "$3" "${4:-}"
}

# An operator-owned UFW rule must never receive a package marker or be purged.
touch "$TEST_ROOT/ufw-rule"
run_script ufw 1 "$TEST_ROOT/postinstall.sh"
test ! -e "$TEST_ROOT/package-state/firewall-ufw-owned"
run_script ufw 0 "$TEST_ROOT/postremove.sh" purge
test -e "$TEST_ROOT/ufw-rule"

# A package-created UFW rule survives ordinary removal and is deleted on purge.
rm -f "$TEST_ROOT/ufw-rule"
mkdir -p "$TEST_ROOT/state"
run_script ufw 1 "$TEST_ROOT/postinstall.sh"
test -e "$TEST_ROOT/ufw-rule"
test -e "$TEST_ROOT/package-state/firewall-ufw-owned"
run_script ufw 0 "$TEST_ROOT/postremove.sh" remove
test -e "$TEST_ROOT/ufw-rule"
run_script ufw 0 "$TEST_ROOT/postremove.sh" purge
test ! -e "$TEST_ROOT/ufw-rule"

# Failed deletion retains ownership so a later purge can retry it.
mkdir -p "$TEST_ROOT/state" "$TEST_ROOT/package-state"
touch "$TEST_ROOT/ufw-rule" "$TEST_ROOT/package-state/firewall-ufw-owned"
TEST_DELETE_FAIL=1 run_script ufw 0 "$TEST_ROOT/postremove.sh" purge
test -e "$TEST_ROOT/ufw-rule"
test -e "$TEST_ROOT/package-state/firewall-ufw-owned"
run_script ufw 0 "$TEST_ROOT/postremove.sh" purge
test ! -e "$TEST_ROOT/ufw-rule"
test ! -e "$TEST_ROOT/package-state/firewall-ufw-owned"

# Firewalld follows the same ownership contract.
mkdir -p "$TEST_ROOT/state"
touch "$TEST_ROOT/firewalld-rule"
run_script firewalld 1 "$TEST_ROOT/postinstall.sh"
test ! -e "$TEST_ROOT/package-state/firewall-firewalld-owned"
run_script firewalld 0 "$TEST_ROOT/postremove.sh" purge
test -e "$TEST_ROOT/firewalld-rule"

rm -f "$TEST_ROOT/firewalld-rule"
mkdir -p "$TEST_ROOT/state"
run_script firewalld 1 "$TEST_ROOT/postinstall.sh"
test -e "$TEST_ROOT/package-state/firewall-firewalld-owned"
run_script firewalld 0 "$TEST_ROOT/postremove.sh" purge
test ! -e "$TEST_ROOT/firewalld-rule"

# A package-created permanent rule is also deleted while firewalld is stopped.
mkdir -p "$TEST_ROOT/package-state"
touch "$TEST_ROOT/firewalld-rule" "$TEST_ROOT/package-state/firewall-firewalld-owned"
run_script none 0 "$TEST_ROOT/postremove.sh" purge
test ! -e "$TEST_ROOT/firewalld-rule"

# Hosts without an active supported firewall create no ownership state.
mkdir -p "$TEST_ROOT/state"
run_script none 1 "$TEST_ROOT/postinstall.sh"
test ! -e "$TEST_ROOT/package-state/firewall-ufw-owned"
test ! -e "$TEST_ROOT/package-state/firewall-firewalld-owned"

echo "package firewall ownership regression tests passed"
