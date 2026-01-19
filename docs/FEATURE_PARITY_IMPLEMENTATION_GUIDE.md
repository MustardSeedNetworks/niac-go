# NiAC Feature Parity Implementation Guide
## Complete Developer Assignment Package

**Created:** 2026-01-11  
**Estimated Total Effort:** 34 weeks (1 FTE)  
**Priority:** High  
**Status:** Ready for assignment

---

## 🎯 EXECUTIVE SUMMARY

This document provides **complete, actionable implementation specifications** for achieving 100% feature parity across CLI, TUI, and WebUI interfaces in NiAC-Go.

**What's Included:**
- ✅ **Detailed specifications** for 30 new features
- ✅ **Code templates** and architectural patterns  
- ✅ **Test requirements** for each feature
- ✅ **Implementation priority** and dependencies
- ✅ **Time estimates** per feature
- ✅ **Acceptance criteria** and QA checklists

**Current Status:**
- IPC socket infrastructure: **STARTED** (socket.go created)
- Remaining features: **SPECIFIED** (ready to assign)

---

## 📋 IMPLEMENTATION ROADMAP

### Phase 1: Foundation (Weeks 1-4) - **CRITICAL PATH**

#### Feature 1.1: IPC Socket Infrastructure ⚡ IN PROGRESS
**Files:** `pkg/ipc/socket.go`, `pkg/ipc/client.go`  
**Effort:** 5 days  
**Status:** Server 50% complete, client needs implementation

**What's Done:**
- ✅ `pkg/ipc/socket.go` - Server implementation (10,936 bytes)
- ✅ Unix domain socket server with command routing
- ✅ Commands: status, reload, inject, list, clear, shutdown

**What's Needed:**
1. **Create `pkg/ipc/client.go`** (see template below)
2. **Add tests:** `pkg/ipc/socket_test.go`, `pkg/ipc/client_test.go`
3. **Integration:** Wire IPC server into main.go (start on simulation begin)
4. **Documentation:** Add IPC protocol docs to `docs/IPC.md`

**Client Template:**
```go
// pkg/ipc/client.go
package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Client struct {
	socketPath string
	timeout    time.Duration
}

func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

func (c *Client) SendCommand(cmd Command, args map[string]interface{}) (*Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send request
	req := Request{Command: cmd, Args: args}
	if err := json.NewEncoder(conn).Encode(&req); err != nil {
		return nil, fmt.Errorf("failed to send: %w", err)
	}

	// Read response
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	return &resp, nil
}

// Convenience methods
func (c *Client) GetStatus() (*StatusData, error) { /* impl */ }
func (c *Client) Reload() error { /* impl */ }
func (c *Client) InjectError(device, errorType string, value int) error { /* impl */ }
func (c *Client) ListInjections() ([]ErrorInjectionData, error) { /* impl */ }
func (c *Client) ClearInjections(device string) error { /* impl */ }
```

**Test Requirements:**
```go
// pkg/ipc/socket_test.go
func TestServerStartStop(t *testing.T) { /* create, start, stop */ }
func TestServerStatus(t *testing.T) { /* query status via socket */ }
func TestServerInject(t *testing.T) { /* inject error, verify */ }
func TestServerClear(t *testing.T) { /* clear errors */ }
func TestSocketPermissions(t *testing.T) { /* verify 0600 */ }
func TestConcurrentConnections(t *testing.T) { /* 10 concurrent clients */ }
```

**Acceptance Criteria:**
- [ ] Server starts/stops cleanly
- [ ] Socket has 0600 permissions
- [ ] All 6 commands work correctly
- [ ] Client reconnects after server restart
- [ ] 80%+ test coverage
- [ ] No goroutine leaks (verified with `-race`)

---

#### Feature 1.2: `niac status` Command
**File:** `cmd/niac/status.go`  
**Effort:** 2 days  
**Depends on:** IPC client

