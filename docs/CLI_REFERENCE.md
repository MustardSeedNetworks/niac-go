# NIAC CLI Reference

Complete command-line reference for NIAC-Go.

## Table of Contents

- [Global Flags](#global-flags)
- [Commands](#commands)
  - [validate](#validate)
  - [template](#template)
  - [interactive](#interactive)
  - [config](#config)
  - [init](#init)
  - [completion](#completion)
  - [man](#man)
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

### config

Configuration management tools for exporting, comparing, and merging NIAC configurations.

```bash
niac config <subcommand>
```

#### Subcommands

##### export

Export a NIAC configuration file to normalized YAML format.

```bash
niac config export <input-file> <output-file>
```

Features:
- Loads and validates the input configuration
- Normalizes all fields and structures
- Exports to clean YAML format
- Useful for converting legacy .cfg to YAML

Examples:
```bash
# Export to new file
niac config export config.yaml normalized.yaml

# Convert legacy .cfg to YAML
niac config export legacy.cfg new-config.yaml

# Validate and normalize
niac config export messy.yaml clean.yaml
```

Error handling:
- Fails if output file already exists (prevents accidental overwrite)
- Shows validation warnings but exports anyway
- Exits with error if input file cannot be loaded

##### diff

Compare two NIAC configuration files and show differences.

```bash
niac config diff <file1> <file2>
```

Compares:
- Device additions/removals
- Device name changes
- MAC/IP address changes
- Device type changes

Output format:
```
+ Device added: new-device
- Device removed: old-device
~ Device router-1: MAC changed from 00:11:22:33:44:55 to 00:11:22:33:44:66
~ Device router-1: Type changed from router to switch
No differences found  (if files are identical)
```

Examples:
```bash
# Compare two configs
niac config diff prod.yaml staging.yaml

# Check for configuration drift
niac config diff baseline.yaml current.yaml

# Compare before/after changes
niac config diff config.yaml config.new.yaml
```

##### merge

Merge two NIAC configuration files with overlay semantics.

```bash
niac config merge <base-file> <overlay-file> <output-file>
```

Merge behavior:
- Devices with same name: overlay replaces base
- New devices in overlay: added to result
- Devices only in base: kept in result

Examples:
```bash
# Merge overlay into base
niac config merge base.yaml overlay.yaml merged.yaml

# Apply environment-specific overrides
niac config merge common.yaml prod-overrides.yaml prod-config.yaml

# Combine device configs from different sources
niac config merge routers.yaml switches.yaml network.yaml
```

Output:
```
Merged configuration written to merged.yaml
Base devices: 5
Overlay devices: 3
Merged devices: 7
```

Error handling:
- Fails if output file already exists
- Exits with error if base or overlay cannot be loaded

##### generate

Interactive configuration generator for NIAC (more detailed than `niac init`).

```bash
niac config generate [output-file]
```

The generator prompts for:
- Network name and subnet
- Number of devices
- Device details (type, name, IP, MAC)
- Protocols to enable (LLDP, CDP, SNMP, DHCP, DNS, etc.)
- Protocol-specific configuration

Examples:
```bash
# Generate configuration interactively
niac config generate

# Generate with specific output file
niac config generate my-network.yaml

# Validate and run
niac config generate network.yaml && niac validate network.yaml
```

Interactive prompts:
```
Step 1: Network Information
  Network name: simulation-network
  Network subnet (CIDR): 192.168.1.0/24
  Path for SNMP walk files: /path/to/walks

Step 2: Device Configuration
  How many devices: 3
  Device 1:
    Type: (1) router (2) switch (3) ap...
    Name: router-1
    IP: 192.168.1.1
    MAC: 02:00:00:00:00:01
    Enable LLDP? (y/n)
    Enable CDP? (y/n)
    ...

Step 3: Save Configuration
  Output filename: config.yaml
```

### init

Interactive template selection wizard for quick project setup.

```bash
niac init [flags]
```

Provides a guided interface for:
1. Selecting a template (minimal, router, switch, etc.)
2. Specifying output filename
3. Optionally editing the configuration

Examples:
```bash
# Start the init wizard
niac init

# Create router config and validate
niac init && niac validate config.yaml
```

Interactive flow:
```
NIAC Configuration Wizard

Select a template:
  1) minimal      - Single device with basic protocols
  2) router       - Enterprise router
  3) switch       - L2/L3 switch
  4) ap           - Access point
  5) server       - Multi-service server
  6) iot          - IoT device
  7) complete     - Multi-device network

Choice [1-7]: 2

Output filename [config.yaml]: my-router.yaml

✓ Created my-router.yaml

Next steps:
  1. Edit configuration: vim my-router.yaml
  2. Validate: niac validate my-router.yaml
  3. Run: sudo niac interactive en0 my-router.yaml
```

### completion

Generate shell completion scripts for niac commands.

```bash
niac completion <shell>
```

Supported shells:
- `bash`
- `zsh`
- `fish`
- `powershell`

#### Installation

##### Bash

```bash
# Generate completion script
niac completion bash > /etc/bash_completion.d/niac

# Or add to your .bashrc
echo 'source <(niac completion bash)' >> ~/.bashrc
source ~/.bashrc
```

##### Zsh

```bash
# Generate completion script
niac completion zsh > "${fpath[1]}/_niac"

# Or add to your .zshrc
echo 'source <(niac completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

##### Fish

```bash
# Generate completion script
niac completion fish > ~/.config/fish/completions/niac.fish

# Reload fish completions
source ~/.config/fish/config.fish
```

##### PowerShell

```powershell
# Add to your PowerShell profile
niac completion powershell | Out-String | Invoke-Expression

# Or save to file
niac completion powershell > niac.ps1
```

#### Features

Once installed, shell completion provides:
- Command completion (`niac va<TAB>` → `niac validate`)
- Subcommand completion (`niac template <TAB>` → shows `list show use`)
- Flag completion (`niac validate --<TAB>` → shows available flags)
- File path completion for config files
- Template name completion for `niac template` commands

### man

Generate Unix man pages for niac commands.

```bash
niac man [output-directory]
```

Generates man pages for all niac commands in the specified directory (defaults to ./man).

Examples:
```bash
# Generate man pages in ./man directory
niac man

# Generate in custom directory
niac man /usr/local/share/man/man1

# Generate and install system-wide (requires sudo)
sudo niac man /usr/share/man/man1
sudo mandb  # Update man page database

# View generated man pages
man niac
man niac-validate
man niac-template
```

Generated pages:
- `niac.1` - Main niac command
- `niac-validate.1` - Validate subcommand
- `niac-template.1` - Template subcommand
- `niac-interactive.1` - Interactive subcommand
- `niac-config.1` - Config management commands
- `niac-init.1` - Init wizard
- `niac-completion.1` - Shell completion

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

#### Performance Profiling Flags
- `--profile, -p` - Enable pprof performance profiling
- `--profile-port <port>` - Port for pprof HTTP server (default: 6060)

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

#### 5. Performance Profiling

```bash
# Enable profiling on default port (6060)
niac en0 config.yaml --profile

# Enable profiling on custom port
niac en0 config.yaml --profile --profile-port 8080

# Collect CPU profile (30 seconds)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Collect memory profile
curl http://localhost:6060/debug/pprof/heap > mem.prof
go tool pprof mem.prof

# Interactive CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Interactive memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# View goroutines
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# Available pprof endpoints:
# http://localhost:6060/debug/pprof/          - Index page
# http://localhost:6060/debug/pprof/profile   - CPU profile
# http://localhost:6060/debug/pprof/heap      - Memory heap profile
# http://localhost:6060/debug/pprof/goroutine - Goroutine stack traces
# http://localhost:6060/debug/pprof/block     - Block profile
# http://localhost:6060/debug/pprof/mutex     - Mutex profile
# http://localhost:6060/debug/pprof/allocs    - Allocation profile
```

**Security Note:** The profiling server binds to `127.0.0.1` (localhost only) for security. Do not expose the profiling port on public networks or production environments.

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
