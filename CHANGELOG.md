# Changelog

All notable changes to NIAC-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Future (v1.6.0+)
- Additional traffic patterns (babble traffic customization)
- SNMP trap generation
- NetFlow/IPFIX export
- DHCPv6 prefix delegation (IA_PD)

## [1.5.0] - 2025-11-05

### 🎉 MILESTONE: Complete YAML Configuration System!

All protocols now fully configurable via YAML with per-protocol debug control and color-coded output.

### Added

#### Phase 1 Features
- **Color-coded debug output**:
  - Color-coded protocol messages for better readability
  - Support for NO_COLOR environment variable
  - `--no-color` flag to disable colors
  - Automatic color detection for terminals

- **Per-protocol debug level control**:
  - 19 protocol-specific debug flags (--debug-arp, --debug-lldp, --debug-dhcpv6, etc.)
  - Independent debug levels for each protocol (0-3)
  - Fallback to global debug level when protocol-specific not set
  - Comprehensive help output showing all debug flags

- **Multiple IPs per device**:
  - Devices can have multiple IPv4 and/or IPv6 addresses
  - Use `ips:` (plural) instead of `ip:` (singular) in YAML
  - Support for dual-stack (IPv4 + IPv6) configurations
  - Multi-homed devices (multiple IPs on different networks)
  - Example: `examples/multi-ip-devices.yaml`

#### Phase 2 Group 1 - Discovery Protocol YAML Configuration
- **LLDP Configuration** (IEEE 802.1AB):
  - `advertise_interval`: How often to send LLDP advertisements (default: 30s)
  - `ttl`: Time-to-live for LLDP information (default: 120s)
  - `system_description`: Device description string
  - `port_description`: Port/interface description
  - `chassis_id_type`: "mac" or "network_address"

- **CDP Configuration** (Cisco Discovery Protocol):
  - `advertise_interval`: Advertisement interval (default: 60s)
  - `holdtime`: Information holdtime (default: 180s)
  - `version`: CDP version (1 or 2)
  - `software_version`: Device software version string
  - `platform`: Platform/model string
  - `port_id`: Port identifier

- **EDP Configuration** (Extreme Discovery Protocol):
  - `advertise_interval`: Advertisement interval (default: 30s)
  - `version_string`: Software version
  - `display_string`: Device model/description

- **FDP Configuration** (Foundry Discovery Protocol):
  - `advertise_interval`: Advertisement interval (default: 60s)
  - `holdtime`: Information holdtime (default: 180s)
  - `software_version`: Device software version
  - `platform`: Platform/model string
  - `port_id`: Port identifier

#### Phase 2 Group 1b - STP YAML Configuration
- **STP Configuration** (Spanning Tree Protocol):
  - `enabled`: Enable/disable STP (default: false)
  - `bridge_priority`: Bridge priority 0-65535 (default: 32768)
  - `hello_time`: Hello BPDU interval in seconds (default: 2)
  - `max_age`: Maximum age in seconds (default: 20)
  - `forward_delay`: Forward delay in seconds (default: 15)
  - `version`: "stp", "rstp", or "mstp" (default: "stp")
  - Example: `examples/layer2/stp-bridge.yaml`

#### Phase 2 Group 2 - Application Protocol YAML Configuration
- **HTTP Server Configuration**:
  - `enabled`: Enable HTTP server
  - `server_name`: Server identification string
  - `endpoints`: Array of endpoint definitions
    - `path`: URL path (e.g., "/api/v1/status")
    - `method`: HTTP method (default: "GET")
    - `status_code`: HTTP status code (default: 200)
    - `content_type`: Response content type
    - `body`: Response body
  - Example: `examples/services/http-server.yaml`

- **FTP Server Configuration**:
  - `enabled`: Enable FTP server
  - `welcome_banner`: FTP welcome message (220 response)
  - `system_type`: System type string (e.g., "UNIX Type: L8")
  - `allow_anonymous`: Allow anonymous login (default: true)
  - `users`: Array of user accounts
    - `username`: Login username
    - `password`: Login password
    - `home_dir`: User home directory
  - Example: `examples/services/ftp-server.yaml`

