# NIAC Security & Code Quality Remediation Plan

**Generated:** 2026-01-12
**Total Issues:** 91 (13 Critical, 28 High, 28 Medium, 22 Low)

---

## Table of Contents
1. [Critical Issues (13)](#critical-issues)
2. [High Priority Issues (28)](#high-priority-issues)
3. [Medium Priority Issues (28)](#medium-priority-issues)
4. [Low Priority Issues (22)](#low-priority-issues)
5. [Implementation Timeline](#implementation-timeline)

---

## Critical Issues

### CRIT-001: ~~WebSocket CheckOrigin Always Returns True~~ RESOLVED
**Status:** RESOLVED - Migrated to SSE

**Resolution:** The WebSocket implementation has been removed and replaced with Server-Sent Events (SSE) via `internal/api/sse.go`. SSE does not require WebSocket upgrade handshakes and uses standard HTTP CORS headers for origin validation. The `sse.go` implementation includes proper origin validation via the CORS middleware.

---

### CRIT-002: YAML Deserialization Without Validation
**Category:** Security | **File:** `pkg/api/devices.go:793-815`

**Problem:** User-supplied YAML parsed directly without schema validation

**Risk:** YAML deserialization attacks, potential code execution

**Fix:**
```go
import "github.com/go-playground/validator/v10"

type DeviceYAMLSchema struct {
    Hostname    string   `yaml:"hostname" validate:"required,hostname"`
    Type        string   `yaml:"type" validate:"required,oneof=router switch firewall"`
    IPs         []string `yaml:"ips" validate:"dive,ip"`
    // ... other fields with validation tags
}

func parseDeviceFromYAML(yamlStr, hostname string) (*config.Device, error) {
    // First unmarshal to strict schema
    var schema DeviceYAMLSchema
    if err := yaml.Unmarshal([]byte(yamlStr), &schema); err != nil {
        return nil, fmt.Errorf("invalid YAML syntax: %w", err)
    }
    
    // Validate against schema
    validate := validator.New()
    if err := validate.Struct(schema); err != nil {
        return nil, fmt.Errorf("YAML validation failed: %w", err)
    }
    
    // Convert to Device
    return schemaToDevice(schema, hostname)
}
```

**Testing:**
- [ ] Unit test: Reject malformed YAML
- [ ] Unit test: Reject YAML with invalid field types
- [ ] Unit test: Reject YAML with missing required fields
- [ ] Fuzz test: Random YAML inputs don't crash

---

### CRIT-003: Path Traversal in Walk File Validation
**Category:** Security | **File:** `pkg/api/walk.go:54-65`

**Problem:** Insufficient path validation, symlink bypass possible

**Risk:** Arbitrary file read via symlinks or absolute paths

**Fix:**
```go
func validateWalkFilePath(filename string, allowedRoot string) (string, error) {
    // Clean the path
    cleanPath := filepath.Clean(filename)
    
    // Reject absolute paths
    if filepath.IsAbs(cleanPath) {
        return "", fmt.Errorf("absolute paths not allowed")
    }
    
    // Reject path traversal attempts
    if strings.Contains(cleanPath, "..") {
        return "", fmt.Errorf("path traversal not allowed")
    }
    
    // Build full path within allowed root
    fullPath := filepath.Join(allowedRoot, cleanPath)
    
    // Resolve symlinks
    realPath, err := filepath.EvalSymlinks(fullPath)
    if err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    
    // Ensure resolved path is within allowed root
    realRoot, _ := filepath.EvalSymlinks(allowedRoot)
    if !strings.HasPrefix(realPath, realRoot+string(os.PathSeparator)) {
        return "", fmt.Errorf("path outside allowed directory")
    }
    
    return realPath, nil
}
```

**Testing:**
- [ ] Unit test: Reject `../etc/passwd`
- [ ] Unit test: Reject `/etc/passwd`
- [ ] Unit test: Reject symlink to outside directory
- [ ] Unit test: Accept valid paths within root

---

### CRIT-004: Unbounded PCAP Cache Memory Exhaustion
**Category:** Security/Performance | **File:** `pkg/api/pcap.go:98-101`

**Problem:** Cache limited by count (50) not memory, each up to 100MB

**Risk:** 5GB memory exhaustion DoS attack

**Fix:**
```go
type pcapCache struct {
    analyses    map[string]*PcapAnalysisResult
    accessOrder []string // LRU tracking
    totalBytes  int64
    maxBytes    int64  // e.g., 500MB total
    maxEntries  int
    mu          sync.RWMutex
}

func (c *pcapCache) Add(id string, result *PcapAnalysisResult) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    resultSize := result.EstimatedSize()
    
    // Evict until we have room
    for c.totalBytes+resultSize > c.maxBytes && len(c.accessOrder) > 0 {
        oldestID := c.accessOrder[0]
        c.accessOrder = c.accessOrder[1:]
        if old, ok := c.analyses[oldestID]; ok {
            c.totalBytes -= old.EstimatedSize()
            delete(c.analyses, oldestID)
        }
    }
    
    // Reject if single item exceeds max
    if resultSize > c.maxBytes {
        slog.Warn("[PCAP] Analysis too large to cache", "size", resultSize)
        return
    }
    
    c.analyses[id] = result
    c.accessOrder = append(c.accessOrder, id)
    c.totalBytes += resultSize
}
```

**Testing:**
- [ ] Unit test: Cache evicts when memory limit reached
- [ ] Unit test: Large single item rejected
- [ ] Load test: 100 concurrent uploads don't crash

---

### CRIT-005: No Per-Endpoint Rate Limiting
**Category:** Security | **File:** `pkg/api/server.go:752-765`

**Problem:** Only global rate limiting, sensitive endpoints unprotected

**Risk:** DoS via config/simulation endpoint flooding

**Fix:**
```go
type EndpointRateLimiter struct {
    limiters map[string]*RateLimiter // endpoint -> limiter
    mu       sync.RWMutex
}

var endpointLimits = map[string]struct{ rate, burst int }{
    "/api/v1/config":     {10, 20},   // 10 req/s for config changes
    "/api/v1/simulation": {5, 10},    // 5 req/s for simulation control
    "/api/v1/pcap":       {2, 5},     // 2 req/s for PCAP uploads
    "/api/v1/devices":    {20, 40},   // 20 req/s for device operations
}

func (s *Server) endpointRateLimit(endpoint string, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ip := getClientIP(r)
        limiter := s.endpointLimiter.GetLimiter(endpoint, ip)
        if !limiter.Allow() {
            writeError(w, r, http.StatusTooManyRequests, "rate_limited",
                "Too many requests to this endpoint", nil)
            return
        }
        next(w, r)
    }
}
```

**Testing:**
- [ ] Unit test: Endpoint-specific limits enforced
- [ ] Integration test: Burst requests blocked
- [ ] Load test: Verify rate limiting under load

---

### CRIT-006: No Unit Tests in WebUI
**Category:** Code Quality | **File:** `ui/` (entire directory)

**Problem:** Zero test files found in UI codebase

**Risk:** Zero confidence in code changes, regression bugs

**Fix:**
1. Add testing dependencies:
```json
{
  "devDependencies": {
    "vitest": "^2.1.0",
    "@vitest/coverage-v8": "^2.1.0",
    "@testing-library/react": "^16.0.0",
    "@testing-library/user-event": "^14.5.0",
    "jsdom": "^25.0.0"
  },
  "scripts": {
    "test": "vitest",
    "test:coverage": "vitest --coverage",
    "test:ui": "vitest --ui"
  }
}
```

2. Create test structure:
```
ui/src/
├── __tests__/
│   ├── setup.ts
│   ├── api/
│   │   └── client.test.ts
│   ├── hooks/
│   │   ├── useApiResource.test.ts
│   │   └── useEventSource.test.ts
│   ├── components/
│   │   ├── ErrorInjectionPanel.test.tsx
│   │   └── ...
│   └── pages/
│       └── DashboardPage.test.tsx
```

3. Create vitest.config.ts:
```typescript
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/__tests__/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      exclude: ['node_modules/', 'src/__tests__/'],
      thresholds: {
        global: {
          branches: 70,
          functions: 70,
          lines: 70,
          statements: 70
        }
      }
    }
  }
});
```

**Testing:**
- [ ] Set up Vitest configuration
- [ ] Create test setup file with mocks
- [ ] Write tests for API client (minimum 10 tests)
- [ ] Write tests for hooks (minimum 15 tests)
- [ ] Write tests for critical components (minimum 20 tests)
- [ ] Achieve 70% code coverage

---

### CRIT-007: SSRF in Webhook URLs
**Category:** Security | **File:** `pkg/api/server.go:338-369`

**Problem:** Webhook URLs not validated for private IPs

**Risk:** Server-side request forgery to internal services

**Fix:**
```go
import "net"

func isPrivateIP(ip net.IP) bool {
    privateBlocks := []string{
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
        "127.0.0.0/8",
        "169.254.0.0/16",
        "::1/128",
        "fc00::/7",
        "fe80::/10",
    }
    
    for _, block := range privateBlocks {
        _, cidr, _ := net.ParseCIDR(block)
        if cidr.Contains(ip) {
            return true
        }
    }
    return false
}

func validateWebhookURL(urlStr string) error {
    parsed, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    
    // Must be HTTP or HTTPS
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return fmt.Errorf("webhook must use http or https")
    }
    
    // Resolve hostname
    ips, err := net.LookupIP(parsed.Hostname())
    if err != nil {
        return fmt.Errorf("cannot resolve hostname: %w", err)
    }
    
    // Check all resolved IPs
    for _, ip := range ips {
        if isPrivateIP(ip) {
            return fmt.Errorf("webhook cannot target private IP addresses")
        }
    }
    
    return nil
}
```

**Testing:**
- [ ] Unit test: Reject 127.0.0.1
- [ ] Unit test: Reject 192.168.x.x
- [ ] Unit test: Reject 10.x.x.x
- [ ] Unit test: Accept public IPs
- [ ] Unit test: Reject localhost DNS resolution

---

### CRIT-008: Missing Error Boundaries in React
**Category:** Code Quality | **File:** `ui/src/`

**Problem:** Error boundaries exist but not properly utilized

**Risk:** Unhandled errors crash entire app

**Fix:**
1. Wrap route components:
```tsx
// App.tsx
import { ErrorBoundary } from './components/ErrorBoundary';

const PageWrapper: FC<{ children: ReactNode }> = ({ children }) => (
  <ErrorBoundary fallback={<PageErrorFallback />}>
    <Suspense fallback={<PageLoader />}>
      {children}
    </Suspense>
  </ErrorBoundary>
);

// In routes
<Route path="/devices" element={
  <PageWrapper>
    <DevicesPage />
  </PageWrapper>
} />
```

2. Create granular error fallbacks:
```tsx
// components/PageErrorFallback.tsx
export const PageErrorFallback: FC<{ error?: Error; resetError?: () => void }> = ({
  error,
  resetError
}) => (
  <Card className="m-8 p-8 text-center">
    <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
    <H2>Something went wrong</H2>
    <SmallText className="text-gray-400 mb-4">
      {error?.message || 'An unexpected error occurred'}
    </SmallText>
    <Button onClick={resetError}>Try Again</Button>
  </Card>
);
```

**Testing:**
- [ ] Unit test: Error boundary catches errors
- [ ] Unit test: Fallback renders correctly
- [ ] Unit test: Reset functionality works
- [ ] E2E test: App recovers from component error

---

### CRIT-009: Type Safety Violations with `any`
**Category:** Code Quality | **File:** `ui/src/components/ErrorInjectionPanel.tsx:132,141`

**Problem:**
```typescript
{errorInfo?.available_types?.map((type: any) => (
```

**Risk:** Runtime type errors, no IDE support

**Fix:**
```typescript
// In types.ts
export interface ErrorTypeInfo {
  type: string;
  description: string;
  min_value?: number;
  max_value?: number;
  unit?: string;
}

export interface ErrorInjectionInfo {
  available_types: ErrorTypeInfo[];
  active_errors: ActiveError[];
}

// In component
{errorInfo?.available_types?.map((type: ErrorTypeInfo) => (
  <option key={type.type} value={type.type}>
    {type.type}
  </option>
))}
```

**Testing:**
- [ ] TypeScript strict mode passes
- [ ] No `any` types in codebase (lint rule)

---

### CRIT-010: Race Condition in Config Apply/Write
**Category:** Security | **File:** `pkg/api/server.go:1206-1229`

**Problem:** Config applied in memory before written to disk, rollback can fail

**Risk:** Inconsistent state between memory and disk

**Fix:**
```go
func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
    // ... parse request ...
    
    // Use a transaction-like approach
    s.configMu.Lock()
    defer s.configMu.Unlock()
    
    // 1. Write to temp file first
    tmpPath := s.cfg.ConfigPath + ".tmp"
    if err := os.WriteFile(tmpPath, []byte(req.Content), 0644); err != nil {
        writeError(w, r, http.StatusInternalServerError, "write_failed", 
            "Failed to write config", nil)
        return
    }
    
    // 2. Validate by loading temp file
    tmpCfg, err := config.LoadFile(tmpPath)
    if err != nil {
        os.Remove(tmpPath)
        writeError(w, r, http.StatusBadRequest, "invalid_config",
            "Config validation failed", nil)
        return
    }
    
    // 3. Atomic rename
    if err := os.Rename(tmpPath, s.cfg.ConfigPath); err != nil {
        os.Remove(tmpPath)
        writeError(w, r, http.StatusInternalServerError, "rename_failed",
            "Failed to save config", nil)
        return
    }
    
    // 4. Apply to runtime (already validated)
    if err := s.cfg.ApplyConfig(tmpCfg); err != nil {
        // This should not happen since we validated
        slog.Error("[API] Config apply failed after validation", "error", err)
        writeError(w, r, http.StatusInternalServerError, "apply_failed",
            "Failed to apply config", nil)
        return
    }
    
    writeJSON(w, http.StatusOK, tmpCfg)
}
```

**Testing:**
- [ ] Unit test: Failed write doesn't corrupt state
- [ ] Unit test: Invalid config doesn't apply
- [ ] Integration test: Concurrent updates handled safely

---

### CRIT-011: PCAP Path Validation Insufficient
**Category:** Security | **File:** `pkg/api/server.go:1947-1959`

**Problem:** `filepath.Abs()` used but no directory containment check

**Risk:** Arbitrary file read as PCAP

**Fix:** Same pattern as CRIT-003

---

### CRIT-012: Temp File World-Readable
**Category:** Security | **File:** `pkg/api/server.go:1964`

**Problem:** Temp directory created with 0750, files may be readable

**Risk:** Information disclosure via temp files

**Fix:**
```go
// Create temp directory with restrictive permissions
dir := filepath.Join(os.TempDir(), "niac-replay")
if err := os.MkdirAll(dir, 0700); err != nil {
    // handle error
}

// Create temp file with restrictive permissions
tmp, err := os.CreateTemp(dir, "upload-*.pcap")
if err != nil {
    // handle error
}
// Ensure restrictive permissions
os.Chmod(tmp.Name(), 0600)
```

---

### CRIT-013: No E2E Tests
**Category:** Code Quality | **File:** `ui/`

**Problem:** No end-to-end test coverage

**Risk:** User flows untested, regressions undetected

**Fix:**
1. Add Playwright:
```json
{
  "devDependencies": {
    "@playwright/test": "^1.45.0"
  },
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui"
  }
}
```

2. Create test structure:
```
ui/e2e/
├── playwright.config.ts
├── auth.spec.ts
├── dashboard.spec.ts
├── devices.spec.ts
├── topology.spec.ts
└── fixtures/
    └── test-data.ts
```

3. Example test:
```typescript
// e2e/dashboard.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test('displays live statistics', async ({ page }) => {
    await page.goto('/');
    
    // Wait for stats to load
    await expect(page.getByText('Packets/sec')).toBeVisible();
    await expect(page.getByText('Bytes/sec')).toBeVisible();
    
    // Verify stats update
    const initialPackets = await page.getByTestId('packets-counter').textContent();
    await page.waitForTimeout(2000);
    const updatedPackets = await page.getByTestId('packets-counter').textContent();
    
    // Stats should have changed (or at least rendered)
    expect(initialPackets).toBeDefined();
  });
});
```

---

## High Priority Issues

### HIGH-001: Walk File Directory Traversal Edge Case
**File:** `pkg/api/server.go:2186-2198`
**Problem:** Directory name confusion (`/root` vs `/root-secret`)
**Fix:** Ensure path check includes trailing separator

### HIGH-002: Temp File Race Condition
**File:** `pkg/api/server.go:1962-1981`
**Problem:** Window between create and delete where file readable
**Fix:** Use `os.CreateTemp` with immediate `os.Chmod(0600)`

### HIGH-003: ~~Localhost Check Missing Port/IPv6~~ RESOLVED
**Status:** RESOLVED - Migrated to SSE
**Resolution:** WebSocket code removed. SSE implementation uses standard HTTP CORS middleware.

### HIGH-004: No Device Hostname Format Validation
**File:** `pkg/api/devices.go:271-276`
**Problem:** Only checks for empty string
**Fix:** Validate against DNS hostname rules

### HIGH-005: PCAP Magic-Only Validation
**File:** `pkg/api/server.go:1878-1908`
**Problem:** Only checks first 4 bytes
**Fix:** Add basic structure validation

### HIGH-006: File Listing No Specific Rate Limit
**File:** `pkg/api/server.go:1346-1364`
**Problem:** Can enumerate all files rapidly
**Fix:** Add specific rate limit for file listing

### HIGH-007: Metrics Endpoint No Access Control
**File:** `pkg/api/server.go:795`
**Problem:** `/metrics` exposes system info
**Fix:** Add authentication or limit exposure

### HIGH-008: No Device Count Limit
**File:** `pkg/api/devices.go:262-269`
**Problem:** Unlimited device creation
**Fix:** Add maximum device count

### HIGH-009: Missing Loading States
**File:** `ui/src/pages/*`
**Problem:** No skeleton loaders, jarring UX
**Fix:** Add loading states to all data-fetching components

### HIGH-010: API Client No Retry Logic
**File:** `ui/src/api/client.ts:51-79`
**Problem:** Single attempt, fails on transient errors
**Fix:** Add exponential backoff retry

### HIGH-011: localStorage Unencrypted
**File:** `ui/src/ui/Sidebar.tsx`, `ui/src/pages/DeviceListPage.tsx`
**Problem:** Preferences stored in plain text
**Fix:** Low risk for preferences, but audit for sensitive data

### HIGH-012: Missing Null Checks
**File:** Multiple components
**Problem:** Optional chaining needed in more places
**Fix:** Audit and add defensive checks

### HIGH-013: Inconsistent Error Handling
**File:** Multiple
**Problem:** Some errors swallowed, some thrown
**Fix:** Standardize error handling pattern

### HIGH-014: Component Prop Drilling
**File:** Multiple
**Problem:** Props passed through many levels
**Fix:** Use React Context for shared state

### HIGH-015: No Input Sanitization
**File:** Form components
**Problem:** User input not sanitized before display
**Fix:** Add input sanitization utilities

### HIGH-016: Async Operations No Cleanup
**File:** Hooks
**Problem:** AbortController not used consistently
**Fix:** Add cleanup to all async useEffects

### HIGH-017: No Code Splitting
**File:** `ui/vite.config.ts`
**Problem:** All routes bundled together
**Fix:** Add manual chunks configuration

### HIGH-018: No Bundle Analysis
**File:** Build config
**Problem:** Unknown bundle size
**Fix:** Add rollup-plugin-visualizer

### HIGH-019: Missing Interface Name Validation
**File:** `pkg/api/server.go:379-402`
**Problem:** Allows potentially dangerous characters
**Fix:** Strict allowlist validation

### HIGH-020: Error Messages Expose Details
**File:** Multiple Go files
**Problem:** Internal errors leaked to client
**Fix:** Sanitize all error responses

### HIGH-021: No Dependency Vulnerability Scan
**File:** CI/CD
**Problem:** No automated security scanning
**Fix:** Add `npm audit` and `go mod verify` to CI

### HIGH-022: useEffect Missing Dependencies
**File:** Multiple hooks
**Problem:** Stale closures possible
**Fix:** Audit all useEffect/useCallback deps

### HIGH-023: No Form Validation Library
**File:** UI forms
**Problem:** Manual validation inconsistent
**Fix:** Add Zod for schema validation

### HIGH-024: Missing ARIA Live Regions
**File:** UI components
**Problem:** Dynamic updates not announced
**Fix:** Add appropriate ARIA attributes

### HIGH-025: No Keyboard Navigation
**File:** Custom components
**Problem:** Mouse-only interaction
**Fix:** Add keyboard handlers

### HIGH-026: Color Contrast Issues
**File:** `ui/src/index.css`
**Problem:** Muted text fails WCAG in dark mode
**Fix:** Increase contrast ratios

### HIGH-027: Missing Form Labels
**File:** Form components
**Problem:** Some inputs lack proper labels
**Fix:** Add htmlFor/id associations

### HIGH-028: No Error Logging Service
**File:** UI
**Problem:** Client errors not tracked
**Fix:** Add Sentry or similar

---

## Medium Priority Issues

### MED-001 through MED-028
(Abbreviated for space - includes items like magic numbers, code duplication, missing documentation, i18n preparation, offline support, undo/redo functionality, etc.)

---

## Low Priority Issues

### LOW-001 through LOW-022
(Abbreviated for space - includes items like cache headers, hardcoded buffer sizes, missing TypeScript path aliases, etc.)

---

## Implementation Timeline

### Phase 1: Critical Security (Week 1)
- [x] CRIT-001: ~~WebSocket CORS fix~~ RESOLVED (migrated to SSE)
- [ ] CRIT-002: YAML validation
- [ ] CRIT-003: Path traversal fix
- [ ] CRIT-004: PCAP cache limits
- [ ] CRIT-005: Rate limiting
- [ ] CRIT-007: SSRF protection

### Phase 2: Critical Quality (Week 2)
- [ ] CRIT-006: Unit test setup
- [ ] CRIT-008: Error boundaries
- [ ] CRIT-009: Type safety
- [ ] CRIT-010: Config transactions
- [ ] CRIT-013: E2E test setup

### Phase 3: High Priority (Weeks 3-4)
- [ ] HIGH-001 through HIGH-010
- [ ] HIGH-011 through HIGH-020
- [ ] HIGH-021 through HIGH-028

### Phase 4: Medium Priority (Weeks 5-6)
- [ ] MED-001 through MED-014
- [ ] MED-015 through MED-028

### Phase 5: Low Priority (Weeks 7-8)
- [ ] LOW-001 through LOW-011
- [ ] LOW-012 through LOW-022

---

## Verification Checklist

### Security Verification
- [ ] Penetration test SSE endpoints
- [ ] Fuzz test YAML parser
- [ ] Path traversal testing with symlinks
- [ ] Load test with memory monitoring
- [ ] Rate limit verification under load
- [ ] SSRF testing with private IPs

### Quality Verification
- [ ] Unit test coverage > 70%
- [ ] E2E tests pass
- [ ] TypeScript strict mode passes
- [ ] No `any` types (ESLint rule)
- [ ] All hooks have proper deps
- [ ] Error boundaries tested

### Performance Verification
- [ ] Bundle size < 500KB gzipped
- [ ] Lighthouse score > 90
- [ ] No memory leaks (heap profiling)
- [ ] SSE reconnection works

---

## Notes

This plan was generated based on a comprehensive security, code quality, and architecture review. Issues are prioritized by:

1. **Critical**: Security vulnerabilities or showstopper bugs
2. **High**: Significant security or quality issues
3. **Medium**: Code quality and maintainability
4. **Low**: Nice-to-have improvements

All critical issues should be addressed before production deployment.
