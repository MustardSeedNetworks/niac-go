# NIAC-Go CLI & UX Improvement Plan

## Current State Analysis

### Current Project Naming
- **Java**: `network_in_a_can` (underscores, verbose)
- **Go**: `niac-go` (hyphen, concise)
- **Issue**: Inconsistent naming convention (underscore vs hyphen)

### Current CLI Issues
1. ❌ Basic flag parsing with minimal options
2. ❌ No version flag
3. ❌ No list-interfaces command
4. ❌ No config validation mode
5. ❌ Debug level not prominently displayed in interactive mode
6. ❌ No verbose/quiet shortcuts
7. ❌ No dry-run mode
8. ❌ No color output control
9. ❌ No log file output
10. ❌ Help text could be more detailed

### Current Interactive Mode Issues
1. ❌ Debug level not visible in status bar
2. ❌ No way to change debug level during runtime
3. ❌ No debug log viewer in interactive mode
4. ❌ No packet capture stats by debug level
5. ❌ No filtering based on debug verbosity

---

## Proposed Improvements

### 1. PROJECT NAMING STANDARDIZATION

**Recommendation: Use consistent "niac-" prefix with language suffix**

```
OLD:
  Java: network_in_a_can
  Go:   niac-go

NEW (Option A - Recommended):
  Java: niac-java
  Go:   niac-go

NEW (Option B - More descriptive):
  Java: niac-jvm
  Go:   niac-native
```

**Rationale**:
- Consistent naming makes it clear they're related projects
- "niac-" prefix groups them together in directory listings
- Language suffix makes it obvious which is which
- Shorter names are easier to type and reference

**Recommendation**: **Option A (niac-java and niac-go)** - Clear, simple, consistent

---

### 2. ENHANCED CLI FLAGS

**Current:**
```bash
niac [-d<n>] [-i|--interactive] <interface> <config>
```

**Proposed:**
```bash
niac [OPTIONS] <interface> <config_file>

OPTIONS:
  -d, --debug <level>       Set debug level (0-3) [default: 1]
  -v, --verbose             Enable verbose output (equivalent to -d 3)
  -q, --quiet               Suppress output (equivalent to -d 0)
  -i, --interactive         Enable interactive TUI mode
  -n, --dry-run             Validate configuration without starting
  -l, --list-interfaces     List available network interfaces
      --list-devices        List devices in config file
  -V, --version             Show version information
  -h, --help                Show this help message

  --no-color                Disable colored output
  --log-file <file>         Write log to file
  --stats-interval <sec>    Statistics update interval [default: 1]

  --babble-interval <sec>   Traffic generation interval [default: 60]
  --no-traffic              Disable background traffic generation

  --snmp-community <str>    Default SNMP community string
  --max-packet-size <int>   Maximum packet size [default: 1514]

EXAMPLES:
  # List available interfaces
  niac --list-interfaces

  # Validate configuration
  niac --dry-run en0 network.cfg

  # Run in interactive mode with full debug
  niac --interactive --debug 3 en0 network.cfg

  # Run in quiet mode with log file
  niac --quiet --log-file niac.log en0 network.cfg

  # Show version
  niac --version
```

**Implementation Benefits:**
- More intuitive flag names
- Better documentation
- More operational modes
- Easier to use

---

### 3. INTERACTIVE MODE ENHANCEMENTS

#### A. Status Bar with Debug Level Display

**Current Status Bar:**
```
NIAC-Go | Packets: 1234 | Errors: 5 | Uptime: 01:23:45
```

**Proposed Status Bar:**
```
NIAC-Go v1.0.0 | Debug: [2] | Packets: 1234 ↓ 5678 ↑ | Errors: 5 | CPU: 12% | Mem: 45MB | Uptime: 01:23:45
```

#### B. Debug Controls in Interactive Mode

**New Keyboard Shortcuts:**
```
CURRENT:
  [i] - Open error injection menu
  [c] - Clear all errors
  [q] - Quit

PROPOSED:
  [i] - Open error injection menu
  [c] - Clear all errors
  [d] - Cycle debug level (0→1→2→3→0)
  [D] - Open debug settings menu
  [l] - View debug log (last 100 lines)
  [s] - View statistics
  [p] - Pause/Resume packet processing
  [h] - Show help/keyboard shortcuts
  [q] - Quit (with confirmation)
  [?] - Quick help overlay
```

