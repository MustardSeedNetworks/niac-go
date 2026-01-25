#!/usr/bin/env bash
# =============================================================================
# File Length Check Script
# =============================================================================
# Enforces file length limits to maintain code quality and readability.
#
# Thresholds:
#   Go source:        Warn >800, Fail >2000
#   TypeScript/React: Warn >500, Fail >1000
#   Test files:       Warn >1000, Fail >2000
#   Config files:     Warn >300, Fail >500
#
# Usage: ./scripts/check-file-length.sh [--strict]
#   --strict: Treat warnings as failures
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Thresholds
GO_WARN=800
GO_FAIL=2000
GO_TEST_WARN=1000
GO_TEST_FAIL=2000
TS_WARN=500
TS_FAIL=1000
# Config files can be legitimately large (golangci.yml, CI workflows)
CONFIG_WARN=500
CONFIG_FAIL=1000

# Counters
WARNINGS=0
FAILURES=0
STRICT_MODE=false

# Parse arguments
if [[ "${1:-}" == "--strict" ]]; then
    STRICT_MODE=true
fi

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}${CYAN}║                         File Length Check                                    ║${NC}"
echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to check file length
check_file() {
    local file="$1"
    local warn_threshold="$2"
    local fail_threshold="$3"
    local file_type="$4"

    if [[ ! -f "$file" ]]; then
        return 0
    fi

    local lines
    lines=$(wc -l < "$file" | tr -d ' ')

    if [[ $lines -gt $fail_threshold ]]; then
        echo -e "${RED}✗ FAIL${NC} ${file} (${lines} lines > ${fail_threshold} ${file_type} limit)"
        FAILURES=$((FAILURES + 1))
    elif [[ $lines -gt $warn_threshold ]]; then
        echo -e "${YELLOW}⚠ WARN${NC} ${file} (${lines} lines > ${warn_threshold} ${file_type} recommended)"
        WARNINGS=$((WARNINGS + 1))
    fi
}

# Check Go source files
echo -e "${BOLD}Checking Go source files...${NC}"
while IFS= read -r -d '' file; do
    # Skip vendor, node_modules, generated files
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/node_modules/"* ]]; then
        continue
    fi

    # Determine if test file
    if [[ "$file" == *"_test.go" ]]; then
        check_file "$file" "$GO_TEST_WARN" "$GO_TEST_FAIL" "test"
    else
        check_file "$file" "$GO_WARN" "$GO_FAIL" "source"
    fi
done < <(find . -name "*.go" -type f -print0 2>/dev/null)

# Check TypeScript/React files
echo ""
echo -e "${BOLD}Checking TypeScript/React files...${NC}"
if [[ -d "ui/src" ]]; then
    while IFS= read -r -d '' file; do
        if [[ "$file" == *"/node_modules/"* ]]; then
            continue
        fi
        check_file "$file" "$TS_WARN" "$TS_FAIL" "component"
    done < <(find ui/src -name "*.ts" -o -name "*.tsx" -type f -print0 2>/dev/null)
fi

# Check config files (YAML)
echo ""
echo -e "${BOLD}Checking config files...${NC}"
while IFS= read -r -d '' file; do
    if [[ "$file" == *"/node_modules/"* ]] || [[ "$file" == *"/vendor/"* ]]; then
        continue
    fi
    # Skip example/template files which are intentionally large
    if [[ "$file" == *"/examples/"* ]] || [[ "$file" == *"/templates/"* ]]; then
        continue
    fi
    check_file "$file" "$CONFIG_WARN" "$CONFIG_FAIL" "config"
done < <(find . -name "*.yaml" -o -name "*.yml" -type f -print0 2>/dev/null)

# Summary
echo ""
echo -e "${BOLD}═══════════════════════════════════════════════════════════════════════════════${NC}"

if [[ $FAILURES -gt 0 ]]; then
    echo -e "${RED}${BOLD}✗ FAILED${NC}: ${FAILURES} file(s) exceed maximum length limits"
    echo ""
    echo "Files exceeding limits need to be split into smaller, focused modules."
    echo "See: https://github.com/krisarmstrong/niac-go/blob/main/docs/CODE_STYLE.md"
    exit 1
elif [[ $WARNINGS -gt 0 ]]; then
    echo -e "${YELLOW}${BOLD}⚠ WARNINGS${NC}: ${WARNINGS} file(s) exceed recommended length"
    if [[ "$STRICT_MODE" == true ]]; then
        echo -e "${RED}Strict mode enabled - treating warnings as failures${NC}"
        exit 1
    fi
    echo ""
    echo "Consider splitting these files to improve maintainability."
    exit 0
else
    echo -e "${GREEN}${BOLD}✓ PASSED${NC}: All files within length limits"
    exit 0
fi