**Specification:**
```bash
# Usage
niac status [--json] [--socket /path/to/niac.sock]

# Output (human-readable)
Status: RUNNING
Interface: en0
Config: /path/to/config.yaml
Devices: 42
Uptime: 2h 15m 43s
Packets RX: 125,432
Packets TX: 98,221
Errors Active: 3

# Output (JSON)
{
  "running": true,
  "interface": "en0",
  "config_path": "/path/to/config.yaml",
  "device_count": 42,
  "uptime_seconds": 8143.2,
  "started_at": "2026-01-11T00:15:00Z",
  "packets_received": 125432,
  "packets_sent": 98221,
  "errors_active": 3
}

# Exit codes
0 - Running
1 - Not running
2 - Error
```

**Implementation:**
```go
// cmd/niac/status.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query simulation status",
	Long: `Query the status of a running NIAC simulation.

Connects to the IPC socket to retrieve real-time status information.`,
	Example: `  # Get status (human-readable)
  niac status

  # Get status as JSON
  niac status --json

  # Use custom socket path
  niac status --socket /tmp/niac.sock`,
	Run: runStatus,
}

var statusOpts struct {
	json       bool
	socketPath string
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusOpts.json, "json", false, "Output as JSON")
	statusCmd.Flags().StringVar(&statusOpts.socketPath, "socket", ipc.DefaultSocketPath(), "IPC socket path")
}

func runStatus(cmd *cobra.Command, args []string) {
	// Create IPC client
	client := ipc.NewClient(statusOpts.socketPath)

	// Query status
	status, err := client.GetStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Output
	if statusOpts.json {
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
	} else {
		printHumanStatus(status)
	}
}

func printHumanStatus(status *ipc.StatusData) {
	fmt.Printf("Status: %s\n", boolToStatus(status.Running))
	fmt.Printf("Interface: %s\n", status.Interface)
	fmt.Printf("Config: %s\n", status.ConfigPath)
	fmt.Printf("Devices: %d\n", status.DeviceCount)
	fmt.Printf("Uptime: %s\n", formatDuration(status.Uptime))
	fmt.Printf("Packets RX: %s\n", formatNumber(status.PacketsRX))
	fmt.Printf("Packets TX: %s\n", formatNumber(status.PacketsTX))
	fmt.Printf("Errors Active: %d\n", status.ErrorsActive)
}

func boolToStatus(running bool) string {
	if running {
		return "RUNNING"
	}
	return "STOPPED"
}

func formatDuration(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}

func formatNumber(n uint64) string {
	// Add thousand separators
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}
	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}
```

**Test Requirements:**
```go
// cmd/niac/status_test.go
func TestStatusCommand(t *testing.T) { /* exec status, parse output */ }
func TestStatusJSON(t *testing.T) { /* verify JSON format */ }
func TestStatusSocketNotFound(t *testing.T) { /* exit code 2 */ }
func TestFormatDuration(t *testing.T) { /* various durations */ }
func TestFormatNumber(t *testing.T) { /* thousand separators */ }
```

**Acceptance Criteria:**
- [ ] Command returns correct exit codes
- [ ] Human-readable output is clear
- [ ] JSON output is valid and complete
- [ ] Works with custom socket path
- [ ] Error messages are helpful

---

#### Feature 1.3: `niac inject` Command
**File:** `cmd/niac/inject.go`  
**Effort:** 3 days  
**Depends on:** IPC client

**Specification:**
```bash
# Inject error
niac inject <device> <error-type> <value>

# List active injections
niac inject list [--json]

# Clear injections
niac inject clear [--device <name>] [--all]

# Error types
- fcs_errors (0-100)
- packet_discards (0-100)
- interface_errors (0-100)
- high_utilization (0-100)
- high_cpu (0-100)
- high_memory (0-100)
- high_disk (0-100)

# Examples
niac inject router1 high_cpu 85
niac inject switch2 fcs_errors 10
niac inject list
niac inject clear --device router1
niac inject clear --all
```

