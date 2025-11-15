# Remaining Issues Summary - Post v2.3.1

**Last Updated:** November 14, 2025
**Critical Issues:** ✅ 0 (All fixed in v2.3.1)
**High Priority:** 🟠 6 issues
**Medium Priority:** 🟡 13 issues

---

## 🟠 HIGH Priority Issues (6 issues) - Fix Next Sprint

**Total Estimated Time:** 6.5 - 11 hours
**Recommendation:** Address in v2.3.2 or v2.4.0

### Security Issues (5)

#### #98 - Goroutine Leak Risk in HTTP Server Startup
- **File:** `pkg/api/server.go` lines 179-183
- **Severity:** HIGH
- **Impact:** Memory leaks over time, resource exhaustion
- **Fix Time:** 2-4 hours
- **Fix:** Use context with timeout for graceful shutdown
```go
func (s *Server) Start(ctx context.Context) error {
    httpServer := &http.Server{...}
    go func() {
        if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("API server error: %v", err)
        }
    }()
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return httpServer.Shutdown(shutdownCtx)
}
```

#### #99 - Missing HTTP Timeouts Allow Slowloris Attacks
- **File:** `pkg/api/server.go`
- **Severity:** HIGH
- **Impact:** DoS via slow connections
- **Fix Time:** 1-2 hours
- **Fix:** Add timeout configuration to HTTP server
```go
httpServer := &http.Server{
    Addr:              s.cfg.Listen,
    Handler:           s.mux,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
    ReadHeaderTimeout: 5 * time.Second,
    MaxHeaderBytes:    1 << 20,
}
```

#### #100 - API Token Comparison Vulnerable to Timing Attacks
- **File:** `pkg/api/server.go` line ~270
- **Severity:** HIGH
- **Impact:** Token could be brute-forced via timing analysis
- **Fix Time:** 30 minutes
- **Fix:** Use constant-time comparison
```go
import "crypto/subtle"

if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

#### #101 - API Token Exposed in CLI Arguments and Shell History
- **File:** `cmd/niac/root.go` line 65
- **Severity:** HIGH
- **Impact:** Token visible in `ps` output and shell history
- **Fix Time:** 1 hour
- **Fix:** Accept token from environment variable
```go
func initConfig() {
    if token := os.Getenv("NIAC_API_TOKEN"); token != "" {
        servicesOpts.apiToken = token
    }
    if cmd.Flags().Changed("api-token") {
        log.Warn("--api-token is deprecated. Use NIAC_API_TOKEN env var")
    }
}
```

#### #102 - Missing Security Headers (CSP, HSTS, Permissions-Policy)
- **File:** `pkg/api/server.go` lines 38-44
- **Severity:** HIGH
- **Impact:** XSS, clickjacking, and other web attacks possible
- **Fix Time:** 2-3 hours
- **Fix:** Add comprehensive security headers
```go
w.Header().Set("Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline'; "+
    "style-src 'self' 'unsafe-inline'; object-src 'none'")
if r.TLS != nil {
    w.Header().Set("Strict-Transport-Security",
        "max-age=31536000; includeSubDomains")
}
w.Header().Set("Referrer-Policy", "no-referrer")
w.Header().Set("Permissions-Policy",
    "geolocation=(), microphone=(), camera=()")
