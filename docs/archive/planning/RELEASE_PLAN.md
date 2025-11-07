# NIAC-Go Release Plan
**Based on Comprehensive Review** - January 7, 2025

---

## v1.13.1 - Critical Patch (Immediate)

**Timeline**: 1-2 days
**Focus**: Security and correctness fixes

### Issues to Address

#### #1 - Fix Version Number Inconsistency
**Priority**: P0 - Critical
**File**: `cmd/niac/main.go:22`
**Problem**: Version hardcoded as "1.9.0" instead of "1.13.0"
**Solution**:
```go
// Remove constants from main.go entirely
// Use build-time linker flags
```
**Build command**:
```bash
go build -ldflags "-X github.com/krisarmstrong/niac-go/cmd/niac.version=v1.13.1" -o niac ./cmd/niac
```

#### #2 - Add Walk File Path Validation (Security)
**Priority**: P0 - Critical (Security)
**File**: `pkg/config/config.go:627`
**Problem**: Path traversal vulnerability - no validation on SNMP walk file paths
**Solution**:
```go
// Add validation function
func validateWalkFilePath(basePath, walkFile string) (string, error) {
    cleanPath := filepath.Clean(walkFile)

    // Prevent directory traversal
    if strings.Contains(cleanPath, "..") {
        return "", fmt.Errorf("walk file path contains invalid characters: %s", walkFile)
    }

    // Build full path
    var fullPath string
    if filepath.IsAbs(cleanPath) {
        fullPath = cleanPath
    } else if basePath != "" {
        fullPath = filepath.Join(basePath, cleanPath)
    } else {
        fullPath = cleanPath
    }

    // Verify file exists
    if _, err := os.Stat(fullPath); err != nil {
        return "", fmt.Errorf("walk file not found: %s", fullPath)
    }

    return fullPath, nil
}
```

#### #3 - Fix RateLimiter Goroutine Leak
**Priority**: P1 - High
**File**: `pkg/capture/capture.go:206`
**Problem**: Goroutine not cleaned up properly
**Solution**:
```go
type RateLimiter struct {
    packetsPerSecond int
    ticker           *time.Ticker
    tokens           chan struct{}
    done             chan struct{}  // Add done channel
}

func NewRateLimiter(packetsPerSecond int) *RateLimiter {
    rl := &RateLimiter{
        packetsPerSecond: packetsPerSecond,
        tokens:           make(chan struct{}, packetsPerSecond),
        done:             make(chan struct{}),  // Initialize
    }

    // Fill initial tokens
    for i := 0; i < packetsPerSecond; i++ {
        rl.tokens <- struct{}{}
    }

    // Start refill goroutine with proper cleanup
    rl.ticker = time.NewTicker(time.Second / time.Duration(packetsPerSecond))
    go func() {
        for {
            select {
            case <-rl.ticker.C:
                select {
                case rl.tokens <- struct{}{}:
                default:
                }
            case <-rl.done:
                return  // Clean exit
            }
        }
    }()

    return rl
}

func (rl *RateLimiter) Stop() {
    rl.ticker.Stop()
    close(rl.done)  // Signal goroutine to exit
}
```

**Testing**: Add test to verify goroutine cleanup
```go
func TestRateLimiter_NoGoroutineLeak(t *testing.T) {
    before := runtime.NumGoroutine()

    rl := NewRateLimiter(100)
    time.Sleep(100 * time.Millisecond)
    rl.Stop()
    time.Sleep(100 * time.Millisecond)

    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("Goroutine leak: %d before, %d after", before, after)
    }
}
```

---

## v1.14.0 - Testing & Quality (1-2 weeks)

**Timeline**: 1-2 weeks
**Focus**: Test coverage, CI/CD, contributor docs

### Testing Infrastructure

#### #4 - Add Tests for cmd/niac (0% → 60%)
**Priority**: P0 - Critical
**Files**: Create `cmd/niac/*_test.go`
**Tests Required**:
```go
// cmd/niac/root_test.go
func TestRootCommand_NoArgs(t *testing.T)
func TestRootCommand_Help(t *testing.T)
func TestRootCommand_Version(t *testing.T)

// cmd/niac/validate_test.go
func TestValidateCommand_ValidConfig(t *testing.T)
func TestValidateCommand_InvalidConfig(t *testing.T)
func TestValidateCommand_MissingFile(t *testing.T)
func TestValidateCommand_JSONOutput(t *testing.T)

// cmd/niac/template_test.go
func TestTemplateList(t *testing.T)
func TestTemplateShow(t *testing.T)
func TestTemplateUse(t *testing.T)

// cmd/niac/config_test.go
func TestConfigExport(t *testing.T)
func TestConfigDiff(t *testing.T)
func TestConfigMerge(t *testing.T)
```

