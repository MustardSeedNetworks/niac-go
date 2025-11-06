#!/bin/bash
# Install git hooks for NIAC-Go

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.githooks"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "📦 Installing git hooks for NIAC-Go..."
echo ""

# Check if .git directory exists
if [ ! -d "$GIT_HOOKS_DIR" ]; then
    echo "❌ .git/hooks directory not found. Are you in a git repository?"
    exit 1
fi

# Check if .githooks directory exists
if [ ! -d "$HOOKS_DIR" ]; then
    echo "❌ .githooks directory not found"
    exit 1
fi

# Install pre-commit hook
if [ -f "$HOOKS_DIR/pre-commit" ]; then
    echo "📝 Installing pre-commit hook..."
    cp "$HOOKS_DIR/pre-commit" "$GIT_HOOKS_DIR/pre-commit"
    chmod +x "$GIT_HOOKS_DIR/pre-commit"
    echo "✅ pre-commit hook installed"
else
    echo "⚠️  pre-commit hook not found in .githooks/"
fi

echo ""
echo "✅ Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will now run on every commit to:"
echo "  - Check code formatting (go fmt)"
echo "  - Run static analysis (go vet)"
echo "  - Run all tests"
echo "  - Check test coverage"
echo "  - Verify the build"
echo ""
echo "To skip the pre-commit hook, use: git commit --no-verify"
