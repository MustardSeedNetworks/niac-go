# NIAC-Go v2.0.0 Code Review Report

**Date:** 2025-11-13
**Branch:** v2.0.0-webui
**Reviewers:** Automated Code Review (Daemon, API Server, WebUI)
**Status:** 🔴 **BLOCKING ISSUES FOUND - CANNOT COMPILE**

---

## Executive Summary

The v2.0.0 release adds critical daemon mode functionality and a full-featured webUI with CLI parity. However, **the code will not compile** due to several critical issues in the daemon implementation. Additionally, there are important thread safety and security issues that must be addressed.

**Overall Assessment:**
- ✅ **Architecture:** Excellent design, good separation of concerns
- ❌ **Implementation:** Critical compilation errors in daemon package
- ⚠️  **Thread Safety:** Race conditions in API server
- ⚠️  **Security:** Missing input validation and error handling
- ⚠️  **Accessibility:** WebUI needs significant accessibility improvements

**Recommendation:** **DO NOT RELEASE** until blocking issues are resolved. Estimated fix time: 4-8 hours.

---

## Critical Issues (Must Fix Before Release)

### 🔴 BLOCKING: Daemon Package Will Not Compile

**Files Affected:**
- `pkg/daemon/daemon.go`
- `cmd/niac/cmd_daemon.go`

#### Issue 1: Non-existent Function `capture.NewEngine()`
**Location:** `pkg/daemon/daemon.go:182`

```go
// BROKEN CODE
engine, err := capture.NewEngine(req.Interface, 0)
```

**Problem:** Function doesn't exist. Correct function is `capture.New()`.

**Fix:**
```go
engine, err := capture.New(req.Interface)
```

---

#### Issue 2: Non-existent Methods `engine.Start()` and `engine.Stop()`
**Locations:**
- `pkg/daemon/daemon.go:192` (Start)
- `pkg/daemon/daemon.go:200, 253` (Stop)

```go
// BROKEN CODE
if err := engine.Start(ctx); err != nil {
    cancel()
    return fmt.Errorf("start capture engine: %w", err)
}

// Later...
engine.Stop()
```

**Problem:** `capture.Engine` has no `Start()` or `Stop()` methods. Only has `Close()`.

**Fix:**
```go
// Remove the Start() call entirely - engine is ready after New()

// For cleanup, use Close() instead of Stop()
if sim.engine != nil {
    sim.engine.Close()
}
```

---

#### Issue 3: Wrong Type for `protocols.NewStack()` Parameter
**Location:** `pkg/daemon/daemon.go:188`

```go
// BROKEN CODE (type mismatch)
stack := protocols.NewStack(engine, cfg, 0)
```

**Problem:** `NewStack()` expects `*logging.DebugConfig` as third parameter, not `int`.

**Expected Signature:**
```go
func NewStack(captureEngine *capture.Engine, cfg *config.Config, debugConfig *logging.DebugConfig) *Stack
```

**Fix:**
```go
// Create proper debug config
debugCfg := &logging.DebugConfig{
    Level: 0,
    // ... other fields
}
stack := protocols.NewStack(engine, cfg, debugCfg)

// OR pass nil if acceptable
stack := protocols.NewStack(engine, cfg, nil)
```

---

### 🔴 CRITICAL: API Server Race Conditions

**File:** `pkg/api/server.go`

#### Issue 4: Nil Pointer Dereference in Handlers
**Locations:** Multiple handlers (handleStats, handleDevices, handleRuntime, etc.)

```go
// UNSAFE CODE
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
    stats := s.cfg.Stack.GetStats()  // Can panic if Stack is nil!
```

**Problem:** In daemon mode, `ClearSimulation()` sets `s.cfg.Stack = nil`. Concurrent requests will panic.

**Fix:**
```go
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
    s.configMu.RLock()
    stack := s.cfg.Stack
    s.configMu.RUnlock()

    if stack == nil {
        http.Error(w, "No simulation running", http.StatusServiceUnavailable)
        return
    }

    stats := stack.GetStats()
    // ... rest of the code
}
```

**Apply this pattern to ALL handlers that access:**
- `s.cfg.Stack`
- `s.cfg.Config`
- `s.cfg.Topology`