#### #5 - Add Tests for pkg/capture (0% → 70%)
**Priority**: P0 - Critical
**Files**: Create `pkg/capture/*_test.go`
**Tests Required**:
```go
// pkg/capture/capture_test.go
func TestEngine_New(t *testing.T)
func TestEngine_SendPacket(t *testing.T)
func TestEngine_SendEthernet(t *testing.T)
func TestEngine_SendARP(t *testing.T)
func TestEngine_SetFilter(t *testing.T)
func TestEngine_Close(t *testing.T)

// Use mock interfaces for pcap
type mockHandle struct {
    packets [][]byte
    written [][]byte
}
```

#### #6 - Add Tests for pkg/interactive (0% → 50%)
**Priority**: P1 - High
**Files**: Create `pkg/interactive/*_test.go`
**Tests Required**:
```go
// pkg/interactive/interactive_test.go
func TestModel_Init(t *testing.T)
func TestModel_Update_KeyPress(t *testing.T)
func TestModel_Update_Tick(t *testing.T)
func TestModel_HandleMenuSelection(t *testing.T)
func TestModel_InjectError(t *testing.T)
func TestModel_View(t *testing.T)
func TestFormatDuration(t *testing.T)
func TestGetDebugLevelName(t *testing.T)

// Use Bubble Tea testing utilities
```

#### #7 - Add Tests for pkg/logging (0% → 80%)
**Priority**: P1 - High
**Files**: Create `pkg/logging/*_test.go`
**Tests Required**:
```go
// pkg/logging/colors_test.go
func TestInitColors(t *testing.T)
func TestInitColors_NO_COLOR(t *testing.T)
func TestError(t *testing.T)
func TestWarning(t *testing.T)
func TestSuccess(t *testing.T)
func TestProtocol(t *testing.T)

// pkg/logging/debug_config_test.go
func TestDebugConfig_SetProtocolLevel(t *testing.T)
func TestDebugConfig_GetProtocolLevel(t *testing.T)
func TestDebugConfig_GetAllLevels(t *testing.T)
func TestDebugConfig_ThreadSafety(t *testing.T)
```

### CI/CD Setup

#### #8 - Create GitHub Actions Workflow
**Priority**: P0 - Critical
**File**: Create `.github/workflows/ci.yml`
```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21', '1.22']

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}

      - name: Install libpcap (Ubuntu)
        if: matrix.os == 'ubuntu-latest'
        run: sudo apt-get install -y libpcap-dev

      - name: Install libpcap (macOS)
        if: matrix.os == 'macos-latest'
        run: brew install libpcap

      - name: Run Tests
        run: go test -v -race -count=1 -coverprofile=coverage.out ./...

      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
          flags: ${{ matrix.os }}-go${{ matrix.go }}

  lint:
    name: Lint
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest

  build:
    name: Build
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build
        run: go build -v ./cmd/niac

      - name: Upload Artifact
        uses: actions/upload-artifact@v3
        with:
          name: niac-${{ matrix.os }}
          path: niac*
```

### Documentation

#### #9 - Create CONTRIBUTING.md
**Priority**: P0 - Critical
**File**: Create `/CONTRIBUTING.md`
**Sections**:
- Code of Conduct
- Development Setup
- Running Tests
- Code Style
- Commit Guidelines
- PR Process

#### #10 - Create docs/ARCHITECTURE.md
**Priority**: P1 - High
**File**: Create `/docs/ARCHITECTURE.md`
**Sections**:
- Package Structure
- Data Flow Diagrams
- Protocol Handler Architecture
- Error Injection Flow
- Configuration Loading
- Extension Points

---

## v1.15.0 - Refactoring & Enhancement (2-3 weeks)

**Timeline**: 2-3 weeks
**Focus**: Code quality, architectural improvements

### Code Refactoring

#### #11 - Refactor runLegacyMode() (290 lines → 3 functions)
**Priority**: P1 - High
**File**: `cmd/niac/main.go:34`
**Approach**:
```go
func runLegacyMode(osArgs []string) {
    flags := parseLegacyFlags(osArgs)
    cfg := loadAndValidateLegacyConfig(flags)
    executeLegacySimulation(flags, cfg)
}

func parseLegacyFlags(osArgs []string) *LegacyFlags { ... }
func loadAndValidateLegacyConfig(flags *LegacyFlags) (*config.Config, error) { ... }
func executeLegacySimulation(flags *LegacyFlags, cfg *config.Config) error { ... }
```

