# Environment Simulation Guide

This guide demonstrates how to simulate complete network environments using NiAC-Go, from small branch offices to large data centers.

## Table of Contents

- [Overview](#overview)
- [Data Center (Spine-Leaf)](#data-center-spine-leaf)
- [Enterprise Campus (3-Tier)](#enterprise-campus-3-tier)
- [Branch Office](#branch-office)
- [Wireless Deployment](#wireless-deployment)
- [Multi-Vendor Environment](#multi-vendor-environment)
- [Service Provider Edge](#service-provider-edge)

## Overview

NiAC-Go can simulate complete network environments with:
- Multiple devices (routers, switches, APs, servers)
- Protocol combinations (LLDP, CDP, DHCP, DNS, SNMP, etc.)
- Topology relationships (port-channels, trunks, VLANs)
- Multi-vendor integration

Each example includes:
- Topology diagram
- Complete YAML configuration
- Usage instructions
- Testing procedures

## Data Center (Spine-Leaf)

### Architecture

Modern data center design with spine-leaf topology for east-west traffic optimization.

```
          [Spine-01]  [Spine-02]
           /    |    X    |    \
          /     |   / \   |     \
    [Leaf-01] [Leaf-02] [Leaf-03]
        |         |         |
    [Servers] [Servers] [Servers]
```

### Features
- 2 Spine switches (Cisco Nexus 9300)
- 3 Leaf switches (Cisco Nexus 9300)
- Port-channels between spine and leaf
- VXLAN overlay ready
- Multi-path ECMP routing

### Configuration

```yaml
devices:
  # Spine Switch 1
  - name: spine-01
    type: switch
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.11"]

    # Port-channels to each leaf
    port_channels:
      - id: 1
        members: ["Ethernet1/1", "Ethernet1/2"]
        mode: "active"
      - id: 2
        members: ["Ethernet1/3", "Ethernet1/4"]
        mode: "active"
      - id: 3
        members: ["Ethernet1/5", "Ethernet1/6"]
        mode: "active"

    # Trunks over port-channels
    trunk_ports:
      - interface: "port-channel1"
        vlans: [1, 10, 20, 30]
        remote_device: "leaf-01"
        remote_interface: "port-channel1"
      - interface: "port-channel2"
        vlans: [1, 10, 20, 30]
        remote_device: "leaf-02"
        remote_interface: "port-channel1"
      - interface: "port-channel3"
        vlans: [1, 10, 20, 30]
        remote_device: "leaf-03"
        remote_interface: "port-channel1"

    lldp:
      enabled: true
      system_name: "spine-01"

    snmp_agent:
      enabled: true
      sysname: "spine-01"

  # Leaf switches (similar configuration)
  - name: leaf-01
    type: switch
    mac: "00:11:22:33:44:11"
    ips: ["10.0.0.21"]

    port_channels:
      - id: 1
        members: ["Ethernet1/49", "Ethernet1/50"]
        mode: "active"

    trunk_ports:
      - interface: "port-channel1"
        vlans: [1, 10, 20, 30]
        remote_device: "spine-01"
        remote_interface: "port-channel1"
```

### Usage
```bash
niac validate examples/environments/datacenter-spine-leaf.yaml
sudo niac en0 examples/environments/datacenter-spine-leaf.yaml
```

## Enterprise Campus (3-Tier)

### Architecture

Traditional hierarchical design: core, distribution, access.

```
      [Core-01]  [Core-02]
         |    X    |
      [Dist-01]  [Dist-02]
       /    |    X    |    \
  [Acc-01] [Acc-02] [Acc-03]
     |        |        |
  [Users]  [Users]  [Users]
```

### Features
- 2 Core switches (Catalyst 9500)
- 2 Distribution switches (Catalyst 9400)
- 3 Access switches (Catalyst 9300)
- STP root bridge on core
- HSRP for gateway redundancy

### Configuration

```yaml
devices:
  # Core Switch 1 (STP Root)
  - name: core-01
    type: switch
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1"]

    stp:
      enabled: true
      bridge_priority: 4096  # Root bridge
      hello_time: 2
      max_age: 20
      forward_delay: 15

    port_channels:
      - id: 1
        members: ["TenGigabitEthernet1/1/1", "TenGigabitEthernet1/1/2"]
        mode: "active"
      - id: 2
        members: ["TenGigabitEthernet1/1/3", "TenGigabitEthernet1/1/4"]
        mode: "active"

    trunk_ports:
      - interface: "port-channel1"
        vlans: [1, 10, 20, 30, 40, 50]
        remote_device: "dist-01"
      - interface: "port-channel2"
        vlans: [1, 10, 20, 30, 40, 50]
        remote_device: "dist-02"

  # Distribution switches
  - name: dist-01
    type: switch
    mac: "00:11:22:33:44:11"
    ips: ["10.0.0.11"]

    stp:
      enabled: true
      bridge_priority: 8192  # Secondary

    # Uplinks to core, downlinks to access
```

## Branch Office

### Architecture

Small office with router, switch, wireless.

```
  [Internet]
      |
  [Router]
      |
  [Switch] --- [AP]
   |  |  |
  PC  PC Phone
```

### Features
- Edge router with DHCP/DNS
- Access switch with PoE
- Wireless access point
- VoIP phone support

### Configuration

```yaml
devices:
  # Edge Router
  - name: branch-rtr-01
    type: router
    mac: "00:11:22:33:44:01"
    ips: ["192.168.1.1", "203.0.113.1"]

    # DHCP Server
    dhcp:
      enabled: true
      pools:
        - network: "192.168.1.0/24"
          range_start: "192.168.1.100"
          range_end: "192.168.1.200"
          gateway: "192.168.1.1"
          dns_servers: ["192.168.1.1", "8.8.8.8"]

    # DNS Server
    dns:
      enabled: true
      forward_records:
        - name: "router.branch.local"
          ip: "192.168.1.1"

    lldp:
      enabled: true

  # Access Switch
  - name: branch-sw-01
    type: switch
    mac: "00:11:22:33:44:02"
    ips: ["192.168.1.2"]

    trunk_ports:
      - interface: "GigabitEthernet0/24"
        vlans: [1, 10, 20]  # Data, Voice, Wireless
        remote_device: "branch-rtr-01"

    lldp:
      enabled: true
```

## Wireless Deployment

### Architecture

Wireless LAN controller managing multiple APs.

```
  [WLC] --- [Switch]
    |          |
  [AP-01]   [AP-02]
```

### Features
- Wireless LAN Controller
- HTTP management API
- DHCP for APs
- DNS for AP discovery

See: `examples/combinations/wireless-controller.yaml`

## Multi-Vendor Environment

### Architecture

Mixed Cisco, Juniper, and Aruba equipment.

```
  [Cisco-Core] --- [Juniper-Edge]
       |                 |
  [Aruba-Switch]   [Aruba-AP]
```

### Features
- LLDP (multi-vendor standard)
- SNMP monitoring all vendors
- Vendor-specific MIBs

### Configuration

```yaml
devices:
  # Cisco Core
  - name: cisco-core-01
    type: switch
    lldp:
      enabled: true
    cdp:
      enabled: true  # Cisco-specific
    snmp_agent:
      enabled: true

  # Juniper Edge
  - name: juniper-edge-01
    type: router
    lldp:
      enabled: true  # Standard
    snmp_agent:
      enabled: true

  # Aruba Access
  - name: aruba-access-01
    type: switch
    lldp:
      enabled: true
    snmp_agent:
      enabled: true
```

## Service Provider Edge

### Architecture

Provider edge with customer segregation.

```
       [PE-01]  [PE-02]
         |    X    |
      [CE-A]    [CE-B]
    (Customer) (Customer)
```

### Features
- VRF/VLAN per customer
- DHCP relay to customer servers
- Isolated management

## Testing Environments

### Verification Steps

For any environment:

1. **Validate Configuration**
```bash
niac validate environment.yaml
```

2. **Deploy**
```bash
sudo niac en0 environment.yaml
```

3. **Verify Topology**
```bash
# LLDP neighbors
lldpcli show neighbors

# CDP neighbors (Cisco)
show cdp neighbors

# Port-channel status
show etherchannel summary
```

4. **Test Protocols**
```bash
# SNMP
snmpwalk -v2c -c public 10.0.0.1 system

# DHCP
sudo dhclient en0

# DNS
dig @10.0.0.1 router.local
```

5. **Monitor**
```bash
# SNMP traps
snmptrapd -f -Lo

# Syslog
tail -f /var/log/syslog
```

## Best Practices

### Design
1. Start with a clear topology diagram
2. Plan IP addressing scheme
3. Define VLAN strategy
4. Document all connections

### Configuration
1. Use consistent naming conventions
2. Group related devices
3. Define remote_device references
4. Enable appropriate discovery protocols

### Testing
1. Validate before deployment
2. Test in stages (core → dist → access)
3. Verify neighbor relationships
4. Monitor for errors/warnings

### Production
1. Document all configurations
2. Implement change control
3. Monitor via SNMP
4. Plan for redundancy

## See Also

- [Topology Guide](TOPOLOGY_GUIDE.md) - Port-channels and trunks
- [Protocol Guide](PROTOCOL_GUIDE.md) - LLDP, CDP, DHCP, DNS, etc.
- [Examples](../examples/) - Ready-to-use configurations