#### C. Debug Log Viewer

**New Screen: Debug Log View**
```
╔══════════════════════════════════════════════════════════════════════════╗
║ Debug Log - Level 2 (Potential Problems)                   [l] to close ║
╠══════════════════════════════════════════════════════════════════════════╣
║ [12:34:01.123] ARP: Reply sent to 192.168.1.10 from Router1            ║
║ [12:34:01.456] ICMP: Echo request from 192.168.1.20                     ║
║ [12:34:02.789] HTTP: GET / from 192.168.1.30 (device: Server1)         ║
║ [12:34:03.012] SNMP: Query for sysDescr.0 (community: public)          ║
║ [12:34:04.345] ERROR: Failed to send packet (queue full)               ║
║                                                                          ║
║ [↑↓] Scroll | [PgUp/PgDn] Page | [Home/End] First/Last | [/] Filter  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

#### D. Enhanced Statistics View

**New Screen: Detailed Statistics**
```
╔══════════════════════════════════════════════════════════════════════════╗
║ Network Statistics                                       [s] to close    ║
╠══════════════════════════════════════════════════════════════════════════╣
║ Protocol Statistics:                                                     ║
║   ARP Requests:     1,234        ARP Replies:        1,198              ║
║   ICMP Requests:      567        ICMP Replies:         543              ║
║   TCP Connections:     89        TCP Resets:            12              ║
║   UDP Packets:        456        HTTP Requests:         78              ║
║   FTP Connections:     12        SNMP Queries:         234              ║
║                                                                          ║
║ Device Statistics:                                                       ║
║   Router1:   ↓ 234 pkt  ↑ 189 pkt  Errors: 0                           ║
║   Switch1:   ↓ 567 pkt  ↑ 534 pkt  Errors: 2                           ║
║   Server1:   ↓ 123 pkt  ↑ 98 pkt   Errors: 0                           ║
║                                                                          ║
║ Error Injection:                                                         ║
║   Active Errors: 5                                                       ║
║   FCS Errors:    12.5%  (Router1:eth0)                                  ║
║   Discards:      5.0%   (Switch1:eth1)                                  ║
║                                                                          ║
║ Performance:                                                             ║
║   CPU Usage:     12.3%          Memory:     45.2 MB                     ║
║   Packets/sec:   234 ↓ 189 ↑    Throughput: 12.3 Mbps                  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

#### E. Debug Level Settings Menu

**New Menu: Debug Settings**
```
╔══════════════════════════════════════════════════════════════════════════╗
║ Debug Settings                                                           ║
╠══════════════════════════════════════════════════════════════════════════╣
║                                                                          ║
║  ► Debug Level: [2] Potential Problems                                  ║
║    • Show timestamps: [✓]                                               ║
║    • Color output: [✓]                                                  ║
║    • Log to file: [ ]                                                   ║
║                                                                          ║
║  Protocol Verbosity:                                                     ║
║    • ARP:  [2]    • TCP:  [1]                                          ║
║    • IP:   [1]    • UDP:  [1]                                          ║
║    • ICMP: [2]    • HTTP: [2]                                          ║
║    • SNMP: [3]    • FTP:  [2]                                          ║
║                                                                          ║
║  Statistics:                                                             ║
║    • Update interval: [1s]                                              ║
║    • Show packet hex dump: [ ]                                          ║
║                                                                          ║
║  [Enter] Apply  [Esc] Cancel                                            ║
╚══════════════════════════════════════════════════════════════════════════╝
```

---

### 4. DEBUG LEVEL INTEGRATION

#### Current Debug Levels
```go
0 - No debug output
1 - Status messages (default)
2 - Potential problems
3 - Full detail
```

#### Proposed Enhanced Debug Levels
```go
0 - QUIET:    Only critical errors
1 - NORMAL:   Status messages (device up/down, connections)
2 - VERBOSE:  Protocol details (ARP, ICMP, HTTP requests)
3 - DEBUG:    Full packet details (headers, payloads, hex dumps)
4 - TRACE:    Everything including internal state changes (NEW)
```

#### Debug Level Application by Component

