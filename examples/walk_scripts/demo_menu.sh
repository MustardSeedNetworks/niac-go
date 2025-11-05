#!/bin/bash

# Interactive Demo Launcher for Network-in-a-Can
# Provides a menu-driven interface for running demos and network walks

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SCENARIO_DIR="$SCRIPT_DIR/../scenario_configs"
WALKS_DIR="$SCRIPT_DIR/../device_walks"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Clear screen function
clear_screen() {
    clear
}

# Display header
show_header() {
    clear_screen
    echo -e "${BOLD}${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${CYAN}║                                                            ║${NC}"
    echo -e "${BOLD}${CYAN}║          Network-in-a-Can Interactive Launcher             ║${NC}"
    echo -e "${BOLD}${CYAN}║                                                            ║${NC}"
    echo -e "${BOLD}${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Main menu
main_menu() {
    while true; do
        show_header
        echo -e "${GREEN}Main Menu${NC}"
        echo "=================================="
        echo "1. Run Scenario Demo"
        echo "2. Run Network Walk"
        echo "3. List All Available Scenarios"
        echo "4. List All Available Walks"
        echo "5. Statistics"
        echo "6. Exit"
        echo "=================================="
        echo -n "Select an option (1-6): "
        read -r choice

        case $choice in
            1) scenario_menu ;;
            2) walk_vendor_menu ;;
            3) list_scenarios ;;
            4) list_all_walks ;;
            5) show_statistics ;;
            6) exit 0 ;;
            *) echo -e "${RED}Invalid option${NC}"; sleep 1 ;;
        esac
    done
}

