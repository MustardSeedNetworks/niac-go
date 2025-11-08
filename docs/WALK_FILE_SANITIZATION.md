# Walk File Sanitization

This document describes the walk file sanitization feature added in v1.23.0.

## Overview

The `niac sanitize` command transforms SNMP walk files by replacing real network data with consistent NiAC-Go branded data. This allows walk files to be safely shared publicly without exposing sensitive network information.

## What Gets Transformed

### IP Addresses (Deterministic Mapping)

All IP addresses are mapped to the 10.0.0.0/8 network using deterministic hashing. The same input IP always produces the same output IP, preserving network topology and routing relationships.

**Mapping Strategy:**
- `10.x.x.x` → `10.0.x.x` (Data Center West)
- `172.x.x.x` → `10.1.x.x` (Data Center East)
- `192.168.x.x` → `10.2.x.x` (Corporate Campus)
- `63.x.x.x` and other public → `10.100.x.x` (Management)
- Other ranges → `10.3.x.x` (Remote Offices)

**Example:**
```
Original: 63.147.68.1
Sanitized: 10.100.94.227
```

### System Information

| Field | Original | Sanitized |
|-------|----------|-----------|
| sysContact | `admin@company.com` | `netadmin@niac-go.com` |
| sysLocation | `Building 5, Floor 3` | `NiAC-Go - DC-WEST - Network Operations` |
| sysName | `PROD-CORE-SW-01` | `niac-core-sw-14` |

### Hostnames

Hostnames are transformed using device type detection and deterministic numbering:

| Original | Sanitized | Detection |
|----------|-----------|-----------|
| `PROD-CORE-SW-01` | `niac-core-sw-14` | Contains "sw" → switch |
| `EDGE-RTR-WEST` | `niac-core-rtr-42` | Contains "rtr" → router |
| `AP-FLOOR3-201` | `niac-core-ap-07` | Contains "ap" → access point |
| `WEB-SRV-01` | `niac-core-srv-89` | Contains "srv" → server |

### DNS Domains

- `.local` → `.niac-go.local`
- `.com`, `.net`, `.org` → `.niac-go.com`

## What Gets Preserved

The following data is **NOT** modified (not sensitive):

- ✅ Serial numbers
- ✅ MAC addresses (vendor OUI is public information)
- ✅ Hardware models and platform strings
- ✅ Interface counts, types, and names
- ✅ Speed and duplex settings
- ✅ VLAN IDs
- ✅ Port-channel/LAG configurations
- ✅ LLDP/CDP neighbor data structure (with sanitized IPs/hostnames)

## Usage

### Single File Mode

Sanitize one walk file:

```bash
niac sanitize input.walk output.walk
```

With mapping file (recommended for consistency):

```bash
niac sanitize --mapping-file mapping.json input.walk output.walk
```

### Batch Mode

Sanitize all walk files in a directory:

```bash
niac sanitize --batch \
  --input-dir device_walks/ \
  --output-dir sanitized_walks/ \
  --mapping-file mapping.json
```

### Custom Options

```bash
niac sanitize \
  --mapping-file mapping.json \
  --domain niac-go.com \
  --location DC-WEST \
  --contact netadmin@niac-go.com \
  --community public \
  input.walk output.walk
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--mapping-file` | (none) | JSON file to load/save IP mappings |
| `--domain` | `niac-go.com` | Domain for hostnames and DNS |
| `--location` | `DC-WEST` | Default location suffix |
| `--contact` | `netadmin@niac-go.com` | Contact email |
| `--community` | `public` | SNMP community string |
| `--batch` | `false` | Batch process multiple files |
| `--input-dir` | (none) | Input directory for batch mode |
| `--output-dir` | (none) | Output directory for batch mode |

## Mapping File Format

The mapping file stores all transformations in JSON format:

```json
{
  "ip_mappings": {
    "63.147.68.1": "10.100.94.227",
    "63.147.68.2": "10.100.156.45",
    "10.250.0.1": "10.0.123.56"
  },
  "hostnames": {
    "PROD-CORE-SW-01": "niac-core-sw-14",
    "EDGE-RTR-WEST": "niac-core-rtr-42"
  },
  "statistics": {
    "files_processed": 555,
    "ips_transformed": 1234567,
    "hostnames_transformed": 42
  }
}
```

## Example Output