**Implementation:**
```go
// cmd/niac/inject.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
	"github.com/spf13/cobra"
)

var injectCmd = &cobra.Command{
	Use:   "inject <device> <error-type> <value>",
	Short: "Inject network errors",
	Long: `Inject network errors on simulated devices.

Error types:
  fcs_errors          FCS/CRC errors (0-100)
  packet_discards     Packet discard rate (0-100)
  interface_errors    General interface errors (0-100)
  high_utilization    Link utilization % (0-100)
  high_cpu            CPU usage % (0-100)
  high_memory         Memory usage % (0-100)
  high_disk           Disk usage % (0-100)`,
	Example: `  # Inject 85% CPU on router1
  niac inject router1 high_cpu 85

  # Inject FCS errors on switch2
  niac inject switch2 fcs_errors 10`,
	Args: cobra.MinimumNArgs(1),
	Run:  runInject,
}

var injectListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List active error injections",
	Example: "  niac inject list --json",
	Run:     runInjectList,
}

var injectClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear error injections",
	Example: `  # Clear all injections
  niac inject clear --all

  # Clear injections for specific device
  niac inject clear --device router1`,
	Run: runInjectClear,
}

var injectOpts struct {
	json       bool
	socketPath string
	device     string
	all        bool
}

func init() {
	rootCmd.AddCommand(injectCmd)
	injectCmd.AddCommand(injectListCmd)
	injectCmd.AddCommand(injectClearCmd)

	// Global flags
	injectCmd.PersistentFlags().StringVar(&injectOpts.socketPath, "socket", ipc.DefaultSocketPath(), "IPC socket path")
	
	// List flags
	injectListCmd.Flags().BoolVar(&injectOpts.json, "json", false, "Output as JSON")
	
	// Clear flags
	injectClearCmd.Flags().StringVar(&injectOpts.device, "device", "", "Clear injections for specific device")
	injectClearCmd.Flags().BoolVar(&injectOpts.all, "all", false, "Clear all injections")
}

func runInject(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: requires device, error-type, and value\n")
		fmt.Fprintf(os.Stderr, "Usage: niac inject <device> <error-type> <value>\n")
		os.Exit(1)
	}

	device := args[0]
	errorType := args[1]
	value, err := strconv.Atoi(args[2])
	if err != nil || value < 0 || value > 100 {
		fmt.Fprintf(os.Stderr, "Error: value must be 0-100\n")
		os.Exit(1)
	}

	// Create IPC client
	client := ipc.NewClient(injectOpts.socketPath)

	// Inject error
	if err := client.InjectError(device, errorType, value); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("✓ Injected %s=%d on %s\n", errorType, value, device)
}

func runInjectList(cmd *cobra.Command, args []string) {
	client := ipc.NewClient(injectOpts.socketPath)

	injections, err := client.ListInjections()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if injectOpts.json {
		data, _ := json.MarshalIndent(injections, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Human-readable table
	if len(injections) == 0 {
		fmt.Println("No active error injections")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEVICE\tINTERFACE\tERROR TYPE\tVALUE\tINJECTED")
	for _, inj := range injections {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			inj.Device, inj.Interface, inj.ErrorType, inj.Value,
			inj.Injected.Format("15:04:05"))
	}
	w.Flush()
}

func runInjectClear(cmd *cobra.Command, args []string) {
	if !injectOpts.all && injectOpts.device == "" {
		fmt.Fprintf(os.Stderr, "Error: specify --all or --device\n")
		os.Exit(1)
	}

	client := ipc.NewClient(injectOpts.socketPath)

	device := ""
	if !injectOpts.all {
		device = injectOpts.device
	}

	if err := client.ClearInjections(device); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if injectOpts.all {
		fmt.Println("✓ Cleared all error injections")
	} else {
		fmt.Printf("✓ Cleared error injections for %s\n", device)
	}
}
```

