#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
TEST_REPO=$(mktemp -d)
trap 'rm -rf "$TEST_REPO"' EXIT

mkdir -p "$TEST_REPO/.husky" "$TEST_REPO/bin" "$TEST_REPO/ui/node_modules"
cp "$REPO_ROOT/.husky/pre-commit" "$REPO_ROOT/.husky/check-frontend" \
  "$REPO_ROOT/.husky/run-tailed" "$TEST_REPO/.husky/"
cp "$REPO_ROOT/.nvmrc" "$REPO_ROOT/package.json" "$TEST_REPO/"

git -C "$TEST_REPO" init -q
git -C "$TEST_REPO" config user.email "hook-test@example.invalid"
git -C "$TEST_REPO" config user.name "Hook Test"
printf 'export const value = 1;\n' >"$TEST_REPO/ui/app.ts"
git -C "$TEST_REPO" add ui/app.ts

cat >"$TEST_REPO/bin/gitleaks" <<'EOF'
#!/usr/bin/env sh
exit 0
EOF

# Derived from .nvmrc and package.json rather than hardcoded, the same way
# .husky/check-frontend derives them. A hardcoded stub silently breaks this
# test on every Node/npm bump.
EXPECTED_NODE=$(cat "$REPO_ROOT/.nvmrc")
EXPECTED_NPM=$(sed -n 's/.*"packageManager": "npm@\([^"]*\)".*/\1/p' "$REPO_ROOT/package.json")

cat >"$TEST_REPO/bin/node" <<EOF
#!/usr/bin/env sh
if [ "\${1:-}" = "--version" ]; then
  echo "v$EXPECTED_NODE"
  exit 0
fi
exit 0
EOF

cat >"$TEST_REPO/bin/npm" <<EOF
#!/usr/bin/env sh
if [ "\${1:-}" = "--version" ]; then
  echo "$EXPECTED_NPM"
  exit 0
fi
if [ "\${1:-}" = "run" ] && [ "\${2:-}" = "typecheck" ]; then
  exit 0
fi
echo "simulated frontend test startup failure" >&2
exit 23
EOF

cat >"$TEST_REPO/bin/npx" <<'EOF'
#!/usr/bin/env sh
exit 0
EOF

chmod +x "$TEST_REPO/bin/"*
mkdir -p "$TEST_REPO/ui/node_modules/.bin"
cat >"$TEST_REPO/ui/node_modules/.bin/biome" <<'EOF'
#!/usr/bin/env sh
exit 0
EOF
chmod +x "$TEST_REPO/ui/node_modules/.bin/biome"

set +e
OUTPUT=$(
  cd "$TEST_REPO"
  CI='' PATH="$TEST_REPO/bin:$PATH" sh .husky/pre-commit 2>&1
)
STATUS=$?
set -e

if [ "$STATUS" -eq 0 ]; then
  echo "expected the pre-commit hook to reject a failed frontend test command"
  echo "$OUTPUT"
  exit 1
fi

if printf '%s\n' "$OUTPUT" | grep -q "Frontend tests passed"; then
  echo "hook reported frontend success after the test command failed"
  echo "$OUTPUT"
  exit 1
fi

printf '%s\n' "$OUTPUT" | grep -q "Frontend tests failed"

# A version that cannot ever coincide with the real pin, so this stays a
# genuine mismatch across future bumps.
cat >"$TEST_REPO/bin/node" <<'EOF'
#!/usr/bin/env sh
if [ "${1:-}" = "--version" ]; then
  echo "v0.0.0"
  exit 0
fi
exit 0
EOF
chmod +x "$TEST_REPO/bin/node"

set +e
OUTPUT=$(
  cd "$TEST_REPO"
  CI='' PATH="$TEST_REPO/bin:$PATH" sh .husky/pre-commit 2>&1
)
STATUS=$?
set -e

if [ "$STATUS" -eq 0 ]; then
  echo "expected the pre-commit hook to reject the wrong Node.js version"
  echo "$OUTPUT"
  exit 1
fi

printf '%s\n' "$OUTPUT" | grep -q "Frontend toolchain mismatch"
if printf '%s\n' "$OUTPUT" | grep -q "Frontend tests passed"; then
  echo "hook ran frontend tests with the wrong Node.js version"
  echo "$OUTPUT"
  exit 1
fi
