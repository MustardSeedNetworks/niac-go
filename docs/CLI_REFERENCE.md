# NIAC CLI Reference

Complete command-line reference for NIAC-Go.

## Table of Contents

- [Global Flags](#global-flags)
- [Commands](#commands)
  - [validate](#validate)
  - [template](#template)
  - [interactive](#interactive)
- [Legacy Mode](#legacy-mode)
- [Examples](#examples)

## Global Flags

```bash
--help, -h      Show help for any command
--version       Show version information
```

## Commands

### validate

Validate a NIAC configuration file for errors and warnings.

```bash
niac validate <config-file> [flags]
```

#### Flags

- `--verbose, -v` - Show detailed validation information
- `--json` - Output validation results as JSON (for CI/CD)

#### Validation Checks

The validator performs comprehensive checks:

**Device Validation:**
- Device name uniqueness
- Required fields (name, type)
- Device type validity (router, switch, ap, access-point, server, host)
- MAC address format and duplicates
- IP address duplicates

**Protocol Validation:**
- SNMP trap threshold ranges (0-100)
- SNMP trap receiver format (IP:port or IP)
- DNS record format (domain names, IPs)
- DNS forward and reverse record completeness

**Output Levels:**
- **Errors**: Critical issues that prevent configuration loading
- **Warnings**: Potential issues that don't prevent loading
- **Info**: Informational messages

#### Exit Codes

- `0` - Configuration is valid
- `1` - Configuration has errors

#### Examples

```bash
# Basic validation
niac validate config.yaml

# Verbose output with details
niac validate config.yaml --verbose

# JSON output for CI/CD pipeline
niac validate config.yaml --json > validation-results.json

# Example validation output (errors)
✗ Configuration errors found: config.yaml

devices[0].mac_address: duplicate MAC address 00:11:22:33:44:55 (also used by router-02)
devices[1].snmp.traps.high_cpu.threshold: threshold must be between 0 and 100, got 150

Summary: 2 error(s), 0 warning(s)

# Example validation output (success)
✓ Configuration is valid: config.yaml
```

### template

Manage configuration templates for quick start scenarios.

```bash
niac template <subcommand>
```

#### Subcommands

##### list

List all available templates with descriptions.

```bash
niac template list
```

Output:
```
Available Templates:

  minimal      - Single device with basic protocols
  router       - Enterprise router with full protocol support
  switch       - Layer 2/3 switch with STP and VLAN support
  ap           - Enterprise Wi-Fi access point
  server       - Multi-service server (DHCP, DNS, HTTP)
  iot          - Lightweight IoT sensor device
  complete     - Multi-device network simulation

Usage:
  niac template show <template-name>
  niac template use <template-name> <output-file>
```

##### show

Display the contents of a template.

```bash
niac template show <template-name>
```

Example:
```bash
niac template show minimal
```

##### use

Copy a template to a new configuration file.

```bash
niac template use <template-name> <output-file>
```

Examples:
```bash
# Create a router configuration
niac template use router my-router.yaml

# Create an IoT device configuration
niac template use iot sensor-network.yaml

# Create a complete multi-device network
niac template use complete lab-network.yaml
```

Error handling:
```bash
# Error if file already exists
$ niac template use router config.yaml
Error: file already exists: config.yaml

# Error if template doesn't exist
$ niac template use invalid config.yaml
Error: template not found: invalid
Available templates: minimal, router, switch, ap, server, iot, complete
```

#### Template Descriptions

| Template | Use Case | Protocols | Devices |
|----------|----------|-----------|---------|
| **minimal** | Quick testing, CI/CD | ARP, ICMP | 1 |
| **router** | Enterprise router | All routing protocols | 1 |
| **switch** | L2/L3 switch | STP, LLDP, CDP, VLANs | 1 |
| **ap** | Wi-Fi access point | LLDP, HTTP, SNMP | 1 |
| **server** | Infrastructure server | DHCP, DNS, HTTP, SNMP | 1 |
| **iot** | IoT/embedded device | Minimal (ARP, ICMP, HTTP) | 1 |
| **complete** | Full network lab | All protocols | 4 |

### interactive

Run NIAC with an interactive Terminal User Interface (TUI).

```bash
niac interactive <interface> <config-file>
```

#### Features

- **Real-time Device Monitoring**: Live status for all simulated devices
- **Live Statistics**: Packet counts, rates, and protocol statistics
- **Interactive Error Injection**: Press 'i' to access error injection menu
- **Device Status Visualization**: Color-coded device states
- **Keyboard Controls**:
  - `q` - Quit
  - `i` - Interactive error injection menu
  - Arrow keys - Navigate devices/options

#### Examples

```bash
# Run interactive mode
niac interactive en0 config.yaml

# With existing --interactive flag (legacy)
niac --interactive en0 config.yaml
```

#### TUI Features

The interactive TUI provides:
- Device list with status indicators
- Per-device packet statistics
- Protocol status (enabled/disabled)
- Error injection controls
- Real-time updates

## Legacy Mode

NIAC maintains backward compatibility with the original command-line interface.

```bash
niac <interface> <config-file> [flags]
```

### Legacy Flags

#### Core Flags
- `--debug <level>` - Set debug level (0-3)
- `--verbose, -v` - Verbose output
- `--quiet, -q` - Quiet mode (errors only)
- `--interactive, -i` - Interactive TUI mode
- `--dry-run` - Validate configuration and exit

#### Information Flags
- `--version` - Show version
- `--list-interfaces` - List network interfaces
- `--list-devices` - List devices in config

#### Output Flags
- `--no-color` - Disable color output
- `--log-file <file>` - Write logs to file
- `--stats-interval <seconds>` - Statistics interval

#### Per-Protocol Debug Flags
- `--debug-arp` - Debug ARP protocol
- `--debug-icmp` - Debug ICMP protocol
- `--debug-lldp` - Debug LLDP protocol
- `--debug-cdp` - Debug CDP protocol
- `--debug-snmp` - Debug SNMP protocol
- And 14 more protocol-specific flags...

### Legacy Examples

```bash
# Basic simulation
niac en0 config.yaml

# With debug output
niac en0 config.yaml --debug 2

# Interactive mode (legacy)
niac en0 config.yaml --interactive

# Dry run validation
niac en0 config.yaml --dry-run --verbose

# Per-protocol debugging
niac en0 config.yaml --debug-lldp --debug-cdp
```

## Examples

### Complete Workflows

#### 1. Quick Start with Template

```bash
# Create a router config from template
niac template use router my-router.yaml

# Validate the configuration
niac validate my-router.yaml

# Run in interactive mode
niac interactive en0 my-router.yaml
```

#### 2. CI/CD Pipeline Integration

```bash
#!/bin/bash
# validate-config.sh

# Validate all configs
for config in configs/*.yaml; do
  echo "Validating $config..."
  niac validate "$config" --json > "results/$(basename $config .yaml).json"

  if [ $? -ne 0 ]; then
    echo "❌ Validation failed: $config"
    exit 1
  fi
done

echo "✅ All configurations valid"
```

#### 3. Development Workflow

```bash
# 1. Create config from template
niac template use complete lab-network.yaml

# 2. Edit configuration
vim lab-network.yaml

# 3. Validate before running
niac validate lab-network.yaml --verbose

# 4. Dry run to check interface
niac --dry-run en0 lab-network.yaml

# 5. Run simulation
niac interactive en0 lab-network.yaml
```

#### 4. Debugging Network Issues

```bash
# Run with LLDP and CDP debugging
niac en0 config.yaml --debug-lldp --debug-cdp

# Full debug with log file
niac en0 config.yaml --debug 3 --log-file debug.log

# Verbose validation
niac validate config.yaml --verbose
```

## Environment Variables

NIAC-Go respects the following environment variables:

- `NO_COLOR` - Disable color output (set to any value)
- `NIAC_DEBUG` - Default debug level (0-3)
- `NIAC_INTERFACE` - Default network interface

Example:
```bash
export NO_COLOR=1
export NIAC_DEBUG=2
export NIAC_INTERFACE=en0

niac my-config.yaml  # Uses environment defaults
```

## Output Formats

### Standard Output

```
✓ Success message
❌ Error message
⚠️  Warning message
ℹ️  Info message
```

### JSON Output (--json flag)

```json
{
  "file": "config.yaml",
  "valid": false,
  "errors": [
    {
      "file": "config.yaml",
      "line": 0,
      "column": 0,
      "field": "devices[0].mac_address",
      "message": "duplicate MAC address",
      "severity": "error"
    }
  ],
  "warnings": []
}
```

## See Also

- [Configuration Reference](CONFIG_REFERENCE.md)
- [Template Guide](TEMPLATES.md)
- [Examples](../examples/)
- [Troubleshooting](TROUBLESHOOTING.md)