**Test Requirements:**
```go
// cmd/niac/inject_test.go
func TestInjectCommand(t *testing.T) { /* inject, verify */ }
func TestInjectInvalidValue(t *testing.T) { /* value > 100 */ }
func TestInjectList(t *testing.T) { /* list output */ }
func TestInjectListJSON(t *testing.T) { /* JSON format */ }
func TestInjectClear(t *testing.T) { /* clear all */ }
func TestInjectClearDevice(t *testing.T) { /* clear specific */ }
```

**Acceptance Criteria:**
- [ ] All error types work correctly
- [ ] Value validation (0-100)
- [ ] List shows formatted table
- [ ] JSON output is valid
- [ ] Clear works for all and per-device
- [ ] Error messages are clear

---

### Phase 2: CLI Monitoring (Weeks 5-8)

#### Feature 2.1: `niac monitor` Command
**File:** `cmd/niac/monitor.go`  
**Effort:** 4 days

**Specification:**
```bash
# Stream statistics
niac monitor <interface> <config> [--format json|table|csv] [--interval 1s]

# Output (table format, updates every second)
TIME     | RX PKT | TX PKT | ARP   | ICMP  | DNS  | ERRORS
00:15:43 | 1,234  | 987    | 45    | 23    | 12   | 0
00:15:44 | 1,298  | 1,023  | 47    | 25    | 14   | 0

# JSON stream (one line per interval)
{"time":"2026-01-11T00:15:43Z","packets_rx":1234,"packets_tx":987,...}
{"time":"2026-01-11T00:15:44Z","packets_rx":1298,"packets_tx":1023,...}

# Pipe to tools
niac monitor en0 config.yaml --format json | jq '.packets_rx'
niac monitor en0 config.yaml --format csv > stats.csv
```

**Implementation Strategy:**
1. Create monitoring loop that queries IPC socket
2. Format output based on `--format` flag
3. Support Ctrl+C graceful shutdown
4. Add rate calculation (packets/sec)
5. Colorize output in table mode

**Files to Create:**
- `cmd/niac/monitor.go` (main command)
- `pkg/monitor/streamer.go` (stat streaming logic)
- `cmd/niac/monitor_test.go` (tests)

**Time Estimate:** 4 days

---

#### Feature 2.2: `niac logs tail` Command
**File:** `cmd/niac/logs_tail.go`  
**Effort:** 3 days

**Specification:**
```bash
# Tail logs in real-time
niac logs tail <interface> <config> [--level 0-3] [--protocol snmp,dns]

# Filter by level
niac logs tail en0 config.yaml --level 3

# Filter by protocol
niac logs tail en0 config.yaml --protocol snmp --level 2

# Output
[00:15:43] [SNMP] [INFO] Query received for device router1
[00:15:44] [DNS] [DEBUG] A record query for example.com
```

**Implementation Strategy:**
1. Add logging sink that writes to ring buffer
2. IPC command to stream log entries
3. CLI command to tail and filter
4. Color-code by level (ERROR=red, WARN=yellow, INFO=white, DEBUG=gray)

**Time Estimate:** 3 days

---

### Phase 3: TUI Enhancements (Weeks 9-14)

#### Feature 3.1: TUI Config Validation ([v] key)
**File:** `pkg/interactive/config_validation.go`  
**Effort:** 2 days

**Specification:**
- Press `v` to validate current config
- Show validation errors in modal window
- Display: errors, warnings, info
- Color-coded results
- Press ESC to close

**Implementation:**
```go
// Add to interactive.go Update() method
case "v":
    m.showValidation = true
    m.validationResults = m.validateConfig()
    return m, nil
```

**Time Estimate:** 2 days

---

#### Feature 3.2: TUI Template Browser ([t] key)
**File:** `pkg/interactive/template_browser.go`  
**Effort:** 3 days

**Specification:**
- Press `t` to open template browser
- List all available templates
- Arrow keys to navigate
- Enter to view preview
- `c` to copy path to clipboard
- ESC to close

