# NIAC-Go Configuration Examples

This directory contains organized example configurations demonstrating all features of NIAC-Go.

## Quick Start

```bash
# Validate any example
niac --dry-run lo0 examples/complete-kitchen-sink.yaml

# Run with an example (requires sudo for packet capture)
sudo niac en0 examples/layer2/lldp-only.yaml

# Run with verbose debug
sudo niac --verbose en0 examples/vendors/cisco-network.yaml
```

## Directory Structure

```
examples/
├── complete-kitchen-sink.yaml    # EVERYTHING - all features demonstrated
├── multi-ip-devices.yaml          # v1.5.0 Feature 3 demo (moved from root)
│
├── layer2/                         # Layer 2 Discovery Protocols
│   ├── lldp-only.yaml             # IEEE 802.1AB LLDP only
│   ├── cdp-only.yaml              # Cisco Discovery Protocol only
│   ├── edp-only.yaml              # Extreme Discovery Protocol only
│   ├── fdp-only.yaml              # Foundry Discovery Protocol only
│   ├── stp-bridge.yaml            # Spanning Tree Protocol (STP/RSTP/MSTP)
│   └── all-discovery-protocols.yaml  # All 4 protocols configured
│
├── dhcp/                           # DHCP Server Configurations
│   ├── dhcpv4-simple.yaml         # Basic DHCP server
│   └── dhcpv4-advanced.yaml       # Multiple pools with options
│
├── services/                       # Application Services
│   ├── dns-server.yaml            # DNS with A, AAAA, CNAME, MX, TXT, SRV, NS, SOA
│   ├── http-server.yaml           # HTTP with custom endpoints and responses
│   ├── ftp-server.yaml            # FTP with welcome banners and users
│   └── netbios-server.yaml        # NetBIOS name service configuration
│
├── network/                        # IP Addressing Schemes
│   ├── ipv4-only.yaml             # Traditional IPv4-only network
│   ├── ipv6-only.yaml             # Modern IPv6-only network
│   └── dual-stack.yaml            # Both IPv4 and IPv6 (most common)
│
└── vendors/                        # Vendor-Specific Configurations
    ├── cisco-network.yaml         # Cisco with CDP and LLDP
    ├── extreme-network.yaml       # Extreme with EDP and LLDP
    └── foundry-network.yaml       # Foundry/Brocade/Ruckus with FDP and LLDP
```

## Examples by Use Case

### Learning NIAC-Go

1. **Start here:** `complete-kitchen-sink.yaml` - See all features in one file
2. **Basic network:** `network/ipv4-only.yaml` - Simple 3-device network
3. **Discovery:** `layer2/lldp-only.yaml` - Add LLDP discovery protocol

### Testing Discovery Protocols

- **Multi-vendor lab:** Use `layer2/all-discovery-protocols.yaml` to simulate different vendor equipment
- **Single protocol:** Use `lldp-only.yaml`, `cdp-only.yaml`, `edp-only.yaml`, or `fdp-only.yaml`
- **Real-world:** Use vendor-specific examples in `vendors/` directory

### DHCP Testing

- **Basic DHCP:** `dhcp/dhcpv4-simple.yaml` - Single pool, straightforward
- **Complex DHCP:** `dhcp/dhcpv4-advanced.yaml` - Multiple pools, custom options

### IPv6 Adoption

- **Pure IPv6:** `network/ipv6-only.yaml` - Test IPv6-only environments
- **Migration:** `network/dual-stack.yaml` - Run both protocols side-by-side
- **Multi-IP:** `multi-ip-devices.yaml` - Devices with multiple addresses

### Vendor Simulation

- **Cisco shop:** `vendors/cisco-network.yaml`
- **Extreme shop:** `vendors/extreme-network.yaml`
- **Brocade/Ruckus shop:** `vendors/foundry-network.yaml`

## Configuration Features by Version

### v1.4.0 (Baseline)
- ✅ Device identity (name, MAC, single IP)
- ✅ DHCP server configuration
- ✅ DNS server configuration
- ✅ SNMP agent configuration
- ✅ Discovery protocols with hardcoded values

### v1.5.0 Features

#### Phase 1 (Completed)
- ✅ **Feature 1:** Color-coded debug output
- ✅ **Feature 2:** Per-protocol debug level control
- ✅ **Feature 3:** Multiple IPs per device (see `multi-ip-devices.yaml`)

#### Phase 2 Group 1 (Completed)
- ✅ **LLDP Config:** All TLVs configurable via YAML
- ✅ **CDP Config:** Version, platform, software, port ID in YAML
- ✅ **EDP Config:** Version string, display string in YAML
- ✅ **FDP Config:** Platform, software, port ID in YAML