- **NetBIOS Configuration**:
  - `enabled`: Enable NetBIOS name service
  - `name`: NetBIOS device name (max 15 characters)
  - `workgroup`: Workgroup/domain name
  - `node_type`: "B" (broadcast), "P" (point-to-point), "M" (mixed), "H" (hybrid)
  - `services`: Array of services ("workstation", "server", "browser", etc.)
  - `ttl`: Name registration TTL in seconds (default: 300)
  - Example: `examples/services/netbios-server.yaml`

#### Phase 2 Group 3 - Network Protocol YAML Configuration
- **ICMP Configuration**:
  - `enabled`: Enable ICMP echo reply (default: true)
  - `ttl`: Time To Live for ICMP packets (default: 64)
    - Common values: 32 (old systems), 64 (Linux/Unix), 128 (Windows), 255 (routers)
  - `rate_limit`: Max ICMP responses per second (default: 0 = unlimited)
  - Example: `examples/network/icmp-config.yaml`

- **ICMPv6 Configuration**:
  - `enabled`: Enable ICMPv6 echo reply (default: true)
  - `hop_limit`: Hop limit for ICMPv6 packets (default: 64)
    - NDP packets ALWAYS use hop limit 255 per RFC 4861 (security requirement)
  - `rate_limit`: Max ICMPv6 responses per second (default: 0 = unlimited)
  - Example: `examples/network/icmpv6-config.yaml`

- **DHCPv6 Server Configuration**:
  - `enabled`: Enable DHCPv6 server (default: false)
  - `pools`: IPv6 address pools
    - `network`: IPv6 network in CIDR notation
    - `range_start`: First address in pool
    - `range_end`: Last address in pool
  - `preferred_lifetime`: Preferred address lifetime in seconds (default: 604800 = 7 days)
  - `valid_lifetime`: Valid address lifetime in seconds (default: 2592000 = 30 days)
  - `preference`: Server preference 0-255 (default: 0, higher is better)
  - `dns_servers`: Array of IPv6 DNS server addresses
  - `domain_list`: Array of DNS search domains
  - `sntp_servers`: Array of SNTP time server addresses
  - `ntp_servers`: Array of NTP server addresses
  - `sip_servers`: Array of SIP server addresses
  - `sip_domains`: Array of SIP domain names
  - Example: `examples/network/dhcpv6-config.yaml`

### Changed
- Protocol handlers now read configuration from device config structs
- ICMP handler uses configurable TTL instead of hardcoded 64
- ICMPv6 handler uses configurable hop limit with RFC 4861 compliance for NDP
- DHCPv6 handler uses configurable server preference
- Discovery protocol handlers use configurable advertisement intervals and values
- STP handler uses configurable bridge priority and timers
- HTTP/FTP/NetBIOS handlers use configurable server parameters

### Documentation
- Created organized example library in `examples/` directory:
  - `examples/EXAMPLES-README.md` - Complete documentation of all examples
  - `examples/complete-kitchen-sink.yaml` - Master example with ALL features
  - `examples/layer2/` - Discovery protocol examples (LLDP, CDP, EDP, FDP, STP)
  - `examples/dhcp/` - DHCP server examples (simple and advanced)
  - `examples/services/` - Application service examples (DNS, HTTP, FTP, NetBIOS)
  - `examples/network/` - Network protocol examples (IPv4, IPv6, dual-stack, ICMP, ICMPv6, DHCPv6)
  - `examples/vendors/` - Vendor-specific examples (Cisco, Extreme, Foundry)
- Updated all example files with comprehensive inline documentation
- Added troubleshooting sections to example files
- Documented all configuration options with defaults and valid ranges

### Technical Details
- Added ICMPConfig, ICMPv6Config, DHCPv6Config structs to pkg/config/config.go
- Added STPConfig, HTTPConfig, FTPConfig, NetBIOSConfig structs
- Updated all YAML parsing in internal/converter/converter.go
- Enhanced config loader with default values for all new options
- Backward compatible - existing configs without new options still work

