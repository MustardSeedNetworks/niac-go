# API Reference

Complete YAML configuration schema reference for NiAC-Go.

## Table of Contents

- [Overview](#overview)
- [Top-Level Structure](#top-level-structure)
- [Device Configuration](#device-configuration)
- [Protocol Configuration](#protocol-configuration)
  - [LLDP](#lldp)
  - [CDP](#cdp)
  - [EDP](#edp)
  - [FDP](#fdp)
  - [STP](#stp)
  - [DHCP (DHCPv4)](#dhcp-dhcpv4)
  - [DHCPv6](#dhcpv6)
  - [DNS](#dns)
  - [HTTP](#http)
  - [FTP](#ftp)
  - [NetBIOS](#netbios)
  - [ICMP](#icmp)
  - [ICMPv6](#icmpv6)
  - [SNMP](#snmp)
- [Topology Configuration](#topology-configuration)
  - [Port Channels](#port-channels)
  - [Trunk Ports](#trunk-ports)
- [Traffic Configuration](#traffic-configuration)
- [Default Values](#default-values)
- [Validation Rules](#validation-rules)

## Overview

NiAC-Go uses YAML configuration files to define network devices and their protocols. This reference documents all available fields, their types, default values, and validation rules.

**Target Go Version:** 1.23+

## Top-Level Structure

```yaml
devices:
  - name: device-01
    # ... device configuration
  - name: device-02
    # ... device configuration
```

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `devices` | array | Yes | [] | List of device configurations |

## Device Configuration

```yaml
devices:
  - name: "switch-01"
    type: "switch"
    mac: "00:11:22:33:44:01"
    ips:
      - "10.0.0.1"
      - "2001:db8::1"

    # Protocol configurations
    lldp: {...}
    cdp: {...}
    edp: {...}
    fdp: {...}
    stp: {...}
    dhcp: {...}
    dhcpv6: {...}
    dns: {...}
    http: {...}
    ftp: {...}
    netbios: {...}
    icmp: {...}
    icmpv6: {...}
    snmp_agent: {...}

    # Topology configurations (v1.23.0)
    port_channels: [...]
    trunk_ports: [...]

    # Traffic configurations (v1.6.0)
    traffic: {...}
```

### Core Device Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique device identifier |
| `type` | string | No | "" | Device type: router, switch, ap, etc. |
| `mac` | string | Yes | - | MAC address (format: 00:11:22:33:44:55) |
| `ips` | string array | No | [] | IPv4 and/or IPv6 addresses |

### Device Type Values

| Type | Description |
|------|-------------|
| `router` | Layer 3 router |
| `switch` | Layer 2/3 switch |
| `ap` | Wireless access point |
| `server` | Generic server |
| `workstation` | End-user device |
| (custom) | Any custom string |

## Protocol Configuration

### LLDP

**Link Layer Discovery Protocol** - IEEE 802.1AB

```yaml
lldp:
  enabled: true
  system_name: "switch-01"
  system_description: "Cisco Catalyst 3850"
  chassis_id: "00:11:22:33:44:01"
  port_description: "GigabitEthernet0/1"
  advertise_interval: 30
  management_address: "10.0.0.1"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable LLDP |
| `system_name` | string | No | device name | System name |
| `system_description` | string | No | "" | Device description |
| `chassis_id` | string | No | device MAC | Chassis ID |
| `port_description` | string | No | "" | Port description |
| `advertise_interval` | integer | No | 30 | Advertisement interval (seconds) |
| `management_address` | string | No | first IP | Management IP address |

#### Constraints

- `advertise_interval`: 1-3600 seconds
- `system_name`: Max 255 characters
- `chassis_id`: Valid MAC address or string

### CDP

**Cisco Discovery Protocol** - Cisco proprietary

```yaml
cdp:
  enabled: true
  platform: "WS-C3850-48P"
  capabilities: "Switch IGMP"
  software_version: "IOS-XE 16.12.4"
  advertise_interval: 60
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable CDP |
| `platform` | string | No | "" | Platform/model identifier |
| `capabilities` | string | No | "Switch" | Device capabilities |
| `software_version` | string | No | "" | Software version |
| `advertise_interval` | integer | No | 60 | Advertisement interval (seconds) |

#### Constraints

- `advertise_interval`: 5-3600 seconds
- Common capabilities: "Router", "Switch", "IGMP", "Host"

### EDP

**Extreme Discovery Protocol** - Extreme Networks proprietary

```yaml
edp:
  enabled: true
  platform: "X460-G2"
  software_version: "ExtremeXOS 30.7.1.4"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable EDP |
| `platform` | string | No | "" | Platform identifier |
| `software_version` | string | No | "" | Software version |

### FDP

**Foundry Discovery Protocol** - Brocade/Foundry proprietary

```yaml
fdp:
  enabled: true
  platform: "ICX7450-48P"
  software_version: "08.0.95"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable FDP |
| `platform` | string | No | "" | Platform identifier |
| `software_version` | string | No | "" | Software version |

### STP

**Spanning Tree Protocol** - IEEE 802.1D/w/s

```yaml
stp:
  enabled: true
  bridge_priority: 4096
  hello_time: 2
  max_age: 20
  forward_delay: 15
  version: "rstp"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable STP |
| `bridge_priority` | integer | No | 32768 | Bridge priority |
| `hello_time` | integer | No | 2 | BPDU interval (seconds) |
| `max_age` | integer | No | 20 | BPDU max age (seconds) |
| `forward_delay` | integer | No | 15 | Forward delay (seconds) |
| `version` | string | No | "rstp" | STP version |

#### Constraints

- `bridge_priority`: 0-61440, increments of 4096
- `hello_time`: 1-10 seconds
- `max_age`: 6-40 seconds (must be >= 2 * (hello_time + 1))
- `forward_delay`: 4-30 seconds (must be >= (max_age / 2) + 1)
- `version`: "stp", "rstp", or "mstp"

### DHCP (DHCPv4)

**Dynamic Host Configuration Protocol** - IPv4 address assignment

```yaml
dhcp:
  enabled: true
  pools:
    - network: "10.0.10.0/24"
      range_start: "10.0.10.100"
      range_end: "10.0.10.200"
      gateway: "10.0.0.1"
      dns_servers: ["10.0.0.1", "8.8.8.8"]
      lease_time: 86400
      domain_name: "corp.example.com"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable DHCP server |
| `pools` | array | Yes | [] | DHCP address pools |

#### Pool Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `network` | string | Yes | - | Network CIDR (e.g., 10.0.0.0/24) |
| `range_start` | string | Yes | - | First IP in range |
| `range_end` | string | Yes | - | Last IP in range |
| `gateway` | string | No | "" | Default gateway IP |
| `dns_servers` | string array | No | [] | DNS server IPs |
| `lease_time` | integer | No | 86400 | Lease time (seconds) |
| `domain_name` | string | No | "" | Domain name |

#### Constraints

- `network`: Valid IPv4 CIDR notation
- `range_start`, `range_end`: Must be within network
- `range_start` must be <= `range_end`
- `lease_time`: 60-31536000 seconds (1 minute to 1 year)

### DHCPv6

**Dynamic Host Configuration Protocol** - IPv6 address assignment

```yaml
dhcpv6:
  enabled: true
  pools:
    - network: "2001:db8:1::/64"
      range_start: "2001:db8:1::100"
      range_end: "2001:db8:1::200"
      dns_servers: ["2001:db8::1"]
      lease_time: 86400
      domain_name: "corp.example.com"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable DHCPv6 server |
| `pools` | array | Yes | [] | DHCPv6 address pools |

#### Pool Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `network` | string | Yes | - | IPv6 network (e.g., 2001:db8::/64) |
| `range_start` | string | Yes | - | First IPv6 in range |
| `range_end` | string | Yes | - | Last IPv6 in range |
| `dns_servers` | string array | No | [] | IPv6 DNS server addresses |
| `lease_time` | integer | No | 86400 | Lease time (seconds) |
| `domain_name` | string | No | "" | Domain name |

#### Constraints

- `network`: Valid IPv6 CIDR notation
- Standard subnet: /64 prefix
- `range_start`, `range_end`: Must be within network

### DNS

**Domain Name System** - Name resolution

```yaml
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
| `forward_records` | array | No | [] | A/AAAA records |

#### Forward Record Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Fully qualified domain name |
| `ip` | string | Yes | - | IPv4 or IPv6 address |
| `ttl` | integer | No | 3600 | Time to live (seconds) |

#### Constraints

- `name`: Valid FQDN (max 255 characters)
- `ip`: Valid IPv4 or IPv6 address
- `ttl`: 0-2147483647 seconds

### HTTP

**Hypertext Transfer Protocol** - Web server

```yaml
http:
  enabled: true
  port: 80
  endpoints:
    - path: "/"
      content: "<h1>Welcome</h1>"
    - path: "/api/info"
      content: '{"status":"ok"}'
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable HTTP server |
| `port` | integer | No | 80 | TCP port |
| `endpoints` | array | No | [] | HTTP endpoints |

#### Endpoint Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | - | URL path (e.g., /api/info) |
| `content` | string | Yes | - | Response content |

#### Constraints

- `port`: 1-65535
- `path`: Must start with /

### FTP

**File Transfer Protocol** - File transfer service

```yaml
ftp:
  enabled: true
  port: 21
  users:
    - username: "admin"
      password: "admin123"
    - username: "guest"
      password: "guest"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable FTP server |
| `port` | integer | No | 21 | TCP port |
| `users` | array | No | [] | User accounts |

#### User Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `username` | string | Yes | - | FTP username |
| `password` | string | Yes | - | FTP password |

#### Constraints

- `port`: 1-65535
- `username`: 1-255 characters, no spaces
- `password`: 1-255 characters

### NetBIOS

**Network Basic Input/Output System** - Windows networking

```yaml
netbios:
  enabled: true
  name: "FILESERVER"
  workgroup: "WORKGROUP"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable NetBIOS |
| `name` | string | No | device name | NetBIOS name |
| `workgroup` | string | No | "WORKGROUP" | Workgroup name |

#### Constraints

- `name`: Max 15 characters, uppercase recommended
- `workgroup`: Max 15 characters

### ICMP

**Internet Control Message Protocol** - IPv4 diagnostics

```yaml
icmp:
  enabled: true
  ttl: 64
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable ICMP responses |
| `ttl` | integer | No | 64 | Time to live |

#### Constraints

- `ttl`: 1-255

### ICMPv6

**Internet Control Message Protocol for IPv6** - IPv6 diagnostics

```yaml
icmpv6:
  enabled: true
  hop_limit: 255
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable ICMPv6 responses |
| `hop_limit` | integer | No | 255 | Hop limit (RFC 4861) |

#### Constraints

- `hop_limit`: 1-255
- **Important**: Use 255 for NDP (RFC 4861 requirement)

### SNMP

**Simple Network Management Protocol** - Network monitoring

```yaml
snmp_agent:
  enabled: true
  community: "public"
  walk_file: "device_walks_sanitized/cisco/niac-cisco-c3850.walk"

  sysname: "switch-01"
  sysdescr: "Cisco Catalyst 3850"
  syscontact: "netadmin@example.com"
  syslocation: "DC-WEST - Rack A01"

  traps:
    enabled: true
    receivers:
      - "10.100.0.100:162"
      - "10.100.0.101:162"
    community: "trap-community"

    cold_start:
      enabled: true
    link_up:
      enabled: true
    link_down:
      enabled: true
    authentication_failure:
      enabled: true

    high_cpu:
      enabled: true
      threshold: 80
    high_memory:
      enabled: true
      threshold: 85
    high_disk:
      enabled: true
      threshold: 90
    interface_errors:
      enabled: true
      threshold: 1000
```

#### Agent Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable SNMP agent |
| `community` | string | No | "public" | Community string |
| `walk_file` | string | No | "" | Path to SNMP walk file |
| `sysname` | string | No | device name | System name |
| `sysdescr` | string | No | "" | System description |
| `syscontact` | string | No | "" | Contact information |
| `syslocation` | string | No | "" | Physical location |
| `traps` | object | No | - | Trap configuration |

#### Trap Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | false | Enable SNMP traps |
| `receivers` | string array | Yes | [] | Trap receiver IPs (IP:port) |
| `community` | string | No | "public" | Trap community string |

#### Event-Based Traps

| Trap | Fields | Description |
|------|--------|-------------|
| `cold_start` | enabled (bool) | Device startup |
| `link_up` | enabled (bool) | Interface up |
| `link_down` | enabled (bool) | Interface down |
| `authentication_failure` | enabled (bool) | SNMP auth failure |

#### Threshold-Based Traps

| Trap | Fields | Description |
|------|--------|-------------|
| `high_cpu` | enabled (bool), threshold (int) | CPU usage > threshold % |
| `high_memory` | enabled (bool), threshold (int) | Memory usage > threshold % |
| `high_disk` | enabled (bool), threshold (int) | Disk usage > threshold % |
| `interface_errors` | enabled (bool), threshold (int) | Interface errors > threshold |

#### Constraints

- `community`: 1-255 characters
- `receivers`: Format "IP:port" (e.g., "10.0.0.1:162")
- `threshold`: 0-100 for percentages, 0+ for counts

## Topology Configuration

### Port Channels

**Port-Channel (LAG) Configuration** - Added in v1.23.0

```yaml
port_channels:
  - id: 1
    members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
    mode: "active"
  - id: 2
    members: ["GigabitEthernet0/3", "GigabitEthernet0/4"]
    mode: "active"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | integer | Yes | - | Port-channel ID |
| `members` | string array | Yes | [] | Member interface names |
| `mode` | string | No | "active" | LACP mode |

#### LACP Modes

| Mode | Description |
|------|-------------|
| `active` | Actively sends LACP packets (recommended) |
| `passive` | Responds to LACP packets |
| `on` | Static LAG without LACP |

#### Constraints

- `id`: Unique per device, 1-4096
- `members`: At least 1 interface, max 8
- Member interfaces must not be in multiple port-channels
- All members must have same speed and duplex

### Trunk Ports

**Trunk Port (VLAN Tagging) Configuration** - Added in v1.23.0

```yaml
trunk_ports:
  - interface: "GigabitEthernet0/10"
    vlans: [1, 10, 20, 30]
    native_vlan: 1
    remote_device: "switch-02"
    remote_interface: "GigabitEthernet0/10"
  - interface: "port-channel1"
    vlans: [1, 10, 20, 30, 40]
    native_vlan: 1
    remote_device: "switch-03"
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `interface` | string | Yes | - | Local interface (physical or port-channel) |
| `vlans` | integer array | Yes | [] | Allowed VLANs |
| `native_vlan` | integer | No | 1 | Untagged VLAN |
| `remote_device` | string | No | "" | Remote device name |
| `remote_interface` | string | No | "" | Remote interface name |

#### Constraints

- `interface`: Valid interface name or "port-channel{N}"
- `vlans`: 1-4094 (avoid 1002-1005, 4095)
- `native_vlan`: 1-4094, must be in vlans list
- `remote_device`: Must exist in devices list (for validation)

## Traffic Configuration

**Traffic Pattern Configuration** - Added in v1.6.0

```yaml
traffic:
  enabled: true
  arp_announcements:
    enabled: true
    interval: 60
  periodic_pings:
    enabled: true
    interval: 120
    payload_size: 32
  random_traffic:
    enabled: true
    interval: 180
    packet_count: 5
    patterns: ["broadcast_arp", "multicast", "udp"]
```

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | No | false | Enable traffic generation |
| `arp_announcements` | object | No | - | Gratuitous ARP config |
| `periodic_pings` | object | No | - | Periodic ICMP config |
| `random_traffic` | object | No | - | Random traffic config |

### ARP Announcements

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | No | false | Enable ARP announcements |
| `interval` | integer | No | 60 | Interval (seconds) |

### Periodic Pings

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | No | false | Enable periodic pings |
| `interval` | integer | No | 120 | Interval (seconds) |
| `payload_size` | integer | No | 32 | Payload size (bytes) |

### Random Traffic

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | boolean | No | false | Enable random traffic |
| `interval` | integer | No | 180 | Interval (seconds) |
| `packet_count` | integer | No | 5 | Packets per interval |
| `patterns` | string array | No | [] | Traffic patterns |

#### Traffic Patterns

- `broadcast_arp`: Broadcast ARP packets
- `multicast`: Multicast packets
- `udp`: Random UDP packets

## Default Values

### Discovery Protocols

| Protocol | Field | Default |
|----------|-------|---------|
| LLDP | advertise_interval | 30 seconds |
| LLDP | ttl | 120 seconds |
| CDP | advertise_interval | 60 seconds |
| CDP | holdtime | 180 seconds |
| EDP | advertise_interval | 30 seconds |
| FDP | advertise_interval | 60 seconds |

### Spanning Tree

| Field | Default |
|-------|---------|
| bridge_priority | 32768 |
| hello_time | 2 seconds |
| max_age | 20 seconds |
| forward_delay | 15 seconds |
| version | "rstp" |

### Network Protocols

| Protocol | Field | Default |
|----------|-------|---------|
| ICMP | ttl | 64 |
| ICMPv6 | hop_limit | 64 (255 for NDP) |
| DHCP | lease_time | 86400 seconds (24 hours) |
| DHCPv6 | lease_time | 86400 seconds (24 hours) |
| DNS | ttl | 3600 seconds (1 hour) |

### Application Protocols

| Protocol | Field | Default |
|----------|-------|---------|
| HTTP | port | 80 |
| FTP | port | 21 |
| SNMP | community | "public" |
| NetBIOS | workgroup | "WORKGROUP" |
| NetBIOS | ttl | 300 seconds |

### SNMP Trap Thresholds

| Trap | Default Threshold |
|------|-------------------|
| high_cpu | 80% |
| high_memory | 90% |
| high_disk | 90% |
| interface_errors | 100 errors |

### Traffic Patterns

| Field | Default |
|-------|---------|
| arp_announcement_interval | 60 seconds |
| periodic_ping_interval | 120 seconds |
| periodic_ping_payload_size | 32 bytes |
| random_traffic_interval | 180 seconds |
| random_traffic_packet_count | 5 packets |

## Validation Rules

### Device Validation

- ✅ Device name must be unique
- ✅ MAC address must be valid format (00:11:22:33:44:55)
- ✅ MAC address must be unique across devices
- ✅ IP addresses must be valid IPv4 or IPv6 format
- ✅ At least one IP address or MAC address required

### Protocol Validation

- ✅ Only one enabled per discovery protocol group
- ✅ SNMP walk_file path must exist if specified
- ✅ DHCP pools must not overlap
- ✅ DNS record names must be valid FQDNs
- ✅ STP bridge_priority must be multiple of 4096

### Topology Validation

- ✅ Port-channel ID must be unique per device
- ✅ Port-channel members must not be in multiple port-channels
- ✅ Trunk VLAN IDs must be 1-4094 (excluding 1002-1005, 4095)
- ✅ Trunk native_vlan must be in vlans list
- ✅ Remote device must exist in configuration
- ✅ Port-channel referenced in trunk must exist

### Trap Validation

- ✅ Trap receivers must be valid IP:port format
- ✅ Thresholds must be 0-100 for percentages
- ✅ Thresholds must be >= 0 for counts
- ✅ At least one receiver required if traps enabled

## Examples

### Minimal Configuration

```yaml
devices:
  - name: device-01
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1"]
```

### Full-Featured Device

```yaml
devices:
  - name: enterprise-router
    type: router
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1", "198.51.100.1"]

    lldp:
      enabled: true
      system_name: "enterprise-router"
      system_description: "Cisco 4331 ISR"
      advertise_interval: 30

    cdp:
      enabled: true
      platform: "Cisco 4331 ISR"
      software_version: "IOS XE 16.12.4"

    dhcp:
      enabled: true
      pools:
        - network: "10.0.10.0/24"
          range_start: "10.0.10.100"
          range_end: "10.0.10.200"
          gateway: "10.0.0.1"
          dns_servers: ["10.0.0.1", "8.8.8.8"]
          lease_time: 86400

    dns:
      enabled: true
      forward_records:
        - name: "router.example.com"
          ip: "10.0.0.1"
          ttl: 3600

    snmp_agent:
      enabled: true
      community: "public"
      sysname: "enterprise-router"
      sysdescr: "Cisco 4331 ISR"
      syscontact: "netadmin@example.com"
      syslocation: "DC-WEST"
      traps:
        enabled: true
        receivers: ["10.100.0.100:162"]
        high_cpu:
          enabled: true
          threshold: 80

    icmp:
      enabled: true
      ttl: 64
```

### Topology Configuration

```yaml
devices:
  - name: core-switch-01
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1"]

    port_channels:
      - id: 1
        members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
        mode: "active"

    trunk_ports:
      - interface: "port-channel1"
        vlans: [1, 10, 20, 30]
        native_vlan: 1
        remote_device: "core-switch-02"
        remote_interface: "port-channel1"

    lldp:
      enabled: true

    snmp_agent:
      enabled: true
```

## See Also

- [Protocol Configuration Guide](PROTOCOL_GUIDE.md) - Detailed protocol documentation
- [Topology Configuration Guide](TOPOLOGY_GUIDE.md) - Port-channels and trunks
- [Environment Simulation Guide](ENVIRONMENTS.md) - Complete examples
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Common issues
- [Examples](../examples/) - Ready-to-use configurations