```

### Documentation (1)

#### #103 - Update Version Badge in README from 2.1.1 to 2.3.0
- **File:** `README.md` line 6
- **Severity:** HIGH (Documentation)
- **Impact:** User confusion about current version
- **Fix Time:** 2 minutes
- **Fix:** Update version badge or use dynamic badge
```markdown
![Version](https://img.shields.io/badge/version-2.3.1-brightgreen.svg)
# Or use dynamic:
![Version](https://img.shields.io/github/v/release/krisarmstrong/niac-go)
```

---

## 🟡 MEDIUM Priority Issues (13 issues) - Fix Next 2 Sprints

**Total Estimated Time:** 35-51 hours

### Security & Code Quality (8)

#### #104 - Implement API Rate Limiting to Prevent Abuse
- **Severity:** MEDIUM
- **Impact:** No protection against brute force or DoS
- **Fix Time:** 4-6 hours
- **Fix:** Add rate limiting middleware
```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}
```

#### #105 - Standardize API Error Response Format
- **Severity:** MEDIUM
- **Impact:** Inconsistent error handling for clients
- **Fix Time:** 2-3 hours
- **Fix:** Create standard error response format
```json
{
  "error": "validation_error",
  "message": "Config validation failed",
  "details": [{"field": "devices[0].name", "issue": "required"}],
  "request_id": "abc-123"
}
```

#### #106 - Silent Error Suppression in Critical Paths
- **File:** `cmd/niac/runtime_services.go` line 118
- **Severity:** MEDIUM
- **Impact:** Errors discarded, debugging difficult
- **Fix Time:** 1 hour
- **Fix:** Log all errors, even during shutdown
```go
if err := rs.replay.Stop(); err != nil {
    log.Debugf("Error stopping replay during shutdown: %v", err)
}
```

#### #107 - Optional API Authentication Allows Unauthenticated Access
- **File:** `pkg/api/server.go` line ~259
- **Severity:** MEDIUM
- **Impact:** Running without token exposes all endpoints
- **Fix Time:** 2 hours
- **Fix:** Warn users, add `--insecure` flag
```go
if s.cfg.Token == "" {
    log.Warn("⚠️  API running without authentication! Use --api-token or set NIAC_API_TOKEN")
}
```

#### #108 - Config File Update Accepts Arbitrary YAML Paths
- **File:** `pkg/api/server.go` line 454
- **Severity:** MEDIUM
- **Impact:** Arbitrary file paths in config could access sensitive files
- **Fix Time:** 2-3 hours
- **Fix:** Validate walk file paths stay within safe directories

#### #109 - No CSRF Protection in API
- **File:** `webui/src/api/client.ts`, `pkg/api/server.go`
- **Severity:** MEDIUM
- **Impact:** Low risk (same-origin policy protects)
- **Fix Time:** 3-4 hours (if needed)
- **Fix:** Implement CSRF tokens if adding cookies/sessions

#### #110 - SNMP Community Strings Stored in Plain Text
- **File:** `pkg/config/config.go` lines 108-109
- **Severity:** MEDIUM
- **Impact:** Sensitive data in plain text config files
- **Fix Time:** 2-3 hours (docs), 4-6 hours (encryption)
- **Fix:** Document security implications, recommend file permissions

#### #111 - Request Body Size Limit Not Enforced
- **File:** `pkg/api/server.go`
- **Severity:** MEDIUM
- **Impact:** DoS via large config uploads
- **Fix Time:** 1-2 hours
- **Fix:** Add MaxBytesReader to all handlers
```go
func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
    // ... rest of handler
}
```

### WebUI Improvements (3)

#### #112 - API Polling Intervals Should Be Configurable
- **Files:** `webui/src/components/*.tsx`
- **Severity:** MEDIUM
- **Impact:** Hard-coded refresh intervals
- **Fix Time:** 2-3 hours
- **Fix:** Add configuration UI for polling intervals

#### #125 - WebUI Component Re-Render Optimization
- **Files:** `webui/src/components/*.tsx`
- **Severity:** MEDIUM
- **Impact:** Unnecessary re-renders impact performance
- **Fix Time:** 3-4 hours
- **Fix:** Add useMemo, useCallback, React.memo optimizations

#### #126 - Large Device Lists Need Virtual Scrolling
- **Files:** `webui/src/`
- **Severity:** MEDIUM
- **Impact:** Slow rendering with >100 devices
- **Fix Time:** 4-6 hours
- **Fix:** Implement react-window for virtualization

### Documentation (2)

#### #113 - Add Docker/Kubernetes Deployment Best Practices Documentation
- **File:** New `docs/DOCKER_K8S_DEPLOYMENT.md`
- **Severity:** MEDIUM
- **Impact:** Missing deployment guidance
- **Fix Time:** 2-4 hours
- **Content:**
  - Example docker-compose.yml
  - Kubernetes deployment manifests
  - Volume mount recommendations
  - Security context configs

#### #114 - Add Performance Tuning Guide
- **File:** New `docs/PERFORMANCE_TUNING.md`
- **Severity:** MEDIUM
- **Impact:** Missing optimization guidance
- **Fix Time:** 3-4 hours
- **Content:**
  - Optimal device counts
  - Advertisement interval tuning
  - Memory/CPU planning
  - Scaling strategies

### Quick Fixes (2)

#### #115 - Update ARCHITECTURE.md Version Metadata
- **File:** `docs/ARCHITECTURE.md` lines 750-751
- **Severity:** MEDIUM
- **Impact:** Outdated version references
- **Fix Time:** 2 minutes
- **Fix:** Update to v2.3.1 and current date

#### #124 - Channel Buffer Sizes May Cause Blocking
- **File:** `pkg/protocols/stack.go` lines 85-86
- **Severity:** MEDIUM
- **Impact:** Packet drops under high load
- **Fix Time:** 3-4 hours
- **Fix:** Make buffer size configurable, add monitoring

---

## 📊 Summary Statistics

| Priority | Count | Time Range | Recommended Release |
|----------|-------|------------|---------------------|
| 🟠 HIGH | 6 | 6.5-11 hrs | v2.3.2 or v2.4.0 |
| 🟡 MEDIUM | 13 | 35-51 hrs | v2.4.0 - v2.5.0 |
| **TOTAL** | **19** | **41.5-62 hrs** | |

---

## 🎯 Recommended Action Plan

### Phase 1: Quick Wins (2.5 hours) - This Week
Fix the easiest high-impact issues:

1. **#103** - Update version badge (2 min)
2. **#115** - Update ARCHITECTURE.md metadata (2 min)
3. **#100** - Fix timing attack (30 min)
4. **#106** - Add error logging (1 hr)

**Result:** v2.3.2 patch release

---

### Phase 2: Security Hardening (7-9 hours) - Next Sprint
Address remaining HIGH security issues:

1. **#99** - Add HTTP timeouts (1-2 hrs)
2. **#101** - Move token to env var (1 hr)
3. **#102** - Add security headers (2-3 hrs)
4. **#98** - Fix goroutine leaks (2-4 hrs)

**Result:** v2.4.0 minor release

---

### Phase 3: API Improvements (12-18 hours) - Sprint +2
Enhance API robustness:

1. **#104** - API rate limiting (4-6 hrs)
2. **#105** - Error response format (2-3 hrs)
3. **#111** - Request size limits (1-2 hrs)
4. **#107** - Auth warnings (2 hrs)
5. **#108** - Config path validation (2-3 hrs)

**Result:** Part of v2.4.0 or v2.5.0

---

### Phase 4: WebUI & Docs (15-25 hours) - Sprint +3
Polish user experience:

1. **#113** - Docker/K8s docs (2-4 hrs)
2. **#114** - Performance guide (3-4 hrs)
3. **#112** - Configurable polling (2-3 hrs)
4. **#125** - Re-render optimization (3-4 hrs)
5. **#126** - Virtual scrolling (4-6 hrs)
6. **#110** - SNMP docs (2-3 hrs)

**Result:** v2.5.0

---

### Phase 5: Performance (6-8 hours) - Future
Optimization work:

1. **#124** - Channel buffer tuning (3-4 hrs)
2. **#109** - CSRF tokens (3-4 hrs, if needed)

**Result:** v2.6.0

---

## 🔄 Continuous Improvement

After fixing HIGH issues, you'll have:

✅ Zero critical vulnerabilities
✅ Zero high-priority security issues
✅ Production-ready deployment
✅ Solid foundation for features

Then you can tackle MEDIUM issues incrementally across multiple releases.

---

## 📈 Progress Tracking

Use GitHub Projects or Milestones:

- **Milestone v2.3.2** - Quick wins (#103, #115, #100, #106)
- **Milestone v2.4.0** - Security hardening (#98, #99, #101, #102)
- **Milestone v2.5.0** - API improvements + WebUI polish
- **Milestone v2.6.0** - Performance optimization

---

## 🎖️ Priority Recommendations

**Must Fix Before Production:**
- #98 - Goroutine leaks (memory issues)
- #99 - HTTP timeouts (DoS protection)
- #100 - Timing attacks (security)
- #101 - Token exposure (security)
- #102 - Security headers (web security)

**Should Fix Soon:**
- #104 - Rate limiting (abuse prevention)
- #107 - Auth warnings (security awareness)
- #111 - Request limits (DoS protection)

**Nice to Have:**
- #105, #112, #113, #114, #125, #126 (UX improvements)
- #106, #108, #110, #124 (code quality)

---

## 📞 Next Steps

1. Review this summary with team
2. Decide on v2.3.2 vs v2.4.0 scope
3. Create GitHub milestones
4. Assign issues to developers
5. Schedule fixes across sprints

---

**Current Status:** ✅ Critical issues resolved, ready for HIGH priority work

*Generated: November 14, 2025*