| Component | Level 0 | Level 1 | Level 2 | Level 3 | Level 4 |
|-----------|---------|---------|---------|---------|---------|
| **Startup** | Errors only | Config summary | Full config | Device details | All init steps |
| **ARP** | - | Requests | Req + Reply | + MAC tables | + Packet hex |
| **ICMP** | - | Ping count | Each ping | + Payload | + Checksums |
| **TCP** | - | Connections | SYN/ACK | + Seq/Ack | + Options |
| **HTTP** | - | Request count | Method + URL | + Headers | + Body |
| **FTP** | - | Connections | Commands | + Responses | + Data |
| **SNMP** | - | Query count | OID requests | + Values | + ASN.1 |
| **Errors** | - | Error count | Error details | + Stack | + State |
| **Traffic Gen** | - | Patterns active | Each packet | + Dest | + Full packet |

#### Implementation in Code
```go
// Each protocol handler checks debug level
func (h *HTTPHandler) HandleRequest(...) {
    if debugLevel >= 1 {
        log.Printf("HTTP request from %s", srcIP)
    }
    if debugLevel >= 2 {
        log.Printf("HTTP %s %s from %s", method, path, srcIP)
    }
    if debugLevel >= 3 {
        log.Printf("HTTP Headers: %v", headers)
    }
    if debugLevel >= 4 {
        log.Printf("HTTP Body: %s", body)
        log.Printf("Packet hex dump: %x", packet)
    }
}
```

---

### 5. COLOR & OUTPUT ENHANCEMENTS

#### A. Color-Coded Output by Log Level

```
Level 0 (QUIET):   [RED]    Critical errors only
Level 1 (NORMAL):  [WHITE]  Status messages
Level 2 (VERBOSE): [CYAN]   Protocol details
Level 3 (DEBUG):   [YELLOW] Packet details
Level 4 (TRACE):   [GRAY]   Internal state
```

#### B. Emoji/Icon Support (Optional)

```
✓ Success messages (green)
✗ Error messages (red)
⚠ Warning messages (yellow)
ℹ Info messages (blue)
→ Packet sent (cyan)
← Packet received (magenta)
⚙ System event (gray)
```

#### C. Progress Indicators

```
[=====>    ] Loading configuration... 50%
[🔄] Starting protocol stack...
[✓] Protocol stack ready
[⚡] Processing packets... 1234/sec
```

---

### 6. ADDITIONAL CLI MODES

#### A. Dry-Run Mode
```bash
$ niac --dry-run en0 network.cfg

✓ Configuration file: network.cfg
✓ Interface: en0 (exists, up)
✓ Devices: 5
  - Router1 (192.168.1.1, 00:11:22:33:44:55)
  - Switch1 (192.168.1.2, 00:11:22:33:44:56)
  - Server1 (192.168.1.10, 00:11:22:33:44:57)
  - AP1 (192.168.1.20, 00:11:22:33:44:58)
  - Workstation1 (192.168.1.30, 00:11:22:33:44:59)
✓ SNMP agents: 5
✓ Walk files: 3 loaded
✗ WARNING: Device Router1 has no walk file

Configuration is valid and ready to run.
```

#### B. List Devices Mode
```bash
$ niac --list-devices network.cfg

Devices in network.cfg:
┌────────────────┬───────────────┬───────────────────┬────────┬───────┐
│ Name           │ IP Address    │ MAC Address       │ Type   │ SNMP  │
├────────────────┼───────────────┼───────────────────┼────────┼───────┤
│ Router1        │ 192.168.1.1   │ 00:11:22:33:44:55 │ router │ Yes   │
│ Switch1        │ 192.168.1.2   │ 00:11:22:33:44:56 │ switch │ Yes   │
│ Server1        │ 192.168.1.10  │ 00:11:22:33:44:57 │ server │ Yes   │
│ AP1            │ 192.168.1.20  │ 00:11:22:33:44:58 │ ap     │ Yes   │
│ Workstation1   │ 192.168.1.30  │ 00:11:22:33:44:59 │ host   │ No    │
└────────────────┴───────────────┴───────────────────┴────────┴───────┘

Total: 5 devices, 4 with SNMP agents
```

#### C. Configuration Generator
```bash
$ niac --generate-config --devices 10 --network 192.168.1.0/24 > network.cfg

Generated configuration with:
  - 2 routers
  - 2 switches
  - 3 servers
  - 3 workstations
  - Network: 192.168.1.0/24
  - SNMP community: public
```

---

