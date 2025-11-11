# Walk Files Documentation

SNMP walk files provide realistic device simulation by containing actual SNMP OID responses from real network equipment.

## Table of Contents

- [Overview](#overview)
- [Supported Vendors](#supported-vendors)
- [Walk File Format](#walk-file-format)
- [Using Walk Files](#using-walk-files)
- [Sanitization](#sanitization)
- [Creating Walk Files](#creating-walk-files)
- [Contributing Walk Files](#contributing-walk-files)
- [Walk File Organization](#walk-file-organization)

## Overview

Walk files contain SNMP MIB data captured from real network devices. They allow NiAC-Go to simulate authentic device responses for SNMP queries, making your simulated environment behave like real hardware.

### What Walk Files Contain

- **System MIB**: sysName, sysDescr, sysContact, sysLocation, sysUpTime
- **Interface MIB**: ifDescr, ifType, ifSpeed, ifPhysAddress, ifOperStatus
- **IP MIB**: ipAdEntAddr, ipAdEntNetMask, ipRouteTable
- **Bridge MIB**: dot1dBaseBridgeAddress, dot1dBaseNumPorts
- **Vendor-specific MIBs**: Cisco MIBs, Juniper MIBs, etc.

### Why Use Walk Files?

1. **Realistic Simulation**: Responses match real device behavior
2. **Vendor Accuracy**: Simulate specific vendor implementations
3. **Model Specificity**: Different models have different MIB support
4. **Testing**: Test monitoring tools against realistic data
5. **Training**: Learn SNMP without physical hardware

## Supported Vendors

NiAC-Go includes 555+ sanitized walk files across 17 vendors:

| Vendor | File Count | Common Models |
|--------|------------|---------------|
| Cisco | 488 | Catalyst 2960, 3560, 3750, 3850, Nexus 5000/7000/9000, ISR 2800/3900, ASR 1000 |
| Extreme | 7 | X-Series (X460, X465), Summit |
| Juniper | 4 | EX Series, MX Series, SRX Series |
| HP | Multiple | ProCurve 2510, 2610, 2920, 5400 |
| Dell | Multiple | PowerConnect 6224, 6248, N-Series |
| Brocade | Multiple | FastIron, ICX Series |
| Huawei | Multiple | S-Series switches |
| Fortinet | 2 | FortiGate, FortiSwitch |
| Meraki | Multiple | MS, MX, MR Series |
| Mikrotik | Multiple | RouterBoard, RouterOS |
| 3Com | Multiple | SuperStack, Baseline |
| NetGear | Multiple | ProSafe, Smart switches |
| Oracle | Multiple | Sun switches |
| VMware | Multiple | NSX, vSphere Distributed Switch |
| ZTE | Multiple | ZXR10 Series |
| VoIP | Multiple | Cisco VoIP gateways, IP phones |
| Misc | Multiple | Various vendors/models |

### Vendor Directory Structure

```
examples/device_walks_sanitized/
├── 3com/              # 3Com switches
├── brocade/           # Brocade/Foundry/Ruckus
├── cisco/             # Cisco (largest collection)
├── dell/              # Dell PowerConnect/N-Series
├── extreme/           # Extreme Networks
├── fortinet/          # FortiGate/FortiSwitch
├── hp/                # HP ProCurve/Aruba
├── huawei/            # Huawei S-Series
├── juniper/           # Juniper EX/MX/SRX
├── meraki/            # Cisco Meraki cloud
├── mikrotik/          # MikroTik RouterOS
├── misc/              # Mixed vendors
├── netgear/           # NetGear ProSafe
├── oracle/            # Oracle/Sun
├── vmware/            # VMware NSX/vDS
├── voip/              # VoIP devices
├── zte/               # ZTE routers/switches
└── mapping.json       # IP sanitization mapping
```

## Walk File Format

### File Naming Convention

**Sanitized Format:** `niac-<vendor>-<model>[-vlan][-ip_octet].walk`

Examples:
- `niac-cisco-c2960-48pst-l.walk` - Cisco Catalyst 2960-48PST-L
- `niac-cisco-c3850-v500.walk` - Cisco Catalyst 3850, VLAN 500
- `niac-juniper-ex4300-48p.walk` - Juniper EX4300-48P
- `niac-extreme-x460-48p.walk` - Extreme X460-48P

**Legacy Format:** `<ip_address>-<identifier>.walk` (raw captures)

### File Content Format

Walk files use standard SNMPwalk output format:

```
.1.3.6.1.2.1.1.1.0 = STRING: Cisco IOS Software, C2960 Software
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1208
.1.3.6.1.2.1.1.3.0 = Timeticks: (123456789) 14 days, 6:56:07.89
.1.3.6.1.2.1.1.4.0 = STRING: Network Administrator
.1.3.6.1.2.1.1.5.0 = STRING: switch-01
.1.3.6.1.2.1.1.6.0 = STRING: Building A, Floor 2, IDF 201
```

### Common OID Prefixes

| OID Prefix | MIB | Description |
|------------|-----|-------------|
| .1.3.6.1.2.1.1 | SNMPv2-MIB::system | System information |
| .1.3.6.1.2.1.2 | IF-MIB::interfaces | Network interfaces |
| .1.3.6.1.2.1.4 | IP-MIB::ip | IP stack information |
| .1.3.6.1.2.1.17 | BRIDGE-MIB | Bridge/switch data |
| .1.3.6.1.2.1.31 | IF-MIB::ifXTable | Extended interface data |
| .1.3.6.1.4.1.9 | CISCO-SMI | Cisco enterprise MIBs |
| .1.3.6.1.4.1.2636 | JUNIPER-SMI | Juniper enterprise MIBs |
| .1.3.6.1.4.1.1916 | EXTREME-BASE-MIB | Extreme enterprise MIBs |

## Using Walk Files

### Basic Configuration

```yaml
devices:
  - name: switch-01
    type: switch
    mac: "00:11:22:33:44:01"
    ips: ["10.0.0.1"]

    snmp_agent:
      enabled: true
      community: "public"
      walk_file: "examples/device_walks_sanitized/cisco/niac-cisco-c3850.walk"

      sysname: "switch-01"
      sysdescr: "Cisco Catalyst 3850"
      syscontact: "netadmin@example.com"
      syslocation: "DC-WEST - Rack A01"
```

### Path Resolution

Walk file paths are resolved relative to the NiAC-Go working directory:

```yaml
# Relative path (from project root)
walk_file: "examples/device_walks_sanitized/cisco/niac-cisco-c3850.walk"

# Absolute path
walk_file: "/full/path/to/walk/files/niac-cisco-c3850.walk"

# Relative to config file location
walk_file: "./walk_files/niac-cisco-c3850.walk"
```

### Choosing the Right Walk File

1. **Match Vendor**: Use walk file from same vendor as device type
2. **Match Model**: Use walk file from same or similar model
3. **Match Role**: Router walk for routers, switch walk for switches
4. **Match Features**: Ensure walk file has required MIBs

**Examples:**

```yaml
# Cisco Core Switch
devices:
  - name: core-sw-01
    walk_file: "examples/device_walks_sanitized/cisco/niac-cisco-c3850.walk"

# Juniper Edge Router
devices:
  - name: edge-rtr-01
    walk_file: "examples/device_walks_sanitized/juniper/niac-juniper-mx240.walk"

# Extreme Access Switch
devices:
  - name: access-sw-01
    walk_file: "examples/device_walks_sanitized/extreme/niac-extreme-x460-48p.walk"
```

### Testing Walk File Usage

```bash
# Validate configuration with walk file
niac validate config.yaml

# Run with SNMP debug
sudo niac --debug-snmp 3 en0 config.yaml

# Query SNMP agent
snmpget -v2c -c public 10.0.0.1 sysName.0
snmpwalk -v2c -c public 10.0.0.1 system
snmpbulkwalk -v2c -c public 10.0.0.1 ifDescr
```

## Sanitization

Walk files from real devices contain sensitive information (IP addresses, hostnames, etc.). NiAC-Go provides sanitization to remove this data.

### What Gets Sanitized

- **IP Addresses**: Replaced with 10.0.0.0/8 network
- **Hostnames**: Replaced with niac-core-<type>-<number>
- **MAC Addresses**: Optionally randomized
- **SNMP Community Strings**: Removed/replaced
- **Serial Numbers**: Optionally obscured
- **Contact Information**: Removed

### Sanitization Process

#### 1. Sanitize Single File

```bash
# Basic sanitization
niac sanitize --input original.walk --output sanitized.walk

# With custom mapping file
niac sanitize \
  --input original.walk \
  --output sanitized.walk \
  --mapping-file mapping.json
```

#### 2. Batch Sanitization

```bash
# Sanitize entire directory
niac sanitize --batch \
  --input-dir examples/device_walks/cisco \
  --output-dir examples/device_walks_sanitized/cisco \
  --mapping-file examples/device_walks_sanitized/mapping.json
```

#### 3. Sanitize All Vendors

```bash
# Process all vendor directories
for dir in examples/device_walks/*/; do
  vendor=$(basename "$dir")
  mkdir -p "examples/device_walks_sanitized/$vendor"

  niac sanitize --batch \
    --input-dir "$dir" \
    --output-dir "examples/device_walks_sanitized/$vendor" \
    --mapping-file examples/device_walks_sanitized/mapping.json
done
```

### Mapping File

The mapping file tracks IP address transformations for consistency:

```json
{
  "version": "1.0",
  "timestamp": "2025-01-08T12:34:56Z",
  "mappings": {
    "192.168.1.1": "10.0.0.1",
    "192.168.1.2": "10.0.0.2",
    "10.10.10.5": "10.0.0.3"
  },
  "statistics": {
    "total_ips_mapped": 1288437,
    "total_files_processed": 555
  }
}
```

**Benefits:**
- Consistent IP mapping across multiple walk files
- Preserves network topology relationships
- Deterministic (same input → same output)

### Sanitization Best Practices

1. **Always sanitize** before sharing walk files publicly
2. **Keep mapping file** for reference and consistency
3. **Verify sanitization** by reviewing output files
4. **Test sanitized files** to ensure functionality
5. **Document source** (vendor, model, version) in filename

## Creating Walk Files

### From Real Devices

#### 1. Capture Full Walk

```bash
# Full SNMP walk (all OIDs)
snmpwalk -v2c -c public 10.0.0.1 .1 > device.walk

# Specific MIB tree
snmpwalk -v2c -c public 10.0.0.1 system > device_system.walk
```

#### 2. Capture Common MIBs

```bash
# System MIB
snmpwalk -v2c -c public 10.0.0.1 SNMPv2-MIB::system

# Interface MIB
snmpwalk -v2c -c public 10.0.0.1 IF-MIB::ifTable
snmpwalk -v2c -c public 10.0.0.1 IF-MIB::ifXTable

# IP MIB
snmpwalk -v2c -c public 10.0.0.1 IP-MIB::ipAddrTable

# Bridge MIB
snmpwalk -v2c -c public 10.0.0.1 BRIDGE-MIB::dot1dBase
```

#### 3. Capture Vendor-Specific MIBs

```bash
# Cisco
snmpwalk -v2c -c public 10.0.0.1 .1.3.6.1.4.1.9 > cisco_enterprise.walk

# Juniper
snmpwalk -v2c -c public 10.0.0.1 .1.3.6.1.4.1.2636 > juniper_enterprise.walk

# Extreme
snmpwalk -v2c -c public 10.0.0.1 .1.3.6.1.4.1.1916 > extreme_enterprise.walk
```

### From Virtual Labs

#### GNS3/EVE-NG Setup

1. **Deploy virtual device** in simulator
2. **Configure SNMP** on device
3. **Bridge network** to host
4. **Capture walk** using snmpwalk

```bash
# Example: Cisco virtual device in GNS3
snmpwalk -v2c -c public 192.168.122.10 .1 > cisco-c3850-virtual.walk
```

#### Vendor Virtual Appliances

- Cisco VIRL/CML
- Juniper vMX/vSRX
- Aruba CX Virtual Switch
- Extreme XOS Virtual
- Fortinet FortiGate VM

### Walk File Quality

**Good Walk Files Include:**

- ✅ Complete system MIB data
- ✅ All interface data (ifTable, ifXTable)
- ✅ Routing information (if router)
- ✅ Bridge data (if switch)
- ✅ Vendor-specific MIBs
- ✅ Representative of model features

**Avoid:**

- ❌ Partial walks (missing important MIBs)
- ❌ Error messages in output
- ❌ Timeout messages
- ❌ Duplicate OIDs
- ❌ Corrupted data

### Validation

```bash
# Verify walk file format
grep -c "^\.1\.3\.6\.1" device.walk  # Should show OID count

# Check for errors
grep -i "error\|timeout\|no such" device.walk

# Test with NiAC-Go
niac validate test-config.yaml
```

## Contributing Walk Files

We welcome walk file contributions for modern network equipment!

### Needed Walk Files

**High Priority:**

- Cisco Catalyst 3650/3850 (modern stackable access)
- Cisco Catalyst 9200/9300/9400/9500 (next-gen)
- Cisco Nexus 9300/9500 (ACI-ready data center)
- Juniper EX4300/EX4600 (access/aggregation)
- Juniper QFX5100/QFX5200 (data center)
- Aruba CX 6200/6300/6400 (campus)
- Aruba CX 8320/8360/9300 (data center)
- Palo Alto PA-Series firewalls
- Fortinet FortiGate (modern models)
- Extreme X-Series (X435, X465, X690)

**See Issue #54** for complete list.

### Contribution Process

1. **Capture walk file** from device (with permission!)
2. **Sanitize** using NiAC-Go sanitize command
3. **Name properly**: `niac-<vendor>-<model>.walk`
4. **Test** with NiAC-Go
5. **Document** vendor, model, software version
6. **Submit PR** with walk file and documentation

### Contribution Guidelines

```markdown
## Walk File Contribution Checklist

- [ ] Captured from real hardware or official virtual appliance
- [ ] Sanitized (no real IP addresses, hostnames, or sensitive data)
- [ ] Named using niac-<vendor>-<model> format
- [ ] Tested with NiAC-Go (validates and responds to queries)
- [ ] Includes complete system and interface MIBs
- [ ] Documented in PR:
  - Vendor and model
  - Software version/release
  - Capture date
  - Device role (router/switch/firewall)
  - Notable features/capabilities
```

### Example PR Description

```markdown
# Add Cisco Catalyst 9300 Walk Files

This PR adds SNMP walk files for Cisco Catalyst 9300 Series switches.

**Devices:**
- Cisco Catalyst 9300-48P (IOS XE 17.6.3)
- Cisco Catalyst 9300-24UX (IOS XE 17.6.3)

**Capture Details:**
- Captured: January 2025
- Source: Physical hardware in lab environment
- Sanitized: Yes, using niac sanitize
- Mapping file: Updated

**Files Added:**
- examples/device_walks_sanitized/cisco/niac-cisco-c9300-48p.walk
- examples/device_walks_sanitized/cisco/niac-cisco-c9300-24ux.walk

**Testing:**
- ✅ Validates with niac validate
- ✅ Responds to snmpwalk queries
- ✅ System MIB complete
- ✅ Interface MIB complete
- ✅ Cisco enterprise MIBs present

Related: #54
```

## Walk File Organization

### Current Statistics

- **Total Files**: 555 sanitized walk files
- **Total Vendors**: 17
- **Total IPs Mapped**: 1,288,437
- **Largest Collection**: Cisco (488 files)

### Directory Sizes

```bash
# Check walk file sizes
du -sh examples/device_walks_sanitized/*

# Typical sizes:
# - Small walk: 10-50 KB (basic switches)
# - Medium walk: 50-200 KB (enterprise switches)
# - Large walk: 200-500 KB (routers with full tables)
# - Very large walk: 500KB-2MB (data center switches)
```

### Loading Performance

| File Size | Load Time | Memory Usage |
|-----------|-----------|--------------|
| < 100 KB | < 100ms | < 10 MB |
| 100-500 KB | 100-500ms | 10-50 MB |
| 500KB-1MB | 500ms-1s | 50-100 MB |
| > 1 MB | > 1s | > 100 MB |

**Optimization Tips:**
- Use device-specific walk files (not full captures)
- Remove unused MIB subtrees
- Compress walk files (gzip) for storage
- Cache parsed walk data in memory

## See Also

- [SNMP Configuration](PROTOCOL_GUIDE.md#snmp) - SNMP protocol configuration
- [Sanitization Tool](../cmd/niac/README.md) - Walk file sanitization
- [GitHub Issue #54](https://github.com/krisarmstrong/niac-go/issues/54) - Modern walk files request
- [Contributing Guide](../CONTRIBUTING.md) - How to contribute
