# NIAC-Go Security Fixes Checklist

## IMMEDIATE FIXES (Before Next Release)

### [ ] 1. Fix Path Traversal in collectFiles()
**File:** `pkg/api/server.go:1196-1240`

**Current Code:**
```go
err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    // No validation that path stays within root
    entries = append(entries, FileEntry{Path: path, ...})
})
```

**Fix:**
```go
// Resolve canonical path
rootAbs, err := filepath.Abs(root)
if err != nil {
    return nil, err
}

var entries []FileEntry
err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return nil
    }
    if d.IsDir() {
        return nil
    }
    
    // Ensure path is within root
    pathAbs, _ := filepath.Abs(path)
    if !strings.HasPrefix(pathAbs, rootAbs + string(os.PathSeparator)) {
        return nil // Skip files outside root
    }
    
    // ... rest of validation
})
```

**Testing:**
- [ ] Test with symlink pointing outside directory
- [ ] Test with ../../../etc/passwd in path
- [ ] Test with directory traversal in walk

---

### [ ] 2. Fix Symlink Attack in Replay File Handling
**File:** `pkg/api/server.go:1036-1047`

**Current Code:**
```go
abs, err := filepath.Abs(req.File)
if err != nil {
    return req, fmt.Errorf("resolve path: %w", err)
}
info, err := os.Stat(abs)
```

**Fix:**
```go
// Resolve path with symlink checking
abs, err := filepath.Abs(req.File)
if err != nil {
    return req, fmt.Errorf("resolve path: %w", err)
}

// Resolve symlinks
canonical, err := filepath.EvalSymlinks(abs)
if err != nil {
    return req, fmt.Errorf("cannot resolve symlinks: %w", err)
}

// Verify canonical path is within allowed directories
allowedDirs := []string{
    filepath.Dir(s.cfg.ConfigPath), // Config directory
    os.TempDir(),                    // Temp directory for uploads
}

allowed := false
for _, dir := range allowedDirs {
    dirAbs, _ := filepath.Abs(dir)
    if strings.HasPrefix(canonical, dirAbs + string(os.PathSeparator)) {
        allowed = true
        break
    }
}

if !allowed {
    return req, fmt.Errorf("file path outside allowed directories: %s", canonical)
}

// Use canonical path instead of abs
req.File = canonical

// Verify file exists and is accessible
info, err := os.Stat(req.File)
if err != nil {
    return req, fmt.Errorf("stat %s: %w", req.File, err)
}
if info.IsDir() {
    return req, fmt.Errorf("%s is a directory", req.File)
}
```

**Testing:**
- [ ] Test with symlink to /etc/passwd
- [ ] Test with symlink to system files
- [ ] Test with valid symlink in allowed dir

---

### [ ] 3. Enforce Request Body Size Limits
**File:** `pkg/api/server.go` (Start method, around line 174)

**Current Code:**
```go
s.httpServer = &http.Server{
    Addr:    s.cfg.Addr,
    Handler: mux,
}
```

**Fix:**
```go
// Create middleware to enforce request body limits
limitedMux := http.NewServeMux()
for pattern, handler := range mux.Handler {
    limitedMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
        // Enforce body size limit
        r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
        handler.ServeHTTP(w, r)
    })
}

s.httpServer = &http.Server{
    Addr:    s.cfg.Addr,
    Handler: limitedMux,
    // Add timeouts
    ReadTimeout:    15 * time.Second,
    WriteTimeout:   15 * time.Second,
    IdleTimeout:    60 * time.Second,
    MaxHeaderBytes: 1 << 20, // 1MB
}
```

**Testing:**
- [ ] Test with 2MB payload (should reject)
- [ ] Test with 900KB payload (should accept)
- [ ] Test with malformed base64

---

### [ ] 4. Fix Token Timing Attack
**File:** `pkg/api/server.go:270`

**Current Code:**
```go
if token != s.cfg.Token {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

**Fix:**
```go
import "crypto/subtle"

// ... in auth function:
if !constantTimeCompare(token, s.cfg.Token) {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}

// Helper function
func constantTimeCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

**Testing:**
- [ ] Verify token validation still works
- [ ] Test with empty token
- [ ] Test with wrong token

---

## HIGH PRIORITY FIXES (Before v2.2.0)

### [ ] 5. Move API Token to Environment Variable
**File:** `cmd/niac/root.go:65`

**Current Code:**
```go
rootCmd.PersistentFlags().StringVar(&servicesOpts.apiToken, "api-token", "", "Bearer token required for API/Web UI access")
```

**Fix:**
```go
// Initialize from environment
func resolveServiceDefaults() {
    if servicesOpts.apiToken == "" {
        servicesOpts.apiToken = os.Getenv("NIAC_API_TOKEN")
    }
}

// Keep flag for override, but prefer env var
rootCmd.PersistentFlags().StringVar(&servicesOpts.apiToken, "api-token", "", "Bearer token (env: NIAC_API_TOKEN)")
```

**Documentation:**
- [ ] Update README with NIAC_API_TOKEN env var
- [ ] Document security best practices
- [ ] Add warning when token is empty

---

### [ ] 6. Add Content-Security-Policy Header
**File:** `pkg/api/server.go:38-44`

**Current Code:**
```go
func addSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
}
```

