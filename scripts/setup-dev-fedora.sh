#!/bin/bash
# =============================================================================
# setup-dev-fedora.sh - Configure Fedora dev server for NIAC builds
# =============================================================================
#
# This script sets up a fresh Fedora dev server with all the tools needed
# to build NIAC packages (.deb, .rpm, .exe, .zip).
#
# Usage: ./scripts/setup-dev-fedora.sh
#
# =============================================================================

set -euo pipefail

echo "=== NIAC Dev Server Setup (Fedora) ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() { echo -e "${GREEN}✓ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠ $1${NC}"; }
error() { echo -e "${RED}✗ $1${NC}"; exit 1; }

# =============================================================================
# Step 1: Fix Go PATH
# =============================================================================
echo "[1/6] Configuring Go PATH..."

if ! grep -q '/usr/local/go/bin' ~/.bashrc 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
    success "Added Go to PATH in ~/.bashrc"
else
    success "Go PATH already configured"
fi

# Source for current session
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# =============================================================================
# Step 2: Install Go tools
# =============================================================================
echo "[2/6] Installing Go tools..."

if command -v gosec &> /dev/null; then
    success "gosec already installed"
else
    echo "  Installing gosec..."
    go install github.com/securego/gosec/v2/cmd/gosec@latest || warn "gosec install failed"
    success "gosec installed"
fi

if command -v golangci-lint &> /dev/null; then
    success "golangci-lint already installed"
else
    echo "  Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest || warn "golangci-lint install failed"
    success "golangci-lint installed"
fi

# =============================================================================
# Step 3: Install system dependencies
# =============================================================================
echo "[3/6] Checking system dependencies..."

DEPS_NEEDED=""
command -v rpmbuild &> /dev/null || DEPS_NEEDED="$DEPS_NEEDED rpm-build"
command -v dpkg-deb &> /dev/null || DEPS_NEEDED="$DEPS_NEEDED dpkg"
command -v node &> /dev/null || DEPS_NEEDED="$DEPS_NEEDED nodejs"
command -v npm &> /dev/null || DEPS_NEEDED="$DEPS_NEEDED npm"

if [ -n "$DEPS_NEEDED" ]; then
    echo "  Installing: $DEPS_NEEDED"
    sudo dnf install -y $DEPS_NEEDED
    success "System dependencies installed"
else
    success "All system dependencies present"
fi

# =============================================================================
# Step 4: Rename project directory to match convention
# =============================================================================
echo "[4/6] Checking project directory structure..."

if [ -d ~/Developer/niac-go ] && [ ! -d ~/Developer/niac/go ]; then
    echo "  Renaming ~/Developer/niac-go to ~/Developer/niac/go..."
    mkdir -p ~/Developer/niac
    mv ~/Developer/niac-go ~/Developer/niac/go
    success "Project directory renamed to ~/Developer/niac/go"
elif [ -d ~/Developer/niac/go ]; then
    success "Project directory already at ~/Developer/niac/go"
else
    warn "Project not found at ~/Developer/niac-go or ~/Developer/niac/go"
fi

# =============================================================================
# Step 5: Verify Node.js version
# =============================================================================
echo "[5/6] Verifying Node.js version..."

if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version | sed 's/v//')
    NODE_MAJOR=$(echo "$NODE_VERSION" | cut -d. -f1)
    if [ "$NODE_MAJOR" -ge 25 ]; then
        success "Node.js $NODE_VERSION (OK)"
    else
        warn "Node.js $NODE_VERSION (recommended: 25.x)"
    fi
else
    warn "Node.js not installed"
fi

# =============================================================================
# Step 6: Verification Summary
# =============================================================================
echo ""
echo "[6/6] Dev Server Setup Verification"
echo "======================================"

# Go version
echo -n "  Go:            "
if command -v go &> /dev/null; then
    go version | awk '{print $3}' | sed 's/go//'
else
    echo "NOT INSTALLED"
fi

# Node version
echo -n "  Node.js:       "
if command -v node &> /dev/null; then
    node --version
else
    echo "NOT INSTALLED"
fi

# npm version
echo -n "  npm:           "
if command -v npm &> /dev/null; then
    npm --version
else
    echo "NOT INSTALLED"
fi

# make version
echo -n "  make:          "
if command -v make &> /dev/null; then
    make --version | head -1
else
    echo "NOT INSTALLED"
fi

# Package tools
echo -n "  rpmbuild:      "
command -v rpmbuild &> /dev/null && echo "OK" || echo "MISSING"
echo -n "  dpkg-deb:      "
command -v dpkg-deb &> /dev/null && echo "OK" || echo "MISSING"

# Go tools
echo -n "  gosec:         "
command -v gosec &> /dev/null && echo "OK" || echo "MISSING"
echo -n "  golangci-lint: "
command -v golangci-lint &> /dev/null && echo "OK" || echo "MISSING"

echo ""
success "Dev server setup complete!"
echo ""
echo "Next steps:"
echo "  1. Run: source ~/.bashrc"
echo "  2. cd ~/Developer/niac/go"
echo "  3. make build"
echo ""
