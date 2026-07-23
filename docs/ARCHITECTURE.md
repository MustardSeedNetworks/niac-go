# NIAC-Go Architecture

This document describes the architectural design of NIAC-Go, a network device simulator written in Go.

## Table of Contents
- [Overview](#overview)
- [Design Principles](#design-principles)
- [Package Structure](#package-structure)
- [Data Flow](#data-flow)
- [Protocol Handler Architecture](#protocol-handler-architecture)
- [Configuration System](#configuration-system)
- [Error Injection System](#error-injection-system)
- [Concurrency Model](#concurrency-model)
- [Extension Points](#extension-points)

---

## Overview

NIAC-Go simulates network devices by:
1. Capturing packets from a network interface (libpcap)
2. Processing packets through protocol handlers
3. Generating appropriate responses
4. Supporting interactive error injection for testing

```
┌─────────────┐
│   Network   │
│  Interface  │
└──────┬──────┘
       │
       v
┌──────────────────┐
│ Capture Engine   │  (gopacket/libpcap)
│  (pkg/capture)   │
└────────┬─────────┘
         │
         v
┌──────────────────┐
│ Protocol Stack   │
│ (pkg/protocols)  │
├──────────────────┤
│ • ARP Handler    │
│ • LLDP Handler   │
│ • SNMP Handler   │
│ • DHCP Handler   │
│ • ... 19 total   │
└────────┬─────────┘
         │
         v
┌──────────────────┐
│ Device Simulator │
│  (pkg/device)    │
└────────┬─────────┘
         │
         v
┌──────────────────┐
│  Error Injection │
│  (pkg/errors)    │
└──────────────────┘
```

---

## Design Principles

### 1. **Modularity**
Each protocol is independent. Adding new protocols doesn't require modifying existing ones.

### 2. **Concurrency-Safe**
All shared state (error injection, SNMP agent) uses proper synchronization.

### 3. **Performance**
- Zero-copy packet handling where possible
- Lock-free reads for hot paths
- Efficient goroutine usage

### 4. **Configurability**
Everything configurable via YAML: protocols, addresses, behaviors, error injection.

### 5. **Testability**
Packages are loosely coupled. Mock interfaces for testing without real network access.

---

## Package Structure

### cmd/niac
**Purpose**: CLI application entry point

**Files**:
- `main.go` - Program entry point (calls Execute())
- `root.go` - Cobra root command and version info
- `legacy.go` - Positional CLI mode
- `validate.go` - Config validation command
- `template.go` - Template management commands
- `interactive.go` - Interactive TUI command
- `config.go` - Config management (export/diff/merge)
- `generate.go` - Interactive config generator
- `init.go` - Template selection wizard
- `completion.go` - Shell completion
- `man.go` - Man page generation

**Responsibilities**:
- Parse command-line arguments and flags (Cobra)
- Load and validate YAML/legacy configuration files
- Initialize capture engine (pkg/capture)
- Start protocol stack (pkg/protocols)
- Handle signals (Ctrl+C, proper shutdown with WaitGroups)
- Manage template system (embedded templates)
- Config file operations (export, diff, merge, generate)
- Shell completion generation (bash, zsh, fish, powershell)
- Man page generation for documentation

---

### pkg/capture
**Purpose**: Low-level packet capture and injection

**Key Types**:
```go
type Engine struct {
    interfaceName string
    handle        *pcap.Handle
    debugLevel    int
}
```

**Responsibilities**:
- Open network interface with libpcap (100ms timeout for responsive shutdown)
- Read packets from wire (non-blocking with timeout handling)
- Send packets to wire (raw Ethernet frames)
- BPF filtering support (SetFilter)
- Rate limiting (RateLimiter with token bucket algorithm)
- Packet statistics (via pcap.Stats)
- Proper cleanup and shutdown (Close method, WaitGroup coordination)

**Dependencies**: `gopacket`, `gopacket/pcap`

**Key Changes** (v1.21.1):
- Fixed Ctrl+C hang by using 100ms timeout instead of BlockForever
- Added comprehensive shutdown tests (capture_shutdown_test.go)
- RateLimiter now properly cleans up goroutines with done channel

---

### pkg/config
**Purpose**: Configuration file parsing

**Key Types**:
```go
type Config struct {
    Devices []Device
    IncludePath string
    CapturePlayback *CapturePlayback
}

type Device struct {
    Name string
    MACAddress net.HardwareAddr
    IPAddresses []net.IP
    SNMPConfig SNMPConfig
    LLDPConfig *LLDPConfig
    // ... 19 protocol configs
}
```

**Responsibilities**:
- Load YAML configurations (primary format)
- Convert Java-DSL `.cfg` input to the canonical YAML model
- Validate device configurations (Validator with 3 severity levels)
- Resolve walk file paths (with security checks against path traversal)
- Config comparison and diff operations
- Config merging with overlay semantics
- Template management (embedded in binary)

**File References**:
- `config.go` - Core configuration structures and loading
- `validator.go` - Comprehensive validation with errors, warnings, info
- `legacy_converter.go` - Legacy .cfg to YAML conversion

**Key Features**:
- Validation with line/column tracking for precise error location
- JSON output support for CI/CD integration
- Three-level validation (error, warning, info)
- Device validation (names, types, MAC/IP duplicates)
- Protocol-specific validation (19 protocols)
- SNMP trap validation (receivers and supported state triggers)
- DNS record validation
- DHCPv4/v6 pool validation (PoolStart, PoolEnd added in v1.19.0)

---

### pkg/protocols
**Purpose**: Protocol packet handlers

**Key Types**:
```go
type Stack struct {
    engine *capture.Engine
    config *config.Config
    handlers []Handler
}

type Handler interface {
    HandlePacket(packet gopacket.Packet, device *config.Device) error
}
```

**Protocol Handlers**:
- `arp.go` - ARP request/reply
- `lldp.go` - LLDP advertisements
- `cdp.go` - Cisco Discovery Protocol
- `edp.go` - Extreme Discovery Protocol
- `fdp.go` - Foundry Discovery Protocol
- `stp.go` - Spanning Tree Protocol
- `dhcp.go` - DHCPv4 server
- `dhcpv6.go` - DHCPv6 server (993 lines, complex)
- `dns.go` - DNS server
- `http.go` - HTTP server
- `ftp.go` - FTP server
- `netbios.go` - NetBIOS name service
- `snmp.go` - SNMP agent integration
- `icmp.go`, `icmpv6.go` - Ping responses

**Responsibilities**:
- Parse incoming packets
- Match packets to simulated devices
- Generate protocol-appropriate responses
- Handle per-protocol configuration

**Key Patterns**:
- Each handler is stateless (except SNMP agent)
- Handlers registered in `stack.go`
- Concurrent packet processing with goroutine-safe state
- Proper shutdown coordination with WaitGroup and stop channels

**Shutdown Architecture** (v1.21.1):
- Stack maintains WaitGroup for all goroutines
- Stop channel signals graceful shutdown
- Stop() method is idempotent (safe to call multiple times)
- Waits for all protocol handlers to complete
- Comprehensive tests in stack_shutdown_test.go

**Key Methods**:
- `Start()` - Begins packet capture and protocol handling
- `Stop()` - Graceful shutdown with WaitGroup coordination
- `initializeDevices()` - Sets up device registry from config

---

### pkg/snmp
**Purpose**: SNMP agent implementation

**Key Types**:
```go
type Agent struct {
    community string
    walkData map[string]string
    mu sync.RWMutex
}
```

**Responsibilities**:
- Load SNMP walk files (standard snmpwalk format)
- Respond to GET/GETNEXT/GETBULK requests
- Generate SNMP traps (v1.6.0+)
- Error injection integration
- Configurable community strings (v1.19.0)

**Walk File Format**: Standard `snmpwalk` output

**Trap Generation** (v1.6.0+):
- State-based traps: coldStart, linkUp, and linkDown
- Multiple trap receivers
- Configurable trap community string (default: "public")

---

### Device Fault State

**Purpose**: Keep injected interface faults in the same authoritative state
store used by management, forwarding, discovery, and SNMP.

Supported faults are FCS errors, packet discards, interface errors, and high
utilization. Multiple fault types can be active on the same interface.

**Flow**:

1. The API or TUI resolves a running device and authored interface.
2. `devicestate.Store` records the fault and emits a state event.
3. Shared per-device SNMP telemetry advances the corresponding IF-MIB,
   IF-MIB-64, and EtherLike-MIB counters.
4. Clearing a fault stops future increments without resetting accumulated
   counters.

**File References**: `internal/devicestate/store_fault.go`,
`internal/protocols/stack_fault.go`, and
`internal/protocols/snmp/fault_telemetry.go`.

---

### pkg/interactive
**Purpose**: Terminal UI for interactive control

**Key Types**:
```go
type model struct {
    cfg *config.Config
    stack *protocols.Stack
    menuVisible bool
    menuItems []string
    selectedItem int
}
```

**Responsibilities**:
- Display device status
- Interactive error injection menu
- Real-time statistics
- Debug log viewer

**Framework**: Bubble Tea (charmbracelet/bubbletea)

---

### pkg/templates
**Purpose**: Embedded configuration templates

**Responsibilities**:
- Embed template files in binary using go:embed
- Template discovery and listing
- Template retrieval by name
- 7 production-ready templates: minimal, router, switch, ap, server, iot, complete

**Templates**:
- `minimal.yaml` - Single device with basic protocols
- `router.yaml` - Enterprise router with full protocol support
- `switch.yaml` - Layer 2/3 switch with STP and VLAN
- `ap.yaml` - Wi-Fi access point
- `server.yaml` - Multi-service server (DHCP, DNS, HTTP)
- `iot.yaml` - Lightweight IoT sensor device
- `complete.yaml` - Multi-device network simulation

**Usage**:
```go
// List all templates
templates := templates.List()

// Get specific template
content, err := templates.Get("router")
```

---

### pkg/logging
**Purpose**: Colored console output and debug control

**Key Types**:
```go
type DebugConfig struct {
    Global int // 0=quiet, 1=normal, 2=verbose, 3=debug
    protocols map[string]int // Per-protocol override
    mu sync.RWMutex
}
```

**Responsibilities**:
- Color-coded output (red=error, green=success, cyan=protocol)
- Per-protocol debug levels
- Respects NO_COLOR environment variable

**Example**:
```go
logging.Protocolf("LLDP", "Sent advertisement from %s", device.Name)
logging.Errorf("Failed to parse config: %v", err)
```

---

## Data Flow

### Packet Reception Flow

```
Network Interface
       ↓
[gopacket Capture]
       ↓
[PacketSource chan]
       ↓
[Stack.Start() goroutine]
       ↓
for packet := range packets {
       ↓
  [Match device by MAC/IP]
       ↓
  [Route to handler based on EtherType/IP Protocol]
       ↓
  [Handler.HandlePacket(packet, device)]
       ↓
  [Check ErrorStateManager]
       ↓
  [Generate Response]
       ↓
  [Engine.SendPacket()]
}
       ↓
Network Interface
```

### Configuration Loading Flow

```
User runs: niac en0 config.yaml
       ↓
[main.go] Parse flags
       ↓
[config.Load(filename)]
       ↓
Detect file extension
       ↓
.yaml? → [LoadYAML()]
       ↓
[converter.LoadYAMLConfig()] Parse YAML
       ↓
[converter.ValidateConfig()] Validate structure
       ↓
[Convert to runtime Config struct]
       ↓
For each device:
  - Parse MAC/IP addresses
  - Load protocol configs (LLDP, CDP, SNMP, etc.)
  - Validate walk file paths (security check)
  - Apply defaults
       ↓
Return *Config
       ↓
[Initialize Protocol Stack]
       ↓
[Start Simulation]
```

---

## Protocol Handler Architecture

### Handler Interface

```go
type Handler interface {
    // HandlePacket processes an incoming packet for a device
    HandlePacket(packet gopacket.Packet, device *config.Device) error
}
```

### Handler Registration

In `pkg/protocols/stack.go`:
```go
func NewStack(engine *capture.Engine, config *config.Config) *Stack {
    s := &Stack{
        engine: engine,
        config: config,
    }

    // Register handlers
    s.RegisterHandler(&ARPHandler{})
    s.RegisterHandler(&LLDPHandler{})
    s.RegisterHandler(&CDPHandler{})
    // ... etc

    return s
}
```

### Example: ARP Handler

```go
type ARPHandler struct{}

func (h *ARPHandler) HandlePacket(packet gopacket.Packet, device *config.Device) error {
    arpLayer := packet.Layer(layers.LayerTypeARP)
    if arpLayer == nil {
        return nil // Not an ARP packet
    }

    arp := arpLayer.(*layers.ARP)

    // Only respond to requests for our device's IP
    if arp.Operation == layers.ARPRequest &&
       bytes.Equal(arp.DstProtAddress, device.IPAddresses[0].To4()) {

        // Send ARP reply
        h.sendARPReply(...)
    }

    return nil
}
```

### Adding New Protocol

1. Create `pkg/protocols/yourprotocol.go`
2. Implement Handler interface
3. Register in `NewStack()`
4. Add config support in `pkg/config/config.go`
5. Add tests `pkg/protocols/yourprotocol_test.go`

---

## Configuration System

### YAML Structure

```yaml
devices:
  - name: router-01
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
      - "2001:db8::1"

    lldp:
      enabled: true
      advertise_interval: 30
      system_description: "Cisco IOS 15.4"

    snmp_agent:
      walk_file: "walks/cisco-router.snmpwalk"
      traps:
        enabled: true
        receivers: ["192.168.1.10:162"]
        link_state:
          enabled: true
          link_down: true
          link_up: true
```

### Security: Path Validation

**Critical**: Walk file paths validated to prevent path traversal attacks.

```go
// pkg/config/config.go:1377
func validateWalkFilePath(basePath, walkFile, deviceName string) (string, error) {
    cleanPath := filepath.Clean(walkFile)

    // Security: Prevent directory traversal
    if strings.Contains(cleanPath, "..") {
        return "", fmt.Errorf("invalid path traversal")
    }

    // Verify file exists and is regular file
    // ... validation logic
}
```

---

## Error Injection System

### Architecture

```text
Web UI / REST API / TUI
          |
          v
protocols.Stack target validation
          |
          v
devicestate.Store active interface faults
          |
          v
shared SNMP ProtocolTelemetry counters
```

### Usage Example

```go
err := stack.SetInterfaceFault(
    "192.168.1.1",
    "GigabitEthernet0/1",
    devicestate.FaultFCS,
    25,
)
```

---

## Concurrency Model

### Goroutines

1. **Main goroutine**: CLI, config loading, signal handling
2. **Packet capture goroutine**: `engine.StartCapture()` - one per interface
3. **Protocol ticker goroutines**: LLDP/CDP advertisements (one per device)
4. **Management event dispatcher**: Coalesced state-change wakeups with lossless event-log replay
5. **TUI goroutine**: Bubble Tea event loop (interactive mode only)

### Thread Safety

- **Device state store**: `sync.RWMutex` for identity, network, routes, and faults
- **DebugConfig**: `sync.RWMutex` for debug levels
- **SNMP Agent**: `sync.RWMutex` for walk data
- **Packet handlers**: Stateless, no shared state

---

## Extension Points

### Adding New Commands

Cobra commands in `cmd/niac/`:
1. Create `yourcommand.go`
2. Define `cobra.Command`
3. Register in `root.go:init()`

### Adding New Configuration Options

1. Add field to `config.Device` struct
2. Add parsing in `LoadYAML()`
3. Add validation in `Validator`
4. Update examples in `examples/`

### Custom Protocol Handlers

Implement `Handler` interface and register in `NewStack()`.

### Error Injection Types

Add new `ErrorType` constant in `pkg/errors/types.go` and handle in SNMP response logic.

---

## Performance Considerations

### Hot Path Optimization

- Packet matching uses map lookups (O(1))
- Lock-free reads with RWMutex (multiple readers)
- Minimal allocations in packet handlers

### Benchmarks

See `pkg/config/config_test.go`, `pkg/errors/state_test.go` for benchmark examples.

Current performance (Apple M2):
- Config parsing: ~1.3µs (770x faster than Java)
- Error injection: 7.7M ops/sec (77x faster than Java)
- Packet handling: <1µs per packet

---

## Future Architecture (v2.0.0)

### Planned: Service Layer

```
┌──────────────┐
│  REST API    │
│  (HTTP/JSON) │
└──────┬───────┘
       │
       v
┌──────────────┐
│ Application  │  Service layer
│   Layer      │  (pkg/app)
└──────┬───────┘
       │
       v
┌──────────────┐
│ Protocol     │
│   Stack      │
└──────────────┘
```

Benefits:
- Multiple UIs (CLI, TUI, Web) share same backend
- Testability without network access
- Clear separation of concerns

---

## Troubleshooting

### "No such interface"
Check: `niac --list-interfaces`

### SNMP walk file not loading
Check: Walk file path validation (logs show reason)

### Performance degradation
Check: Goroutine count (`runtime.NumGoroutine()`), possible leak

### Packet not being handled
Add: `--debug 3 --debug-<protocol> 3` for full trace

---

## References

- **Go**: https://golang.org/doc/effective_go
- **gopacket**: https://pkg.go.dev/github.com/google/gopacket
- **Bubble Tea**: https://github.com/charmbracelet/bubbletea
- **Cobra**: https://github.com/spf13/cobra

---

**Last Updated**: November 14, 2025
**Version**: v2.3.1
**Maintainer**: Kris Armstrong

---

## Recent Changes

### v1.21.1 - Bug Fixes
- Fixed Ctrl+C hang (100ms pcap timeout instead of BlockForever)
- Fixed simulator restart bug (proper WaitGroup shutdown coordination)
- Fixed DHCP broadcast packet handling (255.255.255.255)
- Added configurable DHCP pools (PoolStart, PoolEnd)
- Added configurable SNMP community strings

### v1.21.2 - Testing & Documentation
- Added comprehensive tests for config commands (13 new tests)
- Updated CLI documentation for all commands
- Documented config export, diff, merge, generate commands
- Documented init, completion, and man commands

### v1.21.3 - Architecture & Coverage
- Updated architecture documentation to reflect current implementation
- Added shutdown architecture details
- Documented new command structure
- Improved core package test coverage (in progress)