## [1.4.0] - 2025-01-05

### 🎉 MILESTONE: Complete DHCP/DNS Implementation!

Full-featured DHCP and DNS servers with comprehensive option support.

### Added
- **Complete DHCPv4 Implementation (15 options)**:
  - Basic options: Subnet Mask (1), Router (3), DNS Servers (6), Domain Name (15)
  - Lease management: Lease Time (51), T1 Renewal (58), T2 Rebinding (59)
  - Server identification: Server Identifier (54), Message Type (53)
  - **New High Priority Options**:
    - Hostname (Option 12) - Automatic capture and echo from client requests
    - NTP Servers (Option 42) - Time synchronization
    - Domain Search List (Option 119) - Multiple DNS search domains with RFC 1035 encoding
    - TFTP Server Name (Option 66) - PXE boot support
    - Bootfile Name (Option 67) - Boot image filename for PXE
    - Vendor-Specific Info (Option 43) - Custom vendor data
  - Static DHCP leases with MAC address masks for wildcard matching
  - Configurable via YAML with full end-to-end integration

- **Complete DHCPv6 Implementation (12 options)**:
  - Basic options: Client/Server ID (DUID), IA_NA, IA_Addr, Preference
  - DNS: DNS Servers (23), Domain Search List (24)
  - **New High Priority Options**:
    - SNTP Servers (Option 31) - Simple time synchronization
    - NTP Servers (Option 56) - Full NTP configuration
    - SIP Server Addresses (Option 22) - VoIP IPv6 addresses
    - SIP Domain Names (Option 21) - VoIP domain names
    - FQDN (Option 39) - Client fully qualified domain name
  - Configurable via YAML with full end-to-end integration

- **DNS Server Implementation**:
  - Forward DNS records (A records) - hostname → IPv4
  - Reverse DNS records (PTR records) - IPv4 → hostname
  - Configurable TTL per record
  - Multiple records per device
  - Full YAML configuration support

- **Complete YAML Configuration Support**:
  - All DHCP options loadable from YAML configuration files
  - DNS records configurable in device YAML
  - End-to-end integration: YAML → config parser → runtime → protocol handlers
  - Example configuration: `examples/scenario_configs/complete-reference.yaml`
  - Comprehensive documentation: `examples/scenario_configs/README-complete-reference.md`

- **Example Configuration with 12 Device Types**:
  - Core Router (Cisco 2821) with full DHCP/DNS/SNMP
  - Distribution Switch (Cisco Catalyst 3750)
  - Access Switch (Cisco 2960)
  - Wireless AP (Cisco Aironet)
  - Linux Server (Ubuntu)
  - Juniper Router (multi-vendor support)
  - NetGear Switch (SMB device)
  - VoIP Phone (Cisco IP Phone)
  - Network Printer (HP LaserJet)
  - NAS Storage (Synology DiskStation)
  - Security Camera (Axis)
  - Dual-Stack Server (IPv4/IPv6)

### Changed
- DHCP handler now supports advanced option configuration via `SetAdvancedOptions()`
- DHCPv6 handler now supports advanced option configuration via `SetAdvancedOptions()`
- DNS handler now supports dynamic record addition via `AddRecord()`
- Config loader enhanced with complete DHCP/DNS parsing (lines 374-496 in config.go)
- Main entry point now configures all handlers from YAML (lines 390-440 in main.go)

### Technical Details
- Added DHCPConfig and DNSConfig structs to runtime configuration
- Implemented RFC 1035 DNS label encoding for domain search lists
- Added accessor methods to Stack for protocol handler configuration
- Hostname automatically captured from DHCP requests and echoed in responses
- Vendor-specific data stored as hex strings in YAML, converted to bytes at runtime

### Documentation
- Created comprehensive reference YAML (658 lines) with all features
- Added complete feature documentation with examples and troubleshooting
- Updated all documentation files to proper locations
- Organized planning documents in docs/ folder

## [1.3.0] - 2025-01-05

