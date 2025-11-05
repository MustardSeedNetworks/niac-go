# Complete NIAC-Go Configuration Reference

## Overview

The `complete-reference.yaml` file demonstrates **ALL** available features in NIAC-Go v1.4.0. This serves as both documentation and a working example.

## What's Included

### Global Configuration
- ✓ **Include Path** - Base directory for relative file paths
- ✓ **PCAP Playback** - Replay existing captures with time scaling
- ✓ **Discovery Protocols** - LLDP, CDP, EDP, FDP configuration

### Device Types (12 devices total)
1. **Core Router** - Full-featured Cisco 2821 with DHCP/DNS/SNMP
2. **Distribution Switch** - Cisco Catalyst 3750
3. **Access Switch** - Cisco 2960
4. **Wireless AP** - Cisco Aironet controller
5. **Linux Server** - Ubuntu with system monitoring
6. **Juniper Router** - Multi-vendor environment
7. **NetGear Switch** - Small business device
8. **VoIP Phone** - Cisco IP phone endpoint
9. **Network Printer** - HP LaserJet with printer MIBs
10. **NAS Storage** - Synology DiskStation
11. **Security Camera** - Axis network camera
12. **Dual-Stack Server** - IPv4/IPv6 capable device

### DHCP Server Features

#### DHCPv4 (15 options)
**Basic Options:**
- Subnet Mask (Option 1)
- Router/Gateway (Option 3)
- DNS Servers (Option 6)
- Domain Name (Option 15)
- Lease Time (Option 51)
- T1 Renewal Time (Option 58)
- T2 Rebinding Time (Option 59)
- Server Identifier (Option 54)
- Message Type (Option 53)

**High Priority Options:**
- Hostname (Option 12) - Automatic capture and echo
- NTP Servers (Option 42) - Time synchronization
- Domain Search List (Option 119) - Multiple DNS search domains
- TFTP Server (Option 66) - PXE boot support
- Bootfile Name (Option 67) - Boot image filename
- Vendor-Specific (Option 43) - Custom vendor data

**Static Leases:**
- MAC-to-IP bindings
- MAC address masks for wildcard matching

#### DHCPv6 (12 options)
**Basic Options:**
- Client ID (DUID) (Option 1)
- Server ID (DUID) (Option 2)
- IA_NA (Option 3) - Non-temporary address assignment
- IA_Addr (Option 5) - IPv6 address within IA_NA
- Preference (Option 7) - Server selection priority
- DNS Servers (Option 23)
- Domain Search List (Option 24)

**High Priority Options:**
- SNTP Servers (Option 31) - Simple time sync
- NTP Servers (Option 56) - Full NTP configuration
- SIP Server Addresses (Option 22) - VoIP IPv6 addresses
- SIP Domain Names (Option 21) - VoIP domain names
- FQDN (Option 39) - Client fully qualified domain name

### DNS Server Features
- **Forward Records** (A records) - hostname → IPv4
- **Reverse Records** (PTR records) - IPv4 → hostname
- **Configurable TTL** - Per-record time-to-live

### SNMP Features
- **Walk Files** - Pre-recorded SNMP walks from real devices
- **Custom MIBs** - Override or add specific OIDs
- **MIB Types Supported:**
  - `string` - Text values
  - `integer` - Signed integers
  - `gauge` - Unsigned integers
  - `counter` - 32-bit counters
  - `counter64` - 64-bit counters
  - `timeticks` - Time in 1/100th seconds
  - `ipaddress` - IP addresses
  - `objectid` - OID references

### Device Configuration
- **MAC Addresses** - Unique per device (colon-separated)
- **IP Addresses** - IPv4 (IPv6 as secondary coming soon)
- **VLANs** - 802.1Q tagging (1-4094)
- **Device Names** - Human-readable identifiers

## File Structure