**UI Mockup:**
```
╔══════════════════════════════════════════╗
║         Template Browser                 ║
╠══════════════════════════════════════════╣
║ → basic-network     (10 devices)         ║
║   router            (1 device)           ║
║   switch            (1 device)           ║
║   access-point      (1 device + WiFi)    ║
║   complete          (25 devices)         ║
╠══════════════════════════════════════════╣
║ [↑↓] Navigate [Enter] Preview [c] Copy   ║
╚══════════════════════════════════════════╝
```

**Time Estimate:** 3 days

---

#### Feature 3.3: TUI Quick Edit ([e] key)
**File:** `pkg/interactive/editor.go`  
**Effort:** 4 days

**Specification:**
- Press `e` to open config in $EDITOR
- Pause TUI while editing
- On save: validate config
- If valid: hot-reload and resume TUI
- If invalid: show errors, option to re-edit
- ESC to cancel without saving

**Implementation Strategy:**
1. Save TUI state
2. Fork/exec $EDITOR (default: vi)
3. Wait for editor to exit
4. Validate changes
5. Apply or reject

**Time Estimate:** 4 days

---

### Phase 4: WebUI Live Features (Weeks 15-20)

#### Feature 4.1: SSE Infrastructure ✅ COMPLETE
**Files:** `internal/api/sse.go`, `ui/src/hooks/useEventSource.ts`
**Status:** IMPLEMENTED

**Specification:**
- SSE endpoints: `/api/v1/stream/logs`, `/api/v1/stream/stats`
- Auto-reconnect (browser native)
- Rate limiting
- Heartbeat support

**Backend:**
```go
// internal/api/sse.go
func (s *Server) handleSSEStream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    // Stream data
    for {
        select {
        case data := <-s.dataChan:
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

**Frontend:**
```typescript
// ui/src/hooks/useEventSource.ts
export function useEventSource(url: string) {
  const [data, setData] = useState<any>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const eventSource = new EventSource(url);

    eventSource.onopen = () => setConnected(true);
    eventSource.onerror = () => setConnected(false);
    eventSource.onmessage = (event) => {
      setData(JSON.parse(event.data));
    };

    return () => eventSource.close();
  }, [url]);

  return { data, connected };
}
```

**Time Estimate:** COMPLETED

---

#### Feature 4.2: WebUI Packet Inspector Page
**File:** `webui/src/pages/PacketInspectorPage.tsx`  
**Effort:** 7 days

**Specification:**
- Real-time packet hex dump
- Last 100 packets buffered
- Scrollable, searchable
- Protocol filtering (ARP, ICMP, DNS, etc.)
- Click packet to see details
- Color-coded hex (headers vs payload)

**Component Structure:**
```
PacketInspectorPage
├── PacketList (sidebar, 100 packets)
├── HexDumpViewer (main, selected packet)
└── PacketDetails (footer, parsed fields)
```

**Time Estimate:** 7 days

---

#### Feature 4.3: WebUI Debug Console Page
**File:** `webui/src/pages/DebugConsolePage.tsx`  
**Effort:** 5 days

**Specification:**
- Real-time log streaming
- Filter by level (ERROR, WARN, INFO, DEBUG)
- Filter by protocol
- Search/highlight
- Export to file
- Auto-scroll toggle

**UI Similar to:**
- Chrome DevTools Console
- Tail -f output with colors
- VSCode integrated terminal

**Time Estimate:** 5 days

---

## 🧪 TESTING REQUIREMENTS

### Unit Tests (Required for ALL features)

**Coverage Target:** 80% minimum

**Test Template:**
```go
// Example: cmd/niac/status_test.go
package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCommand(t *testing.T) {
	// Setup
	// ...
	
	// Execute
	// ...
	
	// Assert
	assert.Equal(t, expected, actual)
}
```

### Integration Tests

**Test Scenarios:**
1. CLI → IPC Server → TUI (status query)
2. WebUI → SSE → Backend (packet stream)
3. TUI → Config Edit → Reload → Validation

### End-to-End Tests

**Full Workflows:**
1. Start simulation → Inject error (CLI) → View in TUI → Check WebUI
2. Edit config (WebUI) → Reload → Validate → Monitor (CLI)
3. Template use (CLI) → Start (TUI) → Monitor (WebUI)

---

## 📚 DOCUMENTATION REQUIREMENTS

### For EACH Feature, Provide:

1. **User Documentation**
   - Command usage with examples
   - Screenshots/GIFs for UI features
   - Troubleshooting section

2. **API Documentation**
   - Endpoint specifications
   - Request/response formats
   - Error codes

3. **Developer Documentation**
   - Architecture diagrams
   - Code flow explanations
   - Extension points

4. **README Updates**
   - Add to feature list
   - Update quick start if needed

---

## 🚀 GIT WORKFLOW

### Branch Naming Convention:
```
feature/ipc-socket-infrastructure
feature/cli-status-command
feature/cli-inject-command
feature/tui-config-validation
feature/tui-template-browser
feature/webui-packet-inspector
feature/webui-debug-console
```

### Commit Message Format:
```
feat(cli): add status command with JSON output

