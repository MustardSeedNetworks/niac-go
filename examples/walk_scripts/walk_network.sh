#!/bin/bash

# Network Walk Script
# Performs SNMP walks on network devices using specified walk files
# Usage: ./walk_network.sh [vendor] [device_file]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
WALKS_DIR="$SCRIPT_DIR/../device_walks"
NIAC_JAR="$BASE_DIR/niac-6.0.jar"

# Default network interface
DEFAULT_INTERFACE="en0"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
    echo "Usage: $0 [vendor] [device_file] [network_interface]"
    echo ""
    echo "Available vendors:"
    for vendor_dir in "$WALKS_DIR"/*; do
        if [ -d "$vendor_dir" ]; then
            vendor=$(basename "$vendor_dir")
            count=$(find "$vendor_dir" -type f -name "*.txt" 2>/dev/null | wc -l | tr -d ' ')
            echo "  $vendor ($count walks)"
        fi
    done | sort
    echo ""
    echo "Examples:"
    echo "  $0 cisco                           # List all Cisco walk files"
    echo "  $0 cisco Cisco_10_250_0_1.txt      # Run specific Cisco walk"
    echo "  $0 juniper                         # List all Juniper walk files"
    echo ""
    exit 1
}

list_vendor_walks() {
    local vendor=$1
    local vendor_dir="$WALKS_DIR/$vendor"

    if [ ! -d "$vendor_dir" ]; then
        echo -e "${RED}Error: Vendor '$vendor' not found${NC}"
        exit 1
    fi

    echo -e "${GREEN}Available $vendor walk files:${NC}"
    echo "=================================="

    find "$vendor_dir" -type f \( -name "*.txt" -o -name "*.snap" \) 2>/dev/null | while read file; do
        basename "$file"
    done | sort

    echo ""
    echo "To run a specific walk:"
    echo "  $0 $vendor <filename>"
}

run_walk() {
    local vendor=$1
    local device_file=$2
    local interface="${3:-$DEFAULT_INTERFACE}"
    local vendor_dir="$WALKS_DIR/$vendor"
    local walk_file="$vendor_dir/$device_file"

    # Check if file exists
    if [ ! -f "$walk_file" ]; then
        echo -e "${RED}Error: Walk file not found: $walk_file${NC}"
        echo ""
        list_vendor_walks "$vendor"
        exit 1
    fi

    # Check if niac jar exists
    if [ ! -f "$NIAC_JAR" ]; then
        echo -e "${RED}Error: niac-6.0.jar not found at $NIAC_JAR${NC}"
        exit 1
    fi

    # Display run information
    echo -e "${GREEN}Network Walk Runner${NC}"
    echo "=================================="
    echo "Vendor:    $vendor"
    echo "Device:    $device_file"
    echo "Walk File: $walk_file"
    echo "Interface: $interface"
    echo "=================================="
    echo ""

    # Check for Java
    if ! command -v java &> /dev/null; then
        echo -e "${RED}Error: Java not found. Please install Java to run walks.${NC}"
        exit 1
    fi

    # Run the walk
    echo -e "${YELLOW}Starting network walk...${NC}"
    echo ""

    # Create a temporary config that uses this walk file
    TEMP_CFG=$(mktemp)
    echo "# Auto-generated config for $device_file" > "$TEMP_CFG"
    echo "walk=$walk_file" >> "$TEMP_CFG"

    java -cp "$NIAC_JAR" fluke.niac.Niac "$interface" "$TEMP_CFG"

    EXIT_CODE=$?

    # Cleanup
    rm -f "$TEMP_CFG"

    echo ""
    if [ $EXIT_CODE -eq 0 ]; then
        echo -e "${GREEN}Walk completed successfully${NC}"
    else
        echo -e "${RED}Walk exited with code $EXIT_CODE${NC}"
    fi

    exit $EXIT_CODE
}

# Main logic
if [ $# -eq 0 ]; then
    usage
fi

VENDOR="$1"

# Check if vendor directory exists
if [ ! -d "$WALKS_DIR/$VENDOR" ]; then
    echo -e "${RED}Error: Vendor '$VENDOR' not found${NC}"
    echo ""
    usage
fi

if [ $# -eq 1 ]; then
    # Just vendor specified - list available walks
    list_vendor_walks "$VENDOR"
else
    # Run specific walk
    DEVICE_FILE="$2"
    INTERFACE="${3:-$DEFAULT_INTERFACE}"
    run_walk "$VENDOR" "$DEVICE_FILE" "$INTERFACE"
fi
