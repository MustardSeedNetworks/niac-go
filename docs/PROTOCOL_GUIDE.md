# Protocol Configuration Guide

This guide covers all 21 protocols supported by NIAC-Go, including
configuration examples, use cases, and best practices.

The `examples/` files referenced throughout arrive with the shared demo
catalog: run `./scripts/sync-demo-catalog.sh --sync` to populate `examples/`,
which ships holding only its README.

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
  - [mDNS](#mdns)
  - [SNMP](#snmp)
  - [SSH Device Management](#ssh-device-management)
  - [SIP](#sip)
  - [ONVIF Discovery](#onvif-discovery)
  - [JetDirect](#jetdirect)
  - [DICOM](#dicom)
  - [HL7](#hl7)
  - [Modbus/TCP](#modbustcp)
  - [EtherNet/IP](#ethernetip)
- [Protocol Combinations](#protocol-combinations)
- [Best Practices](#best-practices)

## Overview

NIAC-Go supports 24 network protocols across all layers of the OSI model.
Each protocol can be independently configured per device via YAML
configuration.

### Supported Protocols by Layer

| Layer | Protocols |
| ------- | ----------- |
| Layer 2 (Data Link) | LLDP, CDP, EDP, FDP, STP |
| Layer 3 (Network) | IPv4, IPv6, ARP, ICMP, ICMPv6 |
| Layer 4 (Transport) | TCP, UDP |
| Layer 7 (Application) | DHCP, DHCPv6, DNS, HTTP, FTP, NetBIOS, mDNS, SNMP, SSH, SIP, WS-Discovery, JetDirect, DICOM, HL7, Modbus/TCP, EtherNet/IP |

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
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable LLDP |
| `system_name` | string | No | device name | System name advertised |
| `system_description` | string | No | "" | Device description |
| `chassis_id` | string | No | device MAC | Chassis identifier |
| `port_description` | string | No | interface | Port description |
| `advertise_interval` | integer | No | 30 | Advertisement interval (seconds) |
| `management_address` | string | No | device IP | Management IP address |
| `med` | object | No | none | LLDP-MED (TIA-1057) extensions — see below |

#### LLDP-MED (TIA-1057)

A discovery tool reads LLDP-MED to tell a phone from a printer and to learn the
voice VLAN an endpoint belongs on. Without it an IP phone or access point
appears as an anonymous endpoint. Omit the `med` block and the device sends
plain 802.1AB, which is what a switch or router does.

```yaml
devices:
  - name: clinic-phone-01
    lldp:
      enabled: true
      med:
        device_type: endpoint_class3   # a communication device, e.g. a phone
        network_policies:
          - application: voice
            tagged: true
            vlan_id: 110
            priority: 5                # 802.1p
            dscp: 46                   # expedited forwarding
          - application: voice_signaling
            tagged: true
            vlan_id: 110
            priority: 3
            dscp: 24
        power:
          device_type: pd              # the device drawing power
          source: pse
          priority: high
          value_tenth_watts: 65        # 6.5 W, the TLV's own unit
        inventory:
          hardware_revision: "1.0"
          firmware_revision: "SIP88xx.12-8-1"
          serial_number: "FCH2043E0AB"
          manufacturer: "Cisco Systems"
          model_name: "CP-8841"
```

| Field | Values |
| ------- | -------- |
| `device_type` | `endpoint_class1`, `endpoint_class2`, `endpoint_class3`, `network_connectivity` |
| `network_policies[].application` | `voice`, `voice_signaling`, `guest_voice`, `guest_voice_signaling`, `softphone_voice`, `video_conferencing`, `streaming_video`, `video_signaling` |
| `power.device_type` | `pse` (supplying), `pd` (drawing) |
| `power.source` | `unknown`, `primary`, `backup`, `pse`, `local`, `pse_local` |
| `power.priority` | `unknown`, `critical`, `high`, `low` |

The capabilities TLV is derived from what the device actually advertises rather
than declared separately, so it can never claim a capability the frame does not
carry. An application NIAC does not recognise is dropped rather than sent as
type 0, which a receiver reads as malformed.

The generated vertical packs give every access point a MED block: a
`network_connectivity` class, the voice policy its clients use, its PoE draw,
and its model and serial.

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
| ------- | ------ | ---------- | --------- | ------------- |
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
| --------- | ------ | ----- |
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
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable STP |
| `bridge_priority` | integer | No | 32768 | Bridge priority (0-61440) |
| `hello_time` | integer | No | 2 | BPDU interval (1-10 seconds) |
| `max_age` | integer | No | 20 | BPDU max age (6-40 seconds) |
| `forward_delay` | integer | No | 15 | Forward delay (4-30 seconds) |
| `version` | string | No | "stp" | STP version: stp, rstp, mstp |

#### What the Simulation Sends

Every device with `enabled: true` sends an IEEE 802.1D Configuration BPDU every
`hello_time` seconds, whatever `version` says. The root, the root path cost and
the root port come from one election over the `trunk_ports` that join
STP-enabled devices: the lowest bridge ID (priority, then MAC) is root, and
every other bridge takes its cheapest path there, each link costing 4. The
`dot1dStp` group reports the same tree, so a BPDU and an SNMP poll of the same
switch agree. Received BPDUs are logged, not acted on. The scenario packs make
each site's first core the root (24576) and its second the standby (28672).

#### STP Versions

| Version | Standard | Convergence | Use Case |
| --------- | ---------- | ------------- | ---------- |
| STP | IEEE 802.1D | 30-50 seconds | Legacy compatibility |
| RSTP | IEEE 802.1w | <6 seconds | Modern networks (recommended) |
| MSTP | IEEE 802.1s | <6 seconds | VLAN-aware, complex topologies |

#### Bridge Priority Values

| Priority | Purpose |
| ---------- | --------- |
| 0 or 4096 | Force as root bridge |
| 8192 | Secondary/backup root |
| 32768 | Default (let election decide) |
| 61440 | Never become root (edge switches) |

#### Testing

```bash
# Monitor STP BPDUs
sudo tcpdump -i en0 ether dst 01:80:c2:00:00:00

# Run with STP debug
sudo niac daemon --once -d 3 en0 config.yaml
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
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable ICMP responses |
| `ttl` | integer | No | 64 | IP TTL (1-255) of every IPv4 packet the device sends, unless `os_fingerprint.ttl` sets one |

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
| ------- | ------ | ---------- | --------- | ------------- |
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

#### UDP Reflector

A device can act as a **NetAlly-style UDP reflector** — the endpoint a
performance/TrueSpeed tester bounces probes off to measure round-trip latency,
jitter, and DiffServ handling. When a UDP packet arrives whose payload carries a
reflector signature (`DATA:OT` or `PROBEOT`, 5 bytes into the payload), the
device echoes it back to the sender with source/destination MAC and IP swapped,
the ToS byte wiggled, and the reply optionally delayed. UDP ports are left
unswapped to match the `niac-java` reflector; the tester matches replies by
signature, not the 4-tuple. Replies inherit the request's VLAN tag.

##### Configuration

```yaml
devices:
  - name: reflector
    mac: "bb:bb:bb:00:00:02"
    ips: ["10.20.200.100"]
    reflector:
      latency_ms: 10   # delay before reflecting (0 = immediate)
      jitter_ms: 2     # random +/- delay around latency
      dscp: false      # false: wiggle IP-precedence bit (0x01); true: wiggle DSCP bottom-2 bits (0x03)
```

##### Fields

| Field | Type | Description |
| --- | --- | --- |
| `latency_ms` | integer | One-way delay before the reflected packet is sent, in milliseconds. `0` reflects immediately. |
| `jitter_ms` | integer | Randomises the delay by +/- this many milliseconds, floored at zero. |
| `dscp` | boolean | Selects which ToS bits are toggled: `false` (default) flips the IP-precedence bit, `true` flips the bottom two DSCP bits. |

The presence of the `reflector:` block enables the reflector — there is no
separate enable flag.

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
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable DHCP server |
| `pools` | array | Yes | [] | DHCP address pools |

**Pool Fields:**

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
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
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable DNS server |
| `forward_records` | array | No | [] | A records (hostname -> IP) |
| `forwarders` | array of IP | No | [] | Resolvers asked, in order, for a name this server holds no record for |

**Forward Record Fields:**

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `name` | string | Yes | - | Fully qualified domain name |
| `ip` | string | Yes | - | IPv4 or IPv6 address |
| `ttl` | integer | No | 3600 | Time to live (seconds) |

#### Forwarding

A server answers from its own records first. For any other name it asks its
`forwarders` in order, the way a site resolver forwards to its upstream:

- A forwarder that is another simulated DNS server answers from its own
  records, and its own forwarders, inside the simulation. This is how a site
  resolver resolves internet names: generated scenario packs point every site
  resolver at the lab edge (`10.254.200.1`), which holds the simulated internet
  (`www.google.com` and friends resolve to `8.8.8.8`, which the scenario
  answers ping and HTTP on) plus every scenario host.
- Any other address is a real resolver, asked over the host's network only
  when the simulated chain has no answer. This is opt-in: NIAC makes no
  outbound DNS query unless a forwarder you author points outside the
  simulation. A real answer is passed through non-authoritatively; when no
  real resolver replies within two seconds the client gets SERVFAIL.

```yaml
devices:
  - name: site-dns
    ips: ["10.51.240.10"]
    dns:
      forwarders: ["10.254.200.1"]   # the lab edge, simulated
  - name: lab-edge
    ips: ["10.254.200.1"]
    dns:
      forwarders: ["1.1.1.1"]        # optional: real upstream for everything else
      forward_records:
        - name: "www.google.com"
          ip: "8.8.8.8"
```

A forwarder may not be the server's own address. Real upstream answers point
at real internet addresses the simulation does not route to, so a tester's
follow-on ping or HTTP test to such a name fails; names the scenario models
belong in the edge's records.

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
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable HTTP server |
| `port` | integer | No | 80 | TCP port |
| `endpoints` | array | No | [] | HTTP endpoints |

**Endpoint Fields:**

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
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
      welcome_banner: "220 JD FTP Server Ready"
      system_type: "UNIX Type: L8"
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Serve FTP on TCP 21 |
| `welcome_banner` | string | No | `220 <name> FTP Server (NIAC-Go) ready.` | The 220 greeting sent when a client connects |
| `system_type` | string | No | `UNIX Type: L8` | The `SYST` reply |
| `allow_anonymous`, `users` | | No | | Parsed but not enforced: every login is accepted (#2693) |

#### Behaviour

The server speaks first. After the TCP handshake it sends `welcome_banner`,
then answers each command line in order. Only the control channel is served:
`PASV` and `EPSV` name port 20000, which accepts no connection (#2693). A device
without `ftp.enabled` resets TCP 21. IPv6 is not served.

The HP printers in the scenario packs greet with `220 JD FTP Server Ready`.
That greeting comes from HP's Jetdirect administrator's guide. It is
documented, not captured. No other pack device serves FTP, because no other
vendor in the packs publishes its greeting.

#### Testing

```bash
# The greeting, the login and the failed listing
curl -v ftp://10.0.0.1/

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
| ------- | ------ | ---------- | --------- | ------------- |
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

### mDNS

**Multicast DNS** - link-local name and service advertisement, as published by
Bonjour and Avahi.

#### Use Cases

- Printers, cameras, and appliances that a scanner finds without unicast DNS
- DNS-SD service inventories (`_ipp._tcp`, `_http._tcp`)
- Discovery tools that classify a device from its advertised services
- Networks with no authoritative DNS for the local segment

#### Configuration

```yaml
devices:
  - name: floor2-printer
    ips:
      - "10.0.0.60"
    mdns:
      enabled: true
      hostname: "floor2-printer"
      ttl: 120
      services:
        - type: "_ipp._tcp"
          port: 631
          txt:
            - "rp=ipp/print"
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable mDNS advertisement |
| `hostname` | string | No | device name | Published as `<hostname>.local` |
| `ttl` | integer | No | 120 | Record TTL in seconds |
| `services` | array | No | - | Advertised DNS-SD services |

Each entry in `services` takes:

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `type` | string | Yes | - | DNS-SD service type (e.g. `_ipp._tcp`) |
| `port` | integer | Yes | - | Port the service listens on |
| `txt` | array | No | - | TXT record key=value strings |

#### Testing

```bash
# Browse advertised services (macOS)
dns-sd -B _services._dns-sd._udp local

# Resolve a published host (Linux)
avahi-resolve --name floor2-printer.local

# Monitor mDNS traffic
sudo tcpdump -i en0 port 5353
```

#### Best Practices

- Advertise only the services the device would really publish
- Keep hostnames unique per segment; `.local` collisions are resolved by the client
- Use TXT records that match the vendor's real service metadata
- Pair with SNMP so a discovery tool corroborates the same device two ways

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
          on_startup: true
        link_state:
          enabled: true
          link_down: true
          link_up: true

  # SNMPv3 notifications, acknowledged
  - name: clinic-core-01
    snmpv3:
      enabled: true
      users:
        - username: niac-notify
          auth_protocol: sha256
          auth_password: "authpassphrase"
          priv_protocol: aes
          priv_password: "privpassphrase"
    snmp_agent:
      traps:
        enabled: true
        receivers: ["10.100.0.100:162"]
        version: v3               # v2c (default) or v3
        security_user: niac-notify  # omit to use this device's first user
        inform: true              # acknowledged, rather than fire-and-forget
        inform_retries: 3
        inform_timeout_seconds: 5
        link_state: {enabled: true, link_down: true, link_up: true}

##### Notification version and delivery

A v2c trap carries its community string in the clear, so a manager configured
for v3-only rejects it — a device that can only send v2c is silent to that
manager. Setting `version: v3` sends the notification authenticated, and
encrypted when the named user has a privacy protocol, using the same engine that
answers the device's queries.

A trap is fire-and-forget: if it is dropped, the device believes it reported the
event and the manager never heard of it. An inform is answered with a Response
carrying the same request ID, and an unanswered one is resent
`inform_retries` times at `inform_timeout_seconds` intervals before NIAC logs
that it was never acknowledged.

| Field | Type | Default | Description |
| ------- | ------ | --------- | ------------- |
| `version` | string | `v2c` | `v2c` or `v3` |
| `security_user` | string | first user | Which `snmpv3` user a v3 notification is sent as |
| `inform` | boolean | false | Send an InformRequest instead of a trap |
| `inform_retries` | integer | 3 | Resends of an unacknowledged inform |
| `inform_timeout_seconds` | integer | 5 | Wait before resending |

```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Enable SNMP agent |
| `community` | string | No | "public" | SNMP community string |
| `walk_file` | string | No | "" | Path to SNMP walk file |
| `sysname` | string | No | device name | System name |
| `sysdescr` | string | No | "" | System description |
| `syscontact` | string | No | "" | Contact information |
| `syslocation` | string | No | "" | Physical location |
| `traps` | object | No | - | Trap configuration |

#### SNMPv3 (USM)

SNMPv3 is configured with a device-level `snmpv3:` block (a sibling of
`snmp_agent:`). The simulator acts as an authoritative engine: it performs
engine discovery and answers `noAuthNoPriv`, `authNoPriv`, and `authPriv`
requests. Engine IDs are derived from the device MAC unless `engine_id` (hex)
is set. SNMPv3 is free on all tiers — it is the only SNMP version that does not
send credentials in cleartext.

```yaml
snmpv3:
  enabled: true
  engine_id: ""        # optional hex; auto-derived from MAC when empty
  users:
    - username: "niacadmin"
      auth_protocol: "sha"   # none | md5 | sha | sha224 | sha256 | sha384 | sha512
      auth_password: "authpass123"
      priv_protocol: "aes"   # none | des | aes | aes192 | aes256
      priv_password: "privpass123"
```

The same walk data served over v1/v2c is served over v3 — the security model
only governs authentication and encryption, not the MIB.

The scenario packs author one v3 user on every switch, router, firewall,
access point, controller and server, alongside the v2c community: `netops`,
authPriv, SHA-256 with passphrase `NetAllyDemoAuth` and AES-128 with
`NetAllyDemoPriv`. Endpoint appliances (printers, UPSes, clinical and
industrial devices) stay v2c-only.

A request the engine refuses gets the RFC 3414 Report a real agent sends, so
a manager shows the cause instead of a timeout: an unknown engine ID
(`usmStatsUnknownEngineIDs`, which is also how engine discovery works), an
unknown user (`usmStatsUnknownUserNames`), a security level other than the
user's (`usmStatsUnsupportedSecLevels`), a wrong authentication passphrase
(`usmStatsWrongDigests`), a wrong privacy passphrase
(`usmStatsDecryptionErrors`) or a stale engine time
(`usmStatsNotInTimeWindows`). A user answers only at its configured level: a
`noAuthNoPriv` request naming an authPriv user is refused, not served.

#### Testing

```bash
# SNMP GET
snmpget -v2c -c public 10.0.0.1 sysName.0

# SNMP WALK
snmpwalk -v2c -c public 10.0.0.1 system

# SNMP BULK WALK
snmpbulkwalk -v2c -c public 10.0.0.1 ifDescr

# SNMPv3 authPriv walk (matches the users block above)
snmpwalk -v3 -l authPriv -u niacadmin \
  -a SHA -A authpass123 -x AES -X privpass123 10.0.0.1 system

# Listen for traps
snmptrapd -f -Lo
```

#### Power over Ethernet (POWER-ETHERNET-MIB)

A switch with a `poe` block answers POWER-ETHERNET-MIB (RFC 3621,
`.1.3.6.1.2.1.105`), which is what a tester reads to see how much of a wiring
closet's power budget is in use:

```yaml
devices:
  - name: closet-sw-01
    type: switch
    poe:
      budget_watts: 740
      usage_threshold_percent: 80
```

| Object | Serves |
| ------- | ----------- |
| `pethMainPsePower` | `budget_watts` |
| `pethMainPseConsumptionPower` | What the attached powered devices draw |
| `pethMainPseUsageThreshold` | `usage_threshold_percent` (default 80) |
| `pethPsePortDetectionStatus` | Per port: `deliveringPower`, `searching` or `fault` |
| `pethPsePortPowerClassifications` | The 802.3af class implied by the draw |
| `pethPsePortPowerPriority` | The powered device's advertised priority |

Per-port draw is never authored on the switch. It comes from the attached
device's own LLDP-MED `power` TLV with `device_type: pd`, so one number describes
a phone both in the frame a discovery tool decodes and in the table it walks.
See the PoE section of `AUTHORING_GUIDE.md`.

The `poe_loss` fault reports the port `fault(4)` and removes its draw from the
chassis consumption, which is what distinguishes it from `link_down`: both take
the link down, only one of them explains why.

A device backed by a capture walk that already carries POWER-ETHERNET-MIB keeps
the captured table untouched — a real PSE is the authority on its own group
count, port numbering and consumption.

So `poe_loss` cannot be armed on such a device. A fault would have to override
one port's captured row, and that needs the capture's `group.port` index
mapped to an interface; a PSE port index is not a bridge port index, and no
capture in the corpus states the relationship. PoE faults arm only on PSE
ports NIAC synthesizes, and any other port is refused with
`fault_no_pse_port`, naming why.

#### Printers (Printer-MIB)

A device with `type: printer` answers Printer-MIB (RFC 3805,
`.1.3.6.1.2.1.43`) and lists itself in HOST-RESOURCES-MIB's device table as a
printer. Discovery tools send every device a GETNEXT at `.1.3.6.1.2.1.43` and
file the ones that answer inside it as printers, so this is what makes a
printer read as a printer rather than a host. Nothing is authored for it.

| Object | Serves |
| ------- | ----------- |
| `hrDeviceType` | `hrDevicePrinter` |
| `hrDeviceDescr` | The device's `sysDescr` |
| `hrDeviceStatus` | `running` |
| `hrPrinterStatus` | `idle` |
| `prtGeneralPrinterName` | The device's `sysName` |
| `prtGeneralReset` | `notResetting` |

A walk-backed printer keeps a captured Printer-MIB untouched. One whose capture
carries none gets the rows above at the first `hrDeviceIndex` after the
capture's own devices.

#### UPS (UPS-MIB)

A device whose `sysObjectID` is an APC Network Management Card's
(`1.3.6.1.4.1.318.1.3.27`, set with `properties.sysObjectID`) answers UPS-MIB
(RFC 1628, `.1.3.6.1.2.1.33`). A manager that sees that sysObjectID walks
UPS-MIB next, and a device that answers there is filed as power equipment. It
reports a UPS on mains with a full battery:

| Object | Serves |
| ------- | ----------- |
| `upsIdentManufacturer` | `American Power Conversion Corp.` |
| `upsIdentModel` | `properties.model`, or the device's `sysDescr` |
| `upsIdentAgentSoftwareVersion` | `properties.software` |
| `upsIdentName` | The device's `sysName` |
| `upsBatteryStatus` | `batteryNormal` |
| `upsSecondsOnBattery` | `0` |
| `upsEstimatedMinutesRemaining` | `42` |
| `upsEstimatedChargeRemaining` | `100` |
| `upsOutputSource` | `normal` |
| `upsAlarmsPresent` | `0` |

The input, output and bypass line tables are not served: they need voltages and
loads nothing authors. A walk-backed UPS keeps a captured UPS-MIB untouched; one
whose capture carries only its system group gets the objects above.

#### Best Practices

- Use SNMPv3 in production (authentication and encryption)
- Change default community strings ("public", "private")
- Limit SNMP access with ACLs
- Use separate communities for read-only and traps
- Populate all system MIB fields (sysname, sysdescr, etc.)
- Configure multiple trap receivers for redundancy
- Enable only the cold-start and link-state notifications the lab needs
- Use walk files to simulate real device OIDs

### SSH Device Management

NIAC can terminate SSH on a simulated device address and expose an IOS-like
command session backed by that device's authoritative state. Configure a
username and the name of the environment variable that supplies the password:

```yaml
devices:
  - name: edge-router
    ssh:
      enabled: true
      username: "admin"
      password_env: "NIAC_EDGE_ROUTER_PASSWORD"
```

Set `NIAC_EDGE_ROUTER_PASSWORD` in the environment that launches NIAC, or in
the NIAC service environment, then start or restart NIAC. From a client shell,
connect with `ssh -T admin@10.10.200.1`. The first command profile uses the
client terminal's line handling and does not allocate a remote PTY.

The first command set includes interface status and addressing, static routes,
hostname and VLAN state, running/startup configuration, checkpoints, reload,
and configuration-event inspection. Each command updates shared device state:
hostname changes feed discovery and SNMP, interface and route changes feed
forwarding and SNMP, and enabled SYSLOG receives every state transition. An
operational interface transition also emits an enabled linkUp or linkDown SNMP
notification. The command profile does not execute a vendor image.

`os_fingerprint.ssh_banner` sets the version string the device announces, for
example `SSH-2.0-Cisco-1.25`. Without it the device announces `SSH-2.0-Go`.

`ssh: {enabled: true}` with no `username` and `password_env` serves SSH with no
account. A client completes key exchange and sees the banner and host key, and
every login is refused, so the device needs no secret in the daemon's
environment. The scenario packs serve SSH this way on their IOS XE routers,
switches and wireless controllers. Add `username` and `password_env` to a pack
device to open its command session.

### SIP

**Session Initiation Protocol** - the device answers SIP `OPTIONS` on UDP 5060
with the User-Agent of the handset it models.

A scanner tells a desk phone from a host that merely carries a phone vendor's
MAC prefix by asking it over SIP. NIAC answers `OPTIONS` and nothing else:
calls, registrations and every other method get no reply, and a deep probe
that places a call fails. SIP over IPv6 and TCP is not served.

#### Configuration

```yaml
devices:
  - name: lobby-phone
    type: voip-phone
    ips:
      - "10.0.40.21"
    sip:
      enabled: true
      user_agent: "Cisco-CP8841/12.8.1"
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Answer SIP `OPTIONS` on UDP 5060 |
| `user_agent` | string | When enabled | - | Sent as `User-Agent` and `Server`; printable ASCII, at most 128 characters |

The response echoes the request's `Via`, `From`, `To`, `Call-ID` and `CSeq`.
It adds `received` to the top `Via`, fills in `rport` when the request asked
for it, and goes back to the source port in that case. The scenario packs
enable SIP on their Cisco CP-8841 phones.

#### Testing

```bash
# Identify SIP devices (SIPVicious)
sipvicious_svmap 10.0.40.21

# Monitor SIP traffic
sudo tcpdump -i eth0 -A udp port 5060
```

### ONVIF Discovery

**WS-Discovery** - the device answers a WS-Discovery `Probe` on UDP 3702 with
a `ProbeMatch` naming the ONVIF camera it models.

A video management system or a scanner finds a camera by probing for ONVIF's
`NetworkVideoTransmitter` type. NIAC answers a `Probe` sent to the camera's
address with the camera's types, scopes and device service address, and
answers nothing else. The device service behind that address is not
simulated, so a client that goes on to read the stream profiles or the device
information fails; on the modelled camera those calls need credentials
anyway. Multicast probes, IPv6 and WS-Discovery 1.1 are not served.

#### Configuration

```yaml
devices:
  - name: lobby-camera
    type: iot
    ips:
      - "10.0.40.31"
    onvif:
      enabled: true
      name: "AXIS P3265-LVE"
      hardware: "P3265-LVE"
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Answer WS-Discovery `Probe` on UDP 3702 |
| `name` | string | When enabled | - | The `onvif://www.onvif.org/name/` scope; printable ASCII, at most 64 characters |
| `hardware` | string | When enabled | - | The `onvif://www.onvif.org/hardware/` scope, the model; same limits |

The `ProbeMatch` advertises `dn:NetworkVideoTransmitter tds:Device`, the
`type/video_encoder` and `Profile/Streaming` scopes plus the two authored
ones, and `http://<address>/onvif/device_service`. Its endpoint reference is
a UUID derived from the device name and MAC, so it is the same on every
probe. A `Probe` asking for another type, or for a scope the camera lacks
(RFC 3986 or `strcmp0` matching), gets no reply. The scenario packs enable
ONVIF discovery on their Axis P3265-LVE cameras.

#### Testing

```bash
# Monitor WS-Discovery traffic
sudo tcpdump -i eth0 -A udp port 3702
```

### JetDirect

**Raw printing on TCP 9100** - the device accepts print jobs on the raw port
and, with a PJL ID, answers the Printer Job Language status queries a scanner
identifies a printer by.

Every network printer takes raw jobs on 9100; HP printers and other PJL
printers also answer `@PJL INFO` queries there. NIAC accepts the connection,
acknowledges and discards print data, and answers `@PJL INFO ID` and
`@PJL INFO STATUS`. Other PJL commands, other `INFO` categories and job
rendering are not simulated. JetDirect over IPv6 is not served. A device
without JetDirect resets a connection to 9100.

#### Configuration

```yaml
devices:
  - name: office-printer
    type: printer
    ips:
      - "10.0.40.30"
    jetdirect:
      enabled: true
      pjl_id: "HP LaserJet MFP M634"
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Accept connections and print data on TCP 9100 |
| `pjl_id` | string | No | - | Answer `@PJL INFO ID` with this model and `@PJL INFO STATUS` with ready; printable ASCII without `"`, at most 64 characters. Leave empty for a printer that speaks no PJL |

The scenario packs open 9100 on every printer, which they already advertise as
`_pdl-datastream._tcp`. Only the HP multifunction printers carry a PJL ID; the
Zebra label and Epson receipt printers take raw jobs and answer no PJL query.

#### Testing

```bash
# Identify the printer (port 9100 is excluded from -sV unless --allports)
nmap -sV --allports --version-all -p 9100 --script pjl-ready-message 10.0.40.30

# Ask for the model by hand
printf '@PJL INFO ID\r\n' | nc 10.0.40.30 9100
```

### DICOM

**DICOM verification on TCP 104 and 11112** - the device accepts an
association for the Verification SOP class and answers `C-ECHO`, the check a
PACS archive and the imaging modalities that send to it run against each
other.

NIAC answers `A-ASSOCIATE-RQ` with `A-ASSOCIATE-AC`, echoing both AE titles
and accepting any called AE title. A Verification presentation context is
accepted in Implicit or Explicit VR Little Endian; every other abstract
syntax is refused, so storage, worklist and query fail at association. A
`C-ECHO-RQ` gets a `C-ECHO-RSP` with status Success, and `A-RELEASE-RQ` gets
`A-RELEASE-RP`. PDUs are reassembled across TCP segments, as SCUs such as
dcmtk write them in pieces; a PDU over 64 KiB, or bytes that cannot start a
PDU (a port scanner's HTTP or TLS probe), are answered with `A-ABORT`. DICOM
over IPv6 is not served. A device without DICOM resets a connection to either
port.

#### Configuration

```yaml
devices:
  - name: MED-PACS01
    type: server
    ips:
      - "10.0.50.20"
    dicom:
      enabled: true
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Accept associations and answer `C-ECHO` on TCP 104 and 11112 |

The hospital pack enables DICOM on its MRI systems and on its PACS archive.

#### Testing

```bash
# Verify the archive the way a modality does (dcmtk)
echoscu -v -aec PACS 10.0.50.20 11112

# Or with nmap
nmap -p 104,11112 --script dicom-ping 10.0.50.20
```

### HL7

**HL7 v2 over MLLP on TCP 2575** - the device acknowledges every message
framed in MLLP, the check an interface engine runs against the clinical
systems it feeds.

Each `<VT> message <FS><CR>` block is answered with an original-mode `ACK` in
the same framing. The `ACK` turns the inbound header round (its sending
application and facility become the receiving ones), keeps the sender's
separators, processing ID and version, names the trigger event as
`ACK^<event>^ACK`, and carries its own control ID. `MSA-1` is `AA` and `MSA-2`
the inbound `MSH-10`; a block that does not start with an `MSH` segment, or
whose header has no control ID, gets `AR`. Messages are reassembled across TCP
segments, bytes outside a block are ignored, and a block over 64 KiB is
dropped unanswered. The message is not interpreted: there is no patient,
order or result logic, no enhanced-mode acknowledgement and no IPv6. A device
without HL7 resets a connection to 2575.

#### Configuration

```yaml
devices:
  - name: MED-EMR01
    type: server
    ips:
      - "10.0.50.16"
    hl7:
      enabled: true
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Acknowledge MLLP-framed messages on TCP 2575 |

The hospital pack enables HL7 on its EMR, laboratory and pharmacy systems.

#### Testing

```bash
# Send a message the way an interface engine does (python-hl7)
printf 'MSH|^~\\&|EPIC|MAIN|LAB|MAIN|20261015||ADT^A01|MSG1|P|2.5\r' > adt.hl7
mllp_send --port 2575 --file adt.hl7 10.0.50.16
```

### Modbus/TCP

**Modbus/TCP on TCP 502** - the device answers register reads, the polling a
SCADA master, a historian or an HMI runs against a controller.

Read Holding Registers (function 3) and Read Input Registers (function 4) are
served over addresses 0 up to `holding_registers` and `input_registers`, at
most 125 registers a request, and every register reads zero: the process
values are the control program and are not simulated. Anything else is
answered with the exception a server returns: 01 (illegal function) for every
other function code, including writes and device identification; 03 (illegal
data value) for a quantity of 0 or over 125, or a malformed read; and 02
(illegal data address) for a read past the last register. The answer echoes
the transaction and unit IDs, so any unit ID is served. Requests are
reassembled across TCP segments, a request whose protocol ID is not 0 is
discarded, and a stream whose length field no request can carry is not
answered further. There is no IPv6. A device without Modbus resets a
connection to 502.

#### Configuration

```yaml
devices:
  - name: PLT-PLC01
    type: iot
    ips:
      - "10.0.60.20"
    modbus:
      enabled: true
      holding_registers: 100
      input_registers: 100
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Answer Modbus/TCP on TCP 502 |
| `holding_registers` | integer | Yes | - | Holding registers at addresses 0 up to this count, 0-65536 |
| `input_registers` | integer | Yes | - | Input registers at addresses 0 up to this count, 0-65536 |

The manufacturing pack enables Modbus on its PLCs, with 100 registers of each
type.

#### Testing

```bash
# Read two holding registers from unit 1, then one past the last
mbpoll -m tcp -a 1 -t 4 -r 1 -c 2 -1 10.0.60.20
mbpoll -m tcp -a 1 -t 4 -r 101 -c 1 -1 10.0.60.20   # exception 02
```

### EtherNet/IP

**EtherNet/IP List Identity on TCP and UDP 44818** - the device answers with
its CIP Identity object, the query a commissioning tool or an asset inventory
names a controller by.

The List Identity reply carries one CIP Identity item: encapsulation version
1, the device's own address and port 44818, then the authored vendor ID,
device type, product code, revision, serial number and product name, with
status 0 and state 3 (operational). The sender context and session handle are
echoed. A NOP is never answered. Over TCP every other command, including
Register Session, is answered with status 0x0001 (invalid or unsupported
command); over UDP it is dropped. Requests are reassembled across TCP
segments, and a stream whose length field no message can carry is not
answered further. Sessions, explicit messaging and the implicit I/O a
connection opens on UDP 2222 are the control system and are not simulated.
List Identity sent to a broadcast address and IPv6 are not served. A device
without EtherNet/IP resets a connection to TCP 44818 and answers UDP 44818
with port unreachable.

#### Configuration

```yaml
devices:
  - name: PLT-PLC01
    type: iot
    ips:
      - "10.0.60.20"
    ethernet_ip:
      enabled: true
      vendor_id: 1
      device_type: 14
      product_code: 166
      revision_major: 33
      revision_minor: 11
      serial_number: 3237998081
      product_name: "1756-L83E/B"
```

#### Fields

| Field | Type | Required | Default | Description |
| ------- | ------ | ---------- | --------- | ------------- |
| `enabled` | boolean | Yes | false | Answer List Identity on TCP and UDP 44818 |
| `vendor_id` | integer | Yes | - | ODVA vendor ID, 0-65535 (1 is Rockwell Automation) |
| `device_type` | integer | Yes | - | CIP device type, 0-65535 (14 is a programmable logic controller) |
| `product_code` | integer | Yes | - | Vendor product code, 0-65535 |
| `revision_major` | integer | Yes | - | Firmware major revision, 0-255 |
| `revision_minor` | integer | Yes | - | Firmware minor revision, 0-255 |
| `serial_number` | integer | Yes | - | 32-bit serial number |
| `product_name` | string | Yes | - | 1 to 32 printable ASCII characters |

The manufacturing pack enables EtherNet/IP on its PLCs as ControlLogix 1756-L83E/B
controllers on firmware 33.011, each with a serial number derived from its
name.

#### Testing

```bash
# List Identity over UDP, then over TCP
nmap -sU -p 44818 --script enip-info 10.0.60.20
nmap -sT -p 44818 --script enip-info 10.0.60.20
```

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
4. **Enable only required cold-start and link-state traps** to avoid alert fatigue

### Monitoring and Troubleshooting

1. **Enable debug flags** for specific protocols:

   ```bash
   sudo niac daemon --once -d 3 en0 config.yaml
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
