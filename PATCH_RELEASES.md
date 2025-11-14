# NIAC-Go Patch Release Plan

**Status:** Ready for v2.0.0 Release
**Date:** 2025-11-14

---

## v2.0.0 Release Status

### ✅ COMPLETED - All Blocking Issues Fixed

All 6 blocking/critical and 9 high-priority issues have been resolved:

**Backend Fixes:**
- ✅ Fixed `capture.NewEngine()` → `capture.New()`
- ✅ Fixed `engine.Start()/Stop()` → removed invalid calls, use `Close()`
- ✅ Fixed `protocols.NewStack()` parameter type
- ✅ Added nil checks to all API handlers (handleStats, handleMetrics, handleRuntime, alertLoop)
- ✅ Fixed channel close race condition in `Shutdown()`
- ✅ Added type safety to DaemonController interface
- ✅ Added HTTP method validation to handleInterfaces() and handleRuntime()
- ✅ Fixed uptime calculation with server startTime tracking
- ✅ Added input validation to handleSimulation()
- ✅ Added request body size limits (1MB) to prevent DoS
- ✅ Fixed storage.Close() error handling

**Frontend Fixes:**
- ✅ Added network error handling with timeout (30s)
- ✅ Added getErrorMessage() helper function
- ✅ Improved error messages for network failures

**Build Status:**
- ✅ Backend compiles successfully
- ✅ WebUI builds successfully
- ✅ UI assets deployed to pkg/api/ui/

### Ready for Release

v2.0.0 is now **READY FOR RELEASE** with:
- Full daemon mode support
- Complete CLI parity in webUI
- Thread-safe API handlers
- Proper error handling
- Input validation and security fixes

---

## v2.0.1 - Accessibility & UX Improvements

**Target Date:** 1 week after v2.0.0
**Focus:** WebUI accessibility compliance and user experience enhancements

### Issues to Fix (Medium Priority - 8 total)

#### 1. Accessibility - Form Labels
**Priority:** MEDIUM
**Time:** 1 hour

Add proper `<label>` elements with `htmlFor` attributes to all form inputs:
- Network interface selector (RuntimeControlPage)
- Config path input
- Config file upload
- PCAP file inputs (ReplayPanel)
- Alert configuration inputs

**Files:**
- `webui/src/App.tsx` (lines ~265-301, ~950-1100, ~1150-1200)

---

#### 2. Accessibility - ARIA Attributes
**Priority:** MEDIUM
**Time:** 30 min

Add `role="alert"` and `aria-live="polite"` to all error/success message displays:
- RuntimeControlPage messages
- ReplayPanel messages
- ConfigurationPage status
- AlertConfigCard status

**Files:**
- `webui/src/App.tsx` (lines ~303, ~342, ~1059, etc.)

---

#### 3. Path Traversal Protection in expandPath()
**Priority:** MEDIUM
**Time:** 15 min

Add path validation to prevent directory traversal:

```go
func expandPath(path string) string {
    if len(path) > 0 && path[0] == '~' {
        home, err := os.UserHomeDir()
        if err == nil {
            path = filepath.Join(home, path[1:])
        }
    }
    // Clean the path to remove any .. or . elements
    return filepath.Clean(path)
}
```

**Files:**
- `pkg/daemon/daemon.go:309`

---

#### 4. Magic Numbers - Define Constants
**Priority:** LOW
**Time:** 30 min

Replace magic numbers with named constants:
- Polling intervals (2000, 5000, 15000, 60000 ms)
- Debug levels (0)
- File size limits (1MB)

**Files:**
- `webui/src/App.tsx` (lines ~126, ~171, ~425-428)
- `pkg/daemon/daemon.go` (line ~165)
- `pkg/api/server.go` (line ~629)

---

#### 5. File Upload Feedback & Validation
**Priority:** MEDIUM
**Time:** 1 hour

Add client-side file validation:
- File size limits (with user-friendly messages)
- File type validation (.yaml/.yml, .pcap/.pcapng)
- Upload progress indicators
- Better error messages

**Files:**
- `webui/src/App.tsx` (RuntimeControlPage, ReplayPanel)

---

#### 6. Confirmation Dialogs for Destructive Actions
**Priority:** MEDIUM
**Time:** 30 min

Add confirmation dialogs before:
- Stopping simulation
- Stopping replay
- Deleting/clearing data

**Files:**
- `webui/src/App.tsx` (handleStop functions)

---

