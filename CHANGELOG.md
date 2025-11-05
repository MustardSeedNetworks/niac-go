# Changelog

All notable changes to NIAC-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Future (v1.2.0+)
- IPv6 and ICMPv6 protocol support
- NetBIOS protocol support
- Spanning Tree Protocol (STP) support
- Per-protocol debug level control
- Color-coded debug output

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

[Unreleased]: https://github.com/krisarmstrong/niac-go/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/krisarmstrong/niac-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/krisarmstrong/niac-go/releases/tag/v1.0.0