#### #12 - Refactor runNormalMode() (245 lines → 4 functions)
**Priority**: P1 - High
**File**: `cmd/niac/main.go:530`
**Approach**:
```go
func runNormalMode(interfaceName string, cfg *config.Config, debugConfig *logging.DebugConfig) error {
    engine := initializeCaptureEngine(interfaceName, debugConfig)
    stack := initializeProtocolStack(engine, cfg, debugConfig)
    configureServiceHandlers(stack, cfg, debugConfig)
    return runMainLoop(stack, debugConfig)
}
```

#### #13 - Refactor LoadYAML() (440 lines → multiple functions)
**Priority**: P2 - Medium
**File**: `pkg/config/config.go:520`
**Approach**:
```go
func LoadYAML(filename string) (*Config, error) {
    yamlConfig := loadYAMLFile(filename)
    cfg := createBaseConfig(yamlConfig)

    for _, yamlDevice := range yamlConfig.Devices {
        device := convertYAMLDevice(yamlDevice, cfg)
        cfg.Devices = append(cfg.Devices, device)
    }

    return cfg, validateFinalConfig(cfg)
}

func convertYAMLDevice(yamlDevice YAMLDevice, cfg *Config) Device {
    device := createBaseDevice(yamlDevice)
    device.DHCPConfig = parseDHCPConfig(yamlDevice.Dhcp, yamlDevice.Name)
    device.DNSConfig = parseDNSConfig(yamlDevice.Dns, yamlDevice.Name)
    device.LLDPConfig = parseLLDPConfig(yamlDevice.Lldp)
    // ... etc
    return device
}
```

### Feature Improvements

#### #14 - Add Multi-Device Error Injection to TUI
**Priority**: P2 - Medium
**File**: `pkg/interactive/interactive.go`
**Changes**:
```go
type model struct {
    // ... existing fields
    selectedDeviceIndex int  // NEW: Currently selected device
    deviceMenuVisible   bool // NEW: Device selection menu
}

// Add device selection menu
func (m *model) renderDeviceMenu() string { ... }

// Update error injection to use selected device
func (m *model) injectError(errorType errors.ErrorType, value int) {
    device := m.cfg.Devices[m.selectedDeviceIndex]  // Use selected, not first
    // ...
}
```

#### #15 - Add Configurable Error Values in TUI
**Priority**: P2 - Medium
**File**: `pkg/interactive/interactive.go`
**Changes**:
```go
// Replace hardcoded values with user input
type model struct {
    // ... existing fields
    errorValueInput string // NEW: User input for error value
    inputMode       bool   // NEW: Are we in input mode?
}

// Add numeric input handling
func (m *model) handleNumericInput(key string) { ... }
```

---

## v1.16.0 - Advanced Testing (3-4 weeks)

**Timeline**: 3-4 weeks
**Focus**: Integration tests, benchmarks, fuzz tests

### Integration Tests

#### #16 - Add End-to-End CLI Tests
**Priority**: P1 - High
**Directory**: Create `tests/integration/`
**Tests**:
```go
// tests/integration/cli_test.go
func TestCLI_ValidateWorkflow(t *testing.T)
func TestCLI_TemplateWorkflow(t *testing.T)
func TestCLI_ConfigToolsWorkflow(t *testing.T)

// tests/integration/simulation_test.go
func TestSimulation_MinimalConfig(t *testing.T)
func TestSimulation_MultiDevice(t *testing.T)
func TestSimulation_AllProtocols(t *testing.T)
```

#### #17 - Add Protocol Integration Tests
**Priority**: P1 - High
**Files**: Create `tests/integration/protocols_test.go`
**Tests**:
```go
func TestProtocols_ARPResponse(t *testing.T)
func TestProtocols_LLDPAdvertisement(t *testing.T)
func TestProtocols_SNMPGet(t *testing.T)
func TestProtocols_DHCPLease(t *testing.T)
```

### Performance Testing

#### #18 - Add Performance Benchmarks
**Priority**: P2 - Medium
**Files**: Create `*_bench_test.go` files
**Benchmarks**:
```go
// pkg/config/config_bench_test.go
func BenchmarkLoadYAML(b *testing.B)
func BenchmarkLoadYAML_LargeConfig(b *testing.B)

// pkg/protocols/arp_bench_test.go
func BenchmarkARP_HandleRequest(b *testing.B)

// pkg/protocols/snmp_bench_test.go
func BenchmarkSNMP_GetRequest(b *testing.B)

// pkg/capture/capture_bench_test.go
func BenchmarkEngine_SendPacket(b *testing.B)
```

### Fuzz Testing