---

#### Issue 5: Channel Close Race Condition
**Location:** `pkg/api/server.go:167-169`

```go
// UNSAFE CODE (no lock held)
if s.alertStop != nil {
    close(s.alertStop)  // Could race with updateAlertConfig()
    s.alertStop = nil
}
```

**Fix:**
```go
s.alertMu.Lock()
if s.alertStop != nil {
    close(s.alertStop)
    s.alertStop = nil
}
s.alertMu.Unlock()
```

---

#### Issue 6: Type Safety Violation in DaemonController Interface
**Location:** `pkg/api/server.go:84-88`

```go
// UNSAFE TYPE DEFINITION
type DaemonController interface {
    StartSimulation(req interface{}) error  // Should be strongly typed!
    StopSimulation() error
    GetStatus() interface{}                  // Should be strongly typed!
}
```

**Problem:** Using `interface{}` defeats type safety. Server passes anonymous struct, daemon expects `SimulationRequest`.

**Fix:**
```go
// Option 1: Define types in api package
type SimulationRequest struct {
    Interface  string `json:"interface"`
    ConfigPath string `json:"config_path,omitempty"`
    ConfigData string `json:"config_data,omitempty"`
}

type SimulationStatus struct {
    Running       bool      `json:"running"`
    Interface     string    `json:"interface,omitempty"`
    ConfigPath    string    `json:"config_path,omitempty"`
    ConfigName    string    `json:"config_name,omitempty"`
    DeviceCount   int       `json:"device_count"`
    StartedAt     time.Time `json:"started_at,omitempty"`
    UptimeSeconds float64   `json:"uptime_seconds"`
}

type DaemonController interface {
    StartSimulation(req SimulationRequest) error
    StopSimulation() error
    GetStatus() SimulationStatus
}
```

---

### 🔴 HIGH: Missing HTTP Method Validation

**File:** `pkg/api/server.go`

#### Issue 7: handleInterfaces() Accepts Any HTTP Method
**Location:** `pkg/api/server.go:538`

```go
func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
    // No method check - allows POST, PUT, DELETE, etc.
    ifaces, err := capture.GetAllInterfaces()
```

**Fix:**
```go
func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Allow", "GET")
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // ... rest
}
```

#### Issue 8: handleRuntime() Same Problem
**Location:** `pkg/api/server.go:573`

Apply same fix as above.

---

### 🔴 HIGH: Incorrect Uptime Calculation

**File:** `pkg/api/server.go:585`

```go
// BROKEN CODE
"uptime_seconds": time.Since(time.Now()).Seconds(), // Always returns ~0!
```

**Fix:**
```go
// Add to Server struct
type Server struct {
    // ... existing fields
    startTime time.Time
}

// In NewServer()
func NewServer(cfg ServerConfig) *Server {
    return &Server{
        cfg:       cfg,
        startTime: time.Now(),
    }
}

// In handleRuntime()
"uptime_seconds": time.Since(s.startTime).Seconds(),
```

---

### 🔴 HIGH: WebUI Error Handling Issues

**File:** `webui/src/api/client.ts`

#### Issue 9: No Network Error Handling

```typescript
// UNSAFE CODE - no try-catch for network failures
const response = await fetch(buildURL(path), {
    ...init,
    headers,
    credentials: 'same-origin',
});
```

**Fix:**
```typescript
async function request<T>(path: string, init: RequestInit = {}) {
  try {
    const headers = new Headers(init.headers);
    headers.set('Accept', 'application/json');
    if (API_TOKEN) {
      headers.set('Authorization', `Bearer ${API_TOKEN}`);
    }

    const response = await fetch(buildURL(path), {
      ...init,
      headers,
      credentials: 'same-origin',
      signal: AbortSignal.timeout(30000), // 30s timeout
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || response.statusText);
    }

    return response.json() as Promise<T>;
  } catch (err) {
    if (err instanceof TypeError) {
      throw new Error('Network error: Unable to reach the server');
    }
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new Error('Request timeout');
    }
    throw err;
  }
}
```

---

## High Priority Issues (Should Fix Before Release)

### Resource Management

#### Issue 10: storage.Close() Errors Ignored
**Locations:**
- `pkg/daemon/daemon.go:108`
- `pkg/daemon/daemon.go:132`

