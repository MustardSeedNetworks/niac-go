# NIAC-Go: Network In A Can (Go Edition)

[![CI](https://github.com/krisarmstrong/niac-go/workflows/CI/badge.svg)](https://github.com/krisarmstrong/niac-go/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/)
[![Version](https://img.shields.io/badge/version-2.1.1-brightgreen.svg)](https://github.com/krisarmstrong/niac-go/releases)

**Production-ready network device simulator** - Complete YAML configuration system with per-protocol debug control, multi-IP support, and comprehensive protocol coverage.

**Current Version: 2.1.1** - Security Patch (golang.org/x/net vulnerabilities fixed)

## 🚀 Why Go?

NIAC-Go is a modern rewrite of the original Java-based NIAC, leveraging Go's strengths:

- **🔥 Native Performance**: No JVM overhead, instant startup
- **⚡ Blazing Fast**: 7.7M error injections/sec (vs ~100K/sec in Java)
- **💾 Lightweight**: 6.1MB binary vs 542KB JAR + 200MB JRE
- **🎯 Simple Deployment**: Single binary, no dependencies
- **🧵 Concurrency**: Goroutines make packet handling trivial
- **🎨 Beautiful TUI**: Modern terminal UI with Bubbletea
- **📝 Complete YAML Config**: All protocols fully configurable

## Performance Comparison

| Metric | Java (GraalVM) | Go | Improvement |
|--------|---------------|-----|-------------|
| **Startup** | ~50ms | ~5ms | **10x faster** |
| **Memory** | ~100MB | ~15MB | **6.7x less** |
| **Binary Size** | 16MB | 6.1MB | **2.6x smaller** |
| **Error Injection** | ~100K/sec | 7.7M/sec | **77x faster** |
| **Config Parsing** | ~1ms | ~1.3µs | **770x faster** |
| **Build Time** | 4-5 min | 5 sec | **48-60x faster** |

## Features

### v1.13.0 Highlights 🎉

- **🎯 Enhanced CLI/Help**: Professional CLI experience
  - Shell completion for bash, zsh, fish, and powershell
  - Rich help examples for all commands
  - Unix man pages generation (`niac man`)
  - Installation instructions for all shells
- **🔧 Configuration Management Tools**: DevOps-ready config tools
  - `niac config export` - Normalize and clean YAML configurations
  - `niac config diff` - Compare configurations and detect drift
  - `niac config merge` - Merge base and overlay configurations
  - Environment-specific overrides support

### v1.12.0 Highlights 🎉

- **🎨 Interactive TUI Command**: Modern terminal interface for real-time monitoring
  - New `niac interactive` command with Bubble Tea UI
  - Live device status and statistics
  - Interactive error injection (press 'i')
  - Keyboard controls and beautiful visualization

### v1.11.0 Highlights 🎉

- **📦 Configuration Templates**: Quick start with pre-built templates
  - 7 production-ready templates (minimal, router, switch, ap, server, iot, complete)
  - `niac template list|show|use` commands
  - Templates embedded in binary
  - Instant deployment for common scenarios

### v1.10.0 Highlights 🎉

- **✅ Modern CLI Framework**: Cobra-based command structure
  - `niac validate` command with comprehensive validation
  - Structured error reporting (ConfigError, ConfigErrorList)
  - JSON output for CI/CD integration (--json flag)
  - Verbose mode for detailed insights (--verbose)
  - Beautiful terminal formatting with context and suggestions
- **🔍 Configuration Validator**: Production-ready validation
  - Device validation (names, types, MAC/IP duplicates)
  - SNMP trap validation (thresholds 0-100, receiver format)
  - DNS record validation (domain names, IPs)
  - Three severity levels (error, warning, info)
  - Line/column tracking for precise error location

### v1.7.0 Highlights 🎉

- **✅ Comprehensive Testing Infrastructure**: Production-ready test coverage
  - **87 new tests** added across 4 major packages
  - Config package: **50.6% coverage** (5.2x improvement from 9.8%)
  - Protocol package: **15.4% coverage** (2.75x improvement from 5.6%)
  - Device simulator: **22.0% coverage** (new)
  - SNMP traps: **6.7% coverage** (new)
  - Full validation of v1.6.0 features (traffic patterns, SNMP traps)
- **🔍 Configuration Validator**: Comprehensive config validation tool
  - Three-level validation (Errors, Warnings, Info)
  - Device-level validation (name, type, MAC, IP addresses)
  - Protocol-specific validation (19 protocols)
  - v1.6.0 feature validation (traffic patterns, SNMP traps)
  - Threshold validation (CPU/memory percentages, STP priorities)
  - Detailed error messages with field names and suggestions
  - Integrated with `--dry-run` flag for pre-flight checks
  - Verbose mode for detailed configuration insights
- **⏳ Enhanced CLI/UX**: Improved user experience
  - Progress indicators during startup (capture engine, protocol stack, devices)
  - Feature summary on startup (SNMP agents, traps, traffic generation)
  - Clear error reporting with emojis (✓ ❌ ⚠️)
  - Enhanced `--dry-run` validation output

### Latest Features (v1.23.0) - Topology Configuration

- **🔗 Port-Channels (Link Aggregation)**: Bundle multiple physical interfaces for increased bandwidth and redundancy
  - LACP modes: active, passive, on
  - 2+ member interfaces with automatic validation
  - Trunk over port-channel support
- **🌉 Trunk Ports with VLAN Tagging**: 802.1Q trunk configuration
  - Multiple VLANs per trunk (1-4094)
  - Native VLAN support
  - Remote device references for topology validation
- **🏗️ Multi-Device Topologies**: Complete network environment simulation
  - Spine-leaf data center topologies
  - Enterprise campus (core-distribution-access)
  - Branch office and wireless deployments
  - Multi-vendor integration (Cisco, Juniper, Aruba)
- **📖 Comprehensive Guides**: New documentation for advanced scenarios
  - [Topology Configuration Guide](docs/TOPOLOGY_GUIDE.md) - Port-channels, trunks, VLANs
  - [Environment Simulation Guide](docs/ENVIRONMENTS.md) - Complete network examples
  - See `examples/topology/` and `examples/combinations/` for ready-to-use configs

### Previous Features (v1.6.0)

- **🚦 Configurable Traffic Patterns**: Per-device traffic control
  - ARP announcement intervals
  - Periodic ping intervals and payload sizes
  - Random traffic generation (packet count, intervals, patterns)
- **📡 SNMP Trap Generation**: SNMPv2c trap PDUs
  - Event-based traps (coldStart, linkUp/Down, authenticationFailure)
  - Threshold-based traps (high CPU, high memory, interface errors)
  - Multiple trap receivers with configurable thresholds
- **📝 Complete Protocol & YAML Work**: All protocol configuration complete
  - 19 protocols fully configurable via YAML
  - Per-device traffic and trap configuration
  - Comprehensive examples for all features
- **🎨 Enhanced Examples**: Updated complete-kitchen-sink.yaml with 9 devices showcasing all features

### Previous Features (v1.5.0)

- **🎨 Color-Coded Debug Output**: Protocol messages color-coded for better readability
- **🔧 Per-Protocol Debug Control**: 19 independent debug flags (--debug-arp, --debug-lldp, etc.)
- **🌐 Multiple IPs per Device**: Dual-stack (IPv4/IPv6) and multi-homed configurations
- **📝 Complete YAML Configuration**: All protocols configurable via YAML
  - Discovery Protocols: LLDP, CDP, EDP, FDP with custom values
  - Layer 2: STP bridge priority, timers, and versions
  - Application Services: HTTP endpoints, FTP users, NetBIOS names
  - Network Protocols: ICMP TTL, ICMPv6 hop limits, DHCPv6 pools
- **📚 Comprehensive Examples**: 20+ example files organized by use case
- **🔒 RFC Compliance**: ICMPv6 NDP always uses hop limit 255 per RFC 4861

### Core Features

✅ **Complete Protocol Stack** (19 protocols):
- **Layer 2**: ARP, STP, LLDP, CDP, EDP, FDP
- **Layer 3**: IPv4, IPv6, ICMP, ICMPv6
- **Layer 4**: TCP, UDP
- **Application**: HTTP, FTP, DNS, DHCP (v4/v6), NetBIOS, SNMP

✅ **Advanced Capabilities**:
- Interactive error injection mode with beautiful TUI
- Packet capture and injection (via gopacket/libpcap)
- YAML and legacy .cfg file parsing
- Thread-safe error state management
- Network interface detection
- Multiple error types (FCS, Discards, CPU, Memory, etc.)
- Real-time statistics
- SNMP agent with walk file support
- Comprehensive unit tests
- Performance benchmarks

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/krisarmstrong/niac-go
cd niac-go

# Build
go build -o niac ./cmd/niac

# Install (optional)
sudo cp niac /usr/local/bin/
```

### Requirements

- **Go**: 1.24+ for building
- **libpcap**: For packet capture
  - macOS: `brew install libpcap` (usually pre-installed)
  - Linux: `sudo apt-get install libpcap-dev`
  - Windows: WinPcap or Npcap

## Quick Start

### Interactive Mode

```bash
# Run with interactive error injection (Cobra subcommand)
sudo ./niac interactive en0 examples/layer2/lldp-only.yaml

# Legacy flag is still supported
sudo ./niac --interactive en0 examples/layer2/lldp-only.yaml

# Or try the complete example with all features
sudo ./niac --interactive en0 examples/complete-kitchen-sink.yaml

# Controls:
#   [i] - Open interactive menu
#   [s] - Toggle detailed statistics
#   [N]/[n] - Toggle neighbor discovery table
#   [x] - Toggle packet hex dump viewer
#   [c] - Clear all errors
#   [q] - Quit

# In menu:
#   [↑↓] - Navigate
#   [Enter] - Select
```

Both entrypoints launch the full simulator stack; the TUI now displays live packet statistics, the learned neighbor table (LLDP/CDP/EDP/FDP), and shares the same runtime as the standard CLI workflow.

### Web UI (v2 Preview)

The upcoming 2.x release introduces a React-based control plane that mirrors everything the CLI/TUI can do—statistics, neighbor insights, config management, and (soon) automation. The UI reuses the shared `@krisarmstrong/web-foundation` system that also powers `krisarmstrong-portfolio` and `wi-fi-vigilante`.

```bash
# Install once
cd webui
npm install

# Run the development server
npm run dev

# Build production assets (served by the Go API layer)
npm run build
```

The React app lives in `webui/` with Tailwind CSS, React Router, and the shared component library already wired up. During the v2.0 development cycle it ships alongside the existing CLI/TUI so you can pick whichever entrypoint fits the workflow.

### Basic Usage

```bash
# Validate configuration
./niac --dry-run lo0 examples/network/ipv4-only.yaml

# Run simulation
sudo ./niac en0 examples/vendors/cisco-network.yaml

# Run with verbose debug
sudo ./niac --verbose en0 examples/layer2/lldp-only.yaml

# Debug specific protocol only
sudo ./niac --debug 1 --debug-lldp 3 en0 examples/layer2/lldp-only.yaml
```

### Web UI & REST API

```bash
niac --api-listen :8080 --api-token supersecret en0 config.yaml
```

Visit `http://localhost:8080`, enter the token, and monitor packets, devices, history, and a live topology graph. The built-in YAML editor now talks to `/api/v1/config`, so edits run the same validation pipeline as `niac validate` before landing on disk. Packet replay and alert thresholds can also be managed directly from the Web UI via `/api/v1/replay` and `/api/v1/alerts`. Full API documentation lives in [`docs/REST_API.md`](docs/REST_API.md).

### Reloading Configuration

The 1.x runtime can apply config changes without restarting:

```bash
# Ask a running niac process to reload its config file
kill -HUP $(pgrep -f "niac .*en0")
```

Interactive mode also exposes the shortcut `[r]` to reload the active configuration and refresh the TUI. Behind the scenes both flows call the same hot-reload path that the Web UI/API use, so CLI, TUI, and browser all stay in sync.

### Metrics & Alerting

```bash
niac --api-listen :8080 \
     --metrics-listen :9090 \
     --alert-packets-threshold 100000 \
     --alert-webhook https://hooks.example.com/niac \
     en0 config.yaml
```

Prometheus metrics are exposed at `/metrics` and alerts are pushed to the optional webhook when the configured packet threshold is crossed.

### Run History Storage

NIAC automatically persists recent run metadata (interface, packets, errors) to a BoltDB database so the Web UI can show history. By default it is written to `~/.niac/niac.db`, but you can relocate or disable it:

```bash
# Store history alongside other simulator artifacts
niac --storage-path /var/lib/niac/history.db en0 config.yaml

# Opt out of persistence entirely (stateless containers, CI, etc.)
niac --storage-path disabled en0 config.yaml
```

When disabled, the API simply returns an empty history list, keeping both CLI and TUI behaviour consistent.

### Container & Kubernetes Deployment

Use the provided `Dockerfile`, `docker-compose.yml`, and `deploy/kubernetes/niac-deployment.yaml` manifests to run NIAC inside containers:

```bash
docker compose up --build
# or
kubectl apply -f deploy/kubernetes/niac-deployment.yaml
```

The sample manifests run NIAC on the container's loopback interface and expose the Web UI on port 8080.

### Help

```bash
./niac --help
```

Shows all 50+ command-line options including:
- Core flags: --debug, --verbose, --quiet, --interactive, --dry-run
- Information: --version, --list-interfaces, --list-devices
- Output: --no-color, --log-file, --stats-interval
- Per-protocol debug: --debug-arp, --debug-lldp, --debug-dhcpv6, etc.

### Generating Modern Walk Files

Need an SNMP walk for newer hardware? Use the built-in generator to create realistic synthetic walks for modern switches, firewalls, and routers:

```bash
# Discover the 18 supported vendor/model profiles
python3 scripts/generate_modern_walk.py --list

# Generate a specific device walk file
python3 scripts/generate_modern_walk.py \
  --vendor cisco \
  --model c9300-48p \
  --output examples/device_walks/generated/cisco-c9300.walk

# Use a custom hostname in the generated data
python3 scripts/generate_modern_walk.py \
  --vendor aruba \
  --model cx6300-48g \
  --hostname core-sw-01 \
  --output aruba-cx6300.walk
```

The resulting `.walk` files can be referenced from any device’s `snmp_agent.walk_file` field, right alongside the sanitized captures that ship with NiAC-Go.

## Configuration

### YAML Configuration (v1.5.0+)

NIAC-Go uses YAML for configuration with full support for all protocol options:

```yaml
devices:
  # Cisco router with LLDP and CDP
  - name: cisco-router-01
    mac: "00:1a:2b:3c:4d:01"
    ips:
      - "192.168.1.1"        # IPv4
      - "2001:db8::1"        # IPv6

    # Discovery protocols
    lldp:
      enabled: true
      system_description: "Cisco IOS 15.4"
      port_description: "GigabitEthernet0/0"

    cdp:
      enabled: true
      platform: "Cisco 2921"
      software_version: "IOS 15.4(3)M6a"

    # DHCP server
    dhcp:
      enabled: true
      pools:
        - network: "192.168.1.0/24"
          range_start: "192.168.1.100"
          range_end: "192.168.1.200"
          gateway: "192.168.1.1"
          dns_servers: ["8.8.8.8"]

    # ICMP configuration
    icmp:
      enabled: true
      ttl: 128              # Windows-like TTL
      rate_limit: 100

  # IPv6-only device
  - name: ipv6-server
    mac: "00:11:22:33:44:55"
    ips:
      - "2001:db8::100"

    dhcpv6:
      enabled: true
      pools:
        - network: "2001:db8::/64"
          range_start: "2001:db8::200"
          range_end: "2001:db8::2ff"
      preference: 255       # Highest priority
```

### Example Library

See `examples/` directory for 20+ organized examples:
- **Complete reference**: `complete-kitchen-sink.yaml` (all v1.6.0 features, 9 devices)
- **Traffic patterns** (v1.6.0): `traffic-patterns.yaml` - Configurable ARP, pings, random traffic
- **SNMP traps** (v1.6.0): `snmp-traps.yaml` - Event & threshold-based trap generation
- **Layer 2 protocols**: `layer2/lldp-only.yaml`, `layer2/stp-bridge.yaml`
- **DHCP servers**: `dhcp/dhcpv4-simple.yaml`, `dhcp/dhcpv4-advanced.yaml`
- **Services**: `services/dns-server.yaml`, `services/http-server.yaml`
- **Network configs**: `network/ipv4-only.yaml`, `network/dual-stack.yaml`
- **Vendor examples**: `vendors/cisco-network.yaml`

Full documentation: `examples/EXAMPLES-README.md`

### Topology & Protocol Analysis

```bash
# Export Graphviz topology from an SNMP walk
niac analyze-walk device.walk --graphviz topology.dot
dot -Tpng topology.dot -o topology.png

# Summarise a packet capture
niac analyze-pcap captures/demo.pcap --output json
```

## Development

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run benchmarks
go test ./pkg/... -bench=. -benchmem
```

### Benchmark Results (Apple M2)

```
BenchmarkSetError-8       	 7742178	  154.1 ns/op	  48 B/op	   3 allocs/op
BenchmarkGetError-8       	 6643556	  180.0 ns/op	 144 B/op	   4 allocs/op
BenchmarkGetAllStates-8   	10493102	  114.9 ns/op	 104 B/op	   2 allocs/op
BenchmarkParseSimpleConfig-8  816152	 1302 ns/op	 2024 B/op	  19 allocs/op
```

### Project Structure

```
niac-go/
├── cmd/niac/              # Main application entry point
├── pkg/
│   ├── capture/           # Packet capture & injection
│   ├── config/            # Configuration parsing
│   ├── errors/            # Error injection & state management
│   ├── interactive/       # Interactive TUI
│   ├── protocols/         # Network protocols (ARP, CDP, etc.)
│   └── snmp/              # SNMP agent
├── examples/              # Example configurations
└── README.md
```

## Reference Guides

Need deeper dives? The `docs/` directory ships with dedicated walkthroughs:

- [docs/TOPOLOGY_GUIDE.md](docs/TOPOLOGY_GUIDE.md) – Port-channels, trunk ports, VLAN-aware topologies, and LLDP/CDP neighbor design.
- [docs/ENVIRONMENTS.md](docs/ENVIRONMENTS.md) – Complete data center, enterprise campus, branch, wireless, and multi-vendor simulation scenarios.
- [docs/PROTOCOL_GUIDE.md](docs/PROTOCOL_GUIDE.md) – How to configure LLDP, CDP, DHCP, DNS, SNMP (agents + traps), STP variants, and more.
- [docs/API_REFERENCE.md](docs/API_REFERENCE.md) – The full YAML schema with every field, default, and validation rule.
- [docs/REST_API.md](docs/REST_API.md) – REST endpoints, Web UI, authentication, and alerting options.
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) – Common validation failures, debugging tips, and performance tuning advice.
- [docs/WALK_FILES.md](docs/WALK_FILES.md) – Vendor coverage, synthetic vs. sanitized walks, and contribution guidelines.

Each guide includes paste-ready YAML snippets and command-line workflows so you can adapt the scenarios to your lab quickly.

## Architecture

### Error Injection System

```
┌─────────────────────────────────────────────┐
│         Interactive TUI (Bubbletea)         │
│  • Real-time stats                          │
│  • Menu navigation                          │
│  • Error control                            │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│       StateManager (Thread-Safe)            │
│  • ConcurrentHashMap for device states     │
│  • Goroutine-safe operations               │
│  • 7 error types supported                 │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│       Packet Capture Engine (gopacket)      │
│  • Direct libpcap integration              │
│  • No JNI overhead                         │
│  • Rate limiting                           │
└─────────────────────────────────────────────┘
```

## Error Types

- **FCS Errors**: Frame Check Sequence errors
- **Packet Discards**: Dropped packets
- **Interface Errors**: Generic interface errors
- **High Utilization**: Interface bandwidth saturation
- **High CPU**: Device CPU load
- **High Memory**: Device memory usage
- **High Disk**: Device disk usage

## Why Rewrite?

### Java (Original)
- ✅ Mature, battle-tested (20+ years)
- ✅ Excellent libraries
- ❌ JVM overhead
- ❌ Slow startup
- ❌ Large memory footprint
- ❌ Deployment complexity

### Go (New)
- ✅ Native binary, instant startup
- ✅ Tiny memory footprint
- ✅ Simple deployment
- ✅ Excellent concurrency
- ✅ Modern tooling
- ✅ Fast compile times
- ⚠️  Need to rebuild protocol handlers

## Compatibility

NIAC-Go has achieved feature parity with NIAC-Java and surpassed it:

| Feature | Java | Go | Status |
|---------|------|-----|--------|
| Interactive Mode | ✅ | ✅ | **Complete** |
| Error Injection | ✅ | ✅ | **Complete** |
| Config Parsing | ✅ | ✅ | **Complete + YAML** |
| Packet Capture | ✅ | ✅ | **Complete** |
| SNMP Agent | ✅ | ✅ | **Complete** |
| Protocol Support | ✅ | ✅ | **Complete (19 protocols)** |
| Device Simulation | ✅ | ✅ | **Complete** |
| YAML Configuration | ❌ | ✅ | **Go Only** |
| Per-Protocol Debug | ❌ | ✅ | **Go Only** |
| Color Output | ❌ | ✅ | **Go Only** |
| Multi-IP Devices | ❌ | ✅ | **Go Only** |

## Contributing

Contributions welcome! This is a fun rewrite project to learn Go and modernize NIAC.

## License

Same as original NIAC project.

## Credits

- **Original NIAC**: Kevin Kayes (2002-2015)
- **Java Modernization & Go Rewrite**: Kris Armstrong (2025)

## Related Projects

- [NIAC (Java)](https://github.com/krisarmstrong/network-in-a-can) - Original Java implementation

---

**Built with ❤️ and Go** • Made for network engineers who love fast tools
