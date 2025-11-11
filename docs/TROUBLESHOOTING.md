# Troubleshooting Guide

Common issues, error messages, and solutions for NiAC-Go.

## Table of Contents

- [Configuration Errors](#configuration-errors)
- [Validation Failures](#validation-failures)
- [Protocol Issues](#protocol-issues)
- [Topology Problems](#topology-problems)
- [Performance Issues](#performance-issues)
- [Network Connectivity](#network-connectivity)
- [SNMP Issues](#snmp-issues)
- [Debug Techniques](#debug-techniques)

## Configuration Errors

### Error: "Failed to parse YAML"

**Symptom:**
```
Error: failed to load config: yaml: unmarshal errors:
  line 10: cannot unmarshal !!str `yes` into bool
```

**Cause:** YAML syntax error or incorrect data type.

**Solutions:**

1. **Boolean values** - Use lowercase `true`/`false`, not `yes`/`no`:
   ```yaml
   # ❌ Wrong
   enabled: yes

   # ✅ Correct
   enabled: true
   ```

2. **Indentation** - Use consistent 2-space indentation:
   ```yaml
   # ❌ Wrong (mixed tabs/spaces)
   devices:
   	- name: device-01
     type: switch

   # ✅ Correct
   devices:
     - name: device-01
       type: switch
   ```

3. **Strings with special characters** - Use quotes:
   ```yaml
   # ❌ Wrong
   password: p@ssw0rd:123

   # ✅ Correct
   password: "p@ssw0rd:123"
   ```

4. **Validate YAML syntax** online at https://www.yamllint.com/

### Error: "Invalid MAC address format"

**Symptom:**
```
Error: invalid MAC address format: 00:11:22:33:44
```

**Cause:** MAC address must be 6 octets (48 bits).

**Solution:**
```yaml
# ❌ Wrong (5 octets)
mac: "00:11:22:33:44"

# ❌ Wrong (no colons)
mac: "001122334455"

# ✅ Correct
mac: "00:11:22:33:44:55"
```

### Error: "Invalid IP address format"

**Symptom:**
```
Error: invalid IP address: 10.0.0.256
```

**Cause:** IP address octets must be 0-255.

**Solution:**
```yaml
# ❌ Wrong (octet > 255)
ips: ["10.0.0.256"]

# ❌ Wrong (IPv6 missing segments)
ips: ["2001:db8::"]

# ✅ Correct
ips: ["10.0.0.1"]
ips: ["2001:db8::1"]
```

### Error: "Duplicate device name"

**Symptom:**
```
Error: duplicate device name: switch-01
```

**Cause:** Device names must be unique across all devices.

**Solution:**
```yaml
# ❌ Wrong
devices:
  - name: switch-01
    mac: "00:11:22:33:44:01"
  - name: switch-01  # Duplicate!
    mac: "00:11:22:33:44:02"

# ✅ Correct
devices:
  - name: switch-01
    mac: "00:11:22:33:44:01"
  - name: switch-02
    mac: "00:11:22:33:44:02"
```

### Error: "Duplicate MAC address"

**Symptom:**
```
Error: duplicate MAC address: 00:11:22:33:44:01
```

**Cause:** Each device must have a unique MAC address.

**Solution:**
```yaml
# ❌ Wrong
devices:
  - name: switch-01
    mac: "00:11:22:33:44:01"
  - name: switch-02
    mac: "00:11:22:33:44:01"  # Duplicate!

# ✅ Correct
devices:
  - name: switch-01
    mac: "00:11:22:33:44:01"
  - name: switch-02
    mac: "00:11:22:33:44:02"
```

## Validation Failures

### Error: "Bridge priority must be multiple of 4096"

**Symptom:**
```
Error: STP bridge_priority must be multiple of 4096 (0-61440)
```

**Cause:** STP bridge priority has specific valid values.

**Solution:**
```yaml
# ❌ Wrong
stp:
  bridge_priority: 5000  # Not a multiple of 4096

# ✅ Correct (valid values)
stp:
  bridge_priority: 4096   # Root bridge
  # or 8192, 12288, 16384, 20480, 24576, 28672, 32768, etc.
```

### Error: "Port-channel member already in use"

**Symptom:**
```
Error: interface GigabitEthernet0/1 already belongs to port-channel 1
```

**Cause:** An interface can only be in one port-channel.

**Solution:**
```yaml
# ❌ Wrong
port_channels:
  - id: 1
    members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
  - id: 2
    members: ["GigabitEthernet0/1", "GigabitEthernet0/3"]  # 0/1 duplicate!

# ✅ Correct
port_channels:
  - id: 1
    members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
  - id: 2
    members: ["GigabitEthernet0/3", "GigabitEthernet0/4"]
```

### Error: "Invalid VLAN ID"

**Symptom:**
```
Error: invalid VLAN ID: 5000 (must be 1-4094)
```

**Cause:** VLAN IDs must be within valid range and avoid reserved values.

**Solution:**
```yaml
# ❌ Wrong
trunk_ports:
  - interface: "GigabitEthernet0/1"
    vlans: [1, 10, 5000]  # 5000 > 4094

# ❌ Wrong (reserved VLANs)
trunk_ports:
  - interface: "GigabitEthernet0/1"
    vlans: [1, 1002, 1003]  # 1002-1005 reserved

# ✅ Correct
trunk_ports:
  - interface: "GigabitEthernet0/1"
    vlans: [1, 10, 20, 30, 40]
```

### Error: "Remote device not found"

**Symptom:**
```
Warning: remote device 'switch-03' not found in configuration
```

**Cause:** Referenced device doesn't exist in devices list.

**Solution:**
```yaml
# ❌ Wrong
devices:
  - name: switch-01
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        remote_device: "switch-03"  # Doesn't exist!

# ✅ Correct
devices:
  - name: switch-01
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        remote_device: "switch-02"
  - name: switch-02
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        remote_device: "switch-01"
```

### Error: "DHCP pool overlap"

**Symptom:**
```
Error: DHCP pools overlap: 10.0.10.100-10.0.10.200 and 10.0.10.150-10.0.10.250
```

**Cause:** DHCP address pools must not overlap.

**Solution:**
```yaml
# ❌ Wrong
dhcp:
  pools:
    - network: "10.0.10.0/24"
      range_start: "10.0.10.100"
      range_end: "10.0.10.200"
    - network: "10.0.10.0/24"
      range_start: "10.0.10.150"  # Overlaps with first pool!
      range_end: "10.0.10.250"

# ✅ Correct
dhcp:
  pools:
    - network: "10.0.10.0/24"
      range_start: "10.0.10.100"
      range_end: "10.0.10.200"
    - network: "10.0.10.0/24"
      range_start: "10.0.10.201"  # No overlap
      range_end: "10.0.10.250"
```

## Protocol Issues

### LLDP neighbors not appearing

**Symptoms:**
- `lldpcli show neighbors` shows no results
- Neighbors expected but not discovered

**Diagnosis:**

1. **Check LLDP is enabled:**
   ```yaml
   lldp:
     enabled: true  # Must be true
   ```

2. **Verify NiAC-Go is running:**
   ```bash
   ps aux | grep niac
   ```

3. **Check debug output:**
   ```bash
   sudo niac --debug-lldp 3 en0 config.yaml
   ```

4. **Monitor LLDP traffic:**
   ```bash
   sudo tcpdump -i en0 ether proto 0x88cc
   ```

**Common Causes:**

- LLDP not enabled on remote device
- Wrong network interface specified
- Firewall blocking multicast traffic
- Advertisement interval too long (increase with `--debug`)

**Solution:**
```yaml
# Ensure both ends have LLDP enabled
devices:
  - name: switch-01
    lldp:
      enabled: true
      advertise_interval: 30
  - name: switch-02
    lldp:
      enabled: true
      advertise_interval: 30
```

### CDP not working with LLDP enabled

**Symptom:** CDP neighbors not appearing when LLDP is also enabled.

**Cause:** Only one discovery protocol enabled at a time by default.

**Solution:** Enable both protocols explicitly:
```yaml
devices:
  - name: cisco-switch
    lldp:
      enabled: true
    cdp:
      enabled: true  # Both can be enabled simultaneously
```

### SNMP queries timing out

**Symptoms:**
```bash
$ snmpget -v2c -c public 10.0.0.1 sysName.0
Timeout: No Response from 10.0.0.1
```

**Diagnosis:**

1. **Verify SNMP agent enabled:**
   ```yaml
   snmp_agent:
     enabled: true
     community: "public"
   ```

2. **Check community string:**
   ```bash
   # Wrong community
   snmpget -v2c -c public 10.0.0.1 sysName.0  # ❌

   # Correct community
   snmpget -v2c -c niac-go-ro 10.0.0.1 sysName.0  # ✅
   ```

3. **Verify device is reachable:**
   ```bash
   ping 10.0.0.1
   ```

4. **Check debug output:**
   ```bash
   sudo niac --debug-snmp 3 en0 config.yaml
   ```

**Solution:**
```yaml
snmp_agent:
  enabled: true
  community: "niac-go-ro"  # Use this community string
```

### SNMP traps not received

**Symptoms:** Trap receiver not receiving trap messages.

**Diagnosis:**

1. **Verify trap configuration:**
   ```yaml
   snmp_agent:
     traps:
       enabled: true
       receivers: ["10.100.0.100:162"]  # Correct IP and port?
       community: "trap-community"
   ```

2. **Check trap receiver is listening:**
   ```bash
   # On trap receiver
   sudo snmptrapd -f -Lo
   ```

3. **Verify trap triggers are enabled:**
   ```yaml
   snmp_agent:
     traps:
       high_cpu:
         enabled: true  # Must be true
         threshold: 80
   ```

4. **Inject errors to trigger traps:**
   ```bash
   sudo niac --interactive en0 config.yaml
   # Press 'i' -> Select device -> Set High CPU to 85%
   ```

**Solution:**
```yaml
snmp_agent:
  traps:
    enabled: true
    receivers: ["10.100.0.100:162"]
    community: "trap-community"
    high_cpu:
      enabled: true
      threshold: 80
```

### DHCP clients not receiving leases

**Symptoms:** DHCP clients not getting IP addresses.

**Diagnosis:**

1. **Verify DHCP server enabled:**
   ```yaml
   dhcp:
     enabled: true
   ```

2. **Check pool configuration:**
   ```yaml
   dhcp:
     pools:
       - network: "10.0.10.0/24"
         range_start: "10.0.10.100"
         range_end: "10.0.10.200"
         gateway: "10.0.0.1"
   ```

3. **Monitor DHCP traffic:**
   ```bash
   sudo tcpdump -i en0 port 67 or port 68
   ```

4. **Check debug output:**
   ```bash
   sudo niac --debug-dhcp 3 en0 config.yaml
   ```

**Common Causes:**
- DHCP pool exhausted (all IPs assigned)
- Client on wrong subnet
- Firewall blocking DHCP ports (67/68)
- Gateway IP incorrect

**Solution:**
```yaml
dhcp:
  enabled: true
  pools:
    - network: "10.0.10.0/24"
      range_start: "10.0.10.100"
      range_end: "10.0.10.200"
      gateway: "10.0.10.1"  # Correct gateway in same subnet
      dns_servers: ["10.0.0.1"]
      lease_time: 86400
```

### DNS queries not resolving

**Symptoms:** DNS queries failing or returning NXDOMAIN.

**Diagnosis:**

1. **Verify DNS server enabled:**
   ```yaml
   dns:
     enabled: true
   ```

2. **Check forward records:**
   ```yaml
   dns:
     forward_records:
       - name: "router.example.com"
         ip: "10.0.0.1"
   ```

3. **Test DNS query:**
   ```bash
   # Wrong domain
   dig @10.0.0.1 wrong.example.com  # NXDOMAIN

   # Correct domain
   dig @10.0.0.1 router.example.com  # Should return 10.0.0.1
   ```

4. **Monitor DNS traffic:**
   ```bash
   sudo tcpdump -i en0 port 53
   ```

**Solution:**
```yaml
dns:
  enabled: true
  forward_records:
    - name: "router.example.com"  # Use full FQDN
      ip: "10.0.0.1"
      ttl: 3600
```

## Topology Problems

### Port-channel not forming

**Symptoms:** Port-channel interfaces not coming up.

**Diagnosis:**

1. **Check LACP mode compatibility:**
   ```yaml
   # ❌ Wrong: passive on both ends (needs one active)
   device-01:
     port_channels:
       - mode: "passive"
   device-02:
     port_channels:
       - mode: "passive"

   # ✅ Correct: active on both ends
   device-01:
     port_channels:
       - mode: "active"
   device-02:
     port_channels:
       - mode: "active"
   ```

2. **Verify member interfaces exist:**
   ```yaml
   port_channels:
     - id: 1
       members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
   ```

3. **Check member interface status:**
   ```bash
   sudo niac --debug-lacp 3 en0 config.yaml
   ```

**LACP Mode Combinations:**

| Device 1 | Device 2 | Result |
|----------|----------|--------|
| active | active | ✅ Works (best) |
| active | passive | ✅ Works |
| passive | passive | ❌ Fails |
| on | on | ✅ Works (no LACP) |
| active | on | ❌ Fails (mismatch) |

**Solution:**
```yaml
# Use active mode on both ends
devices:
  - name: switch-01
    port_channels:
      - id: 1
        members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
        mode: "active"
  - name: switch-02
    port_channels:
      - id: 1
        members: ["GigabitEthernet0/1", "GigabitEthernet0/2"]
        mode: "active"
```

### VLANs not passing traffic on trunk

**Symptoms:** VLAN traffic not crossing trunk link.

**Diagnosis:**

1. **Verify VLAN in allowed list:**
   ```yaml
   trunk_ports:
     - interface: "GigabitEthernet0/1"
       vlans: [1, 10, 20, 30]  # Is your VLAN here?
   ```

2. **Check both ends of trunk:**
   ```yaml
   # Both devices must allow same VLANs
   device-01:
     trunk_ports:
       - vlans: [1, 10, 20, 30]
   device-02:
     trunk_ports:
       - vlans: [1, 10, 20, 30]  # Must match
   ```

3. **Verify native VLAN matches:**
   ```yaml
   device-01:
     trunk_ports:
       - native_vlan: 1
   device-02:
     trunk_ports:
       - native_vlan: 1  # Must match
   ```

**Solution:**
```yaml
devices:
  - name: switch-01
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        vlans: [1, 10, 20, 30, 40]  # Add missing VLANs
        native_vlan: 1
        remote_device: "switch-02"

  - name: switch-02
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        vlans: [1, 10, 20, 30, 40]  # Match exactly
        native_vlan: 1
        remote_device: "switch-01"
```

### STP topology not converging

**Symptoms:** STP taking too long to converge or loops forming.

**Diagnosis:**

1. **Check STP version:**
   ```yaml
   stp:
     version: "rstp"  # Use RSTP for fast convergence
   ```

2. **Verify root bridge priority:**
   ```yaml
   # Root bridge should have lowest priority
   root-switch:
     stp:
       bridge_priority: 4096  # Lowest

   secondary-switch:
     stp:
       bridge_priority: 8192  # Higher than root
   ```

3. **Check STP timers:**
   ```yaml
   stp:
     hello_time: 2     # Default
     max_age: 20       # Default
     forward_delay: 15 # Default
   ```

4. **Monitor STP:**
   ```bash
   sudo niac --debug-stp 3 en0 config.yaml
   sudo tcpdump -i en0 ether dst 01:80:c2:00:00:00
   ```

**Solution:**
```yaml
devices:
  - name: root-bridge
    stp:
      enabled: true
      bridge_priority: 4096  # Force as root
      version: "rstp"        # Fast convergence

  - name: backup-bridge
    stp:
      enabled: true
      bridge_priority: 8192  # Backup root
      version: "rstp"

  - name: access-switch
    stp:
      enabled: true
      bridge_priority: 32768  # Default
      version: "rstp"
```

## Performance Issues

### High CPU usage

**Symptoms:** NiAC-Go consuming excessive CPU.

**Diagnosis:**

1. **Check packet rate:**
   ```bash
   # Monitor packet rate
   sudo niac --stats en0 config.yaml
   ```

2. **Identify traffic sources:**
   - Too many devices
   - Aggressive advertisement intervals
   - Excessive random traffic

**Solutions:**

1. **Increase advertisement intervals:**
   ```yaml
   lldp:
     advertise_interval: 60  # Instead of 30
   cdp:
     advertise_interval: 120  # Instead of 60
   ```

2. **Reduce random traffic:**
   ```yaml
   traffic:
     random_traffic:
       enabled: false  # Disable if not needed
   ```

3. **Limit device count:**
   - Consider splitting configuration into multiple runs

### High memory usage

**Symptoms:** NiAC-Go consuming excessive memory.

**Diagnosis:**

1. **Check walk file sizes:**
   ```bash
   ls -lh examples/device_walks_sanitized/cisco/*.walk
   ```

2. **Monitor memory:**
   ```bash
   ps aux | grep niac
   top -pid $(pgrep niac)
   ```

**Solutions:**

1. **Use smaller walk files:**
   - Remove unnecessary OIDs from walk files
   - Use device-specific walk files

2. **Reduce device count:**
   - Split large configurations

### Slow startup time

**Symptoms:** NiAC-Go takes long time to start.

**Causes:**
- Large walk files loading
- Many devices configured
- Complex topology validation

**Solutions:**

1. **Use `--dry-run` for validation only:**
   ```bash
   niac --dry-run lo0 config.yaml
   ```

2. **Optimize walk files:**
   - Use sanitized walk files (smaller)
   - Remove unused MIB tables

## Network Connectivity

### Cannot ping device IPs

**Symptoms:** `ping 10.0.0.1` fails even though ICMP enabled.

**Diagnosis:**

1. **Verify ICMP enabled:**
   ```yaml
   icmp:
     enabled: true
   ```

2. **Check device is running:**
   ```bash
   ps aux | grep niac
   ```

3. **Verify IP address:**
   ```yaml
   ips: ["10.0.0.1"]  # Correct IP?
   ```

4. **Check network interface:**
   ```bash
   # Are you running on correct interface?
   sudo niac en0 config.yaml  # Not lo0!
   ```

5. **Check routing:**
   ```bash
   # Is device on same network?
   ip route get 10.0.0.1
   ```

**Solution:**
```bash
# Run NiAC-Go on correct interface
sudo niac en0 config.yaml  # Physical interface

# Enable ICMP
devices:
  - name: device-01
    ips: ["10.0.0.1"]
    icmp:
      enabled: true
```

### Interface permission denied

**Symptom:**
```
Error: failed to open interface en0: permission denied
```

**Cause:** Packet capture requires root/administrator privileges.

**Solution:**
```bash
# ❌ Wrong
niac en0 config.yaml

# ✅ Correct
sudo niac en0 config.yaml
```

### Interface not found

**Symptom:**
```
Error: interface en0 not found
```

**Diagnosis:**

1. **List available interfaces:**
   ```bash
   # macOS
   ifconfig -l

   # Linux
   ip link show

   # Windows
   ipconfig /all
   ```

2. **Common interface names:**
   - macOS: `en0`, `en1`
   - Linux: `eth0`, `ens33`, `ens192`
   - Windows: `Ethernet`, `Wi-Fi`

**Solution:**
```bash
# Use correct interface name
sudo niac eth0 config.yaml  # Linux
sudo niac en0 config.yaml   # macOS
```

## SNMP Issues

### Walk file not found

**Symptom:**
```
Error: walk file not found: device_walks/cisco/c3850.walk
```

**Cause:** Walk file path incorrect or file doesn't exist.

**Solution:**

1. **Check file exists:**
   ```bash
   ls -l examples/device_walks_sanitized/cisco/
   ```

2. **Use correct path:**
   ```yaml
   # ❌ Wrong (missing examples/)
   snmp_agent:
     walk_file: "device_walks_sanitized/cisco/niac-cisco-c3850.walk"

   # ✅ Correct (from project root)
   snmp_agent:
     walk_file: "examples/device_walks_sanitized/cisco/niac-cisco-c3850.walk"

   # ✅ Correct (absolute path)
   snmp_agent:
     walk_file: "/full/path/to/niac-cisco-c3850.walk"
   ```

### Walk file parse errors

**Symptom:**
```
Error: failed to parse walk file: invalid OID format
```

**Cause:** Corrupted or malformed walk file.

**Solutions:**

1. **Use sanitized walk files:**
   ```yaml
   # Prefer sanitized versions
   walk_file: "examples/device_walks_sanitized/cisco/niac-cisco-c3850.walk"
   ```

2. **Regenerate walk file:**
   ```bash
   # Sanitize walk file
   niac sanitize --input original.walk --output sanitized.walk
   ```

3. **Capture new walk:**
   ```bash
   # Capture from real device
   snmpwalk -v2c -c public 10.0.0.1 . > device.walk
   ```

## Debug Techniques

### Enable protocol debug output

```bash
# Debug specific protocols
sudo niac --debug-lldp 3 en0 config.yaml
sudo niac --debug-cdp 3 en0 config.yaml
sudo niac --debug-snmp 3 en0 config.yaml
sudo niac --debug-dhcp 3 en0 config.yaml
sudo niac --debug-stp 3 en0 config.yaml

# Debug multiple protocols
sudo niac --debug-lldp 3 --debug-snmp 3 en0 config.yaml

# Debug levels: 1 (errors), 2 (warnings), 3 (info), 4 (debug), 5 (trace)
```

### Capture and analyze traffic

```bash
# Capture all traffic
sudo tcpdump -i en0 -w capture.pcap

# Capture specific protocols
sudo tcpdump -i en0 ether proto 0x88cc  # LLDP
sudo tcpdump -i en0 ether dst 01:00:0c:cc:cc:cc  # CDP
sudo tcpdump -i en0 port 161 or port 162  # SNMP
sudo tcpdump -i en0 port 67 or port 68   # DHCP
sudo tcpdump -i en0 port 53              # DNS
sudo tcpdump -i en0 icmp                 # ICMP

# Analyze in Wireshark
wireshark capture.pcap
```

### Use dry-run mode

```bash
# Validate configuration without running
niac --dry-run lo0 config.yaml

# Shows:
# - Configuration validation results
# - Enabled protocols
# - Device summary
# - Topology validation
```

### Interactive error injection

```bash
# Launch interactive mode
sudo niac --interactive en0 config.yaml

# Controls:
# [i] - Open interactive menu
# [c] - Clear all errors
# [q] - Quit

# Use to:
# - Test SNMP trap triggers
# - Simulate interface failures
# - Test error conditions
```

### Check logs and output

```bash
# Run with verbose output
sudo niac -v en0 config.yaml

# Monitor system logs (macOS)
log show --predicate 'process == "niac"' --last 1h

# Monitor system logs (Linux)
journalctl -u niac -f

# Check for errors
sudo niac en0 config.yaml 2>&1 | grep -i error
```

## Getting Help

If you're still experiencing issues:

1. **Check GitHub Issues:** https://github.com/krisarmstrong/niac-go/issues
2. **Review Examples:** `examples/` directory
3. **Read Documentation:**
   - [Protocol Guide](PROTOCOL_GUIDE.md)
   - [Topology Guide](TOPOLOGY_GUIDE.md)
   - [API Reference](API_REFERENCE.md)
4. **Create Issue:** Include configuration, error messages, and debug output

## See Also

- [Protocol Configuration Guide](PROTOCOL_GUIDE.md)
- [Topology Configuration Guide](TOPOLOGY_GUIDE.md)
- [API Reference](API_REFERENCE.md)
- [Environment Simulation Guide](ENVIRONMENTS.md)
