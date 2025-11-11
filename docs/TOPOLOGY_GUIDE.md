# Topology Configuration Guide

This guide covers network topology configuration in NiAC-Go, including port-channels (Link Aggregation Groups), trunk ports with VLAN tagging, and multi-device topologies.

**Added in:** v1.23.0

## Table of Contents

- [Port-Channels (Link Aggregation)](#port-channels-link-aggregation)
- [Trunk Ports](#trunk-ports)
- [Multi-Device Topologies](#multi-device-topologies)
- [VLAN Configuration](#vlan-configuration)
- [Neighbor Discovery](#neighbor-discovery)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Port-Channels (Link Aggregation)

Port-channels (also known as LAGs or EtherChannels) bundle multiple physical interfaces into a single logical interface for increased bandwidth and redundancy.

### Basic Configuration

```yaml
devices:
  - name: core-switch-01
    type: switch
    mac: "00:11:22:33:44:01"
    ips:
      - "10.0.0.11"

    port_channels:
      - id: 1
        members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
        mode: "active"
```

### Port-Channel Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | integer | Yes | Port-channel ID (must be unique per device) |
| `members` | string array | Yes | Physical interface names to aggregate |
| `mode` | string | No | LACP mode: `active`, `passive`, or `on` |

### LACP Modes

- **`active`**: Actively sends LACP packets. Best for both ends to use active mode for fastest convergence.
- **`passive`**: Responds to LACP packets. Requires the other end to be active.
- **`on`**: Static LAG without LACP negotiation. Both ends must use `on` mode.

**Recommendation:** Use `active` mode on both ends for production deployments.

### Member Interface Rules

1. All member interfaces must have the same speed and duplex
2. An interface can only be a member of one port-channel
3. At least one member interface is required
4. Member interfaces should be adjacent ports for cable management

### Validation

The validator checks:
- Port-channel ID uniqueness within a device
- No duplicate member interfaces
- Member interfaces aren't in multiple port-channels
- Valid LACP mode values

## Trunk Ports

Trunk ports carry traffic for multiple VLANs using 802.1Q tagging. They're essential for inter-switch links and router-on-a-stick configurations.

### Basic Configuration

```yaml
devices:
  - name: core-switch-01
    type: switch

    trunk_ports:
      - interface: "GigabitEthernet0/10"
        vlans: [1, 10, 20, 30]
        native_vlan: 1
        remote_device: "core-switch-02"
        remote_interface: "GigabitEthernet0/10"
```

### Trunk Port Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `interface` | string | Yes | Local interface name (physical or port-channel) |
| `vlans` | integer array | Yes | List of allowed VLANs (1-4094) |
| `native_vlan` | integer | No | Untagged VLAN (default: 1) |
| `remote_device` | string | No | Remote device name for topology validation |
| `remote_interface` | string | No | Remote interface name for neighbor discovery |

### VLAN Ranges

- Valid VLAN IDs: 1-4094
- VLAN 1: Default native VLAN (management)
- VLANs 1002-1005: Reserved for Token Ring/FDDI (avoid)
- VLAN 4095: Reserved (invalid)

### Native VLAN

The native VLAN carries untagged traffic. Best practices:
- Use VLAN 1 for management (most common)
- Or use a dedicated management VLAN
- **Security:** Change native VLAN from 1 to prevent VLAN hopping attacks
- Ensure native VLAN matches on both ends of trunk

### Trunk Over Port-Channel

Combine port-channels with trunks for high-bandwidth inter-switch links:

```yaml
port_channels:
  - id: 1
    members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
    mode: "active"

trunk_ports:
  - interface: "port-channel1"  # Trunk on the port-channel
    vlans: [1, 10, 20, 30, 40]
    native_vlan: 1
```

## Multi-Device Topologies

Use remote device references to create multi-device topologies with proper neighbor discovery.

### Two-Switch Topology

```yaml
devices:
  - name: switch-01
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        vlans: [1, 10, 20]
        remote_device: "switch-02"
        remote_interface: "GigabitEthernet0/1"

  - name: switch-02
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        vlans: [1, 10, 20]
        remote_device: "switch-01"
        remote_interface: "GigabitEthernet0/1"
```

### Remote Device Validation

The validator checks:
- Referenced device exists in the configuration
- Both ends of the link reference each other (optional but recommended)
- VLAN lists match (warning if different)

## VLAN Configuration

While port-channels and trunk ports define the physical topology, VLANs segment your network logically.

### VLAN Design Patterns

1. **Flat Network** (Small deployments)
   - VLAN 1: Management and all user traffic
   - Simple but limited scalability

2. **Segmented Network** (Recommended)
   - VLAN 1: Management
   - VLAN 10: Servers
   - VLAN 20: Workstations
   - VLAN 30: Wireless
   - VLAN 40: VoIP
   - VLAN 99: Native VLAN (security)

3. **Multi-Tenant** (Service providers)
   - VLANs 1-99: Infrastructure
   - VLANs 100-999: Tenant A
   - VLANs 1000-1999: Tenant B
   - etc.

## Neighbor Discovery

Topology configuration integrates with LLDP and CDP for automatic neighbor discovery.

### LLDP Configuration

```yaml
devices:
  - name: switch-01
    lldp:
      enabled: true
      system_name: "switch-01"
      system_description: "Cisco Catalyst 3850"
      chassis_id: "00:11:22:33:44:01"
      advertise_interval: 30

    trunk_ports:
      - interface: "GigabitEthernet0/1"
        remote_device: "switch-02"  # LLDP will advertise this relationship
```

### CDP Configuration

```yaml
devices:
  - name: switch-01
    cdp:
      enabled: true
      platform: "WS-C3850-48P"
      capabilities: "Switch IGMP"
      software_version: "IOS-XE 16.12.4"

    trunk_ports:
      - interface: "GigabitEthernet0/1"
        remote_device: "switch-02"
```

## Best Practices

### Port-Channels

1. **Use LACP active mode** on both ends for production
2. **Start with 2 members** (minimum for redundancy)
3. **Keep members on the same module** (hardware consistency)
4. **Monitor member status** with SNMP traps
5. **Document port-channel mappings** for troubleshooting

### Trunk Ports

1. **Explicitly define allowed VLANs** (don't trunk all VLANs)
2. **Change native VLAN** from 1 for security (use 99 or 999)
3. **Match configurations** on both ends of trunk
4. **Limit VLAN count** per trunk (performance/security)
5. **Use port-channels** for critical trunks (redundancy)

### Multi-Device Topologies

1. **Document all connections** with remote_device fields
2. **Use consistent naming** (switch-01, switch-02, not random names)
3. **Plan IP addressing** (management subnets, VLAN SVIs)
4. **Enable LLDP and CDP** for verification
5. **Validate configurations** before deployment

### Validation Workflow

```bash
# 1. Validate configuration
niac validate topology.yaml

# 2. Check for errors
# Fix any validation errors before deployment

# 3. Deploy to test environment first
sudo niac en0 topology.yaml

# 4. Verify neighbor relationships
lldpcli show neighbors
show cdp neighbors

# 5. Monitor SNMP traps
snmptrapd -f -Lo
```

## Examples

### Example 1: Basic Two-Switch Trunk

See: `examples/topology/two-switch-trunk.yaml`

Two core switches with a simple trunk link carrying 4 VLANs.

```yaml
devices:
  - name: niac-core-sw-01
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        vlans: [1, 10, 20, 30]
        native_vlan: 1
        remote_device: "niac-core-sw-02"
```

### Example 2: Port-Channel with Trunk

See: `examples/topology/port-channel-lab.yaml`

Two switches connected via 2-member port-channel with trunk.

```yaml
devices:
  - name: niac-core-sw-01
    port_channels:
      - id: 1
        members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
        mode: "active"
    trunk_ports:
      - interface: "port-channel1"
        vlans: [1, 10, 20, 30, 40]
```

### Example 3: Data Center Spine-Leaf

```yaml
devices:
  # Spine Switch 1
  - name: spine-01
    type: switch
    port_channels:
      - id: 1
        members: ["Ethernet1/1", "Ethernet1/2"]
        mode: "active"
      - id: 2
        members: ["Ethernet1/3", "Ethernet1/4"]
        mode: "active"
    trunk_ports:
      - interface: "port-channel1"
        vlans: [1, 10, 20, 30]
        remote_device: "leaf-01"
        remote_interface: "port-channel1"
      - interface: "port-channel2"
        vlans: [1, 10, 20, 30]
        remote_device: "leaf-02"
        remote_interface: "port-channel1"

  # Leaf Switch 1
  - name: leaf-01
    type: switch
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

## Troubleshooting

### Port-Channel Issues

**Problem:** Port-channel not forming
- Check LACP mode compatibility (active/passive/on)
- Verify member interfaces have same speed/duplex
- Ensure interfaces are enabled (admin status up)

**Problem:** Member interface not joining
- Check for interface already in another port-channel
- Verify interface configuration matches other members
- Check for STP blocking

### Trunk Issues

**Problem:** VLANs not passing traffic
- Verify VLAN in allowed list on both ends
- Check native VLAN matches
- Ensure VLAN exists on device

**Problem:** Validation warnings about remote device
- Check device name spelling
- Ensure both devices defined in same config file
- Verify remote_interface matches actual interface

### Validation Errors

**Error:** "duplicate port-channel ID"
- Each port-channel ID must be unique per device
- Use different IDs for different port-channels

**Error:** "interface already belongs to port-channel X"
- Remove interface from other port-channel first
- Use unique member interfaces

**Error:** "invalid VLAN ID"
- Use VLANs 1-4094 only
- Avoid reserved VLANs (1002-1005, 4095)

## See Also

- [Protocol Combinations Guide](PROTOCOL_GUIDE.md) - Using topology with LLDP/CDP/SNMP
- [Environment Simulation Guide](ENVIRONMENTS.md) - Complete topology examples
- [API Reference](API_REFERENCE.md) - Complete YAML schema
- [Examples](../examples/topology/) - Ready-to-use configurations
