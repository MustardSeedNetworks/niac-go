# NiAC-Go Examples

Example YAML configurations demonstrating NiAC-Go's protocol simulation and network topology features.

## Table of Contents

- [Quick Start](#quick-start)
- [Example Categories](#example-categories)
- [Usage Instructions](#usage-instructions)
- [Directory Structure](#directory-structure)
- [Walk Files](#walk-files)
- [Creating Custom Examples](#creating-custom-examples)

## Quick Start

```bash
# Validate an example
niac validate examples/layer2/lldp-only.yaml

# Run an example (requires root for packet capture)
sudo niac en0 examples/layer2/lldp-only.yaml

# Run with debug output
sudo niac --debug-lldp 3 en0 examples/layer2/lldp-only.yaml

# Test configuration without running
niac --dry-run lo0 examples/complete-kitchen-sink.yaml
```

## Example Categories

### Layer 2 Discovery Protocols

Layer 2 neighbor discovery configurations.

| Example | Description | Protocols |
|---------|-------------|-----------|
| [lldp-only.yaml](layer2/lldp-only.yaml) | LLDP neighbor discovery (multi-vendor) | LLDP |
| [cdp-only.yaml](layer2/cdp-only.yaml) | Cisco Discovery Protocol | CDP |
| [edp-only.yaml](layer2/edp-only.yaml) | Extreme Discovery Protocol | EDP |
| [fdp-only.yaml](layer2/fdp-only.yaml) | Foundry/Brocade Discovery Protocol | FDP |
| [all-discovery-protocols.yaml](layer2/all-discovery-protocols.yaml) | All discovery protocols combined | LLDP, CDP, EDP, FDP |
| [stp-bridge.yaml](layer2/stp-bridge.yaml) | Spanning Tree Protocol (STP/RSTP/MSTP) | STP |

**Usage:**
```bash
# LLDP discovery
sudo niac en0 examples/layer2/lldp-only.yaml
lldpcli show neighbors

# CDP discovery
sudo niac en0 examples/layer2/cdp-only.yaml
show cdp neighbors

# STP with debug
sudo niac --debug-stp 3 en0 examples/layer2/stp-bridge.yaml
```

### Network Layer

IPv4, IPv6, ICMP configurations.

| Example | Description | Protocols |
|---------|-------------|-----------|
| [ipv4-only.yaml](network/ipv4-only.yaml) | IPv4-only device | IPv4, ARP, ICMP |
| [ipv6-only.yaml](network/ipv6-only.yaml) | IPv6-only device | IPv6, ICMPv6, NDP |
| [dual-stack.yaml](network/dual-stack.yaml) | IPv4 + IPv6 dual-stack | IPv4, IPv6, ICMP, ICMPv6 |
| [icmp-config.yaml](network/icmp-config.yaml) | ICMP/ping configuration | ICMP |
| [icmpv6-config.yaml](network/icmpv6-config.yaml) | ICMPv6/ping6 configuration | ICMPv6 |
| [dhcpv6-config.yaml](network/dhcpv6-config.yaml) | DHCPv6 server | DHCPv6 |

**Usage:**
```bash
# Dual-stack device
sudo niac en0 examples/network/dual-stack.yaml
ping 10.0.0.1
ping6 2001:db8::1

# ICMP configuration
sudo niac en0 examples/network/icmp-config.yaml
ping 10.0.0.1
```

### DHCP

Dynamic Host Configuration Protocol servers.

| Example | Description | Features |
|---------|-------------|----------|
| [dhcpv4-simple.yaml](dhcp/dhcpv4-simple.yaml) | Basic DHCPv4 server | Single pool |
| [dhcpv4-advanced.yaml](dhcp/dhcpv4-advanced.yaml) | Advanced DHCPv4 server | Multiple pools, custom options |

**Usage:**
```bash
# Run DHCP server
sudo niac en0 examples/dhcp/dhcpv4-simple.yaml

# Request IP (on client)
sudo dhclient en0

# Monitor DHCP traffic
sudo tcpdump -i en0 port 67 or port 68
```

### Services

Application layer services.

| Example | Description | Protocol |
|---------|-------------|----------|
| [dns-server.yaml](services/dns-server.yaml) | DNS name resolution | DNS |
| [http-server.yaml](services/http-server.yaml) | HTTP web server | HTTP |
| [ftp-server.yaml](services/ftp-server.yaml) | FTP file server | FTP |
| [netbios-server.yaml](services/netbios-server.yaml) | NetBIOS/Windows networking | NetBIOS |

**Usage:**
```bash
# DNS server
sudo niac en0 examples/services/dns-server.yaml
dig @10.0.0.1 router.example.com

# HTTP server
sudo niac en0 examples/services/http-server.yaml
curl http://10.0.0.1/

# FTP server
sudo niac en0 examples/services/ftp-server.yaml
ftp 10.0.0.1
```

### SNMP

SNMP agent and trap configurations.

| Example | Description | Features |
|---------|-------------|----------|
| [snmp-agent-basic.yaml](snmp/snmp-agent-basic.yaml) | Basic SNMP agent | Walk file, system MIB |
| [snmp-traps-all.yaml](snmp/snmp-traps-all.yaml) | All SNMP trap types | Event + threshold traps |
| [snmp-multiple-communities.yaml](snmp/snmp-multiple-communities.yaml) | Multiple SNMP communities | RO, RW, trap communities |
| [snmp-complete-network.yaml](snmp/snmp-complete-network.yaml) | Complete SNMP network | 4 devices, traps, walk files |

**Usage:**
```bash
# SNMP agent
sudo niac en0 examples/snmp/snmp-agent-basic.yaml
snmpwalk -v2c -c public 10.0.0.1 system

# SNMP traps
sudo niac --interactive en0 examples/snmp/snmp-traps-all.yaml
# Press 'i' to inject errors and trigger traps

# Listen for traps (on receiver)
snmptrapd -f -Lo
```

### Topology (v1.23.0)

Port-channels, trunk ports, and multi-device topologies.

| Example | Description | Features |
|---------|-------------|----------|
| [two-switch-trunk.yaml](topology/two-switch-trunk.yaml) | Two switches with trunk link | Trunk ports, VLANs |
| [port-channel-lab.yaml](topology/port-channel-lab.yaml) | Port-channel/LAG configuration | LACP, trunk over port-channel |

**Usage:**
```bash
# Two-switch trunk
sudo niac en0 examples/topology/two-switch-trunk.yaml
lldpcli show neighbors  # See both switches

# Port-channel
sudo niac en0 examples/topology/port-channel-lab.yaml
show etherchannel summary  # View port-channel status
```

### Protocol Combinations

Real-world protocol combinations for different network roles.

| Example | Description | Protocols |
|---------|-------------|-----------|
| [enterprise-router.yaml](combinations/enterprise-router.yaml) | Full-featured edge router | LLDP, CDP, DHCP, DNS, SNMP, ICMP |
| [datacenter-switch.yaml](combinations/datacenter-switch.yaml) | Data center core switch | STP, LLDP, CDP, SNMP, ICMP |
| [access-switch.yaml](combinations/access-switch.yaml) | Access layer switch | LLDP, CDP, STP, SNMP |
| [wireless-controller.yaml](combinations/wireless-controller.yaml) | Wireless LAN controller | HTTP, DHCP, DNS, SNMP, ICMP |

**Usage:**
```bash
# Enterprise router (full protocol suite)
sudo niac en0 examples/combinations/enterprise-router.yaml

# Test all protocols:
lldpcli show neighbors           # LLDP
snmpwalk -v2c -c public 10.0.0.1 # SNMP
dig @10.0.0.1 router.local       # DNS
sudo dhclient en0                # DHCP
ping 10.0.0.1                    # ICMP
```

### Vendor-Specific

Vendor-specific configurations.

| Example | Description | Vendor |
|---------|-------------|--------|
| [cisco-network.yaml](vendors/cisco-network.yaml) | Cisco devices | Cisco |
| [extreme-network.yaml](vendors/extreme-network.yaml) | Extreme Networks | Extreme |
| [foundry-network.yaml](vendors/foundry-network.yaml) | Brocade/Foundry | Foundry |

**Usage:**
```bash
# Cisco network
sudo niac en0 examples/vendors/cisco-network.yaml
```

### Advanced Features

Traffic patterns, error injection, and complex scenarios.

| Example | Description | Features |
|---------|-------------|----------|
| [traffic-patterns.yaml](traffic-patterns.yaml) | Traffic generation patterns | ARP announcements, periodic pings, random traffic |
| [snmp-traps.yaml](snmp-traps.yaml) | SNMP trap generation | Event and threshold traps |
| [error-injection-demo.yaml](error-injection-demo.yaml) | Error injection demo | Interactive error injection |
| [multi-ip-devices.yaml](multi-ip-devices.yaml) | Devices with multiple IPs | Multi-homed, dual-stack |
| [complete-kitchen-sink.yaml](complete-kitchen-sink.yaml) | All features demo | 9 devices, all protocols, all features |

**Usage:**
```bash
# Traffic generation
sudo niac en0 examples/traffic-patterns.yaml

# Error injection (interactive)
sudo niac --interactive en0 examples/error-injection-demo.yaml
# Press 'i' to open menu

# Complete demo
sudo niac en0 examples/complete-kitchen-sink.yaml
```

## Synthetic Walk Files

NiAC-Go can generate modern SNMP walk files rather than relying solely on captured data. The generator script lives in `scripts/generate_modern_walk.py`:

```bash
# Discover every supported vendor/model
python3 scripts/generate_modern_walk.py --list

# Create a Meraki MS390 walk file
python3 scripts/generate_modern_walk.py \
  --vendor meraki \
  --model ms390-48uxb \
  --output examples/device_walks/generated/meraki-ms390.walk
```

Reference the generated file in any example by pointing a device’s `snmp_agent.walk_file` setting to the new path. This is a quick way to simulate current-generation hardware without curated captures.

## Usage Instructions

### Validation

Always validate before running:

```bash
# Validate configuration
niac validate examples/layer2/lldp-only.yaml

# Validate with dry-run
niac --dry-run lo0 examples/layer2/lldp-only.yaml
```

### Running Examples

```bash
# Basic run
sudo niac <interface> <config-file>

# With debug output
sudo niac --debug-<protocol> <level> <interface> <config-file>

# Interactive mode
sudo niac --interactive <interface> <config-file>
```

**Debug Levels:**
- 1: Errors only
- 2: Warnings
- 3: Info (recommended)
- 4: Debug
- 5: Trace

**Available Debug Flags:**
- `--debug-lldp`
- `--debug-cdp`
- `--debug-stp`
- `--debug-dhcp`
- `--debug-dns`
- `--debug-snmp`
- `--debug-icmp`

### Testing Protocols

```bash
# LLDP
lldpcli show neighbors

# CDP (Cisco CLI)
show cdp neighbors

# SNMP
snmpwalk -v2c -c public 10.0.0.1 system

# DNS
dig @10.0.0.1 hostname.example.com

# DHCP
sudo dhclient en0

# HTTP
curl http://10.0.0.1/

# ICMP
ping 10.0.0.1
```

### Monitoring Traffic

```bash
# All traffic
sudo tcpdump -i en0 -w capture.pcap

# LLDP
sudo tcpdump -i en0 ether proto 0x88cc

# CDP
sudo tcpdump -i en0 ether dst 01:00:0c:cc:cc:cc

# DHCP
sudo tcpdump -i en0 port 67 or port 68

# DNS
sudo tcpdump -i en0 port 53

# SNMP
sudo tcpdump -i en0 port 161 or port 162

# HTTP
sudo tcpdump -i en0 port 80
```

## Directory Structure

```
examples/
├── README.md                      # This file
├── layer2/                        # Layer 2 discovery protocols
│   ├── lldp-only.yaml
│   ├── cdp-only.yaml
│   ├── edp-only.yaml
│   ├── fdp-only.yaml
│   ├── all-discovery-protocols.yaml
│   └── stp-bridge.yaml
├── network/                       # Network layer (IPv4/IPv6)
│   ├── ipv4-only.yaml
│   ├── ipv6-only.yaml
│   ├── dual-stack.yaml
│   ├── icmp-config.yaml
│   ├── icmpv6-config.yaml
│   └── dhcpv6-config.yaml
├── dhcp/                          # DHCP servers
│   ├── dhcpv4-simple.yaml
│   └── dhcpv4-advanced.yaml
├── services/                      # Application services
│   ├── dns-server.yaml
│   ├── http-server.yaml
│   ├── ftp-server.yaml
│   └── netbios-server.yaml
├── snmp/                          # SNMP agent and traps
│   ├── snmp-agent-basic.yaml
│   ├── snmp-traps-all.yaml
│   ├── snmp-multiple-communities.yaml
│   └── snmp-complete-network.yaml
├── topology/                      # Topology features (v1.23.0)
│   ├── two-switch-trunk.yaml
│   └── port-channel-lab.yaml
├── combinations/                  # Protocol combinations
│   ├── enterprise-router.yaml
│   ├── datacenter-switch.yaml
│   ├── access-switch.yaml
│   └── wireless-controller.yaml
├── vendors/                       # Vendor-specific examples
│   ├── cisco-network.yaml
│   ├── extreme-network.yaml
│   └── foundry-network.yaml
├── traffic-patterns.yaml          # Traffic generation
├── snmp-traps.yaml                # SNMP trap demo
├── error-injection-demo.yaml      # Error injection
├── multi-ip-devices.yaml          # Multi-IP devices
├── complete-kitchen-sink.yaml     # All features
└── device_walks_sanitized/        # SNMP walk files (555 files)
    ├── cisco/ (488 files)
    ├── extreme/ (7 files)
    ├── juniper/ (4 files)
    ├── fortinet/ (2 files)
    ├── hp/
    ├── dell/
    ├── brocade/
    ├── huawei/
    ├── meraki/
    ├── mikrotik/
    ├── 3com/
    ├── netgear/
    ├── oracle/
    ├── vmware/
    ├── voip/
    ├── zte/
    ├── misc/
    └── mapping.json
```

## Walk Files

SNMP walk files provide realistic device simulation. NiAC-Go includes 555+ sanitized walk files across 17 vendors.

### Using Walk Files

```yaml
devices:
  - name: switch-01
    snmp_agent:
      enabled: true
      walk_file: "examples/device_walks_sanitized/cisco/niac-cisco-c3850.walk"
```

### Walk File Locations

- **Cisco**: `examples/device_walks_sanitized/cisco/` (488 files)
  - Catalyst 2960, 3560, 3750, 3850
  - Nexus 5000, 7000, 9000
  - ISR 2800, 3900
  - ASR 1000

- **Juniper**: `examples/device_walks_sanitized/juniper/` (4 files)
  - EX Series, MX Series, SRX Series

- **Extreme**: `examples/device_walks_sanitized/extreme/` (7 files)
  - X-Series (X460, X465), Summit

- **Other Vendors**: HP, Dell, Brocade, Huawei, Fortinet, Meraki, etc.

### Walk File Documentation

See [docs/WALK_FILES.md](../docs/WALK_FILES.md) for:
- Walk file format specification
- Capturing walk files from real devices
- Sanitizing walk files
- Contributing walk files
- Vendor coverage details

## Creating Custom Examples

### Basic Template

```yaml
devices:
  - name: my-device
    type: switch
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1"]

    # Enable protocols as needed
    lldp:
      enabled: true
      system_name: "my-device"

    snmp_agent:
      enabled: true
      community: "public"
      sysname: "my-device"

    icmp:
      enabled: true
```

### Tips for Custom Examples

1. **Start simple** - Begin with one device and one protocol
2. **Validate frequently** - Use `niac validate` to catch errors early
3. **Use dry-run** - Test with `--dry-run` before running live
4. **Enable debug** - Use `--debug-<protocol> 3` to see what's happening
5. **Reference existing examples** - Copy and modify similar examples
6. **Test protocols** - Verify each protocol works independently
7. **Document** - Add comments explaining configuration choices

### Example Workflow

```bash
# 1. Create configuration
cat > my-example.yaml << 'EOF'
devices:
  - name: test-switch
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1"]
    lldp:
      enabled: true
    icmp:
      enabled: true
EOF

# 2. Validate
niac validate my-example.yaml

# 3. Test with dry-run
niac --dry-run lo0 my-example.yaml

# 4. Run with debug
sudo niac --debug-lldp 3 en0 my-example.yaml

# 5. Test functionality
ping 10.0.0.1
lldpcli show neighbors
```

## Statistics

- **Total Examples**: 35+ YAML configurations
- **Walk Files**: 555 sanitized files
- **Vendors**: 17 vendors represented
- **Protocols**: 19 protocols demonstrated
- **Categories**: 9 example categories

## Documentation

### Comprehensive Guides

- **[Protocol Guide](../docs/PROTOCOL_GUIDE.md)** - Complete protocol configuration reference
- **[API Reference](../docs/API_REFERENCE.md)** - YAML schema and all configuration options
- **[Topology Guide](../docs/TOPOLOGY_GUIDE.md)** - Port-channels, trunks, VLANs
- **[Environment Guide](../docs/ENVIRONMENTS.md)** - Complete network environment examples
- **[Troubleshooting Guide](../docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Walk Files Guide](../docs/WALK_FILES.md)** - SNMP walk file documentation

### Quick References

- **[Main README](../README.md)** - Project overview and features
- **[Contributing](../CONTRIBUTING.md)** - How to contribute examples

## Support

- **GitHub Issues**: https://github.com/krisarmstrong/niac-go/issues
- **Documentation**: https://github.com/krisarmstrong/niac-go/tree/main/docs
- **Examples**: This directory

---

**Last Updated:** 2025-01-11 (v1.23.0)
