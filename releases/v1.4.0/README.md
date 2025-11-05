# NIAC-Go v1.4.0 Release

**Release Date:** January 5, 2025
**Git Commit:** 56c3624

## Downloads

### Main Binary (niac)
- **macOS ARM64** (Apple Silicon): `niac-1.4.0-darwin-arm64` (7.3 MB)

**Note:** The main `niac` binary requires libpcap and must be built on the target platform due to CGO dependencies. For Linux and Windows builds, compile from source on the target platform or use Docker.

### Configuration Converter (niac-convert)
Cross-platform compatible (no CGO dependencies):
- **Linux AMD64**: `niac-convert-1.4.0-linux-amd64` (3.3 MB)
- **Linux ARM64**: `niac-convert-1.4.0-linux-arm64` (3.3 MB)
- **macOS Intel**: `niac-convert-1.4.0-darwin-amd64` (3.3 MB)
- **macOS ARM64**: `niac-convert-1.4.0-darwin-arm64` (3.3 MB)
- **Windows AMD64**: `niac-convert-1.4.0-windows-amd64.exe` (3.4 MB)

## Installation

### macOS (ARM64)
```bash
# Download and install main binary
chmod +x niac-1.4.0-darwin-arm64
sudo mv niac-1.4.0-darwin-arm64 /usr/local/bin/niac

# Download and install converter
chmod +x niac-convert-1.4.0-darwin-arm64
sudo mv niac-convert-1.4.0-darwin-arm64 /usr/local/bin/niac-convert

# Verify installation
niac --version
niac-convert --help
```

### Linux / Windows
For Linux and Windows, please build from source:

```bash
# Clone repository
git clone https://github.com/krisarmstrong/niac-go
cd niac-go
git checkout v1.4.0

# Build (requires Go 1.21+ and libpcap-dev)
go build -o niac ./cmd/niac
go build -o niac-convert ./cmd/niac-convert

# Install
sudo cp niac niac-convert /usr/local/bin/
```

## Verification

Verify downloads using SHA256 checksums in `SHA256SUMS`:
```bash
shasum -a 256 -c SHA256SUMS
```

## What's New in v1.4.0

### 🎉 Complete DHCP/DNS Implementation

**DHCPv4 (15 options):**
- Basic: Subnet Mask, Router, DNS, Domain Name, Lease Time, T1/T2, Server ID
- High Priority: Hostname, NTP Servers, Domain Search, TFTP Server, Bootfile, Vendor-Specific
- Static leases with MAC address masks

**DHCPv6 (12 options):**
- Basic: Client/Server ID, IA_NA, IA_Addr, Preference, DNS, Domain Search
- High Priority: SNTP Servers, NTP Servers, SIP Servers/Domains, FQDN

**DNS Server:**
- Forward DNS (A records)
- Reverse DNS (PTR records)
- Configurable TTL

**Configuration:**
- Complete YAML support for all DHCP/DNS options
- Example configuration with 12 device types
- End-to-end integration from YAML to runtime

### Documentation
- New comprehensive reference YAML (658 lines)
- Complete feature documentation
- Organized docs/ folder structure
- Updated README and CHANGELOG

## Requirements

### Runtime
- **libpcap**: Required for packet capture
  - macOS: Pre-installed or `brew install libpcap`
  - Linux: `sudo apt-get install libpcap-dev`
  - Windows: WinPcap or Npcap

### Build
- **Go**: 1.21 or higher
- **libpcap-dev**: Development headers
- **gcc**: C compiler for CGO

## Quick Start

```bash
# Validate configuration
./niac --dry-run eth0 examples/scenario_configs/complete-reference.yaml

# Run normally
sudo ./niac eth0 examples/scenario_configs/complete-reference.yaml

# Run with debug output
sudo ./niac --debug 3 eth0 examples/scenario_configs/complete-reference.yaml

# Run in interactive mode
sudo ./niac --interactive eth0 examples/scenario_configs/complete-reference.yaml
```

## Features

- ✅ 13 network protocols (ARP, IP, ICMP, IPv6, ICMPv6, UDP, TCP, DNS, DHCP, HTTP, FTP, NetBIOS, STP)
- ✅ 4 discovery protocols (LLDP, CDP, EDP, FDP)
- ✅ Complete DHCP/DHCPv6 servers with 27 options
- ✅ DNS server with A and PTR records
- ✅ SNMP agent with walk file support
- ✅ Interactive TUI with error injection
- ✅ Traffic generation
- ✅ Device simulation

## Support

- **Documentation**: https://github.com/krisarmstrong/niac-go/tree/v1.4.0/docs
- **Examples**: https://github.com/krisarmstrong/niac-go/tree/v1.4.0/examples
- **Issues**: https://github.com/krisarmstrong/niac-go/issues

## Credits

- **Original NIAC**: Kevin Kayes (2002-2015)
- **Go Rewrite**: Kris Armstrong (2025)

## License

Same as original NIAC project.
