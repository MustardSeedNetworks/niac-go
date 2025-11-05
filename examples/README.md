# Network-in-a-Can Demo Configurations

Consolidated and organized demo configurations for the Network-in-a-Can project.

## Quick Start

### Run a Demo Scenario
```bash
cd walk_scripts
./run_demo.sh nexus           # Run Nexus scenario
./run_demo.sh brocade         # Run Brocade scenario
./run_demo.sh demo            # Run main demo
```

### Run a Network Walk
```bash
cd walk_scripts
./walk_network.sh cisco                        # List all Cisco walks
./walk_network.sh cisco Cisco_10_250_0_1.txt   # Run specific Cisco walk
./walk_network.sh juniper                      # List all Juniper walks
```

### Interactive Menu
```bash
cd walk_scripts
./demo_menu.sh
```

## Directory Structure

```
demo_configs/
├── device_walks/         # Device walk files by vendor (565 files)
│   ├── cisco/           # Cisco devices (488 files)
│   ├── dell/            # Dell devices
│   ├── juniper/         # Juniper devices
│   ├── extreme/         # Extreme Networks
│   ├── brocade/         # Brocade devices
│   ├── hp/              # HP devices
│   └── [15 more vendors]
├── scenario_configs/     # Scenario .cfg files (14 files)
├── walk_scripts/         # Executable scripts
│   ├── run_demo.sh      # Unified demo runner
│   ├── walk_network.sh  # Network walk tool
│   └── demo_menu.sh     # Interactive launcher
├── captures/             # Network capture files (8 files)
└── binaries/             # Binary dependencies (1 file)
```

## Available Scripts

### run_demo.sh
Unified demo runner - replaces 14 individual .bat files

**Usage:** `./run_demo.sh <scenario> [interface]`

**Examples:**
- `./run_demo.sh nexus`
- `./run_demo.sh brocade eth0`

### walk_network.sh
Network walk execution tool

**Usage:** `./walk_network.sh [vendor] [device_file] [interface]`

**Examples:**
- `./walk_network.sh cisco` - List all Cisco walks
- `./walk_network.sh cisco Cisco_10_250_0_1.txt` - Run specific walk

### demo_menu.sh
Interactive menu-driven launcher - easiest way to get started!

**Usage:** `./demo_menu.sh`

## Statistics

- **Total Files:** 691 files (~1.3 GB)
- **Device Walks:** 565 files across 20 vendors
- **Scenarios:** 14 configurations
- **Vendors:** Cisco, Dell, Juniper, HP, Extreme, Brocade, Fortinet, Huawei, and more

## Requirements

- Bash 3.2+
- Java (for running demos)
- Network interface access

## Documentation

See [CONSOLIDATION_REPORT.md](CONSOLIDATION_REPORT.md) for detailed information about:
- Complete migration details
- Vendor file counts
- Script features
- Space analysis
- Usage examples

## Vendor Coverage

| Vendor | Walks | Vendor | Walks |
|--------|-------|--------|-------|
| Cisco | 488 | Fortinet | 4 |
| Misc | 33 | Huawei | 4 |
| Brocade | 8 | Juniper | 4 |
| Extreme | 8 | Dell | 3 |
| Netgear | 4 | HP | 2 |

Plus: VMware, Meraki, MikroTik, Oracle, VoIP, ZTE

---

**Last Updated:** 2025-11-04
