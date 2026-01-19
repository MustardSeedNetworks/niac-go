# SNMP Walk File Workflow

## Overview

SNMP walk files contain OID-value pairs that NIAC-Go uses to simulate real device SNMP responses. This guide shows how to create, optimize, and contribute walk files.

## Creating Walk Files

### From Real Device

```bash
# Full MIB walk
snmpwalk -v2c -c public device.example.com .1 > device-full.walk

# Specific subtree
snmpwalk -v2c -c public device.example.com .1.3.6.1.2.1.1 > device-system.walk
```

### From SNMPv3

```bash
snmpwalk -v3 -l authPriv -u username -a SHA -A authpass -x AES -X privpass \
  device.example.com .1 > device.walk
```

## Walk File Format

```
iso.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS Software, Version 15.2"
iso.3.6.1.2.1.1.2.0 = OID: enterprises.9.1.1208
iso.3.6.1.2.1.1.3.0 = Timeticks: (12345678) 1 day, 10:17:36.78
iso.3.6.1.2.1.1.4.0 = STRING: "admin@example.com"
```

## Optimizing Walk Files

### Remove Unnecessary OIDs

```bash
# Keep only system and interface MIBs
grep -E '(\.1\.3\.6\.1\.2\.1\.1\.|\.1\.3\.6\.1\.2\.1\.2\.)' full.walk > optimized.walk
```

### Common Essential OIDs

```bash
# System group (1.3.6.1.2.1.1.*)
grep '\.1\.3\.6\.1\.2\.1\.1\.' full.walk > system.walk

# Interface group (1.3.6.1.2.1.2.*)
grep '\.1\.3\.6\.1\.2\.1\.2\.' full.walk > interfaces.walk

# IP group (1.3.6.1.2.1.4.*)
grep '\.1\.3\.6\.1\.2\.1\.4\.' full.walk > ip.walk
```

### Combine Multiple Walks

```bash
cat system.walk interfaces.walk ip.walk > combined.walk
sort -u combined.walk > device.walk
```

## Using in NIAC-Go

```yaml
devices:
  - name: router1
    type: router
    ip_addresses:
      - 192.168.1.1
    snmp_config:
      community: public
      walk_file: /path/to/device.walk
```

## Contributing Walk Files

### Repository Structure

```
examples/walks/
├── cisco/
│   ├── ios-router-15.2.walk
│   └── catalyst-switch.walk
├── juniper/
│   └── junos-router.walk
└── README.md
```

### Sanitization

Before contributing, sanitize sensitive data:

```bash
# Remove community strings
sed -i 's/private/public/g' device.walk

# Remove contact info
sed -i 's/admin@company.com/admin@example.com/g' device.walk

# Remove location strings
sed -i 's/Building 5, Floor 2/Data Center/g' device.walk

# Remove serial numbers
sed -i 's/S/N: ABC123/S/N: XXXXX/g' device.walk
```

### Contribution Checklist

- [ ] Walk file is from legitimate device testing
- [ ] Sensitive information removed (emails, locations, serials)
- [ ] Community strings sanitized
- [ ] File size optimized (< 5MB preferred)
- [ ] Device model/version documented in filename
- [ ] README.md updated with device info

### Pull Request Template

```markdown
## SNMP Walk Contribution

**Device Information:**
- Manufacturer: Cisco
- Model: Catalyst 3750
- Software Version: 15.2(4)E
- MIBs Included: BRIDGE-MIB, IF-MIB, IP-MIB

**Collection Method:**
- SNMPv2c walk of .1 subtree
- Sanitized location/contact information
- File size: 2.3MB

**Testing:**
- [ ] Tested with NIAC-Go v2.6.0
- [ ] No sensitive data included
- [ ] Essential OIDs present
```

## Device-Specific Examples

### Cisco IOS Router

```bash
snmpwalk -v2c -c public router .1 | grep -E '(sysDescr|ifDescr|ipAdEntAddr|cisco)' > cisco-ios.walk
```

### Juniper JUNOS

```bash
snmpwalk -v2c -c public juniper .1 | grep -E '(sysDescr|ifDescr|jnx)' > juniper.walk
```

### Linux Server (net-snmp)

```bash
snmpwalk -v2c -c public server .1 | grep -E '(sysDescr|hrSystem|ucd)' > linux-server.walk
```

## Troubleshooting

### Walk File Too Large

```bash
# Get file size
ls -lh device.walk

# Count OIDs
wc -l device.walk

# Reduce to essential MIBs only
grep -E '(\.1\.3\.6\.1\.2\.1\.(1|2|4|31)\.)' device.walk > essential.walk
```

### Invalid Format

```bash
# Validate format
awk -F' = ' 'NF!=2{print "Line " NR ": Invalid format"}' device.walk

# Fix common issues
sed -i 's/iso\./\.1\./g' device.walk  # Normalize OID format
```

### Missing Critical OIDs

Essential OIDs for basic operation:
```
.1.3.6.1.2.1.1.1.0  # sysDescr
.1.3.6.1.2.1.1.2.0  # sysObjectID
.1.3.6.1.2.1.1.3.0  # sysUpTime
.1.3.6.1.2.1.2.1.0  # ifNumber
.1.3.6.1.2.1.2.2.1.2.X  # ifDescr for each interface
```

## Best Practices

1. **Document Source**: Include device model/version in filename
2. **Sanitize Data**: Remove all sensitive information
3. **Optimize Size**: Include only essential OIDs
4. **Test Before Contributing**: Verify walk works with NIAC-Go
5. **Update Regularly**: Keep walks current with software versions
6. **Use Meaningful Names**: `cisco-ios-15.2-router.walk` not `device1.walk`