#### #19 - Add Fuzz Tests for Protocol Parsers
**Priority**: P2 - Medium
**Files**: Create `*_fuzz_test.go` files
**Fuzz Tests**:
```go
// pkg/protocols/dhcpv6_fuzz_test.go
func FuzzDHCPv6_ParseMessage(f *testing.F)

// pkg/protocols/dns_fuzz_test.go
func FuzzDNS_ParseQuery(f *testing.F)

// pkg/config/config_fuzz_test.go
func FuzzConfig_LoadYAML(f *testing.F)
```

### SNMP Coverage Improvement

#### #20 - Improve SNMP Test Coverage (6.7% → 50%)
**Priority**: P1 - High
**Files**: Add to `pkg/snmp/*_test.go`
**Tests Required**:
```go
// pkg/snmp/agent_test.go
func TestAgent_HandleGetRequest(t *testing.T)
func TestAgent_HandleGetNextRequest(t *testing.T)
func TestAgent_HandleGetBulkRequest(t *testing.T)
func TestAgent_LoadWalkFile(t *testing.T)
func TestAgent_GenerateTrap_ColdStart(t *testing.T)
func TestAgent_GenerateTrap_LinkDown(t *testing.T)
func TestAgent_GenerateTrap_HighCPU(t *testing.T)
func TestAgent_ErrorInjection_Integration(t *testing.T)
```

---

## v2.0.0 - Web API & Architecture (2-3 months)

**Timeline**: 2-3 months
**Focus**: REST API, HTMX UI, architectural improvements

### Architectural Changes

#### #21 - Introduce Service Layer
**Priority**: P0 - Critical (for v2.0)
**Create**: `pkg/app/application.go`
**Approach**:
```go
// pkg/app/application.go
type Application struct {
    config        *config.Config
    captureEngine *capture.Engine
    protocolStack *protocols.Stack
    stateManager  *errors.StateManager
    server        *http.Server  // NEW for v2.0
}

func New(configFile string, iface string) (*Application, error) { ... }
func (app *Application) Start(ctx context.Context) error { ... }
func (app *Application) Stop() error { ... }
func (app *Application) GetStats() Stats { ... }
```

#### #22 - Add Structured Logging
**Priority**: P1 - High
**Replace**: `pkg/logging` with zerolog or zap
**Approach**:
```go
import "github.com/rs/zerolog/log"

log.Info().
    Str("device", deviceName).
    Str("protocol", "ARP").
    Msg("Sent ARP reply")
```

#### #23 - Add Prometheus Metrics
**Priority**: P1 - High
**Create**: `pkg/metrics/metrics.go`
**Metrics**:
```go
var (
    packetsReceived = prometheus.NewCounter(...)
    packetsSent     = prometheus.NewCounter(...)
    errorsInjected  = prometheus.NewCounterVec(...)
)
```

---

## Summary Timeline

| Version  | Timeline      | Focus                        | Issues     |
|----------|---------------|------------------------------|------------|
| v1.13.1  | 1-2 days      | Critical fixes (security)    | #1, #2, #3 |
| v1.14.0  | 1-2 weeks     | Testing & CI/CD              | #4-#10     |
| v1.15.0  | 2-3 weeks     | Refactoring & features       | #11-#15    |
| v1.16.0  | 3-4 weeks     | Advanced testing             | #16-#20    |
| v2.0.0   | 2-3 months    | Web API & architecture       | #21-#23    |

---

## Metrics & Goals

### Test Coverage Goals
- **v1.13.0 (Current)**: 38% overall
- **v1.14.0 Target**: 60% overall
- **v1.15.0 Target**: 70% overall
- **v1.16.0 Target**: 80% overall

### Package-Specific Coverage
| Package           | Current | v1.14.0 | v1.15.0 | v1.16.0 |
|-------------------|---------|---------|---------|---------|
| cmd/niac          | 0%      | 60%     | 70%     | 75%     |
| pkg/capture       | 0%      | 70%     | 80%     | 85%     |
| pkg/config        | 52%     | 65%     | 75%     | 80%     |
| pkg/device        | 24%     | 50%     | 60%     | 70%     |
| pkg/errors        | 95%     | 95%     | 95%     | 95%     |
| pkg/interactive   | 0%      | 50%     | 60%     | 70%     |
| pkg/logging       | 0%      | 80%     | 85%     | 90%     |
| pkg/protocols     | 45%     | 55%     | 65%     | 75%     |
| pkg/snmp          | 7%      | 50%     | 60%     | 70%     |

---

**Next Action**: Create GitHub issues for v1.13.1 (P0 items) and start implementation