#### 7. Inconsistent Error Message Format
**Priority:** LOW
**Time:** 30 min

Standardize error messages to lowercase:
- `"failed to start simulation"` (not `"Failed to..."`)
- `"invalid request"` (not `"Invalid..."`)

**Files:**
- `pkg/api/server.go` (throughout)

---

#### 8. Missing Focus Management
**Priority:** LOW
**Time:** 1 hour

Implement keyboard focus management:
- Focus first input when opening panels
- Return focus after modal/panel close
- Trap focus within modals

**Files:**
- `webui/src/App.tsx` (all panel components)

---

### Total Estimated Time: 5.5 hours

---

## v2.0.2 - Code Quality & Organization

**Target Date:** 2 weeks after v2.0.1
**Focus:** Code refactoring, organization, and maintainability

### Issues to Fix (Medium/Low Priority - 6 total)

#### 1. Split Large App.tsx File (1358 lines)
**Priority:** MEDIUM
**Time:** 3-4 hours

Refactor into:
```
webui/src/
├── App.tsx (routes and shell only)
├── pages/
│   ├── DashboardPage.tsx
│   ├── RuntimeControlPage.tsx
│   ├── DevicesPage.tsx
│   ├── TopologyPage.tsx
│   ├── AnalysisPage.tsx
│   └── AutomationPage.tsx
├── components/
│   ├── DeviceTable.tsx
│   ├── NeighborTable.tsx
│   ├── ReplayPanel.tsx
│   ├── AlertConfigCard.tsx
│   └── ...
└── utils/
    ├── formatters.ts
    ├── fileHelpers.ts
    └── errors.ts
```

---

#### 2. Extract Duplicated Code Patterns
**Priority:** MEDIUM
**Time:** 2 hours

Create custom hooks:
- `useAsyncHandler` - for try-catch-finally patterns
- `usePolling` - for interval-based data fetching
- `useFileUpload` - for file upload logic

---

#### 3. Improve File Conversion Performance
**Priority:** LOW
**Time:** 30 min

Replace `fileToBase64()` with FileReader for better performance:

```typescript
async function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      const base64 = result.split(',')[1];
      resolve(base64);
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}
```

---

#### 4. Add Loading Skeletons
**Priority:** LOW
**Time:** 2 hours

Create skeleton loading states for:
- Device table
- Topology visualization
- History table
- Statistics cards

---

#### 5. Split server.go (989 lines)
**Priority:** MEDIUM
**Time:** 2 hours

Refactor into:
```
pkg/api/
├── server.go (core server, routing)
├── handlers.go (HTTP handlers)
├── handlers_daemon.go (daemon-specific handlers)
├── helpers.go (helper functions)
```

---

#### 6. Add Request/Response Logging
**Priority:** LOW
**Time:** 1 hour

Add middleware for request logging:
- Request method, path, duration
- Error responses
- Optional debug mode for request/response bodies

---

### Total Estimated Time: 10.5-11.5 hours

---

## v2.0.3 - Performance & Security

**Target Date:** 3 weeks after v2.0.2
**Focus:** Performance optimization and security hardening

### Issues to Fix (Low Priority - 6 total)

#### 1. Replace Polling with WebSockets
**Priority:** MEDIUM
**Time:** 4-6 hours

Implement WebSocket endpoints for real-time updates:
- Live statistics
- Device status changes
- Log streaming
- Simulation status

**Benefits:**
- Reduced server load
- Lower latency
- Better user experience
- Lower bandwidth usage

---

#### 2. Add Rate Limiting
**Priority:** MEDIUM
**Time:** 2 hours

Implement rate limiting middleware:
- Per-IP rate limits
- Per-endpoint limits (e.g., simulation start/stop)
- Configurable thresholds
- Return 429 Too Many Requests

---

#### 3. Remove Token Query Parameter Support
**Priority:** MEDIUM (Security)
**Time:** 30 min

Remove `?token=` support, require Authorization header only:

```go
// Remove this fallback
} else {
    token = r.URL.Query().Get("token")  // REMOVE THIS
}
```

Add warning in documentation about security implications.

---

#### 4. Add CSRF Protection
**Priority:** MEDIUM (Security)
**Time:** 2 hours

Implement CSRF tokens:
- Generate token on session start
- Validate on state-changing operations (POST/PUT/DELETE)
- Return 403 Forbidden on validation failure

---

#### 5. Add Security Headers
**Priority:** LOW (Security)
**Time:** 30 min

