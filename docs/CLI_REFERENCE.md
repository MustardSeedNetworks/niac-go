# NIAC CLI Reference

Complete command-line reference for NIAC-Go.

## Table of Contents

- [Global Flags](#global-flags)
- [Commands](#commands) — generated from the command tree
- [Platform-specific commands](#platform-specific-commands)
- [Direct Invocation Mode](#direct-invocation-mode)
- [Examples](#examples)

## Global Flags

```bash
--help, -h      Show help for any command
--version       Show version information
```

<!-- BEGIN GENERATED COMMANDS -->

## Commands

- [`niac analyze-pcap`](#niac-analyze-pcap) — summarise a packet capture by protocol
- [`niac analyze-walk`](#niac-analyze-walk) — analyze an SNMP walk file: device, interfaces, and LLDP/CDP neighbors
- [`niac completion`](#niac-completion) — generate completion script
- [`niac config`](#niac-config) — configuration management tools
- [`niac config diff`](#niac-config-diff) — compare two configurations
- [`niac config export`](#niac-config-export) — export configuration to YAML
- [`niac config generate`](#niac-config-generate) — interactive configuration generator
- [`niac config interface`](#niac-config-interface) — manage device interface metadata
- [`niac config interface set`](#niac-config-interface-set) — set speed, duplex, and status for a device interface
- [`niac config merge`](#niac-config-merge) — merge two configurations
- [`niac content`](#niac-content) — install and inspect the on-disk content library
- [`niac content install`](#niac-content-install) — install a content bundle into the library
- [`niac content list`](#niac-content-list) — list what's installed in the library
- [`niac daemon`](#niac-daemon) — run NIAC in daemon mode with web UI control
- [`niac dump`](#niac-dump) — dump captured packets from a running NIAC simulation
- [`niac init`](#niac-init) — interactive template wizard for quick configuration setup
- [`niac install-ca`](#niac-install-ca) — install NIAC's self-signed root certificate into the OS trust store
- [`niac interactive`](#niac-interactive) — run NIAC in interactive TUI mode
- [`niac list`](#niac-list) — list interfaces and demo content
- [`niac list captures`](#niac-list-captures) — list packet captures
- [`niac list interfaces`](#niac-list-interfaces) — list available network interfaces
- [`niac list scenarios`](#niac-list-scenarios) — list runnable scenarios
- [`niac list walks`](#niac-list-walks) — list SNMP walk files
- [`niac logs`](#niac-logs) — view and stream simulation logs
- [`niac logs tail`](#niac-logs-tail) — stream logs from a running simulation
- [`niac man`](#niac-man) — generate man pages
- [`niac mibzip`](#niac-mibzip) — convert SNMP walk files to and from MibZip format
- [`niac mibzip compress`](#niac-mibzip-compress) — compress a text SNMP walk file
- [`niac mibzip expand`](#niac-mibzip-expand) — expand a MibZip file to text
- [`niac mibzip inspect`](#niac-mibzip-inspect) — inspect a file for MibZip format
- [`niac monitor`](#niac-monitor) — stream real-time statistics from a running NIAC simulation
- [`niac neighbors`](#niac-neighbors) — display neighbor discovery table from LLDP/CDP protocols
- [`niac neighbors watch`](#niac-neighbors-watch) — watch neighbor table for live updates
- [`niac run`](#niac-run) — run network simulation
- [`niac sanitize`](#niac-sanitize) — sanitize SNMP walk files with NIAC branding
- [`niac status`](#niac-status) — query the status of a running NIAC simulation
- [`niac template`](#niac-template) — manage configuration templates
- [`niac template apply`](#niac-template-apply) — validate and display template information
- [`niac template list`](#niac-template-list) — list available templates
- [`niac template show`](#niac-template-show) — show template contents
- [`niac template use`](#niac-template-use) — copy template to a new file
- [`niac topology`](#niac-topology) — network topology management commands
- [`niac topology export`](#niac-topology-export) — export current network topology
- [`niac validate`](#niac-validate) — validate a NIAC configuration file
- [`niac version`](#niac-version) — print the version, commit and build metadata

### `niac analyze-pcap`

Summarise a packet capture by protocol.

```text
niac analyze-pcap <pcap-file> [flags]
```

```text
Parse a PCAP file and emit protocol counters for rapid troubleshooting.
The tool classifies packets into ARP, LLDP, CDP, STP, IPv4, IPv6, TCP, UDP,
and generic application protocols.
```

Flags:

```text
      --output string   Output format (text, json, yaml) (default "text")
```

Examples:

```bash
# Summarise a capture (text output)
niac analyze-pcap capture.pcap

# Machine-readable JSON output
niac analyze-pcap --output json capture.pcap

# YAML output (handy for diffing two captures)
niac analyze-pcap --output yaml capture.pcap
```

### `niac analyze-walk`

Analyze an SNMP walk file: device, interfaces, and LLDP/CDP neighbors.

```text
niac analyze-walk <walk-file> [flags]
```

```text
Analyze an SNMP walk file and extract device identity, the interface
inventory, and LLDP/CDP neighbor adjacencies.

The tool parses these standard SNMP MIBs:
  • SNMPv2-MIB        (system identity)
  • IF-MIB + ifXTable (interfaces: index, name, type, speed, status, MAC)
  • LLDP-MIB          (LLDP neighbors)
  • CISCO-CDP-MIB     (CDP neighbors)
```

Flags:

```text
      --graphviz string   Write Graphviz (DOT) neighbor graph to file (use '-' for stdout)
      --output string     Output format (yaml, json, text) (default "yaml")
      --show-neighbors    Show neighbor relationships only
```

Examples:

```bash
# Analyze and output as YAML
niac analyze-walk device.walk

# Output as JSON
niac analyze-walk --output json device.walk

# Show only neighbor relationships
niac analyze-walk --show-neighbors device.walk

# Write a Graphviz (DOT) neighbor graph
niac analyze-walk --graphviz topology.dot device.walk
```

### `niac completion`

Generate completion script.

```text
niac completion [bash|zsh|fish|powershell]
```

```text
Generate shell completion script for NIAC.

To load completions:

Bash:
  $ source <(niac completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ niac completion bash > /etc/bash_completion.d/niac
  # macOS:
  $ niac completion bash > $(brew --prefix)/etc/bash_completion.d/niac

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ niac completion zsh > "${fpath[1]}/_niac"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ niac completion fish | source

  # To load completions for each session, execute once:
  $ niac completion fish > ~/.config/fish/completions/niac.fish

PowerShell:
  PS> niac completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> niac completion powershell > niac.ps1
  # and source this file from your PowerShell profile.
```

### `niac config`

Configuration management tools.

```text
niac config
```

```text
Tools for exporting, comparing, and merging NIAC configurations.
```

Examples:

```bash
# Export configuration to new file
niac config export input.yaml output.yaml

# Compare two configurations
niac config diff config1.yaml config2.yaml

# Merge configurations
niac config merge base.yaml overlay.yaml merged.yaml
```

### `niac config diff`

Compare two configurations.

```text
niac config diff <file1> <file2>
```

```text
Compare two NIAC configuration files and show differences.

Compares:
- Device additions/removals
- Device name changes
- MAC/IP address changes
- Protocol configuration changes
```

Examples:

```bash
# Compare two configs
niac config diff prod.yaml staging.yaml

# Check for drift
niac config diff baseline.yaml current.yaml

# Compare before/after changes
niac config diff config.yaml config.new.yaml
```

### `niac config export`

Export configuration to YAML.

```text
niac config export <input-file> <output-file>
```

```text
Export a NIAC configuration file to normalized YAML format.

This command:
- Loads and validates the input configuration
- Normalizes all fields and structures
- Exports to clean YAML format
- Useful for converting legacy .cfg to YAML
```

Examples:

```bash
# Export to new file
niac config export config.yaml normalized.yaml

# Convert legacy .cfg to YAML
niac config export legacy.cfg new-config.yaml

# Validate and normalize
niac config export messy.yaml clean.yaml
```

### `niac config generate`

Interactive configuration generator.

```text
niac config generate [output-file]
```

```text
Interactive configuration generator for NIAC.

Prompts you for all configuration details and generates a complete YAML
configuration file. More detailed than 'niac init' template wizard.

The generator will ask you for:
  - Network name and subnet
  - Number of devices
  - Device details (type, name, IP, MAC)
  - Protocols to enable (LLDP, CDP, SNMP, DHCP, DNS, etc.)
  - Protocol-specific configuration
```

Examples:

```bash
# Generate configuration interactively
niac config generate

# Generate with specific output file
niac config generate my-network.yaml

# Validate and run
niac config generate network.yaml && niac validate network.yaml
```

### `niac config interface`

Manage device interface metadata.

```text
niac config interface
```

```text
Manage device interface metadata in a NIAC configuration.

Interface metadata controls simulated speed, duplex, administrative status,
operational status, descriptions, and VLAN hints used by runtime and topology
views.
```

Examples:

```bash
# Set speed and duplex in place
niac config interface set network.yaml switch-a Ethernet1/1 --speed 1000 --duplex full

# Write the updated config to a new file
niac config interface set network.yaml switch-a Ethernet1/1 --admin-status down --output updated.yaml
```

### `niac config interface set`

Set speed, duplex, and status for a device interface.

```text
niac config interface set <config-file> <device> <interface> [flags]
```

```text
Set speed, duplex, and status for a device interface in a NIAC config.

If the named interface does not exist on the device, it is created. By default
the input file is updated in place; pass --output to write to a separate file.
```

Flags:

```text
      --admin-status string   Admin status: up or down
      --duplex string         Interface duplex: full, half, or auto
      --oper-status string    Operational status: up, down, or testing
      --output string         Output config path; defaults to updating input in place
      --speed int             Interface speed in Mbps
```

Examples:

```bash
niac config interface set network.yaml switch-a Ethernet1/1 --speed 1000 --duplex full
niac config interface set network.yaml switch-a Ethernet1/1 --admin-status down --oper-status down
```

### `niac config merge`

Merge two configurations.

```text
niac config merge <base-file> <overlay-file> <output-file>
```

```text
Merge two NIAC configuration files.

The overlay file takes precedence:
- Devices with same name are replaced
- New devices are added
- Base devices not in overlay are kept
```

Examples:

```bash
# Merge overlay into base
niac config merge base.yaml overlay.yaml merged.yaml

# Apply environment-specific overrides
niac config merge common.yaml prod-overrides.yaml prod-config.yaml

# Combine device configs
niac config merge routers.yaml switches.yaml network.yaml
```

### `niac content`

Install and inspect the on-disk content library.

```text
niac content
```

```text
Manage the content library that the daemon serves to the UI.

The library lives at ~/.niac/library by default (or /var/lib/niac/library
on packaged installs) and contains three sibling directories:

  networks/   YAML network configs
  walks/      SNMP walk files
  pcaps/      packet captures

Content ships as local bundles (embedded essentials, the niac-content
deb/rpm package, or a bundle uploaded through the UI) — there is no
network fetch. Use 'niac content install --bundle path.tar.gz' to
install one.
```

Examples:

```bash
# Show what's in the library right now
niac content list

# Install a local bundle
niac content install --bundle /tmp/niac-content.tar.gz
```

### `niac content install`

Install a content bundle into the library.

```text
niac content install [flags]
```

```text
Install a versioned content bundle (gzip-tar) into the local library
from a local file — no network access is made.

The bundle's top-level directories must be one of: networks, walks,
pcaps. Anything else is rejected. Each entry is re-rooted under
<library>/<kind>/ before any file is touched, so a malicious bundle
cannot escape the library.
```

Flags:

```text
      --bundle string   Local bundle file to install (required)
      --dry-run         Print what would be installed without writing files
      --force           Overwrite your own files and any bundle file you have edited (default: preserve them)
      --root string     Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)
```

Examples:

```bash
# Install from a local bundle
niac content install --bundle ./niac-content-v0.66.41.tar.gz

# Install into a custom root
niac content install --bundle ./niac-content.tar.gz --root /var/lib/niac/library

# Preview what would be installed
niac content install --bundle ./niac-content.tar.gz --dry-run
```

### `niac content list`

List what's installed in the library.

```text
niac content list [flags]
```

```text
Print every kind (networks / walks / pcaps) currently in the library
along with the file count and on-disk size for each, plus a TOTAL row.
```

Flags:

```text
      --root string   Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)
```

Examples:

```bash
# List the default library
niac content list

# Inspect a non-default library
niac content list --root /var/lib/niac/library
```

### `niac daemon`

Run NIAC in daemon mode with web UI control.

```text
niac daemon [flags]
```

```text
Start NIAC as a daemon process that serves the web UI and allows
starting/stopping simulations dynamically without restarting the daemon.

The daemon runs the API server and web UI independently from the simulation
engine, allowing you to:
  - Start/stop simulations from the web UI
  - Change network interfaces without restarting
  - Switch between different configuration files
  - Replace the active simulation without restarting the daemon
  - Run several scenarios at once, one per physical VLAN, on a trunk attachment

The daemon serves HTTPS on 127.0.0.1:8445 by default. Binding to a
non-loopback address (e.g. --listen 0.0.0.0) requires an API token via
NIAC_API_TOKEN or --api-token.
```

Flags:

```text
      --api-token string                Bearer token (preferred: NIAC_API_TOKEN). Required when --listen is non-loopback.
      --attachment-policy stringArray   Operator-approved routed attachment (repeatable): INTERFACE=direct, INTERFACE=access:VLAN, or INTERFACE=trunk:VLAN,...
      --cert-dir string                 Directory holding the self-signed cert and key (default: certs/ relative to CWD; override with NIAC_CERT_DIR)
      --listen string                   Address to listen on for the HTTPS API and web UI (default: 127.0.0.1:8445)
      --storage string                  Path to run history database (use 'disabled' to disable) (default "~/.niac/niac.db")
      --storage-keep int                Run history records to keep, pruned oldest first on start (0 keeps every run) (default 500)
      --token-file string               Path to a 0600 JSON file with scoped tokens (overrides --api-token / NIAC_API_TOKEN). Schema: {"tokens":[{"value":"...","scope":"read-only|read-write"}]}. Re-read on SIGHUP.
      --webhook-allowed-host strings    Hostname allowed as alert webhook destination (repeatable; if any are set, all webhook URLs must match exactly). When unset, the existing private-IP/blocked-hostname filter is used.
```

Examples:

```bash
# Default: HTTPS on 127.0.0.1:8445 (loopback only, no token needed)
niac daemon

# Listen on all interfaces — requires an API token
export NIAC_API_TOKEN=$(openssl rand -base64 32)
niac daemon --listen 0.0.0.0

# Use a token file with scoped tokens (read-only / read-write)
niac daemon --token-file /etc/niac/tokens.json

# Permit routed labs on an operator-managed access port
niac daemon --attachment-policy eth0=access:200

# Permit concurrent scenario VLANs on an operator-managed trunk
niac daemon --attachment-policy eth0=trunk:200,201,202,203,204,205,299

# Permit a directly connected untagged tester
niac daemon --attachment-policy eth1=direct

# Disable run-history persistence
niac daemon --storage disabled
```

### `niac dump`

Dump captured packets from a running NIAC simulation.

```text
niac dump [flags]
```

```text
Dump packets from a running NIAC simulation, read from the daemon's live stream.

This command connects to a running NIAC simulation and retrieves
hex dumps of recently captured packets. The output format is similar
to the standard hexdump or xxd utilities.

Packets can be filtered by device name or interface name. Use the
--count flag to limit the number of packets returned.

Exit codes:
  0 - Success
  1 - Connection failed (no daemon answered)
  2 - Error occurred (request failed, parse error, etc.)
```

Flags:

```text
      --api string         Daemon API address (default: https://127.0.0.1:8445, or NIAC_API_URL)
      --cacert string      Daemon certificate to trust (default: the local daemon's own, when visible)
      --count int          Maximum number of packets to display (0 = all)
      --device string      Filter by device name
      --insecure           Skip TLS verification, for a daemon whose certificate this host cannot see
      --interface string   Filter by interface name
      --json               Output packets as JSON
```

Examples:

```bash
# Dump all captured packets
niac dump

# Dump packets for a specific device
niac dump --device router-1

# Dump packets for a specific interface
niac dump --interface eth0

# Limit output to 10 packets
niac dump --count 10

# Output as JSON
niac dump --json

# Combine filters
niac dump --device router-1 --interface eth0 --count 5

# Read from a daemon on another address
niac dump --api https://10.0.0.5:8445
```

### `niac init`

Interactive template wizard for quick configuration setup.

```text
niac init [output-file]
```

```text
Interactive wizard that helps you choose the right template and create
a configuration file for your network simulation needs.

The wizard will ask about your network type, size, and requirements,
then suggest the most appropriate template.
```

Examples:

```bash
# Start interactive wizard
niac init

# Start wizard with specific output file
niac init my-network.yaml

# Quick workflow
niac init && niac validate config.yaml
```

### `niac install-ca`

Install NIAC's self-signed root certificate into the OS trust store.

```text
niac install-ca [flags]
```

```text
Install NIAC's self-signed root certificate into the operating
system's trust store so browsers stop showing the "not secure" warning when
visiting the NIAC UI over HTTPS.

The cert NIAC generates on first launch (at certs/server.crt) is a single-
tier self-signed root: it is both the leaf served on the TLS handshake and
its own issuer. Installing it as a trusted root tells the OS to accept it
for SSL.

Run niac daemon at least once before install-ca so the certificate file
exists.

Supported platforms:
  macOS    System Keychain (requires sudo)
  Linux    System CA bundle via update-ca-certificates / update-ca-trust
  Windows  LocalMachine\Root (requires elevated shell)

Verification:
  niac install-ca --print-fingerprint
  curl -k https://localhost:8445/__version | jq -r .tlsFingerprint
The two values must match.
```

Flags:

```text
      --cert string         Path to the PEM-encoded certificate to install (default "certs/server.crt")
      --print-fingerprint   Print the SHA-256 fingerprint of the certificate and exit without modifying the trust store
      --uninstall           Remove NIAC's certificate from the OS trust store
```

Examples:

```bash
# Install NIAC's self-signed root into the OS trust store
sudo niac install-ca

# Print the cert's SHA-256 fingerprint (no trust-store change)
niac install-ca --print-fingerprint

# Remove the previously installed root from the OS trust store
sudo niac install-ca --uninstall

# Install a non-default certificate file
sudo niac install-ca --cert /etc/niac/certs/server.crt
```

### `niac interactive`

Run NIAC in interactive TUI mode.

```text
niac interactive <interface> <config-file> [flags]
```

```text
Run NIAC with an interactive Terminal User Interface (TUI).

The TUI provides:
- Real-time device monitoring
- Live statistics and packet counts
- Interactive error injection (press 'i')
- Device status visualization
- Keyboard controls (q to quit)
```

Flags:

```text
  -d, --debug int   Debug level (0-3) (default 1)
      --no-color    Disable colored output
  -q, --quiet       Quiet mode (equivalent to -d 0)
  -v, --verbose     Verbose output (equivalent to -d 3)
```

Examples:

```bash
# Run interactive mode
sudo niac interactive en0 config.yaml

# Quick start with template
niac template use router router.yaml
sudo niac interactive en0 router.yaml

# Controls during runtime:
#   i - Interactive error injection menu
#   q - Quit
#   ↑↓ - Navigate devices
```

### `niac list`

List interfaces and demo content.

```text
niac list
```

```text
List the same operator-facing resources the legacy Java demo
wrapper exposed: network interfaces, runnable scenarios, SNMP walks, and
packet captures.

Scenario output includes built-in templates and installed library networks.
Walk and capture output reads the on-disk content library.
```

Flags:

```text
      --root string   Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)
```

Examples:

```bash
# List usable network interfaces
niac list interfaces

# List runnable scenarios
niac list scenarios

# List SNMP walks, optionally scoped by vendor/path prefix
niac list walks
niac list walks cisco

# List packet captures
niac list captures
```

### `niac list captures`

List packet captures.

```text
niac list captures
```

```text
List packet captures from the content library. Captures are the
PCAP/PCAPNG/CAP files used by replay, packet inspection, and offline analysis
workflows.
```

Examples:

```bash
# List installed captures
niac list captures
```

### `niac list interfaces`

List available network interfaces.

```text
niac list interfaces [flags]
```

```text
List network interfaces available to NIAC. By default this shows
usable interfaces only (ethernet, Wi-Fi, and loopback). Use --all to include
every interface returned by libpcap.
```

Flags:

```text
      --all   Show all interfaces instead of only usable ones
```

Examples:

```bash
# List usable interfaces
niac list interfaces

# Include every libpcap-visible interface
niac list interfaces --all
```

### `niac list scenarios`

List runnable scenarios.

```text
niac list scenarios
```

```text
List runnable scenario sources. Built-in templates are always
available. Installed library networks are shown when the content library can
be opened.
```

Examples:

```bash
# List built-in and installed scenarios
niac list scenarios

# Inspect a non-default library
niac list scenarios --root /var/lib/niac/library
```

### `niac list walks`

List SNMP walk files.

```text
niac list walks [vendor-or-prefix]
```

```text
List SNMP walk files from the content library. If a prefix is
provided, only matching walk paths are shown. This mirrors the Java demo
wrapper's vendor browsing flow while preserving Go's library layout.
```

Examples:

```bash
# List all SNMP walks
niac list walks

# List Cisco walks
niac list walks cisco
```

### `niac logs`

View and stream simulation logs.

```text
niac logs
```

```text
View and stream logs from a running NIAC simulation.

The logs command provides access to simulation logs including device activity,
protocol messages, and error injections. Logs can be filtered by level and
text pattern, and can be streamed in real-time with the tail subcommand.
```

Examples:

```bash
# View recent logs
niac logs tail

# Stream logs in real-time
niac logs tail --follow

# Filter by log level
niac logs tail --level warn

# Filter by text pattern
niac logs tail --filter "device"

# Output as JSON
niac logs tail --json
```

### `niac logs tail`

Stream logs from a running simulation.

```text
niac logs tail [flags]
```

```text
Stream logs from a running NIAC simulation in real-time.

The tail command reads the daemon's live log stream over its HTTPS API and
displays log messages. Use --follow to continuously stream new logs.

Log levels (from most to least verbose):
  - debug: All messages including detailed debugging information
  - info:  Informational messages about normal operation
  - warn:  Warning messages about potential issues
  - error: Error messages about failures

The --filter option performs case-insensitive substring matching on log messages.
```

Flags:

```text
      --api string      Daemon API address (default: https://127.0.0.1:8445, or NIAC_API_URL)
      --cacert string   Daemon certificate to trust (default: the local daemon's own, when visible)
  -n, --count int       Number of recent logs to display (default: 100) (default 100)
      --filter string   Filter logs by text pattern (case-insensitive)
  -f, --follow          Continuously stream new logs (like tail -f)
      --insecure        Skip TLS verification, for a daemon whose certificate this host cannot see
      --json            Output logs as JSON (one object per line)
  -l, --level string    Minimum log level: debug, info, warn, error (default "info")
```

Examples:

```bash
# View recent logs (one-shot)
niac logs tail

# Stream logs continuously
niac logs tail --follow

# Show only warnings and errors
niac logs tail --level warn

# Filter for specific device
niac logs tail --filter "router-1"

# Combine options
niac logs tail --follow --level info --filter "LLDP"

# JSON output for scripting
niac logs tail --json | jq '.message'

# Read from a daemon on another address
niac logs tail --api https://10.0.0.5:8445
```

### `niac man`

Generate man pages.

```text
niac man [flags]
```

```text
Generate Unix man pages for NIAC commands.
```

Flags:

```text
  -o, --output string   output directory (relative paths resolve against the current working directory) (default "docs/man")
```

Examples:

```bash
# Generate man pages to ./docs/man/ (default)
niac man

# Generate to a specific directory
niac man --output /tmp/niac-man

# Install man pages (requires sudo)
sudo cp docs/man/* /usr/local/share/man/man1/
sudo mandb
```

### `niac mibzip`

Convert SNMP walk files to and from MibZip format.

```text
niac mibzip
```

```text
Convert SNMP walk files to and from NIAC MibZip format.

MibZip is the compact binary SNMP walk format used by the legacy Java
implementation. Use this command to preserve legacy walk workflows while
keeping walk conversion available in the modern Go toolchain.
```

Examples:

```bash
# Compress a text snmpwalk file
niac mibzip compress cisco.walk cisco.mz

# Expand a MibZip file back to text
niac mibzip expand cisco.mz cisco-expanded.walk

# Inspect whether a file is MibZip
niac mibzip inspect cisco.mz
```

### `niac mibzip compress`

Compress a text SNMP walk file.

```text
niac mibzip compress <walk-file> <mibzip-file>
```

```text
Compress a text SNMP walk file into NIAC MibZip binary format.

The input file is parsed as a standard snmpwalk-style text file. The output
file is created with owner-only permissions.
```

Examples:

```bash
niac mibzip compress walks/cisco-c9300.walk walks/cisco-c9300.mz
```

### `niac mibzip expand`

Expand a MibZip file to text.

```text
niac mibzip expand <mibzip-file> <walk-file>
```

```text
Expand a NIAC MibZip binary file back into a text SNMP walk file.

The expanded file is useful for validating compressed walks, reviewing legacy
assets, or converting MibZip content back into the shared walk catalog.
```

Examples:

```bash
niac mibzip expand walks/cisco-c9300.mz walks/cisco-c9300-expanded.walk
```

### `niac mibzip inspect`

Inspect a file for MibZip format.

```text
niac mibzip inspect <file>
```

```text
Inspect a file and report whether it uses NIAC MibZip format.

For MibZip files, the command also reports the number of expanded walk entries
so operators can quickly verify the compressed asset is readable.
```

Examples:

```bash
niac mibzip inspect walks/cisco-c9300.mz
```

### `niac monitor`

Stream real-time statistics from a running NIAC simulation.

```text
niac monitor [flags]
```

```text
Monitor a running NIAC simulation in real-time.

This command reads statistics from a running NIAC daemon over its HTTPS API
and displays them continuously, similar to 'top' or 'watch'.

The monitor supports multiple output formats:
  - table: Human-readable table with auto-refresh (default)
  - json:  JSON Lines format (one JSON object per interval)
  - csv:   CSV format with header (suitable for piping)

The table format clears the screen and redraws on each update, while
JSON and CSV formats append new lines for pipe-friendly output.
```

Flags:

```text
      --api string        Daemon API address (default: https://127.0.0.1:8445, or NIAC_API_URL)
      --cacert string     Daemon certificate to trust (default: the local daemon's own, when visible)
      --format string     Output format: table, json, or csv (default "table")
      --insecure          Skip TLS verification, for a daemon whose certificate this host cannot see
      --interval string   Update interval (e.g., 1s, 500ms, 2s) (default "1s")
      --session string    Scenario session to watch (default: whichever the daemon has selected)
```

Examples:

```bash
# Monitor with default settings (table format, 1s interval)
niac monitor

# Monitor with JSON output for piping to jq
niac monitor --format json | jq '.packets_rx'

# Monitor with 2-second interval
niac monitor --interval 2s

# Monitor with CSV output, redirect to file
niac monitor --format csv > stats.csv

# Read from a daemon on another address
niac monitor --api https://10.0.0.5:8445

# Monitor with fast 500ms updates
niac monitor --interval 500ms
```

### `niac neighbors`

Display neighbor discovery table from LLDP/CDP protocols.

```text
niac neighbors [watch]
```

```text
Display the neighbor discovery table from a running NIAC simulation.

This command shows network neighbors discovered via LLDP (Link Layer Discovery
Protocol) and CDP (Cisco Discovery Protocol) from simulated devices.

The table displays:
  - Device:    Local device name
  - Interface: Local interface where neighbor was discovered
  - Neighbor:  Remote device name/chassis ID
  - Protocol:  Discovery protocol (LLDP, CDP, EDP, FDP)
  - LastSeen:  When the neighbor was last seen

Without arguments, shows the current neighbor table snapshot.
Use the 'watch' subcommand for continuous live updates.
```

Flags:

```text
      --api string        Daemon API address (default: https://127.0.0.1:8445, or NIAC_API_URL)
      --cacert string     Daemon certificate to trust (default: the local daemon's own, when visible)
      --device string     Filter by device name
      --insecure          Skip TLS verification, for a daemon whose certificate this host cannot see
      --json              Output in JSON format
      --protocol string   Filter by protocol: lldp, cdp, or all (default "all")
```

Examples:

```bash
# Show current neighbors table
niac neighbors

# Show neighbors in JSON format
niac neighbors --json

# Filter by device
niac neighbors --device router-1

# Filter by protocol
niac neighbors --protocol lldp

# Watch for live updates
niac neighbors watch

# Watch with filters
niac neighbors watch --device switch-1 --protocol cdp

# Read from a daemon on another address
niac neighbors --api https://10.0.0.5:8445
```

### `niac neighbors watch`

Watch neighbor table for live updates.

```text
niac neighbors watch
```

```text
Watch the neighbor discovery table in real-time.

This subcommand continuously monitors the neighbor table and displays
updates as neighbors are discovered or expire. Similar to 'watch' command,
the display refreshes periodically to show current state.

Press Ctrl+C to stop watching.
```

Examples:

```bash
# Watch all neighbors
niac neighbors watch

# Watch with JSON output
niac neighbors watch --json

# Watch specific device
niac neighbors watch --device router-1

# Watch only LLDP neighbors
niac neighbors watch --protocol lldp
```

### `niac run`

Run network simulation.

```text
niac run <interface> <config-file> [flags]
```

```text
Run NIAC network simulation with an optional terminal UI.

By default, runs in headless mode. Add --tui for interactive terminal UI.
Use "niac daemon" for the HTTPS web UI and API.
```

Flags:

```text
  -d, --debug int   Debug level (0-3) (default 1)
  -n, --dry-run     Validate config without starting simulation
      --no-color    Disable colored output
  -q, --quiet       Quiet mode (equivalent to -d 0)
      --tui         Enable interactive Terminal UI
  -v, --verbose     Verbose output (equivalent to -d 3)
```

Examples:

```bash
# Headless simulation
sudo niac run en0 config.yaml

# With Terminal UI (TUI)
sudo niac run en0 config.yaml --tui

# Validate config without running
niac run en0 config.yaml --dry-run
```

### `niac sanitize`

Sanitize SNMP walk files with NIAC branding.

```text
niac sanitize <input-walk> <output-walk> [flags]
```

```text
Sanitize SNMP walk files by replacing real network data with consistent
NiAC-Go branded data. IP addresses are mapped deterministically so the
same input IP always produces the same output IP.

What is KEPT (not sensitive):
  • Serial numbers
  • MAC addresses
  • Hardware models
  • Interface counts/types
  • VLAN IDs

What is TRANSFORMED (deterministic):
  • IP addresses → 10.0.0.0/8 (NiAC-Go network)
  • Hostnames → niac-<location>-<type>-<number>
  • DNS domains → niac-go.com / niac-go.local
  • Contact info → netadmin@niac-go.com
  • Location strings → NiAC-Go - DC-WEST
  • Community strings → public or niac-go-ro
```

Flags:

```text
      --batch                 Batch process multiple files
      --check                 Report walks that are not safe to ship instead of writing sanitized copies
      --community string      SNMP community string (default "public")
      --contact string        Contact email (default "netadmin@niac-go.com")
      --domain string         Domain for hostnames and DNS (default "niac-go.com")
      --input-dir string      Input directory for batch mode
      --location string       Default location suffix (default "DC-WEST")
      --mapping-file string   JSON file to load/save IP mappings
      --output-dir string     Output directory for batch mode
```

Examples:

```bash
# Sanitize a single walk file
niac sanitize device.walk device-sanitized.walk

# Batch mode - sanitize all walks in a directory
niac sanitize --batch --input-dir walks/ --output-dir sanitized/

# Use persistent mapping file
niac sanitize --mapping-file ip-map.json device.walk output.walk

# Check walks are safe to ship (exit 1 when any is not)
niac sanitize --check internal/library/starter/walks/*.walk
```

### `niac status`

Query the status of a running NIAC simulation.

```text
niac status [flags]
```

```text
Query the status of a running NIAC simulation.

This command connects to a running NIAC simulation and retrieves
current status information including:
  - Simulation state (running/stopped)
  - Network interface in use
  - Configuration file path
  - Device count
  - Uptime
  - Packet statistics (RX/TX)
  - Active error injections

Exit codes:
  0 - Simulation is running
  1 - Simulation is not running (socket not found or connection refused)
  2 - Error occurred (socket error, parse error, etc.)
```

Flags:

```text
      --api string      Daemon API address (default: https://127.0.0.1:8445, or NIAC_API_URL)
      --cacert string   Daemon certificate to trust (default: the local daemon's own, when visible)
      --insecure        Skip TLS verification, for a daemon whose certificate this host cannot see
      --json            Output status as JSON
```

Examples:

```bash
# Check simulation status
niac status

# Output status as JSON
niac status --json

# Use a custom socket path
niac status --socket /var/run/niac.sock

# Use in scripts
if niac status > /dev/null 2>&1; then
  echo "NIAC is running"
else
  echo "NIAC is not running"
fi
```

### `niac template`

Manage configuration templates.

```text
niac template
```

```text
List, show, and use pre-built configuration templates for common scenarios.
```

Examples:

```bash
# List all available templates
niac template list

# Show template contents
niac template show basic-network

# Create config from template
niac template use small-office office.yaml

# Apply template directly (validate and display info)
niac template apply data-center
```

### `niac template apply`

Validate and display template information.

```text
niac template apply <template-name>
```

```text
Validate a template and display its configuration details.
This command loads the template, validates it, and shows what devices
and protocols it contains without creating a file.
```

Examples:

```bash
# Validate basic network template
niac template apply basic-network

# Check data center template
niac template apply data-center

# Verify IoT network configuration
niac template apply iot-network
```

### `niac template list`

List available templates.

```text
niac template list
```

```text
Print every bundled template name with a one-line description.
Templates cover common scenarios (basic-network, small-office, data-center,
iot-network, etc.) and are the fastest path to a runnable YAML config.
```

Examples:

```bash
# List all templates with descriptions
niac template list
```

### `niac template show`

Show template contents.

```text
niac template show <template-name>
```

```text
Print the YAML body of a named template to stdout. Useful for
inspecting what a template will produce or piping it into another tool
without writing to disk.
```

Examples:

```bash
# Show basic network template
niac template show basic-network

# Show small office template
niac template show small-office

# Pipe to file
niac template show data-center > my-config.yaml
```

### `niac template use`

Copy template to a new file.

```text
niac template use <template-name> <output-file>
```

```text
Copy a named template's body into a new YAML file at the given
output path. The output file becomes the starting point you edit and run
with 'niac run'; the template itself is unchanged.
```

Examples:

```bash
# Create small office config
niac template use small-office office.yaml

# Create IoT network config
niac template use iot-network sensors.yaml

# Create data center config
niac template use data-center dc.yaml

# Quick workflow
niac template use basic-network config.yaml && niac validate config.yaml
```

### `niac topology`

Network topology management commands.

```text
niac topology
```

```text
Network topology management commands for NIAC simulations.

These commands allow you to export and visualize the current network
topology from a running NIAC simulation.
```

Examples:

```bash
# Export topology in DOT format for Graphviz
niac topology export --format dot

# Export topology as JSON
niac topology export --format json

# Export topology to a file
niac topology export --format dot --output network.dot

# Generate a PNG using Graphviz
niac topology export --format dot | dot -Tpng -o network.png
```

### `niac topology export`

Export current network topology.

```text
niac topology export [flags]
```

```text
Export the current network topology from a running NIAC simulation.

Supported output formats:
  dot   - Graphviz DOT format for visualization (can be rendered with Graphviz tools)
  json  - JSON format for programmatic use
  yaml  - YAML format for programmatic use

The topology includes:
  - Devices (nodes): name, type (router, switch, ap, etc.)
  - Connections (links): interfaces, VLANs, link type, speed, status
  - Protocols: discovered neighbors via LLDP, CDP, etc.

Exit codes:
  0 - Success
  1 - Connection failed (simulation not running)
  2 - Error occurred (export failed, invalid format, etc.)
```

Flags:

```text
      --api string      Daemon API address (default: https://127.0.0.1:8445, or NIAC_API_URL)
      --cacert string   Daemon certificate to trust (default: the local daemon's own, when visible)
  -f, --format string   Output format: dot, json, yaml (default "dot")
      --insecure        Skip TLS verification, for a daemon whose certificate this host cannot see
  -o, --output string   Output file path (default: stdout)
```

Examples:

```bash
# Export to stdout in DOT format (default)
niac topology export

# Export as JSON
niac topology export --format json

# Export as YAML
niac topology export --format yaml

# Save to file
niac topology export --format dot --output topology.dot

# Read from a daemon on another address
niac topology export --api https://10.0.0.5:8445 --format json

# Generate visualization with Graphviz
niac topology export --format dot | dot -Tpng -o network.png
niac topology export --format dot | dot -Tsvg -o network.svg

# Process with jq
niac topology export --format json | jq '.nodes[] | .name'
```

### `niac validate`

Validate a NIAC configuration file.

```text
niac validate <config-file> [flags]
```

```text
Validate a NIAC configuration file for errors and warnings.

This command performs comprehensive validation including:
- Device name uniqueness
- MAC address format and duplicates
- IP address duplicates
- SNMP trap configurations (thresholds, receivers)
- DNS record formats
- Protocol-specific validation

Exit codes:
  0 - Configuration is valid
  1 - Configuration has errors
```

Flags:

```text
      --json      Output validation results as JSON
  -v, --verbose   Show detailed validation information
```

Examples:

```bash
# Validate a configuration file
niac validate config.yaml

# Verbose output with details
niac validate config.yaml --verbose

# JSON output for CI/CD pipeline
niac validate config.yaml --json > validation-results.json

# Use in a CI/CD script
if niac validate config.yaml; then
  echo "Config is valid, deploying..."
else
  echo "Config validation failed!"
  exit 1
fi
```

### `niac version`

Print the version, commit and build metadata.

```text
niac version [flags]
```

```text
Print the build metadata compiled into this binary.

The fields are the ones the daemon serves at /__version — version, commit,
buildTime, uiBuildHash and releaseTrain — so a deployment check can compare
the binary on disk against the daemon it started.
```

Flags:

```text
      --json   Print version metadata as JSON
```

Examples:

```bash
# Human-readable
niac version

# For scripts
niac version --json
```

<!-- END GENERATED COMMANDS -->

## Platform-specific commands

`niac service` registers only on Windows, so it is absent from the generated
reference above, which is produced on Linux and macOS. It manages the Windows
service: `niac service install`, `uninstall`, `start`, `stop` and `status`.

## Direct Invocation Mode

The original positional command form remains available in the current pre-1.0
binary. It is not a compatibility guarantee for future pre-1.0 releases.

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

**Security Note:** The profiling server binds to `127.0.0.1` (localhost only)
for security. Do not expose the profiling port on public networks or in
production environments.

## Environment Variables

NIAC-Go respects the following environment variables:

- `NO_COLOR` - Disable color output (set to any value)
- `NIAC_DEBUG` - Default debug level (0-3)
- `NIAC_INTERFACE` - Default network interface
- Per-device SSH password variables named by `devices[].ssh.password_env`

Example:

```bash
export NO_COLOR=1
export NIAC_DEBUG=2
export NIAC_INTERFACE=en0

niac my-config.yaml  # Uses environment defaults
```

## Simulated Device CLI

When a device enables `ssh`, connect to any of its simulated IPv4 addresses
with the configured username. The password is read from the environment
variable named by `password_env`; passwords are never stored in the scenario.

The IOS-like profile supports `enable`, `configure terminal`, `show ip interface
brief`, `show ip route`, running/startup configuration display and save,
interface `shutdown`/`no shutdown`, IPv4 address changes, static routes,
hostname and VLAN changes, checkpoints, rollback, reload, and configuration
event display. Type `?` or a command prefix followed by `?` for contextual help.

## Output Formats

### Standard Output

```text
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

- [Configuration schema](schemas/niac.schema.json)
- [Examples](../examples/)
- [Troubleshooting](TROUBLESHOOTING.md)
