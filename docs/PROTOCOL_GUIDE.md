# Protocol Configuration Guide

This guide covers all 19 protocols supported by NiAC-Go, including configuration examples, use cases, and best practices.

## Table of Contents

- [Overview](#overview)
- [Layer 2 Protocols](#layer-2-protocols)
  - [LLDP](#lldp)
  - [CDP](#cdp)
  - [EDP](#edp)
  - [FDP](#fdp)
  - [STP](#stp)
- [Network Layer](#network-layer)
  - [ARP](#arp)
  - [ICMP](#icmp)
  - [ICMPv6](#icmpv6)
- [Transport Layer](#transport-layer)
  - [TCP](#tcp)
  - [UDP](#udp)
- [Application Protocols](#application-protocols)
  - [DHCP (DHCPv4)](#dhcp-dhcpv4)
  - [DHCPv6](#dhcpv6)
  - [DNS](#dns)
  - [HTTP](#http)
  - [FTP](#ftp)
  - [NetBIOS](#netbios)
  - [SNMP](#snmp)
- [Protocol Combinations](#protocol-combinations)
- [Best Practices](#best-practices)

## Overview

NiAC-Go supports 19 network protocols across all layers of the OSI model. Each protocol can be independently configured per device via YAML configuration.

### Supported Protocols by Layer

| Layer | Protocols |
|-------|-----------|
| Layer 2 (Data Link) | LLDP, CDP, EDP, FDP, STP |
| Layer 3 (Network) | IPv4, IPv6, ARP, ICMP, ICMPv6 |
| Layer 4 (Transport) | TCP, UDP |
| Layer 7 (Application) | DHCP, DHCPv6, DNS, HTTP, FTP, NetBIOS, SNMP |

## Layer 2 Protocols

### LLDP

**Link Layer Discovery Protocol** - IEEE 802.1AB standard for network device discovery.

#### Use Cases
- Multi-vendor network discovery
- Network topology mapping
- Neighbor relationship verification
- Automated device inventory

#### Configuration

```yaml
devices:
  - name: switch-01
    lldp:
      enabled: true
      system_name: "switch-01"
      system_description: "Cisco Catalyst 3850 - Core Switch"
      chassis_id: "00:11:22:33:44:01"
      port_description: "GigabitEthernet0/1"
      advertise_interval: 30  # Seconds between advertisements
      management_address: "10.0.0.1"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable LLDP |
| `system_name` | string | No | device name | System name advertised |
| `system_description` | string | No | "" | Device description |
| `chassis_id` | string | No | device MAC | Chassis identifier |
| `port_description` | string | No | interface | Port description |
| `advertise_interval` | integer | No | 30 | Advertisement interval (seconds) |
| `management_address` | string | No | device IP | Management IP address |

#### Testing

```bash
# View LLDP neighbors
lldpcli show neighbors

# View detailed neighbor info
lldpcli show neighbors detail

# Monitor LLDP traffic
sudo tcpdump -i en0 ether proto 0x88cc
```

#### Best Practices
- Use LLDP for multi-vendor environments (industry standard)
- Set advertise_interval to 30 seconds (default, RFC recommended)
- Always configure system_description for identification
- Use management_address for out-of-band management
- Enable on all network infrastructure devices

### CDP

**Cisco Discovery Protocol** - Cisco proprietary neighbor discovery protocol.

#### Use Cases
- Cisco device discovery
- Legacy network compatibility
- Cisco-specific network management
- VoIP phone power negotiation

#### Configuration

```yaml
devices:
  - name: switch-01
    cdp:
      enabled: true
      platform: "WS-C3850-48P"
      capabilities: "Switch IGMP"
      software_version: "IOS-XE 16.12.4"
      advertise_interval: 60  # Seconds between advertisements
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable CDP |
| `platform` | string | No | "" | Platform/model identifier |
| `capabilities` | string | No | "Switch" | Device capabilities |
| `software_version` | string | No | "" | Software version string |
| `advertise_interval` | integer | No | 60 | Advertisement interval (seconds) |

#### Testing

```bash
# View CDP neighbors (Cisco CLI)
show cdp neighbors
show cdp neighbors detail

# Monitor CDP traffic
sudo tcpdump -i en0 ether dst 01:00:0c:cc:cc:cc
```

#### Best Practices
- Use CDP for Cisco-only environments
- Disable CDP on edge ports facing untrusted networks (security)
- Set advertise_interval to 60 seconds (Cisco default)
- Populate platform and software_version for accurate inventory
- Consider LLDP instead for multi-vendor compatibility

#### LLDP vs CDP Comparison

| Feature | LLDP | CDP |
|---------|------|-----|
| Standard | IEEE 802.1AB (open) | Cisco proprietary |
| Multi-vendor | ✅ Yes | ❌ Cisco only |
| Default interval | 30 seconds | 60 seconds |
| Security | Configurable | Less granular |
| VoIP PoE | Limited | ✅ Enhanced |
| Use when | Multi-vendor | Cisco-only |

### EDP

**Extreme Discovery Protocol** - Extreme Networks proprietary discovery protocol.

#### Use Cases
- Extreme Networks device discovery
- ExtremeXOS/EXOS switch management
- Legacy Extreme network compatibility

#### Configuration

```yaml
devices:
  - name: extreme-switch-01
    edp:
      enabled: true
      platform: "X460-G2"
      software_version: "ExtremeXOS 30.7.1.4"
```

#### Best Practices
- Use EDP for Extreme Networks devices
- Enable alongside LLDP for maximum compatibility
- Primarily used in Extreme-only networks

### FDP

**Foundry Discovery Protocol** - Brocade/Foundry proprietary discovery protocol.

#### Use Cases
- Brocade/Foundry device discovery
- Legacy Foundry network compatibility
- Ruckus ICX switch management (post-acquisition)

#### Configuration

```yaml
devices:
  - name: foundry-switch-01
    fdp:
      enabled: true
      platform: "ICX7450-48P"
      software_version: "08.0.95"
```

#### Best Practices
- Use FDP for Brocade/Foundry/Ruckus devices
- Enable alongside LLDP for multi-vendor compatibility

### STP

**Spanning Tree Protocol** - IEEE 802.1D/802.1w/802.1s loop prevention protocol.

#### Use Cases
- Loop prevention in Layer 2 networks
- Network redundancy and failover
- Topology optimization

#### Configuration

```yaml
devices:
  - name: core-switch-01
    stp:
      enabled: true
      bridge_priority: 4096    # 0-61440, increments of 4096
      hello_time: 2            # Seconds between BPDUs
      max_age: 20              # BPDU info retention time
      forward_delay: 15        # Time in listening/learning
      version: "rstp"          # stp, rstp, or mstp
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable STP |
| `bridge_priority` | integer | No | 32768 | Bridge priority (0-61440) |
| `hello_time` | integer | No | 2 | BPDU interval (1-10 seconds) |
| `max_age` | integer | No | 20 | BPDU max age (6-40 seconds) |
| `forward_delay` | integer | No | 15 | Forward delay (4-30 seconds) |
| `version` | string | No | "rstp" | STP version: stp, rstp, mstp |

#### STP Versions

| Version | Standard | Convergence | Use Case |
|---------|----------|-------------|----------|
| STP | IEEE 802.1D | 30-50 seconds | Legacy compatibility |
| RSTP | IEEE 802.1w | <6 seconds | Modern networks (recommended) |
| MSTP | IEEE 802.1s | <6 seconds | VLAN-aware, complex topologies |

#### Bridge Priority Values

| Priority | Purpose |
|----------|---------|
| 0 or 4096 | Force as root bridge |
| 8192 | Secondary/backup root |
| 32768 | Default (let election decide) |
| 61440 | Never become root (edge switches) |

#### Testing

```bash
# Monitor STP BPDUs
sudo tcpdump -i en0 ether dst 01:80:c2:00:00:00

# Run with STP debug
sudo niac --debug-stp 3 en0 config.yaml
```

#### Best Practices
- Use RSTP (802.1w) for modern networks (faster convergence)
- Set root bridge priority to 4096, backup to 8192
- Use 32768 (default) for access switches
- Set edge switches to 61440 (never root)
- Match hello_time, max_age, forward_delay across all switches
- Enable STP on all switches in a Layer 2 domain

## Network Layer

### ARP

**Address Resolution Protocol** - Maps IPv4 addresses to MAC addresses.

#### Use Cases
- IP to MAC address resolution
- Network reachability testing
- ARP table population

#### Configuration

```yaml
devices:
  - name: device-01
    # ARP is automatically enabled when IPv4 is configured
    ips:
      - "10.0.0.1"
```

ARP is implicitly enabled when IPv4 addresses are configured. No explicit configuration needed.

#### Testing

```bash
# View ARP table
arp -a

# Monitor ARP traffic
sudo tcpdump -i en0 arp
```

### ICMP

**Internet Control Message Protocol** - IPv4 diagnostic and error reporting.

#### Use Cases
- Ping (echo request/reply)
- Network troubleshooting
- Connectivity verification
- Path MTU discovery

#### Configuration

```yaml
devices:
  - name: device-01
    ips:
      - "10.0.0.1"
    icmp:
      enabled: true
      ttl: 64  # Time to live (1-255)
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable ICMP responses |
| `ttl` | integer | No | 64 | Time to live (1-255) |

#### Testing

```bash
# Ping device
ping 10.0.0.1

# Ping with specific count
ping -c 4 10.0.0.1

# Traceroute
traceroute 10.0.0.1
```

#### Best Practices
- Enable ICMP for network troubleshooting
- Use TTL of 64 (standard for most systems)
- Consider disabling ICMP on WAN-facing interfaces (security)
- Allow ICMP echo for connectivity monitoring

### ICMPv6

**Internet Control Message Protocol for IPv6** - IPv6 diagnostic and error reporting.

#### Use Cases
- IPv6 ping (echo request/reply)
- Neighbor Discovery Protocol (NDP)
- Router advertisements
- Path MTU discovery

#### Configuration

```yaml
devices:
  - name: device-01
    ips:
      - "2001:db8::1"
    icmpv6:
      enabled: true
      hop_limit: 255  # RFC 4861 requires 255 for NDP
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable ICMPv6 responses |
| `hop_limit` | integer | No | 255 | Hop limit (RFC 4861) |

#### Testing

```bash
# Ping IPv6 device
ping6 2001:db8::1

# View IPv6 neighbor cache
ndp -a

# Monitor ICMPv6 traffic
sudo tcpdump -i en0 icmp6
```

#### Best Practices
- Always use hop_limit of 255 for NDP (RFC 4861 requirement)
- Enable ICMPv6 for IPv6 networks (required for NDP)
- ICMPv6 is more critical than ICMP (don't disable)
- Use link-local addresses (fe80::) for neighbor discovery

## Transport Layer

### TCP

**Transmission Control Protocol** - Reliable, connection-oriented transport.

#### Use Cases
- HTTP servers
- FTP servers
- Reliable data transfer
- Connection-oriented services

TCP is automatically used by application protocols (HTTP, FTP) that require it. No explicit configuration needed.

### UDP

**User Datagram Protocol** - Connectionless, unreliable transport.

#### Use Cases
- DNS queries
- DHCP
- SNMP
- Low-latency applications

UDP is automatically used by application protocols (DNS, DHCP, SNMP) that require it. No explicit configuration needed.

## Application Protocols

### DHCP (DHCPv4)

**Dynamic Host Configuration Protocol** - Automatic IPv4 address assignment.

#### Use Cases
- Automatic IP address assignment
- Network parameter distribution (gateway, DNS)
- PXE boot servers
- IP address management (IPAM)

#### Configuration

```yaml
devices:
  - name: dhcp-server
    ips:
      - "10.0.0.1"
    dhcp:
      enabled: true
      pools:
        - network: "10.0.10.0/24"
          range_start: "10.0.10.100"
          range_end: "10.0.10.200"
          gateway: "10.0.0.1"
          dns_servers: ["10.0.0.1", "8.8.8.8"]
          lease_time: 86400  # 24 hours in seconds
          domain_name: "corp.example.com"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable DHCP server |
| `pools` | array | Yes | [] | DHCP address pools |

**Pool Fields:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `network` | string | Yes | - | Network CIDR (e.g., 10.0.0.0/24) |
| `range_start` | string | Yes | - | First IP in range |
| `range_end` | string | Yes | - | Last IP in range |
| `gateway` | string | No | "" | Default gateway |
| `dns_servers` | string array | No | [] | DNS server IPs |
| `lease_time` | integer | No | 86400 | Lease time in seconds |
| `domain_name` | string | No | "" | Domain name |

#### Testing

```bash
# Request DHCP lease
sudo dhclient en0

# Release lease
sudo dhclient -r en0

# Monitor DHCP traffic
sudo tcpdump -i en0 port 67 or port 68
```

#### Best Practices
- Use lease_time of 86400 (24 hours) for workstations
- Use shorter lease_time (3600-7200) for guest networks
- Always provide gateway and dns_servers
- Reserve IPs outside DHCP range for servers
- Use multiple pools for network segmentation

### DHCPv6

**Dynamic Host Configuration Protocol for IPv6** - Automatic IPv6 address assignment.

#### Use Cases
- IPv6 address assignment
- IPv6 network parameters
- Dual-stack networks

#### Configuration

```yaml
devices:
  - name: dhcpv6-server
    ips:
      - "2001:db8::1"
    dhcpv6:
      enabled: true
      pools:
        - network: "2001:db8:1::/64"
          range_start: "2001:db8:1::100"
          range_end: "2001:db8:1::200"
          dns_servers: ["2001:db8::1", "2001:4860:4860::8888"]
          lease_time: 86400
          domain_name: "corp.example.com"
```

#### Best Practices
- Use DHCPv6 for stateful IPv6 addressing
- Consider SLAAC for stateless autoconfiguration
- Use /64 prefix for subnets (standard)
- Provide IPv6 DNS servers

### DNS

**Domain Name System** - Name resolution service.

#### Use Cases
- Hostname to IP resolution
- Internal domain names
- Split-horizon DNS
- Service discovery

#### Configuration

```yaml
devices:
  - name: dns-server
    ips:
      - "10.0.0.1"
    dns:
      enabled: true
      forward_records:
        - name: "router.example.com"
          ip: "10.0.0.1"
          ttl: 3600
        - name: "server.example.com"
          ip: "10.0.0.10"
          ttl: 3600
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable DNS server |
| `forward_records` | array | No | [] | A records (hostname -> IP) |

**Forward Record Fields:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Fully qualified domain name |
| `ip` | string | Yes | - | IPv4 or IPv6 address |
| `ttl` | integer | No | 3600 | Time to live (seconds) |

#### Testing

```bash
# Query DNS server
dig @10.0.0.1 router.example.com

# Query with nslookup
nslookup router.example.com 10.0.0.1

# Monitor DNS traffic
sudo tcpdump -i en0 port 53
```

#### Best Practices
- Use TTL of 3600 (1 hour) for internal records
- Use shorter TTL (300-600) for frequently changing records
- Always use fully qualified domain names (FQDNs)
- Provide DNS in DHCP offers
- Consider split-horizon DNS for internal/external separation

### HTTP

**Hypertext Transfer Protocol** - Web server protocol.

#### Use Cases
- Web-based management interfaces
- API endpoints
- Configuration portals
- Device status pages

#### Configuration

```yaml
devices:
  - name: web-server
    ips:
      - "10.0.0.1"
    http:
      enabled: true
      port: 80
      endpoints:
        - path: "/"
          content: "<h1>Welcome to NiAC-Go</h1>"
        - path: "/status"
          content: '{"status":"ok","uptime":12345}'
        - path: "/api/info"
          content: '{"device":"switch-01","model":"Catalyst 3850"}'
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable HTTP server |
| `port` | integer | No | 80 | TCP port |
| `endpoints` | array | No | [] | HTTP endpoints |

**Endpoint Fields:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | - | URL path (e.g., /api/info) |
| `content` | string | Yes | - | Response content (HTML/JSON) |

#### Testing

```bash
# Access HTTP endpoint
curl http://10.0.0.1/

# Test API endpoint
curl http://10.0.0.1/api/info

# Monitor HTTP traffic
sudo tcpdump -i en0 port 80
```

#### Best Practices
- Use port 80 for standard HTTP
- Use port 443 for HTTPS (if implementing)
- Return JSON for API endpoints
- Return HTML for management interfaces
- Consider authentication for production systems

### FTP

**File Transfer Protocol** - File transfer service.

#### Use Cases
- Firmware upgrades
- Configuration backups
- File distribution
- Log collection

#### Configuration

```yaml
devices:
  - name: ftp-server
    ips:
      - "10.0.0.1"
    ftp:
      enabled: true
      port: 21
      users:
        - username: "admin"
          password: "admin123"
        - username: "readonly"
          password: "guest"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable FTP server |
| `port` | integer | No | 21 | TCP port |
| `users` | array | No | [] | FTP user accounts |

**User Fields:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `username` | string | Yes | - | FTP username |
| `password` | string | Yes | - | FTP password |

#### Testing

```bash
# Connect to FTP server
ftp 10.0.0.1

# Monitor FTP traffic
sudo tcpdump -i en0 port 21
```

#### Best Practices
- Use SFTP or FTPS instead of FTP (security)
- Create separate accounts for different access levels
- Use strong passwords (production)
- Consider disabling FTP on production systems
- Use for simulation/testing purposes

### NetBIOS

**Network Basic Input/Output System** - Windows network name service.

#### Use Cases
- Windows network browsing
- SMB/CIFS file sharing
- Legacy Windows compatibility
- Workgroup discovery

#### Configuration

```yaml
devices:
  - name: windows-server
    ips:
      - "10.0.0.1"
    netbios:
      enabled: true
      name: "FILESERVER"
      workgroup: "WORKGROUP"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable NetBIOS |
| `name` | string | No | device name | NetBIOS name (15 chars max) |
| `workgroup` | string | No | "WORKGROUP" | Workgroup name |

#### Testing

```bash
# Query NetBIOS names (Linux)
nmblookup -A 10.0.0.1

# Query NetBIOS names (Windows)
nbtstat -A 10.0.0.1

# Monitor NetBIOS traffic
sudo tcpdump -i en0 port 137 or port 138 or port 139
```

#### Best Practices
- Use NetBIOS only for Windows compatibility
- Limit NetBIOS name to 15 characters
- Disable NetBIOS on non-Windows networks (security)
- Consider modern alternatives (DNS, LLMNR)

### SNMP

**Simple Network Management Protocol** - Network monitoring and management.

#### Use Cases
- Network device monitoring
- Performance metrics collection
- Alerting and notifications (traps)
- Inventory management

#### Configuration

```yaml
devices:
  - name: switch-01
    ips:
      - "10.0.0.1"
    snmp_agent:
      enabled: true
      community: "public"  # Read-only community
      walk_file: "device_walks_sanitized/cisco/niac-cisco-c3850.walk"

      # System MIB values
      sysname: "switch-01"
      sysdescr: "Cisco Catalyst 3850 - Core Switch"
      syscontact: "netadmin@example.com"
      syslocation: "DC-WEST - Row 1 Rack A01"

      # SNMP Trap configuration
      traps:
        enabled: true
        receivers:
          - "10.100.0.100:162"
          - "10.100.0.101:162"
        community: "trap-community"

        # Event-based traps
        cold_start:
          enabled: true
        link_up:
          enabled: true
        link_down:
          enabled: true
        authentication_failure:
          enabled: true

        # Threshold-based traps
        high_cpu:
          enabled: true
          threshold: 80  # Percentage
        high_memory:
          enabled: true
          threshold: 85  # Percentage
        high_disk:
          enabled: true
          threshold: 90  # Percentage
        interface_errors:
          enabled: true
          threshold: 1000  # Error count
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable SNMP agent |
| `community` | string | No | "public" | SNMP community string |
| `walk_file` | string | No | "" | Path to SNMP walk file |
| `sysname` | string | No | device name | System name |
| `sysdescr` | string | No | "" | System description |
| `syscontact` | string | No | "" | Contact information |
| `syslocation` | string | No | "" | Physical location |
| `traps` | object | No | - | Trap configuration |

#### Testing

```bash
# SNMP GET
snmpget -v2c -c public 10.0.0.1 sysName.0

# SNMP WALK
snmpwalk -v2c -c public 10.0.0.1 system

# SNMP BULK WALK
snmpbulkwalk -v2c -c public 10.0.0.1 ifDescr

# Listen for traps
snmptrapd -f -Lo
```

#### Best Practices
- Use SNMPv3 in production (authentication and encryption)
- Change default community strings ("public", "private")
- Limit SNMP access with ACLs
- Use separate communities for read-only and traps
- Populate all system MIB fields (sysname, sysdescr, etc.)
- Configure multiple trap receivers for redundancy
- Set appropriate thresholds for threshold-based traps
- Use walk files to simulate real device OIDs

## Protocol Combinations

Different network scenarios require specific protocol combinations.

### Enterprise Edge Router

**Protocols:** LLDP, CDP, DHCP, DNS, SNMP, ICMP

```yaml
devices:
  - name: edge-router
    lldp: {enabled: true}
    cdp: {enabled: true}
    dhcp: {enabled: true, pools: [...]}
    dns: {enabled: true, forward_records: [...]}
    snmp_agent: {enabled: true, traps: {...}}
    icmp: {enabled: true}
```

**Use Case:** Gateway router providing network services and monitoring.

See: `examples/combinations/enterprise-router.yaml`

### Data Center Core Switch

**Protocols:** STP, LLDP, CDP, SNMP, ICMP

```yaml
devices:
  - name: core-switch
    stp: {enabled: true, bridge_priority: 4096, version: "rstp"}
    lldp: {enabled: true}
    cdp: {enabled: true}
    snmp_agent: {enabled: true, traps: {...}}
    icmp: {enabled: true}
```

**Use Case:** Core switch with loop prevention and monitoring.

See: `examples/combinations/datacenter-switch.yaml`

### Access Layer Switch

**Protocols:** LLDP, CDP, STP, SNMP

```yaml
devices:
  - name: access-switch
    lldp: {enabled: true}
    cdp: {enabled: true}
    stp: {enabled: true, bridge_priority: 32768}
    snmp_agent: {enabled: true}
```

**Use Case:** Access switch for end-user connectivity.

See: `examples/combinations/access-switch.yaml`

### Wireless Controller

**Protocols:** HTTP, DHCP, DNS, SNMP, ICMP

```yaml
devices:
  - name: wireless-controller
    http: {enabled: true, endpoints: [...]}
    dhcp: {enabled: true, pools: [...]}
    dns: {enabled: true, forward_records: [...]}
    snmp_agent: {enabled: true}
    icmp: {enabled: true}
```

**Use Case:** Wireless LAN controller with management and services.

See: `examples/combinations/wireless-controller.yaml`

## Best Practices

### Protocol Selection

1. **Always Enable:**
   - LLDP (multi-vendor discovery)
   - SNMP (monitoring)
   - ICMP (troubleshooting)

2. **Enable When Needed:**
   - CDP (Cisco-only environments)
   - STP (Layer 2 networks with redundancy)
   - DHCP (dynamic addressing required)
   - DNS (name resolution needed)

3. **Enable Sparingly:**
   - EDP, FDP (vendor-specific)
   - HTTP, FTP (management interfaces only)
   - NetBIOS (Windows compatibility only)

### Security Considerations

1. **Disable unnecessary protocols** on edge interfaces
2. **Change default community strings** for SNMP
3. **Use SNMPv3** in production environments
4. **Disable CDP** on untrusted ports
5. **Implement access control** for SNMP, HTTP, FTP
6. **Use strong passwords** for FTP users
7. **Consider disabling ICMP** on WAN interfaces

### Performance Optimization

1. **Adjust advertisement intervals** based on network size
   - Small networks: 30s (LLDP), 60s (CDP)
   - Large networks: 60s (LLDP), 120s (CDP)

2. **Limit SNMP walk files** to necessary OIDs
3. **Use RSTP instead of STP** for faster convergence
4. **Configure appropriate trap thresholds** to avoid alert fatigue

### Monitoring and Troubleshooting

1. **Enable debug flags** for specific protocols:
   ```bash
   sudo niac --debug-lldp 3 --debug-snmp 3 en0 config.yaml
   ```

2. **Use tcpdump** to monitor protocol traffic:
   ```bash
   sudo tcpdump -i en0 ether proto 0x88cc  # LLDP
   sudo tcpdump -i en0 port 161 or port 162  # SNMP
   ```

3. **Validate configurations** before deployment:
   ```bash
   niac validate config.yaml
   ```

4. **Use dry-run mode** to test configurations:
   ```bash
   niac --dry-run lo0 config.yaml
   ```

## See Also

- [Topology Configuration Guide](TOPOLOGY_GUIDE.md) - Port-channels, trunks, VLANs
- [Environment Simulation Guide](ENVIRONMENTS.md) - Complete network examples
- [API Reference](API_REFERENCE.md) - Complete YAML schema
- [Examples](../examples/) - Ready-to-use configurations