Add security headers to all responses:
```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Strict-Transport-Security", "max-age=31536000")
```

---

#### 6. Add Caching Strategy
**Priority:** LOW
**Time:** 2 hours

Implement intelligent caching:
- Cache static data (version, interfaces, error types)
- Use `stale-while-revalidate` pattern
- Add cache invalidation on configuration changes
- Add ETag support for conditional requests

---

### Total Estimated Time: 11-13 hours

---

## Summary

| Version | Issues | Estimated Time | Focus |
|---------|--------|----------------|-------|
| v2.0.0 | 0 open | **COMPLETE** | Core functionality & CLI parity |
| v2.0.1 | 8 | 5.5 hours | Accessibility & UX |
| v2.0.2 | 6 | 10.5-11.5 hours | Code quality & organization |
| v2.0.3 | 6 | 11-13 hours | Performance & security |
| **Total** | **20** | **~27-30 hours** | **~4-6 weeks** |

---

## Release Process for Each Version

### 1. Development
- Create feature branch from `main`
- Implement fixes from patch plan
- Code review each PR

### 2. Testing
- Unit tests
- Integration tests
- Manual QA testing
- Cross-platform testing (macOS, Linux, Windows)

### 3. Documentation
- Update CHANGELOG.md
- Update README.md if needed
- Add migration notes if needed

### 4. Release
- Merge to `main`
- Tag release (`git tag v2.0.X`)
- Build binaries for all platforms
- Publish GitHub release
- Update version in code

---

## Beyond v2.0.x - Future Enhancements

Items deferred to v2.1.0 and beyond (per VERSION_ROADMAP.md):

**v2.1.0 - Essential WebUI Features** (7 issues):
- Live log streaming (#86)
- Complete error injection API (#88)
- Hex dump packet viewer (#87)
- Traffic injection controls (#90)
- SNMP trap generation (#76)
- VLAN-aware ARP (#77)
- Monaco Editor for YAML (#89)

**v2.2.0 - Advanced Analysis** (5 issues):
- PCAP analysis tools (#91)
- Configuration bundle export (#92)
- Traffic generation patterns (#78)
- Complete IPv6 support (#80)
- Modern equipment walk files (#54)

**v2.3.0 - Testing & Production** (6 issues):
- Test coverage improvements (#74, #73, #55, #47)
- Performance monitoring (#36)
- Container/K8s deployment (#35)

**v2.4.0 - API Documentation** (1 issue):
- OpenAPI/Swagger docs (#31)
- API hardening

**v3.0.0+ - Enterprise Features** (3 issues):
- Database persistence (#32)
- Advanced protocol analyzers (#34)
- Advanced topology visualization (#37)
- **DEFERRED:** Multi-user support (#33)

---

## Notes

- All patch releases (v2.0.1-v2.0.3) are **non-breaking changes**
- Backward compatibility maintained throughout v2.x series
- Each patch can be released independently as fixes are ready
- No need to wait for all items in a version to be complete
- Critical security fixes can be released as needed (v2.0.X)

---

## Current Status

**✅ THREE RELEASES COMPLETED AND PUBLISHED!**

### Released Versions

**v2.0.1** - Released 2025-11-14
- ✅ All 8 accessibility & UX issues fixed
- ✅ WCAG AA compliance achieved
- ✅ Committed, tagged, and pushed

**v2.0.2** - Released 2025-11-14
- ✅ Performance optimizations complete
- ✅ Request logging implemented
- ✅ Committed, tagged, and pushed

**v2.1.0** - Released 2025-11-14 (formerly v2.0.3)
- ✅ Security hardening complete
- ✅ Breaking change: Removed query parameter auth
- ✅ Security headers implemented
- ✅ Committed, tagged, and pushed
- ⚠️ **BREAKING CHANGE** - Bumped to v2.1.0 per semver

### Git Status
```
✅ Commit 18a5c3b: v2.0.1 - Accessibility & UX Improvements
✅ Commit 72aadec: v2.0.2 - Code Quality & Performance
✅ Commit d837407: v2.1.0 - Security Hardening
✅ Tags pushed: v2.0.1, v2.0.2, v2.1.0
✅ Branch: v2.0.0-webui (up to date with origin)
```

### Next Steps
1. Create GitHub releases for each tag
2. Close completed issues from VERSION_ROADMAP.md
3. Update README.md badges to v2.1.0
4. Begin work on remaining VERSION_ROADMAP.md features