```yaml
include_path: "../device_walks"

capture_playbacks:
  - file_name: "..."
    loop_time: 5000
    scale_time: 1.0

discovery_protocols:
  lldp: { enabled: true, interval: 30 }
  cdp:  { enabled: true, interval: 60 }
  edp:  { enabled: true, interval: 30 }
  fdp:  { enabled: true, interval: 60 }

devices:
  - name: "device-name"
    mac: "00:1a:2b:3c:4d:01"
    ip: "192.168.1.1"
    vlan: 1

    snmp_agent:
      walk_file: "path/to/walk/file.walk"
      add_mibs:
        - oid: "1.3.6.1.2.1.1.5.0"
          type: "string"
          value: "hostname"

    dhcp:
      # Basic options
      subnet_mask: "255.255.255.0"
      router: "192.168.1.1"
      domain_name_server: "192.168.1.1"

      # Advanced DHCPv4
      ntp_servers: ["192.168.1.100"]
      domain_search: ["example.com"]
      tftp_server_name: "tftp.example.com"
      bootfile_name: "pxeboot.img"

      # DHCPv6
      sntp_servers_v6: ["2001:db8::100"]
      ntp_servers_v6: ["2001:db8::200"]
      sip_servers_v6: ["2001:db8::300"]
      sip_domains_v6: ["voip.example.com"]

      # Static leases
      client_leases:
        - client_ip: "192.168.1.50"
          mac_addr_value: "aa:bb:cc:dd:ee:01"

    dns:
      forward_records:
        - name: "host.example.com"
          ip: "192.168.1.10"
          ttl: 3600

      reverse_records:
        - ip: "192.168.1.10"
          name: "host.example.com"
          ttl: 3600
```

## Usage Examples

### Basic Validation
```bash
go run ./cmd/niac --dry-run eth0 examples/scenario_configs/complete-reference.yaml
```

### Run with Normal Output
```bash
sudo ./niac eth0 examples/scenario_configs/complete-reference.yaml
```

### Run with Debug Output
```bash
sudo ./niac --debug 3 eth0 examples/scenario_configs/complete-reference.yaml
```

### Run in Interactive TUI Mode
```bash
sudo ./niac --interactive eth0 examples/scenario_configs/complete-reference.yaml
```

## Creating SNMP Walk Files

### From Real Device
```bash
snmpwalk -v2c -c public 192.168.1.1 > device.walk
```

### From Java NIAC Config
```bash
./niac-convert legacy-config.cfg output.yaml
```

## Customization Tips

1. **Start Simple** - Copy one device block and modify
2. **Test Incrementally** - Use `--dry-run` to validate
3. **Use Comments** - YAML supports `#` comments
4. **Organize by Function** - Group similar devices
5. **Version Control** - Keep configs in git

## Common Device Templates

### Minimal Device (SNMP only)
```yaml
- name: "simple-switch"
  mac: "00:11:22:33:44:55"
  ip: "192.168.1.10"

  snmp_agent:
    walk_file: "cisco/switch.walk"
```

### DHCP Server Device
```yaml
- name: "dhcp-server"
  mac: "00:11:22:33:44:66"
  ip: "192.168.1.1"

  dhcp:
    subnet_mask: "255.255.255.0"
    router: "192.168.1.1"
    domain_name_server: "192.168.1.1"
    ntp_servers: ["192.168.1.100"]
```

### DNS Server Device
```yaml
- name: "dns-server"
  mac: "00:11:22:33:44:77"
  ip: "192.168.1.2"

  dns:
    forward_records:
      - name: "www.example.com"
        ip: "192.168.1.20"
```

## Troubleshooting

### YAML Syntax Errors
- Ensure proper indentation (2 spaces recommended)
- Check for missing colons or quotes
- Use a YAML validator: `yamllint config.yaml`

### File Not Found
- Check `include_path` is correct
- Use absolute paths if needed
- Verify walk files exist

### Validation Failures
- Run with `--dry-run` flag
- Check debug output with `-d 3`
- Verify MACs are unique
- Ensure IPs don't conflict

## Features by Version

### v1.4.0 (Current)
- ✓ Complete DHCPv4 with 15 options
- ✓ Complete DHCPv6 with 12 options
- ✓ DNS server (A and PTR records)
- ✓ All discovery protocols (LLDP, CDP, EDP, FDP)
- ✓ SNMP walk file support
- ✓ Custom MIB overrides
- ✓ PCAP playback
- ✓ Interactive TUI mode
- ✓ Normal daemon mode

### Coming Soon
- IPv6 as primary device address
- Multiple IPs per device
- DHCPv6 prefix delegation (IA_PD)
- HTTP/FTP server config in YAML
- SNMP trap generation
- NetFlow/IPFIX export

## Support

- **Documentation**: See `/docs` directory
- **Examples**: See `/examples/scenario_configs`
- **Issues**: https://github.com/krisarmstrong/niac-go/issues

## License

This configuration file and NIAC-Go are released under the same license as the main project.

Original NIAC by Kevin Kayes (2002-2015)
Go rewrite by Kris Armstrong (2025)