# Scenario menu
scenario_menu() {
    show_header
    echo -e "${GREEN}Available Scenarios${NC}"
    echo "=================================="

    # Build array of scenarios
    scenarios=()
    i=1
    for cfg in "$SCENARIO_DIR"/*.cfg; do
        if [ -f "$cfg" ]; then
            scenario=$(basename "$cfg" .cfg)
            scenarios+=("$scenario")
            echo "$i. $scenario"
            ((i++))
        fi
    done
    echo "0. Back to Main Menu"
    echo "=================================="
    echo -n "Select scenario (0-$((${#scenarios[@]}))): "
    read -r choice

    if [ "$choice" = "0" ]; then
        return
    fi

    if [ "$choice" -ge 1 ] && [ "$choice" -le "${#scenarios[@]}" ]; then
        scenario="${scenarios[$((choice-1))]}"
        echo ""
        echo -e "${YELLOW}Running scenario: $scenario${NC}"
        echo ""
        "$SCRIPT_DIR/run_demo.sh" "$scenario"
        echo ""
        echo -e "${CYAN}Press Enter to continue...${NC}"
        read -r
    else
        echo -e "${RED}Invalid selection${NC}"
        sleep 1
    fi
}

# Walk vendor selection menu
walk_vendor_menu() {
    show_header
    echo -e "${GREEN}Select Vendor${NC}"
    echo "=================================="

    # Build array of vendors with file counts
    vendors=()
    i=1
    for vendor_dir in "$WALKS_DIR"/*; do
        if [ -d "$vendor_dir" ]; then
            vendor=$(basename "$vendor_dir")
            count=$(find "$vendor_dir" -type f \( -name "*.txt" -o -name "*.snap" \) 2>/dev/null | wc -l | tr -d ' ')
            if [ "$count" -gt 0 ]; then
                vendors+=("$vendor")
                printf "%2d. %-15s (%3d walks)\n" "$i" "$vendor" "$count"
                ((i++))
            fi
        fi
    done
    echo " 0. Back to Main Menu"
    echo "=================================="
    echo -n "Select vendor (0-$((${#vendors[@]}))): "
    read -r choice

    if [ "$choice" = "0" ]; then
        return
    fi

    if [ "$choice" -ge 1 ] && [ "$choice" -le "${#vendors[@]}" ]; then
        vendor="${vendors[$((choice-1))]}"
        walk_file_menu "$vendor"
    else
        echo -e "${RED}Invalid selection${NC}"
        sleep 1
    fi
}

# Walk file selection menu
walk_file_menu() {
    local vendor=$1
    local vendor_dir="$WALKS_DIR/$vendor"

    while true; do
        show_header
        echo -e "${GREEN}$vendor Walk Files${NC}"
        echo "=================================="

        # Build array of walk files
        walks=()
        i=1
        while IFS= read -r file; do
            walks+=("$file")
            filename=$(basename "$file")
            # Truncate long filenames
            if [ ${#filename} -gt 50 ]; then
                filename="${filename:0:47}..."
            fi
            echo "$i. $filename"
            ((i++))
        done < <(find "$vendor_dir" -type f \( -name "*.txt" -o -name "*.snap" \) 2>/dev/null | sort)

        echo " 0. Back to Vendor Selection"
        echo "=================================="
        echo -n "Select walk file (0-$((${#walks[@]}))): "
        read -r choice

        if [ "$choice" = "0" ]; then
            return
        fi

        if [ "$choice" -ge 1 ] && [ "$choice" -le "${#walks[@]}" ]; then
            walk_file="${walks[$((choice-1))]}"
            filename=$(basename "$walk_file")
            echo ""
            echo -e "${YELLOW}Running walk: $filename${NC}"
            echo ""
            "$SCRIPT_DIR/walk_network.sh" "$vendor" "$filename"
            echo ""
            echo -e "${CYAN}Press Enter to continue...${NC}"
            read -r
        else
            echo -e "${RED}Invalid selection${NC}"
            sleep 1
        fi
    done
}

# List all scenarios
list_scenarios() {
    show_header
    echo -e "${GREEN}All Available Scenarios${NC}"
    echo "=================================="

    for cfg in "$SCENARIO_DIR"/*.cfg; do
        if [ -f "$cfg" ]; then
            scenario=$(basename "$cfg" .cfg)
            size=$(du -h "$cfg" 2>/dev/null | cut -f1)
            printf "%-20s %8s\n" "$scenario" "$size"
        fi
    done | sort

    echo ""
    echo -e "${CYAN}Press Enter to continue...${NC}"
    read -r
}

# List all walks
list_all_walks() {
    show_header
    echo -e "${GREEN}All Available Walks by Vendor${NC}"
    echo "=================================="

    for vendor_dir in "$WALKS_DIR"/*; do
        if [ -d "$vendor_dir" ]; then
            vendor=$(basename "$vendor_dir")
            count=$(find "$vendor_dir" -type f \( -name "*.txt" -o -name "*.snap" \) 2>/dev/null | wc -l | tr -d ' ')
            if [ "$count" -gt 0 ]; then
                printf "%-15s: %3d walks\n" "$vendor" "$count"
            fi
        fi
    done | sort

    echo ""
    echo -e "${CYAN}Press Enter to continue...${NC}"
    read -r
}

# Show statistics
show_statistics() {
    show_header
    echo -e "${GREEN}Network-in-a-Can Statistics${NC}"
    echo "=================================="
    echo ""

    # Count scenarios
    scenario_count=$(find "$SCENARIO_DIR" -type f -name "*.cfg" 2>/dev/null | wc -l | tr -d ' ')
    echo -e "${BOLD}Scenarios:${NC} $scenario_count"

    # Count walks by vendor
    echo ""
    echo -e "${BOLD}Device Walks by Vendor:${NC}"
    total_walks=0
    for vendor_dir in "$WALKS_DIR"/*; do
        if [ -d "$vendor_dir" ]; then
            vendor=$(basename "$vendor_dir")
            count=$(find "$vendor_dir" -type f \( -name "*.txt" -o -name "*.snap" \) 2>/dev/null | wc -l | tr -d ' ')
            if [ "$count" -gt 0 ]; then
                printf "  %-15s: %3d walks\n" "$vendor" "$count"
                ((total_walks+=count))
            fi
        fi
    done | sort
    echo "  --------------------------------"
    echo -e "  ${BOLD}Total:${NC}          $total_walks walks"

    # Disk usage
    echo ""
    echo -e "${BOLD}Disk Usage:${NC}"
    du -sh "$WALKS_DIR" 2>/dev/null | awk '{print "  Device walks:   " $1}'
    du -sh "$SCENARIO_DIR" 2>/dev/null | awk '{print "  Scenarios:      " $1}'
    du -sh "$SCRIPT_DIR/../captures" 2>/dev/null | awk '{print "  Captures:       " $1}'
    du -sh "$SCRIPT_DIR/.." 2>/dev/null | awk '{print "  Total:          " $1}'

    echo ""
    echo -e "${CYAN}Press Enter to continue...${NC}"
    read -r
}

# Start the menu
main_menu
