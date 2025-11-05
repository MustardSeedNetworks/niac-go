#!/bin/bash

# Unified Network-in-a-Can Demo Runner
# Replaces individual .bat files with a single, flexible script
# Usage: ./run_demo.sh <scenario> [network_interface]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SCENARIO_DIR="$SCRIPT_DIR/../scenario_configs"
NIAC_JAR="$BASE_DIR/niac-6.0.jar"

# Default network interface (can be overridden)
DEFAULT_INTERFACE="en0"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 <scenario> [network_interface]"
    echo ""
    echo "Available scenarios:"
    for cfg in "$SCENARIO_DIR"/*.cfg; do
        basename "$cfg" .cfg
    done | sort
    echo ""
    echo "Examples:"
    echo "  $0 nexus"
    echo "  $0 brocade eth0"
    echo "  $0 demo Demo"
    exit 1
}

# Check if scenario provided
if [ $# -lt 1 ]; then
    usage
fi

SCENARIO="$1"
INTERFACE="${2:-$DEFAULT_INTERFACE}"

# Find scenario config file
if [ "$SCENARIO" = "demo" ] || [ "$SCENARIO" = "Demo" ]; then
    CONFIG_FILE="$SCENARIO_DIR/Demo.cfg"
elif [ -f "$SCENARIO_DIR/$SCENARIO.cfg" ]; then
    CONFIG_FILE="$SCENARIO_DIR/$SCENARIO.cfg"
else
    echo -e "${RED}Error: Scenario '$SCENARIO' not found${NC}"
    echo ""
    usage
fi

# Check if niac jar exists
if [ ! -f "$NIAC_JAR" ]; then
    echo -e "${RED}Error: niac-6.0.jar not found at $NIAC_JAR${NC}"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: Config file not found: $CONFIG_FILE${NC}"
    exit 1
fi

# Display run information
echo -e "${GREEN}Network-in-a-Can Demo Runner${NC}"
echo "=================================="
echo "Scenario:  $SCENARIO"
echo "Config:    $CONFIG_FILE"
echo "Interface: $INTERFACE"
echo "=================================="
echo ""

# Check for Java
if ! command -v java &> /dev/null; then
    echo -e "${RED}Error: Java not found. Please install Java to run this demo.${NC}"
    exit 1
fi

# Run the demo
echo -e "${YELLOW}Starting demo...${NC}"
echo ""

java -cp "$NIAC_JAR" fluke.niac.Niac "$INTERFACE" "$CONFIG_FILE"

EXIT_CODE=$?

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}Demo completed successfully${NC}"
else
    echo -e "${RED}Demo exited with code $EXIT_CODE${NC}"
fi

exit $EXIT_CODE
