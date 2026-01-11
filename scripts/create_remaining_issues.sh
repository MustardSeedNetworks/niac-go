#!/bin/bash

# Medium Priority Issues (continuing from #9)

gh issue create --title "[MEDIUM] Standardize API Error Response Format" --label "enhancement" --body "## Severity
🟡 **MEDIUM** - Fix Next Sprint

## Location
- **File:** \`pkg/api/server.go\`
- **Lines:** Various error responses

## Description
API error responses are inconsistent - some return plain text, others JSON, without a standardized format for error codes and details.

## Current Issues
- Mixed plain text and JSON error responses
- No error codes for client-side handling
- No structured validation errors
- Difficult for clients to parse errors programmatically

## Recommended Format
\`\`\`json
{
  \"error\": \"validation_error\",
  \"message\": \"Config validation failed\",
  \"details\": [{
    \"field\": \"devices[0].name\",
    \"issue\": \"required\"
  }],
  \"request_id\": \"abc-123\"
}
\`\`\`

## Fix Priority
**MEDIUM**

## Estimated Fix Time
2-3 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 5.2"

gh issue create --title "[MEDIUM] Silent Error Suppression in Critical Paths" --label "bug" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`cmd/niac/runtime_services.go\`
- **Line:** 118

## Description
Errors discarded with blank identifiers during shutdown, making debugging difficult.

## Vulnerable Code
\`\`\`go
_, _ = rs.replay.Stop()  // Errors discarded
\`\`\`

## Recommended Fix
\`\`\`go
if err := rs.replay.Stop(); err != nil {
    log.Debugf(\"Error stopping replay during shutdown: %v\", err)
}
\`\`\`

## Estimated Fix Time
1 hour

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 1.1"

gh issue create --title "[MEDIUM] Channel Buffers May Cause Blocking Under High Load" --label "bug,performance" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`pkg/protocols/stack.go\`
- **Lines:** 85-86

## Code
\`\`\`go
sendQueue: make(chan *Packet, 1000)
recvQueue: make(chan *Packet, 1000)
\`\`\`

## Risk
Under high load (>1000 packets/sec), buffers fill causing blocking and dropped packets.

## Recommended Fix
- Monitor queue depth
- Log warnings when >80% full
- Consider backpressure handling
- Make buffer size configurable

## Estimated Fix Time
3-4 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 1.2"

gh issue create --title "[MEDIUM] Optional API Authentication Allows Unauthenticated Access" --label "bug" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`pkg/api/server.go\`
- **Line:** ~259

## Code
\`\`\`go
if s.cfg.Token == \"\" {
    next(w, r)
    return
}
\`\`\`

## Risk
Running without token exposes all API endpoints (stats, config, devices) without authentication.

## Recommended Fix
- Warn users when running without authentication
- Consider requiring authentication by default
- Add \`--insecure\` flag to explicitly allow unauthenticated mode

## Estimated Fix Time
2 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 2.1"

gh issue create --title "[MEDIUM] Config File Update Accepts Arbitrary YAML Paths" --label "bug" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`pkg/api/server.go\`
- **Line:** 454

## Description
Configuration updates don't validate that walk file paths remain within safe directories.

## Risk
Arbitrary file paths in config could reference sensitive files outside intended directories.

## Recommended Fix
Validate walk file paths stay within allowed directories before accepting config.

## Estimated Fix Time
2-3 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 2.2"

gh issue create --title "[MEDIUM] No CSRF Protection in API" --label "enhancement" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`webui/src/api/client.ts\`
- **File:** \`pkg/api/server.go\`

## Description
API lacks CSRF protection, though risk is low due to same-origin policy.

## Recommendation
If adding cookies/sessions in future, implement CSRF tokens using double-submit cookie pattern.

## Estimated Fix Time
3-4 hours (if needed)

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 2.8"

gh issue create --title "[MEDIUM] SNMP Community Strings Stored in Plain Text" --label "enhancement,documentation" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`pkg/config/config.go\`
- **Lines:** 108-109

## Description
SNMP community strings stored unencrypted in configuration files.

## Recommended Actions
1. Document security implications in README
2. Recommend file permission restrictions (chmod 600)
3. Consider encryption for sensitive values (future enhancement)

## Estimated Fix Time
2-3 hours (documentation)
4-6 hours (encryption implementation)

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 2.6"

gh issue create --title "[MEDIUM] Request Body Size Limit Not Enforced" --label "bug" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`pkg/api/server.go\`
- **Line:** MaxRequestBodySize defined but not enforced

## Recommended Fix
\`\`\`go
func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
    // ... rest of handler
}
\`\`\`

## Estimated Fix Time
1-2 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 5.3"

gh issue create --title "[MEDIUM] Large Device Lists May Cause Slow WebUI Rendering" --label "enhancement,performance" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`webui/src/\` (component rendering)

## Description
Device lists with >100 items may render slowly without virtualization.

## Recommended Fix
Implement virtual scrolling using \`react-window\` or \`react-virtualized\`:

\`\`\`bash
npm install react-window
\`\`\`

## Estimated Fix Time
4-6 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 4.5"

gh issue create --title "[MEDIUM] API Polling Intervals Should Be Configurable" --label "enhancement" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`webui/src/components/ErrorInjectionPanel.tsx\` (line 20)
- **File:** \`webui/src/components/ReplayControlPanel.tsx\` (line 22)

## Description
Hard-coded refresh intervals (2s, 5s) should be user-configurable.

## Current Code
\`\`\`typescript
{ intervalMs: 2000 }  // Hard-coded
{ intervalMs: 5000 }  // Hard-coded
\`\`\`

## Recommended Fix
Add configuration settings in WebUI for polling intervals.

## Estimated Fix Time
2-3 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 4.5"

gh issue create --title "[MEDIUM] Component Re-Render Optimization Needed" --label "enhancement,performance" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`webui/src/\` (React components)

## Description
Need to verify and optimize component re-renders for performance.

## Recommended Actions
1. Profile with React DevTools
2. Add useMemo/useCallback for expensive operations
3. Implement React.memo for pure components

## Estimated Fix Time
3-4 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 4.1"

gh issue create --title "[MEDIUM] Add Docker/Kubernetes Deployment Best Practices Documentation" --label "documentation,enhancement" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`README.md\` mentions Docker/K8s but minimal docs
- **File:** \`deploy/kubernetes/README.md\` (minimal content)

## Missing Documentation
- Example docker-compose.yml with best practices
- Kubernetes deployment manifests
- Volume mount recommendations
- Security context configurations
- Resource limits and requests

## Estimated Fix Time
2-4 hours

## References
- DOCUMENTATION_REVIEW.md, Section 3"

gh issue create --title "[MEDIUM] Add Performance Tuning Guide" --label "documentation,enhancement" --body "## Severity
🟡 **MEDIUM**

## Description
Need comprehensive performance tuning guide covering:
- Optimal device counts per instance
- Advertisement interval tuning
- Traffic pattern optimization
- Memory and CPU resource planning
- Scaling strategies

## Location
New file: \`docs/PERFORMANCE_TUNING.md\`

## Estimated Fix Time
3-4 hours

## References
- DOCUMENTATION_REVIEW.md, Section 3"

gh issue create --title "[MEDIUM] WebUI Accessibility Improvements Needed" --label "enhancement,accessibility" --body "## Severity
🟡 **MEDIUM**

## Location
- **Files:** \`webui/src/components/*.tsx\`

## Required Improvements
1. Add ARIA labels on interactive elements
2. Test keyboard navigation
3. Ensure semantic HTML (buttons not divs)
4. Verify color contrast ratios (WCAG AA: 4.5:1)
5. Add focus indicators

## Testing Tools
- axe DevTools
- Lighthouse accessibility audit
- WAVE browser extension

## Estimated Fix Time
4-6 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 4.6"

gh issue create --title "[MEDIUM] Update ARCHITECTURE.md Version Metadata" --label "documentation" --body "## Severity
🟡 **MEDIUM**

## Location
- **File:** \`docs/ARCHITECTURE.md\`
- **Lines:** 750-751

## Current (Outdated)
\`\`\`
Last Updated: January 8, 2025
Version: v1.21.3
\`\`\`

## Should Be
\`\`\`
Last Updated: November 14, 2025
Version: v2.3.0
\`\`\`

## Estimated Fix Time
2 minutes

## References
- DOCUMENTATION_REVIEW.md, Section 6"

# Low Priority Issues

gh issue create --title "[LOW] Add Godoc Comments for Utility Functions" --label "documentation" --body "## Severity
🔵 **LOW**

## Location
Utility functions like \`padRight()\`, \`formatDuration()\` in main.go

## Description
Some exported functions lack godoc comments.

## Recommendation
Add brief godoc comments for all exported functions.

## Estimated Fix Time
1-2 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 1.5"

gh issue create --title "[LOW] Add OpenAPI/Swagger Specification" --label "enhancement,documentation" --body "## Severity
🔵 **LOW**

## Description
Generate OpenAPI 3.0 specification for API endpoints.

## Benefits
- Auto-generate client libraries
- Interactive API documentation
- Better API discoverability

## Tools
- swaggo/swag for Go annotation-based generation
- Or manually maintain openapi.yaml

## Estimated Fix Time
6-8 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 5.1"

gh issue create --title "[LOW] Add Request Tracing IDs" --label "enhancement" --body "## Severity
🔵 **LOW**

## Description
Add unique request IDs for log correlation and debugging.

## Implementation
\`\`\`go
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        ctx := context.WithValue(r.Context(), \"request_id\", requestID)
        w.Header().Set(\"X-Request-ID\", requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
\`\`\`

## Estimated Fix Time
2-3 hours

## References
- CODE_REVIEW_COMPREHENSIVE.md, Low Priority Recommendations"

gh issue create --title "[LOW] Log Goroutine Count for Debugging" --label "enhancement" --body "## Severity
🔵 **LOW**

## Description
Add periodic logging of goroutine count for debugging long-running simulations.

## Implementation
\`\`\`go
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        log.Debugf(\"Active goroutines: %d\", runtime.NumGoroutine())
    }
}()
\`\`\`

## Estimated Fix Time
30 minutes

## References
- CODE_REVIEW_COMPREHENSIVE.md, Section 3.2"

gh issue create --title "[LOW] Add FAQ Section to Documentation" --label "documentation,enhancement" --body "## Severity
🔵 **LOW**

## Description
Create FAQ section combining common troubleshooting questions.

## Location
New file: \`docs/FAQ.md\`

## Content Ideas
- Common configuration errors
- Performance questions
- Deployment best practices
- Comparison with other tools

## Estimated Fix Time
2-3 hours

## References
- DOCUMENTATION_REVIEW.md, Section 12"

gh issue create --title "[LOW] Add WebUI Dedicated Documentation with Screenshots" --label "documentation,enhancement" --body "## Severity
🔵 **LOW**

## Description
Create dedicated WebUI documentation with screenshots and walkthrough.

## Location
New file: \`docs/WEBUI_GUIDE.md\`

## Content
- Feature overview with screenshots
- Page-by-page walkthrough
- Common workflows
- Tips and tricks

## Estimated Fix Time
3-4 hours

## References
- DOCUMENTATION_REVIEW.md, Section 3"

gh issue create --title "[LOW] Add More API Usage Examples (Python/curl)" --label "documentation,enhancement" --body "## Severity
🔵 **LOW**

## Description
Expand API documentation with practical examples in multiple languages.

## Examples Needed
- Python client library usage
- curl command examples for all endpoints
- JavaScript/TypeScript examples
- Error handling examples

## Estimated Fix Time
2-3 hours

## References
- DOCUMENTATION_REVIEW.md, Section 3"

gh issue create --title "[LOW] Clarify Go Version Requirement (1.24.0 vs 1.24+)" --label "documentation" --body "## Severity
🔵 **LOW**

## Location
- README.md badge shows \"1.24+\"
- go.mod specifies \"1.24.0\"

## Recommendation
Decide on one approach and use consistently:
- If 1.24.0 minimum: Update badge to \"1.24.0+\"
- If any 1.24.x works: Keep badge as is

## Estimated Fix Time
5 minutes

## References
- DOCUMENTATION_REVIEW.md, Section 2"

echo "All 39 issues created successfully!"