### Added
- **Discovery Protocol Support (4 protocols)**:
  - LLDP (Link Layer Discovery Protocol) - IEEE 802.1AB
  - CDP (Cisco Discovery Protocol)
  - EDP (Extreme Discovery Protocol)
  - FDP (Foundry Discovery Protocol)
  - All protocols configurable via YAML
  - Periodic advertisement transmission
  - Neighbor discovery and tracking

## [1.2.0] - 2025-01-05

### 🎉 MILESTONE: 100% Protocol Coverage Achieved!

All 13 network protocols now fully implemented - complete feature parity with Java NIAC.

### Added
- **IPv6 and ICMPv6 Protocol Support** (678 lines):
  - Complete IPv6 packet handling with extension header chain walking
  - ICMPv6 Echo Request/Reply (ping6)
  - Neighbor Discovery Protocol (NDP) with Neighbor Solicitation/Advertisement
  - Router Solicitation handling
  - IPv6 multicast MAC mapping per RFC 2464 (33:33:xx:xx:xx:xx)
  - IPv6 pseudo-header checksum calculation
  - Device config parser now accepts "ipv6" keyword
  - Comprehensive unit test coverage
- **NetBIOS Protocol Support** (536 lines):
  - NetBIOS Name Service (NBNS) on UDP port 137
  - NetBIOS Datagram Service (NBDS) on UDP port 138
  - NetBIOS name encoding/decoding (first-level encoding)
  - Support for all name types (workstation, file server, browser, master, etc.)
  - Device name matching against NetBIOS queries
  - Full RFC 1001/1002 compliance
- **Spanning Tree Protocol Support** (509 lines):
  - STP Configuration BPDU handling
  - Topology Change Notification (TCN) BPDU processing
  - Bridge ID management (priority + MAC address)
  - BPDU transmission for simulated switches/bridges
  - Port state tracking (Disabled, Blocking, Listening, Learning, Forwarding)
  - RSTP support with port roles and rapid convergence flags
  - IEEE 802.1D and 802.1w compliance
  - Multicast MAC address handling (01:80:C2:00:00:00)

### Changed
- Protocol stack dispatcher now handles STP via multicast MAC detection
- UDP handler routes NetBIOS packets to appropriate ports (137, 138)
- Device table enhanced with GetByIPv6() and GetAll() methods

### Complete Protocol Suite (13/13)
1. ✅ ARP (Address Resolution Protocol)
2. ✅ IP (Internet Protocol v4)
3. ✅ ICMP (Internet Control Message Protocol)
4. ✅ IPv6 (Internet Protocol v6) **NEW**
5. ✅ ICMPv6 (ICMP for IPv6) **NEW**
6. ✅ UDP (User Datagram Protocol)
7. ✅ TCP (Transmission Control Protocol)
8. ✅ DNS (Domain Name System)
9. ✅ DHCP (Dynamic Host Configuration Protocol)
10. ✅ HTTP (Hypertext Transfer Protocol)
11. ✅ FTP (File Transfer Protocol)
12. ✅ NetBIOS (Network Basic Input/Output System) **NEW**
13. ✅ STP (Spanning Tree Protocol) **NEW**

### Performance
- Total lines added: ~1,723 lines across 9 new files
- All unit tests passing (100% test coverage maintained)
- No performance degradation with additional protocols

## [1.1.0] - 2025-01-05

### Added
- **Enhanced CLI**:
  - `--version` flag with detailed build information
  - `--list-interfaces` to show available network interfaces
  - `--list-devices` to display device table from config file
  - `--dry-run` for configuration validation without starting
  - `--verbose` / `-v` shortcut for debug level 3
  - `--quiet` / `-q` shortcut for debug level 0
  - Additional output flags: `--no-color`, `--log-file`, `--stats-interval`
  - Advanced flags: `--babble-interval`, `--no-traffic`, `--snmp-community`, `--max-packet-size`
  - Improved help text with comprehensive examples
  - Beautiful banner on startup