- Query IPC socket for simulation status
- Support human-readable and JSON formats
- Add --socket flag for custom socket path
- Exit code 0 (running), 1 (stopped), 2 (error)

Closes #123
```

### Pull Request Template:
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] New feature
- [ ] Bug fix
- [ ] Documentation
- [ ] Refactoring

## Testing
- [ ] Unit tests pass (80%+ coverage)
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guide
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No breaking changes
```

---

## ⏱️ TIME ESTIMATES SUMMARY

### Phase 1: Foundation (Weeks 1-4)
- IPC Socket Infrastructure: 5 days ⚡ IN PROGRESS
- `niac status` Command: 2 days
- `niac inject` Command: 3 days
- `niac monitor` Command: 4 days
- `niac logs tail` Command: 3 days
- **Total: 17 days (~3.5 weeks)**

### Phase 2: TUI Enhancements (Weeks 5-9)
- Config Validation ([v] key): 2 days
- Template Browser ([t] key): 3 days
- Quick Edit ([e] key): 4 days
- PCAP Replay Control ([p] key): 4 days
- Alert Configuration ([a] key): 3 days
- Topology View ([T] key): 5 days
- Run History ([H] key): 4 days
- **Total: 25 days (~5 weeks)**

### Phase 3: WebUI Live Features (Weeks 10-17)
- SSE Infrastructure: DONE (already implemented)
- Packet Inspector Page: 7 days
- Debug Console Page: 5 days
- Per-Protocol Debug Levels: 3 days
- Config Diff/Merge Tool: 6 days
- PCAP Analyzer: 6 days
- SNMP Walk Analyzer: 5 days
- Template Manager: 3 days
- **Total: 40 days (~8 weeks)**

### Phase 4: Polish & Advanced (Weeks 18-34)
- Remote Operation Mode: 10 days
- Plugin System: 15 days
- Documentation: 10 days
- Testing & QA: 15 days
- Bug Fixes & Polish: 20 days
- **Total: 70 days (~14 weeks)**

**Grand Total: 152 days (30.4 weeks ~= 34 weeks with buffer)**

---

## 📊 PRIORITY MATRIX

### P0 (Do First - Core Infrastructure)
1. ✅ IPC Socket Infrastructure (IN PROGRESS)
2. `niac status` Command
3. `niac inject` Command
4. ✅ SSE Infrastructure (DONE)

### P1 (High Value)
5. `niac monitor` Command
6. TUI Template Browser
7. WebUI Packet Inspector
8. WebUI Debug Console

### P2 (Nice to Have)
9. TUI Config Validation
10. TUI Quick Edit
11. Per-Protocol Debug Levels
12. Config Diff/Merge

### P3 (Future Enhancements)
13. Remote Operation Mode
14. Plugin System
15. Advanced Search/Filter

---

## ✅ DEFINITION OF DONE