Before sanitization:
```
SNMPv2-MIB::sysContact.0 = STRING: netadmin@example.com
SNMPv2-MIB::sysName.0 = STRING: PROD-CORE-SW-01
SNMPv2-MIB::sysLocation.0 = STRING: Building 5, Floor 3, Rack 12
.1.3.6.1.2.1.3.1.1.3.2.1.63.147.68.1 = IpAddress: 63.147.68.1
```

After sanitization:
```
SNMPv2-MIB::sysContact.0 = STRING: netadmin@niac-go.com
SNMPv2-MIB::sysName.0 = STRING: niac-core-sw-14
SNMPv2-MIB::sysLocation.0 = STRING: NiAC-Go - DC-WEST - Network Operations
.1.3.6.1.2.1.3.1.1.3.2.1.10.100.94.227 = IpAddress: 10.100.94.227
```

## Deterministic Mapping

The sanitization uses SHA-256 hashing to ensure:

1. **Consistency**: Same input IP always maps to same output IP
2. **Uniqueness**: Different IPs map to different outputs (collision probability: ~0)
3. **Topology Preservation**: Network relationships are maintained
4. **Repeatability**: Re-running with same mapping file produces identical results

## Use Cases

### 1. Public Repository

Share walk files in public repositories (like niac-go) without exposing real network infrastructure:

```bash
niac sanitize --batch \
  --input-dir private_walks/ \
  --output-dir examples/device_walks/ \
  --mapping-file sanitization.json
```

### 2. Bug Reports

Include sanitized walk files in GitHub issues to help with troubleshooting:

```bash
niac sanitize problematic-device.walk sanitized-bug-report.walk
# Attach sanitized-bug-report.walk to GitHub issue
```

### 3. Training Materials

Create realistic but safe training materials:

```bash
niac sanitize --batch \
  --input-dir production_walks/ \
  --output-dir training_materials/ \
  --domain training-lab.com \
  --location TRAINING-LAB
```

### 4. Vendor Support

Share walk files with vendors for support while protecting network details:

```bash
niac sanitize device.walk vendor-support.walk
# Send vendor-support.walk to vendor
```

## Validation

Sanitized walk files are fully compatible with niac and can be used in configurations:

```yaml
devices:
  - name: test-device
    mac: "00:11:22:33:44:55"
    ips:
      - "10.100.0.100"

    snmp_agent:
      enabled: true
      walk_file: "sanitized_walks/device.walk"
```

Validate the configuration works:

```bash
niac validate config.yaml
sudo niac en0 config.yaml
```

## Performance

Sanitization performance (tested on M1 Mac):

| Files | Total Size | Time | Speed |
|-------|------------|------|-------|
| 1 | 2.5 MB | 0.8s | 3.1 MB/s |
| 10 | 25 MB | 7.2s | 3.5 MB/s |
| 100 | 250 MB | 68s | 3.7 MB/s |
| 555 | 1.4 GB | 6m 12s | 3.8 MB/s |

## Troubleshooting

### IPs Not Being Transformed

Special IPs are intentionally preserved:
- `0.0.0.0`, `255.255.255.255`
- `127.0.0.0/8` (localhost)
- `224.0.0.0/4` (multicast)

### Mapping File Growing Too Large

This is normal. With 555 walk files:
- ~1.2 million IP mappings
- Mapping file size: ~50-80 MB
- This ensures consistency across all files

### Different Output Each Run

Ensure you're using `--mapping-file` for consistency:

```bash
# First run - creates mapping
niac sanitize --mapping-file map.json file1.walk out1.walk

# Subsequent runs - reuses mapping
niac sanitize --mapping-file map.json file2.walk out2.walk
```

## Security Notes

1. **Deterministic != Reversible**: While mappings are deterministic, SHA-256 prevents reversing sanitized IPs back to originals without the mapping file.

2. **Protect Mapping Files**: The mapping file contains the key to reverse transformations. Keep it private.

3. **Review Output**: Always review sanitized files before sharing publicly to ensure all sensitive data was transformed.

4. **Special Cases**: Some custom SNMP OIDs may contain sensitive data not caught by standard patterns. Review vendor-specific MIBs.

## Related Documentation

- [SNMP Agent Configuration](../examples/snmp/)
- [Walk File Format](WALK_FILE_FORMAT.md)
- [NiAC-Go Branding Guidelines](NIAC_GO_CORP_BRANDING.md)
- [CLI Reference](CLI_REFERENCE.md)

## Version History

- **v1.23.0**: Initial release
  - Deterministic IP mapping
  - Batch processing
  - Mapping file persistence
  - 555 walk files sanitized in main repository