**Fix:**
```go
func addSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    
    // Content-Security-Policy
    csp := "default-src 'self'; " +
           "script-src 'self' 'wasm-unsafe-eval'; " +
           "style-src 'self' 'unsafe-inline'; " +
           "img-src 'self' data:; " +
           "font-src 'self'; " +
           "connect-src 'self'; " +
           "frame-ancestors 'none';"
    w.Header().Set("Content-Security-Policy", csp)
    
    // Permissions-Policy
    w.Header().Set("Permissions-Policy", 
        "geolocation=(), microphone=(), camera=(), payment=()")
    
    // Referrer-Policy
    w.Header().Set("Referrer-Policy", "no-referrer")
}
```

**Testing:**
- [ ] Verify headers present in all responses
- [ ] Test CSP with various content types
- [ ] Verify no CSP violations in browser console

---

### [ ] 7. Add HSTS Support (with configuration)
**File:** `pkg/api/server.go:38-44`

**Fix:**
```go
type ServerConfig struct {
    // ... existing fields ...
    EnableHSTS bool       // New field
    HSTPSMaxAge int       // Default 31536000 (1 year)
}

func addSecurityHeaders(w http.ResponseWriter, enableHSTS bool, maxAge int) {
    // ... existing headers ...
    if enableHSTS {
        w.Header().Set("Strict-Transport-Security", 
            fmt.Sprintf("max-age=%d; includeSubDomains; preload", maxAge))
    }
}
```

**Configuration:**
- [ ] Add --enable-hsts flag (false by default)
- [ ] Add HST_MAX_AGE env var
- [ ] Document HTTPS requirement for HSTS

---

## MEDIUM PRIORITY FIXES (Next Sprint)

### [ ] 8. Implement API Rate Limiting
**New File:** `pkg/api/ratelimit.go`

**Implementation:**
```go
package api

import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
    }
}

func (rl *RateLimiter) Limit(ip string) bool {
    rl.mu.RLock()
    limiter, exists := rl.limiters[ip]
    rl.mu.RUnlock()
    
    if !exists {
        limiter = rate.NewLimiter(rate.Every(time.Minute/60), 1) // 60 req/min
        rl.mu.Lock()
        rl.limiters[ip] = limiter
        rl.mu.Unlock()
    }
    
    return limiter.Allow()
}
```

**Integration:**
- [ ] Wrap API handlers with rate limit middleware
- [ ] Return 429 Too Many Requests
- [ ] Make limit configurable

**Testing:**
- [ ] Test with rapid requests
- [ ] Test with multiple IPs
- [ ] Verify cleanup of old limiters

---

### [ ] 9. Standardize Error Response Format
**File:** `pkg/api/server.go`

**New Response Type:**
```go
type ErrorResponse struct {
    Code    string              `json:"code"`
    Message string              `json:"message"`
    Details []ErrorDetail       `json:"details,omitempty"`
    Request string              `json:"request,omitempty"`
}

type ErrorDetail struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

// Helper function
func (s *Server) writeError(w http.ResponseWriter, code string, message string, details ...ErrorDetail) {
    // Determine status code from error code
    statusCode := 400
    if strings.HasPrefix(code, "not_found") {
        statusCode = 404
    } else if strings.HasPrefix(code, "unauthorized") {
        statusCode = 401
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(ErrorResponse{
        Code:    code,
        Message: message,
        Details: details,
    })
}
```

**Testing:**
- [ ] Update all error responses
- [ ] Test with curl for proper format
- [ ] Update documentation

---

## TESTING ADDITIONS

### [ ] 10. Add Security Test Cases
**New File:** `pkg/api/server_security_test.go`

```go
func TestPathTraversalAttack(t *testing.T) {
    // Test ../../../etc/passwd
    // Test symlinks
    // Test mixed traversal
}

func TestSymlinkAttack(t *testing.T) {
    // Create symlink to outside directory
    // Verify it's rejected
}

func TestAuthenticationBypass(t *testing.T) {
    // Test empty token
    // Test timing attack resistance
    // Test token in query param (should fail)
}

func TestRequestBodyLimits(t *testing.T) {
    // Test with oversized config
    // Test with oversized PCAP upload
    // Test with valid sizes
}

func TestRateLimiting(t *testing.T) {
    // Test rapid requests
    // Test per-IP isolation
    // Test limit reset
}
```

---

## DOCUMENTATION UPDATES

### [ ] 11. Update README with Security Info
- [ ] Add "Running in Production" section
- [ ] Document NIAC_API_TOKEN environment variable
- [ ] Warn about default "public" SNMP community string
- [ ] Recommend HTTPS setup
- [ ] Document file permission requirements

### [ ] 12. Create SECURITY.md
- [ ] Responsible disclosure policy
- [ ] Known limitations (default SNMP community, etc.)
- [ ] Best practices guide
- [ ] Deployment checklist

---

## COMPLETION CHECKLIST

**Week 1 (Immediate):**
- [ ] Fix path traversal (collectFiles)
- [ ] Fix symlink attack
- [ ] Add request body limits
- [ ] Fix token timing attack

**Week 2 (High Priority):**
- [ ] Move token to env var
- [ ] Add security headers
- [ ] Add HSTS support
- [ ] Write security tests

**Week 3 (Medium Priority):**
- [ ] Implement rate limiting
- [ ] Standardize error responses
- [ ] Improve frontend accessibility
- [ ] Update documentation

**Week 4 (Review & Release):**
- [ ] Security audit review
- [ ] Penetration testing
- [ ] Final testing
- [ ] Release v2.2.0

---

## Success Criteria

- [ ] All IMMEDIATE fixes completed
- [ ] Security tests pass
- [ ] No critical vulnerabilities remain
- [ ] OWASP Top 10 addressed
- [ ] Code review approval from security team
- [ ] Documentation updated
- [ ] Deployment guide created