**Fix:** Log errors instead of ignoring them.

#### Issue 11: Missing Input Validation in handleSimulation()
**Location:** `pkg/api/server.go:610-621`

**Fix:** Validate that interface and config are provided before calling daemon.

---

### Security Concerns

#### Issue 12: Token Exposure via Query Parameters
**Location:** `pkg/api/server.go:221`

```go
} else {
    token = r.URL.Query().Get("token")  // Appears in logs!
}
```

**Recommendation:** Remove query parameter support or add security warning in docs.

#### Issue 13: No Request Size Limits
**Location:** `pkg/api/server.go:610` (handleSimulation), `pkg/api/server.go:377` (handleConfig)

**Fix:**
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
```

---

### WebUI Accessibility

#### Issue 14: Missing Form Labels
**Location:** `webui/src/App.tsx:265-276`

All form inputs need proper `<label>` elements with `htmlFor` attributes.

#### Issue 15: Missing ARIA Attributes
**Location:** `webui/src/App.tsx:303-307`

Error/success messages need `role="alert"` and `aria-live="polite"`.

---

## Medium Priority Issues (Fix in v2.0.1 Patch)

1. **Path Traversal Risk** in `expandPath()` - Add `filepath.Clean()`
2. **Magic Numbers** - Define constants for debug levels, poll intervals
3. **Inconsistent Error Messages** - Standardize capitalization
4. **Large File Size** - Split `App.tsx` (1358 lines) into smaller components
5. **Missing Confirmation Dialogs** - Add confirms for stop/delete actions
6. **No Loading Skeletons** - Add skeleton states for better UX
7. **Performance** - Replace polling with WebSockets where appropriate

---

## Low Priority Issues (Backlog for v2.1+)

1. **API Documentation** - No OpenAPI/Swagger docs (planned for v2.4.0)
2. **Test Coverage** - No tests for new endpoints
3. **CSRF Protection** - Add CSRF tokens for state-changing operations
4. **Rate Limiting** - Add rate limiting to simulation endpoints
5. **Telemetry** - Add error tracking and analytics
6. **Code Organization** - Refactor large files into smaller modules

---

## Required Fixes Summary

### Must Fix Before ANY Release (Blocking)

| # | Issue | File | Lines | Severity | Time |
|---|-------|------|-------|----------|------|
| 1 | `capture.NewEngine()` doesn't exist | daemon.go | 182 | 🔴 BLOCKER | 5 min |
| 2 | `engine.Start()/Stop()` don't exist | daemon.go | 192, 200, 253 | 🔴 BLOCKER | 10 min |
| 3 | Wrong type for `NewStack()` param | daemon.go | 188 | 🔴 BLOCKER | 10 min |
| 4 | Nil pointer dereference in handlers | server.go | Multiple | 🔴 CRITICAL | 30 min |
| 5 | Channel close race condition | server.go | 167 | 🔴 CRITICAL | 5 min |
| 6 | Type safety in DaemonController | server.go | 84-88 | 🔴 CRITICAL | 20 min |

**Total Blocking Fixes:** ~1.5 hours

### Should Fix Before v2.0.0 Release

| # | Issue | File | Severity | Time |
|---|-------|------|----------|------|
| 7-8 | Missing HTTP method validation | server.go | 🟡 HIGH | 10 min |
| 9 | Network error handling | client.ts | 🟡 HIGH | 20 min |
| 10 | Incorrect uptime calculation | server.go | 🟡 HIGH | 15 min |
| 11 | Input validation | server.go | 🟡 HIGH | 20 min |
| 12-13 | Security issues | server.go | 🟡 HIGH | 30 min |
| 14-15 | Accessibility issues | App.tsx | 🟡 HIGH | 1 hour |

**Total High Priority Fixes:** ~2.5 hours

**Grand Total:** ~4 hours to fix all blocking and high-priority issues

---

## Testing Checklist

After fixes are applied, test these scenarios:

### Daemon Mode Testing

- [ ] Start daemon with `niac daemon`
- [ ] Verify API server responds at http://localhost:8080
- [ ] Start simulation via webUI
- [ ] Verify simulation is running
- [ ] Stop simulation via webUI
- [ ] Start new simulation with different config
- [ ] Stop daemon gracefully (SIGTERM)
- [ ] Test concurrent API requests during start/stop

### Error Scenarios

- [ ] Try starting simulation with invalid interface
- [ ] Try starting with missing config
- [ ] Try starting with malformed YAML
- [ ] Try stopping when nothing is running
- [ ] Test large file upload (>1MB)
- [ ] Test network timeout/disconnect
- [ ] Test rapid start/stop cycles

### Cross-Platform Testing

- [ ] macOS (development platform)
- [ ] Linux (Ubuntu 22.04, CentOS 8)
- [ ] Windows (if supported)

### Browser Testing

- [ ] Chrome/Edge (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)

### Accessibility Testing

- [ ] Keyboard navigation (Tab, Enter, Escape)
- [ ] Screen reader (VoiceOver, NVDA, JAWS)
- [ ] Color contrast (WCAG AA compliance)
- [ ] Form validation feedback

---

## Positive Observations

Despite the critical issues, the codebase shows many strengths:

### Architecture
✅ Excellent separation of daemon from API server
✅ Clean interface design for extensibility
✅ Good use of contexts for cancellation
✅ Proper mutex usage (when applied correctly)

### Code Quality
✅ Consistent error wrapping with `%w`
✅ Defensive nil checks throughout
✅ Good resource cleanup patterns
✅ Clear variable naming

### API Design
✅ RESTful conventions followed
✅ Versioned API (`/api/v1/`)
✅ Appropriate HTTP status codes
✅ Good error messages (when shown)

### WebUI
✅ Modern React with TypeScript
✅ Good component structure
✅ Consistent styling with Tailwind
✅ Responsive design

---

## Recommendations

### Immediate Actions (Today)

1. **Fix all blocking issues** (Issues #1-6) - ~1.5 hours
2. **Add nil checks to all handlers** - Critical for stability
3. **Fix type safety** - Prevents runtime errors
4. **Test compilation** - Verify daemon package builds

### Before Release (This Week)

1. **Fix high-priority issues** (Issues #7-15) - ~2.5 hours
2. **Add basic tests** for new endpoints
3. **Update documentation** - README, API docs
4. **Cross-platform testing** - At least macOS + Linux
5. **Create CHANGELOG** for v2.0.0

### Post-Release (v2.0.1 Patch)

1. **Address medium-priority issues**
2. **Improve accessibility** - Full WCAG AA compliance
3. **Add integration tests**
4. **Performance optimization**

### Future Releases

Follow the VERSION_ROADMAP.md:
- v2.1.0: Essential webUI features (logs, error injection)
- v2.2.0: Advanced analysis & protocols
- v2.3.0: Testing & production readiness
- v2.4.0: API documentation

---

## Approval Status

- ❌ **Code Review:** FAILED (blocking issues)
- ⏳ **QA Testing:** PENDING (blocked by compilation errors)
- ⏳ **Documentation Review:** PENDING
- ⏳ **Security Review:** PENDING

**Current Status:** 🔴 **NOT APPROVED FOR RELEASE**

**Next Steps:**
1. Fix blocking issues #1-6
2. Re-compile and test
3. Fix high-priority issues #7-15
4. Run full test suite
5. Re-submit for approval

---

## Estimated Timeline

| Milestone | Duration | Status |
|-----------|----------|--------|
| Fix blocking issues | 1.5 hours | 🔴 Not Started |
| Fix high-priority issues | 2.5 hours | 🔴 Not Started |
| Testing & validation | 2 hours | ⏳ Pending |
| Documentation updates | 1 hour | ⏳ Pending |
| **Total to v2.0.0 Release** | **7 hours** | **1 day** |

**Realistic Release Date:** Tomorrow (2025-11-14) if started today.

---

## Contact

For questions about this review, please refer to:
- GitHub Issues: https://github.com/krisarmstrong/niac-go/issues
- This Review Document: `/Users/krisarmstrong/Developer/projects/niac-go/CODE_REVIEW_V2.0.0.md`
- Version Roadmap: `/Users/krisarmstrong/Developer/projects/niac-go/VERSION_ROADMAP.md`