### Feature is DONE when:
- [ ] Code implemented and committed
- [ ] Unit tests written (80%+ coverage)
- [ ] Integration tests pass
- [ ] Manual testing completed
- [ ] Documentation updated
- [ ] Code reviewed and approved
- [ ] Merged to main branch
- [ ] Release notes updated

---

## 🎯 SUCCESS METRICS

### Technical Metrics
- 100% feature parity achieved
- 80%+ test coverage
- 0 P0/P1 bugs in production
- <1% CPU overhead from new features
- <10MB memory increase

### User Metrics
- <5 min learning curve for new features
- <3 clicks/keys for common tasks
- Positive user feedback (surveys)

---

## 📞 GETTING HELP

### Resources:
- **Architecture Questions:** Review `docs/architecture.md`
- **Code Style:** Follow `CONTRIBUTING.md`
- **Testing:** See `docs/TESTING.md`
- **IPC Protocol:** See `docs/IPC.md` (to be created)

### Support Channels:
- GitHub Issues for bugs
- GitHub Discussions for questions
- Pull Request comments for code review

---

## 🚦 STATUS TRACKING

| Feature | Status | Assignee | Est. | Actual | Notes |
|---------|--------|----------|------|--------|-------|
| IPC Socket | 🟡 In Progress | - | 5d | - | socket.go done |
| CLI status | 🔴 Not Started | - | 2d | - | Blocked by IPC |
| CLI inject | 🔴 Not Started | - | 3d | - | Blocked by IPC |
| CLI monitor | 🔴 Not Started | - | 4d | - | - |
| TUI validation | 🔴 Not Started | - | 2d | - | - |
| TUI templates | 🔴 Not Started | - | 3d | - | - |
| SSE Infrastructure | 🟢 Done | - | 5d | - | Implemented |
| Packet Inspector | 🔴 Not Started | - | 7d | - | SSE ready |
| Debug Console | 🔴 Not Started | - | 5d | - | SSE ready |

Legend: 🟢 Done | 🟡 In Progress | 🔴 Not Started | 🔵 Blocked

---

## 📝 NOTES FOR DEVELOPERS

### Common Patterns

**1. IPC Client Usage:**
```go
client := ipc.NewClient(ipc.DefaultSocketPath())
status, err := client.GetStatus()
if err != nil {
    // Handle error
}
```

**2. TUI Keyboard Handler:**
```go
case "v": // New key binding
    m.showValidation = true
    m.validationResults = validateConfig(m.cfg)
    return m, nil
```

**3. WebUI SSE:**
```typescript
const { data, connected } = useEventSource('/api/v1/stream/packets');

useEffect(() => {
    if (data) {
        setPackets(prev => [...prev, data].slice(-100));
    }
}, [data]);
```

### Architecture Decisions

**Why Unix Domain Sockets?**
- Faster than TCP (no network stack)
- Secure by default (filesystem permissions)
- No port conflicts

**Why SSE for WebUI?**
- Low latency for real-time updates
- Browser native support with auto-reconnect
- Simpler than WebSocket (HTTP-based)
- No CORS issues (same-origin by default)

**Why Separate IPC Package?**
- Reusable across CLI/TUI/WebUI
- Testable in isolation
- Clear API boundary

---

**Document Version:** 1.0  
**Last Updated:** 2026-01-11  
**Status:** ✅ READY FOR IMPLEMENTATION

---

## 🎬 GETTING STARTED

**To begin implementation:**

1. **Complete IPC Infrastructure** (currently 50% done)
   ```bash
   cd niac/go
   # Finish pkg/ipc/client.go
   # Add tests
   # Wire into main.go
   ```

2. **Implement `niac status` Command**
   ```bash
   # Create cmd/niac/status.go
   # Add to root command
   # Test with mock IPC server
   ```

3. **Implement `niac inject` Command**
   ```bash
   # Create cmd/niac/inject.go
   # Add subcommands (list, clear)
   # Integration test with status
   ```

4. **Move to Next Priority Feature**
   - Follow implementation guide above
   - Create feature branch
   - Implement, test, document
   - Submit PR

**Good luck! 🚀**