#### Phase 2 Group 1b (Completed)
- ✅ **STP Config:** Bridge priority, timers, version (see `stp-bridge.yaml`)

#### Phase 2 Group 2 (Completed)
- ✅ **HTTP Server:** Server name, endpoints, responses (see `http-server.yaml`)
- ✅ **FTP Server:** Welcome banner, system type, users (see `ftp-server.yaml`)
- ✅ **NetBIOS:** Name, workgroup, node type, services (see `netbios-server.yaml`)

#### Phase 2 Group 3 (Pending)
- ⏳ **ICMP/ICMPv6:** TTL values, rate limiting (coming soon)
- ⏳ **DHCPv6:** IPv6 address assignment (coming soon)

## Common Configuration Patterns

### Enable Just One Protocol

```yaml
devices:
  - name: switch-01
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.10"

    lldp:
      enabled: true  # Only LLDP will advertise
```

### Multi-Vendor Environment

```yaml
devices:
  # Cisco switch
  - name: cisco-sw-01
    mac: "00:1a:2b:3c:4d:01"
    ip: "192.168.1.10"
    lldp:
      enabled: true
    cdp:
      enabled: true  # Cisco-specific

  # Extreme switch
  - name: extreme-sw-01
    mac: "00:04:96:ab:cd:02"
    ip: "192.168.1.20"
    lldp:
      enabled: true
    edp:
      enabled: true  # Extreme-specific
```

### Multiple IP Addresses

```yaml
devices:
  - name: server-01
    mac: "00:11:22:33:44:55"
    ips:  # Note: 'ips' (plural) not 'ip'
      - "192.168.1.100"      # IPv4
      - "192.168.2.100"      # IPv4 on another network
      - "2001:db8::100"      # IPv6
```

### DHCP + DNS Server

```yaml
devices:
  - name: infrastructure-server
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"

    dhcp:
      enabled: true
      pools:
        - network: "192.168.1.0/24"
          range_start: "192.168.1.100"
          range_end: "192.168.1.200"
          gateway: "192.168.1.1"
          dns_servers: ["192.168.1.1"]
          domain: "local"
          lease_time: 86400

    dns:
      enabled: true
      records:
        - name: "server.local"
          type: "A"
          value: "192.168.1.1"
          ttl: 3600
```

## Testing Examples

```bash
# Validate configuration syntax
niac --dry-run lo0 examples/layer2/lldp-only.yaml

# List devices in configuration
niac --list-devices examples/complete-kitchen-sink.yaml

# Run with debug for specific protocol
sudo niac --debug 1 --debug-lldp 3 en0 examples/layer2/lldp-only.yaml

# Run in interactive TUI mode
sudo niac --interactive en0 examples/vendors/cisco-network.yaml

# Save output to log file
sudo niac --log-file niac.log en0 examples/dhcp/dhcpv4-simple.yaml
```

## Creating Your Own Configurations

1. **Start with an example** - Copy the closest matching example
2. **Modify devices** - Change names, MACs, and IPs to match your needs
3. **Enable protocols** - Set `enabled: true` for protocols you want
4. **Customize values** - Update descriptions, versions, platform strings
5. **Validate** - Use `--dry-run` to check syntax
6. **Test** - Run with `--verbose` to see detailed output

## Troubleshooting

### Configuration Doesn't Load
- Check YAML syntax (indentation matters!)
- Validate with `--dry-run`
- Review error messages carefully

### Protocol Not Advertising
- Verify `enabled: true` is set
- Check debug output: `--debug-lldp 3` (for LLDP, etc.)
- Ensure device has MAC address

### DHCP Not Working
- Server needs static IP on the network
- Pool range must be within network subnet
- Gateway should be reachable

### Multiple IPs Not Working
- Use `ips:` (plural) not `ip:`
- Provide array of addresses with `-` prefix
- See `multi-ip-devices.yaml` for examples

## Additional Resources

- **Main README:** `../README.md` - Full NIAC-Go documentation
- **Protocol Details:** Each example file contains protocol-specific notes
- **Debug Help:** Use `--help` to see all debug options
- **GitHub Issues:** Report problems at https://github.com/krisarmstrong/niac-go/issues

## Contributing Examples

Have a useful configuration? Submit a pull request!

Good example contributions:
- Real-world scenarios (campus network, data center, home lab)
- Edge cases and complex setups
- Integration examples (NIAC-Go + other tools)
- Testing and QA configurations