- **Interactive Mode Enhancements**:
  - Debug level now displayed in status bar
  - `[d]` key for debug level cycling (0→1→2→3→0)
  - `[h]` and `[?]` keys for comprehensive help overlay
  - `[l]` key for debug log viewer (shows last 10 logs)
  - `[s]` key for detailed statistics viewer
  - Debug logging system (keeps last 100 entries)
  - Timestamped log entries
  - Enhanced status messages
  - Updated controls display in footer

### Changed
- Version bumped to 1.1.0
- Status bar now shows: "Debug: X (LEVELNAME)"
- Interactive mode initial message now includes help hint
- All error injections and actions are now logged

## [1.0.0] - 2025-01-05

### Added
- Initial production release of NIAC-Go
- Complete protocol stack implementation:
  - ARP (Address Resolution Protocol)
  - IP (Internet Protocol v4)
  - ICMP (Internet Control Message Protocol)
  - TCP (Transmission Control Protocol)
  - UDP (User Datagram Protocol)
  - HTTP (HyperText Transfer Protocol) with multiple endpoints
  - FTP (File Transfer Protocol) with 17 commands
  - DNS (Domain Name System) - stub implementation
  - DHCP (Dynamic Host Configuration Protocol) - stub implementation
- SNMP agent with full functionality:
  - GET operations
  - GET-NEXT operations
  - GET-BULK operations
  - Community string authentication
  - MIB-II system group support
  - Walk file import/export
  - Dynamic OIDs (sysUpTime, etc.)
- Interactive error injection mode:
  - Beautiful terminal UI using Bubbletea
  - 7 error types (FCS errors, packet discards, interface errors, high utilization, high CPU, high memory, high disk)
  - Real-time error injection via keyboard
  - Interface configuration (speed, duplex)
  - Statistics display
- Device behavior simulation:
  - Per-device state management (up, down, starting, stopping, maintenance)
  - Type-specific behavior (router, switch, AP, server)
  - Device counters for all protocol types (10 counter types)
  - Periodic behavior loops (every 30 seconds)
  - SNMP agent per device
- Network traffic generation:
  - Gratuitous ARP announcements (every 60 seconds)
  - Periodic pings between devices (every 120 seconds)
  - Random traffic patterns (every 180 seconds):
    - Broadcast ARP requests
    - Multicast packets
    - Random UDP traffic
- Configuration file parser:
  - Compatible with Java NIAC config file format
  - Device properties (name, type, IP, MAC, SNMP settings)
  - Interface configuration
  - SNMP walk file loading
- Comprehensive test suite:
  - 23 unit tests covering all major components
  - Config parsing tests
  - Error injection tests
  - Protocol stack tests
  - 100% test pass rate
- Complete documentation:
  - README with usage instructions
  - FINAL_SUMMARY with all features and statistics
  - PROGRESS_REPORT with development timeline
  - JAVA_VS_GO_VALIDATION with detailed comparison

### Performance
- Binary size: 6.1 MB (2.6x smaller than Java + JRE)
- Startup time: ~5ms (10x faster than Java)
- Memory usage: ~15MB (6.7x less than Java)
- Error injection: 7.7M ops/sec (77x faster than Java)
- Config parsing: ~1.3µs (770x faster than Java)
- Build time: ~5 seconds (48-60x faster than Java)
- Code size: 6,216 lines (3.3x less than Java's 20,380 lines)

### Notes
- First production-ready release
- Feature parity with Java NIAC on all core protocols
- Four major enhancements over Java:
  1. Advanced HTTP server (vs Java's "Yo Dude" response)
  2. Complete FTP server (not present in Java)
  3. Advanced device simulation with state management
  4. Comprehensive traffic generation (3 patterns vs Java's basic babble)
- Compatible with all Java NIAC configuration files and SNMP walk files
- Modern architecture using Go idioms (goroutines, channels, clean packages)

[Unreleased]: https://github.com/krisarmstrong/niac-go/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/krisarmstrong/niac-go/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/krisarmstrong/niac-go/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/krisarmstrong/niac-go/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/krisarmstrong/niac-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/krisarmstrong/niac-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/krisarmstrong/niac-go/releases/tag/v1.0.0