### 7. HELP SYSTEM IMPROVEMENTS

#### A. Contextual Help in Interactive Mode

```
Press [h] for help:

╔══════════════════════════════════════════════════════════════════════════╗
║ NIAC-Go Keyboard Shortcuts                               [h] to close    ║
╠══════════════════════════════════════════════════════════════════════════╣
║                                                                          ║
║  Navigation & Control:                                                   ║
║    [q]        Quit (with confirmation)                                   ║
║    [h] [?]    Show this help                                            ║
║    [Esc]      Close current menu/dialog                                 ║
║                                                                          ║
║  Error Injection:                                                        ║
║    [i]        Open error injection menu                                  ║
║    [c]        Clear all errors                                           ║
║                                                                          ║
║  Debugging:                                                              ║
║    [d]        Cycle debug level (0→1→2→3→0)                            ║
║    [D]        Open debug settings                                        ║
║    [l]        View debug log (last 100 lines)                           ║
║                                                                          ║
║  Statistics & Info:                                                      ║
║    [s]        View detailed statistics                                   ║
║    [v]        View device information                                    ║
║    [p]        Pause/Resume packet processing                             ║
║                                                                          ║
║  Current State:                                                          ║
║    Debug Level: 2 (Verbose)                                             ║
║    Packet Processing: Running                                            ║
║    Active Errors: 5                                                      ║
║                                                                          ║
╚══════════════════════════════════════════════════════════════════════════╝
```

#### B. Man Page Style Help
```bash
$ niac --help

NAME
    niac - Network In A Can - Network device simulator

SYNOPSIS
    niac [OPTIONS] <interface> <config_file>
    niac --list-interfaces
    niac --version

DESCRIPTION
    NIAC simulates network devices and their behavior on a network interface.
    It can respond to ARP, ICMP, TCP, UDP, HTTP, FTP, SNMP, and other protocols.

    Devices are configured in a configuration file that specifies their IP
    addresses, MAC addresses, and behavior. SNMP agents can be configured with
    walk files for realistic responses.

OPTIONS
    -d, --debug <level>
        Set debug verbosity level (0-4)
        0 = Quiet (errors only)
        1 = Normal (status messages) [default]
        2 = Verbose (protocol details)
        3 = Debug (packet details)
        4 = Trace (internal state)

    -v, --verbose
        Enable verbose output (equivalent to --debug 3)

    -q, --quiet
        Suppress output (equivalent to --debug 0)

    ... (full man page)

EXAMPLES
    Start in interactive mode with verbose debugging:
        $ sudo niac --interactive --verbose en0 network.cfg

    Validate configuration without starting:
        $ niac --dry-run en0 network.cfg

    Run quietly with log file:
        $ sudo niac --quiet --log-file niac.log en0 network.cfg

SEE ALSO
    tcpdump(8), wireshark(1), snmpwalk(1)

AUTHOR
    Original Java version by Kevin Kayes (2002-2015)
    Go rewrite by Kris Armstrong (2025)
```

---

## Implementation Priority

### Phase 1: Critical UX Improvements (Do First)
1. ✅ Rename project folder (Java: network_in_a_can → niac-java)
2. ✅ Enhanced CLI flags (--version, --list-interfaces, --dry-run)
3. ✅ Debug level display in interactive status bar
4. ✅ Keyboard shortcut [d] to cycle debug level
5. ✅ Color-coded output by level

### Phase 2: Enhanced Features
6. ✅ Debug log viewer ([l] key)
7. ✅ Statistics viewer ([s] key)
8. ✅ Help overlay ([h] key)
9. ✅ Debug settings menu ([D] key)
10. ✅ Per-protocol debug levels

### Phase 3: Advanced Features
11. ⬜ Config generator (--generate-config)
12. ⬜ Log file output (--log-file)
13. ⬜ Packet hex dump viewer
14. ⬜ Network traffic graphs
15. ⬜ Export statistics to JSON/CSV

---

## Success Metrics

After implementation, the CLI should:
- ✅ Be discoverable (users can figure it out without docs)
- ✅ Be intuitive (common patterns like -v, --version work)
- ✅ Provide feedback (debug level visible, progress indicators)
- ✅ Support power users (keyboard shortcuts, advanced options)
- ✅ Be consistent (naming, flags, output format)

---

**Next Step**: Implement Phase 1 improvements immediately!
