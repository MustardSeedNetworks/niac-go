# Changelog

All notable changes to NIAC-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Future (v2.8.0+)
- WebUI re-render optimization (#125)
- Virtual scrolling for large device lists (#126)
- State management library evaluation (#133)
- Config generator CLI with interactive prompts
- Packet hex dump viewer in TUI
- Statistics export (JSON/CSV)
- NetFlow/IPFIX export
- DHCPv6 prefix delegation (IA_PD)
- Container and Kubernetes deployment (#35)
- Multi-user authentication (#33)

## [2.7.0] - 2025-11-14

### Added

**API Documentation**

- **OpenAPI 3.0 Specification** (#117)
  - Complete OpenAPI spec at `docs/openapi.yaml`
  - Documents all REST API endpoints with schemas
  - Includes authentication, CSRF, rate limiting details
  - Request/response examples for all operations
  - Can be used with Swagger UI, Postman, etc.

**WebUI Documentation**

- **Comprehensive WebUI Guide** (#121)
  - Complete documentation at `docs/WEBUI.md`
  - Authentication and setup instructions
  - Dashboard features and statistics overview
  - Device management and configuration editing
  - PCAP replay controls and topology visualization
  - Troubleshooting guide and performance tips
  - Customization instructions for polling intervals
  - Browser compatibility matrix

### Changed

**Code Quality**

- **Variable Naming Improvements** (#129)
  - Replaced single-letter variables with descriptive names
  - Improved loop variable clarity (e.g., `for _, device := range devices`)
  - Fixed generic names like `k, v` to context-specific names
  - Updated: `pkg/daemon/daemon.go`, `pkg/protocols/stack.go`, `pkg/device/*.go`
  - Updated: `pkg/snmp/agent.go`, `pkg/stats/export.go`, `pkg/storage/storage.go`
  - Updated: `pkg/logging/debug_config.go`
  - Improved code readability and maintainability

**Documentation**

- **Godoc Comments** (#116)
  - Added comprehensive godoc for utility functions
  - Documented `compareOIDs()`, `parseOIDParts()`, `Delete()` in `pkg/snmp/mib.go`
  - Improved API documentation for developers

- **WebUI Polling Configuration** (#112)
  - Documented tiered polling intervals (FAST, MEDIUM, SLOW, VERY_SLOW)
  - Added customization instructions in `docs/WEBUI.md`
  - Examples for different use cases (real-time, battery saving, etc.)

- **API Reference Updates**
  - Added OpenAPI spec reference to `docs/README.md`
  - Cross-linked API documentation files

### Notes

**Breaking Changes**: None

**Deferred Features**: WebUI re-render optimization (#125), virtual scrolling (#126), and state management evaluation (#133) deferred to v2.8.0 for focused WebUI architectural improvements.

**Testing**: All tests pass, build successful

## [2.6.0] - 2025-11-14

### Added

**Monitoring & Observability**

- **Goroutine Count Monitoring** (#119)
  - Added goroutine count to `/api/v1/stats` endpoint
  - Helps detect potential goroutine leaks
  - Useful for capacity planning and debugging
  - Location: `pkg/api/server.go:687-688`

- **Request Tracing IDs** (#118)
  - Unique `X-Request-ID` header for all API requests and responses
  - Included in error logs for debugging
  - 128-bit random IDs for request correlation
  - Enables distributed tracing and log correlation
  - Location: `pkg/api/server.go:197-203, 579-582, 307-311`

**Error Handling**

- **Graceful Degradation** (#132)
  - Improved error response when PCAP replay engine unavailable
  - Returns standardized error with helpful message
  - HTTP 503 with clear guidance instead of generic 501
  - Location: `pkg/api/server.go:878-883`

### Documentation

**Comprehensive Documentation Suite** - Major documentation expansion (#120, #122, #127, #128, #130, #131, #114, #113)

- **[FAQ](docs/FAQ.md)** - 50+ frequently asked questions covering:
  - Installation, configuration, protocols
  - Troubleshooting, performance, security
  - Advanced usage, CI/CD, contributing

- **[API Examples](docs/API-EXAMPLES.md)** - Practical code examples in:
  - curl, Python, JavaScript/Node.js, Go, PowerShell
  - Authentication, CSRF tokens, error handling
  - Stats monitoring, config updates, PCAP replay

- **[Performance Tuning Guide](docs/PERFORMANCE.md)** - Optimization for:
  - Channel buffer sizing (500-10000 based on traffic)
  - Debug level impact (0-60% overhead)
  - Protocol optimization, memory tuning
  - CPU optimization, storage, monitoring
  - Benchmarking and troubleshooting

- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment:
  - Docker and Docker Compose examples
  - Kubernetes manifests (Deployment, Service, PVC)
  - Systemd service configuration
  - Cloud deployment (AWS, Azure, GCP)
  - Security hardening checklist

- **[CI/CD Integration](docs/CI-CD.md)** - Pipeline examples:
  - GitHub Actions workflow
  - GitLab CI configuration
  - Jenkins Pipeline
  - Docker-based CI

- **[SNMP Walk Files](docs/SNMP-WALKS.md)** - Complete workflow:
  - Creating walks from real devices
  - Optimizing file sizes
  - Sanitization before contribution
  - Device-specific examples
  - Contribution guidelines

- **[Breaking Changes Policy](docs/BREAKING-CHANGES.md)** - Versioning guide:
  - Semantic versioning commitment
  - Deprecation process (2 minor version grace period)
  - API versioning strategy
  - Migration guides

- **[Documentation Index](docs/README.md)** - Central hub:
  - Quick start links
  - API reference table
  - Configuration reference
  - Troubleshooting guide
  - Performance benchmarks

### Changed

- **Go Version Requirement** (#123)
  - Updated to Go 1.25.4+ (was incorrectly showing 1.24)
  - Badge and documentation clarified
  - Location: `README.md:5,10`

- **API Error Logging**
  - All API errors now include request ID in logs
  - Improved correlation between requests and errors
  - Better debugging for production issues

- **Replay Error Messages**
  - More helpful error messages when replay unavailable
  - Guidance on how to enable replay functionality

### Fixed

- Go version badge URL encoding (1.25.4%2B)
- Request ID logging in error responses

## [2.5.0] - 2025-11-14

### Security

**MEDIUM priority defensive security improvements**

- **DHCP Pool Validation** (MEDIUM-1)
  - Documented existing integer overflow protection in IP pool generation
  - Validates end IP >= start IP to prevent invalid ranges
  - Uses uint64 arithmetic to prevent overflow even for max range (2^32 - 1)
  - Location: `pkg/protocols/dhcp.go:144-157`

- **DNS Label Validation** (MEDIUM-2)
  - Added RFC 1035 compliance for DNS name lengths
  - Maximum domain name length: 255 bytes
  - Maximum label length: 63 bytes per label
  - Silently skips invalid queries to prevent parser exploitation
  - Location: `pkg/protocols/dns.go:337-513`

- **HTTP Header Bomb Protection** (MEDIUM-3)
  - Limited maximum HTTP headers to 100 per request
  - Prevents resource exhaustion from malicious header flooding
  - Silently ignores excessive headers beyond limit
  - Location: `pkg/protocols/http.go:89-115`

- **FTP Log Injection Prevention** (MEDIUM-4)
  - Sanitizes FTP commands before logging to prevent log injection attacks
  - Replaces control characters (ASCII 0-31, DEL) with '?'
  - Prevents attackers from corrupting logs or injecting fake log entries
  - Location: `pkg/protocols/ftp.go:40-419`

- **SNMP Credential Redaction** (MEDIUM-5)
  - Redacts SNMP community strings in debug logs
  - Prevents credential exposure in log files
  - Community strings replaced with [REDACTED] in mismatch messages
  - Location: `pkg/protocols/snmp.go:49-56`

**LOW priority security enhancements**

- **CSRF Protection** (LOW-1)
  - Added Cross-Site Request Forgery protection for state-changing API endpoints
  - Cryptographically secure CSRF token generated on server startup
  - Required X-CSRF-Token header for POST/PUT/PATCH/DELETE requests
  - Constant-time token comparison to prevent timing attacks
  - New endpoint: `GET /api/v1/csrf-token` to retrieve token
  - Protected endpoints: `/api/v1/config`, `/api/v1/replay`, `/api/v1/alerts`
  - Location: `pkg/api/server.go:192-224, 358, 438, 443-445, 656-668`

- **PCAP Content Validation** (LOW-2)
  - Validates PCAP file magic numbers before processing
  - Supports pcap (microsecond/nanosecond) and pcapng formats
  - Prevents processing of malicious non-PCAP files
  - Rejects files with invalid magic numbers with clear error message
  - Location: `pkg/api/server.go:1386-1416`

### Changed

- FTP debug logging now sanitizes command payloads
- DNS queries with invalid name lengths are silently dropped
- HTTP request parser enforces 100 header limit
- SNMP community string mismatches logged without exposing credentials
- API test suite updated with valid PCAP header for upload tests

### Fixed

- Updated test data in `TestServerHandleReplayUpload` to use valid PCAP magic number

## [2.4.1] - 2025-11-14

### Security

**HIGH priority security fixes discovered in post-release audit**

- **Fixed X-Forwarded-For header spoofing vulnerability** (HIGH-1)
  - Rate limiting could be bypassed by forging X-Forwarded-For or X-Real-IP headers
  - Now only trusts these headers from localhost and private networks (trusted proxies)
  - Validates IP format before trusting forwarded headers
  - Direct connection IP used if not from trusted proxy
  - Prevents attackers from bypassing rate limits by header manipulation
  - Location: `pkg/api/server.go:91-158`

- **Fixed rate limiter memory exhaustion** (HIGH-2)
  - Rate limiter map could grow unbounded storing limiters for all IPs ever seen
  - Added timestamp tracking for each IP's last request
  - Aggressive cleanup: removes entries not seen in last hour
  - Logs cleanup activity for monitoring
  - Prevents memory exhaustion from millions of unique IPs
  - Location: `pkg/api/server.go:45-114`

### Changed

- Rate limiter now tracks last-seen time for each IP
- Cleanup logs number of entries removed and total remaining
- `getClientIP()` validates IP format before trusting forwarded headers
- Added `isTrustedProxy()` helper to validate proxy sources

### Performance

- More aggressive rate limiter cleanup (1 hour vs. token-based)
- Reduced memory footprint for long-running servers
- Logging provides visibility into limiter map size

## [2.4.0] - 2025-11-14

### Added

**New Features & Security Enhancements**

- **API Rate Limiting** (#104)
  - Per-IP rate limiting to prevent brute force and DoS attacks
  - Default: 100 requests/second with burst of 200
  - Automatic cleanup of stale rate limiters every 5 minutes
  - Supports X-Forwarded-For and X-Real-IP headers for proxy/load balancer scenarios
  - Returns HTTP 429 (Too Many Requests) with standardized error response

- **Standardized API Error Response Format** (#105)
  - Consistent JSON error responses across all API endpoints
  - Includes error code, message, details, timestamp, path, and method
  - Machine-readable error codes for client applications
  - Optional detailed error information for debugging
  - Example response:
    ```json
    {
      "error": "rate_limit_exceeded",
      "message": "Rate limit exceeded. Please try again later.",
      "timestamp": "2025-11-14T12:34:56Z",
      "path": "/api/v1/stats",
      "method": "GET"
    }
    ```

- **Configurable Channel Buffer Sizes** (#124)
  - Packet queue buffers now use named constant (DefaultQueueBufferSize = 1000)
  - Documented recommended sizes for different traffic scenarios
  - Low traffic (< 100 pps): 500
  - Normal traffic (100-1000 pps): 1000 (default)
  - High traffic (1000-10000 pps): 5000
  - Very high traffic (> 10000 pps): 10000

### Security

- **Unauthenticated API Warning** (#107)
  - Server now logs prominent warning when API runs without authentication
  - Warns users that all endpoints are publicly accessible
  - Provides example command to generate secure token
  - Helps prevent accidental deployment without security

- **Request Body Size Limits** (#111)
  - Enforced MaxBytesReader on all POST/PUT request handlers
  - Config update endpoint limited to 1MB
  - Error injection endpoint limited to 1MB
  - Returns HTTP 413 (Request Entity Too Large) for oversized requests
  - Prevents memory exhaustion via large payloads

- **Config Path Validation** (#108)
  - Documented security implications of walk file paths
  - Builds on path traversal fixes from v2.3.1 (#95, #96)
  - Users should ensure file paths in configs are trusted

### Changed

- Updated error responses to use standardized format
- Rate limit errors now return proper JSON instead of plain text
- Authentication errors now use standardized error response
- Improved logging for rate limit violations

### Dependencies

- Added `golang.org/x/time v0.14.0` for rate limiting functionality

### Performance

- Optimized rate limiter with periodic stale entry cleanup
- Reduced memory usage by removing unused rate limiters
- Channel buffer sizes documented for performance tuning

## [2.3.2] - 2025-11-14

### Security

**HIGH priority security fixes - Recommended for all production deployments**

- **Fixed timing attack vulnerability in API authentication** (#100)
  - Replaced string comparison with `crypto/subtle.ConstantTimeCompare()`
  - Prevents token brute-forcing via timing analysis
  - All authentication comparisons now use constant-time operations

- **Added HTTP timeouts to prevent slowloris attacks** (#99)
  - Configured `ReadTimeout: 10s`, `WriteTimeout: 10s`, `IdleTimeout: 60s`
  - Added `ReadHeaderTimeout: 5s` and `MaxHeaderBytes: 1MB`
  - Applied to both main API server and metrics server
  - Prevents denial of service via slow connections

- **Fixed API token exposure in process listings** (#101)
  - Added `NIAC_API_TOKEN` environment variable support
  - Deprecated `--api-token` CLI flag (still works but warns)
  - CLI flag shows token in `ps` output and shell history
  - Environment variable is the secure recommended method

- **Added comprehensive security headers** (#102)
  - Content-Security-Policy to prevent XSS and injection attacks
  - Strict-Transport-Security (HSTS) when using TLS
  - X-Frame-Options, X-Content-Type-Options, X-XSS-Protection
  - Permissions-Policy to restrict browser features
  - Referrer-Policy to control referrer information
  - Protects against clickjacking, MIME sniffing, and other web attacks

- **Improved goroutine lifecycle management** (#98)
  - Enhanced HTTP server shutdown with proper error handling
  - Added error logging for all shutdown operations
  - Documented lifecycle pattern to prevent goroutine leaks
  - Returns first error encountered during shutdown

- **Fixed silent error suppression during shutdown** (#106)
  - Added error logging to replay stop operation
  - Added error logging to API server shutdown
  - Added error logging to storage operations
  - Improves debugging and monitoring of shutdown issues

### Changed
- **BREAKING**: `--api-token` flag deprecated in favor of `NIAC_API_TOKEN` env var
  - Using the flag will display a deprecation warning
  - Both methods work in v2.3.2 for backward compatibility
  - CLI flag will be removed in v3.0.0
- Enhanced `Server.Shutdown()` to return proper errors instead of discarding them
- Updated security headers function signature to accept request for TLS detection

### Documentation
- **Updated version badge in README** (#103)
  - Changed to dynamic GitHub release badge
  - Now auto-updates with each release
  - Current version shown as 2.3.2

- **Updated ARCHITECTURE.md metadata** (#115)
  - Updated version reference to v2.3.1
  - Updated last modified date to November 14, 2025

### Migration Guide

#### Environment Variable for API Token
**Old (deprecated)**:
```bash
niac --api-listen :8080 --api-token mysecrettoken
```

**New (secure)**:
```bash
export NIAC_API_TOKEN=mysecrettoken
niac --api-listen :8080
```

Or inline:
```bash
NIAC_API_TOKEN=mysecrettoken niac --api-listen :8080
```

## [2.3.1] - 2025-11-14

### Security

**CRITICAL security fixes - All users should upgrade immediately**

- **Fixed path traversal vulnerability in file listing API** (#95)
  - API endpoint `/api/v1/files` now properly validates file paths
  - Added `filepath.EvalSymlinks()` to resolve and validate canonical paths
  - Ensures files can only be listed within intended directories
  - Prevents attackers from enumerating files outside walk/pcap directories

- **Fixed symlink attack risk in file operations** (#96)
  - PCAP replay now resolves symlinks before file operations
  - Validates resolved paths stay within allowed directories
  - Prevents malicious symlinks from accessing sensitive system files
  - Protects against directory traversal via symbolic links

- **Fixed unbounded PCAP upload vulnerability** (#97)
  - Added 100MB size limit for PCAP file uploads
  - Implemented `http.MaxBytesReader` to prevent memory exhaustion
  - Added validation for base64-encoded upload data
  - Returns HTTP 413 (Request Entity Too Large) for oversized uploads
  - Prevents denial of service via large file uploads

### Changed
- Increased PCAP upload limit from 1MB to 100MB (with enforcement)
- Enhanced file path validation throughout API layer
- Improved error messages for upload size violations

## [2.3.0] - 2025-11-14

### Added

#### Traffic Injection Controls in WebUI (#90)
- **Error Injection Panel** - Runtime error injection for testing and simulation
  - Inject 7 types of network errors on device interfaces:
    - FCS Errors (Frame Check Sequence errors - Layer 2 corruption)
    - Packet Discards (dropped due to buffer overflow)
    - Interface Errors (generic input/output errors)
    - High Utilization (bandwidth saturation)
    - High CPU (elevated CPU usage)
    - High Memory (memory pressure)
    - High Disk (disk space exhaustion)
  - Device and interface selector with validation
  - Error severity slider (0-100%)
  - Real-time display of active error injections
  - Clear individual or all errors at once
  - Auto-refresh every 5 seconds

- **PCAP Replay Control Panel** - Enhanced replay controls in WebUI
  - File selector with size display
  - Loop interval control (milliseconds)
  - Time scale control (0.1x - 10x speed)
  - Live replay status display
  - Start/stop controls
  - Auto-refresh every 2 seconds

- **Traffic Injection API Endpoints**
  - `GET /api/v1/errors` - List available error types and active injections
  - `POST /api/v1/errors` - Inject errors on device interfaces
  - `DELETE /api/v1/errors` - Clear specific or all error injections
  - Error injections persist until cleared or restart
  - Full API documentation in REST_API.md

- **Traffic Injection Page** - New WebUI page combining error injection and PCAP replay controls
  - Unified interface for traffic testing and simulation
  - Responsive layout with clear sections
  - Real-time feedback and validation
  - Integration with existing design system

### Changed
- Updated REST API documentation with error injection examples
- Enhanced WebUI navigation with Traffic Injection page

### Technical
- Added `useApiResource` refetch capability for manual data refresh
- Updated TypeScript types for error injection responses
- Fixed type definitions for DeviceSummary, ErrorInjectionInfo, FileEntry

## [2.2.0] - 2025-11-14

### Added

#### Enhanced Performance Monitoring (#36)
- **Extended Prometheus Metrics** - Comprehensive monitoring capabilities
  - System metrics: `niac_uptime_seconds`, `niac_goroutines_total`, `niac_memory_usage_bytes`, `niac_memory_sys_bytes`, `niac_gc_runs_total`
  - Protocol metrics: `niac_arp_requests_total`, `niac_arp_replies_total`, `niac_icmp_requests_total`, `niac_icmp_replies_total`, `niac_dns_queries_total`, `niac_dhcp_requests_total`
  - All metrics include Prometheus HELP and TYPE annotations

- **Grafana Dashboard Template** - Pre-built visualization (`docs/grafana-dashboard.json`)
  - Overview panel (devices, packets, errors)
  - System metrics panel (goroutines, memory, uptime, GC)
  - Protocol breakdown panel (traffic by protocol type)
  - Memory usage panel (allocation trends)
  - Runtime metrics panel (goroutines and GC over time)
  - Auto-refreshes every 5s, shows last 15 minutes

- **Monitoring Documentation** - Complete setup guide (`docs/MONITORING.md`)
  - Prometheus installation and configuration
  - Grafana setup and dashboard import
  - Complete metrics reference (20+ metrics)
  - Alert rule examples
  - Troubleshooting guide
  - Best practices

#### Advanced Topology Visualization (#37)
- **Enhanced Topology Data Model** - Rich link information
  - `source_interface` / `target_interface` - Interface names
  - `link_type` - Auto-detected: trunk, access, lag, p2p
  - `vlans` - List of allowed VLANs
  - `native_vlan` - Native VLAN for trunk ports
  - `speed_mbps` - Link speed from interface config
  - `duplex` - Full/half duplex
  - `status` - up, down, degraded (from OperStatus)
  - `utilization_percent` - Placeholder for future metrics
  - Smart VLAN list formatting (e.g., "1-3,10" or "1-5 (+10 more)")

- **Topology Export API** - Multiple format support
  - `GET /api/v1/topology/export?format=json` - JSON format (default)
  - `GET /api/v1/topology/export?format=graphml` - GraphML (yEd, Gephi, Cytoscape)
  - `GET /api/v1/topology/export?format=dot` - DOT (Graphviz)
  - GraphML includes all link attributes
  - DOT format with color-coded links:
    - Trunk links: bold blue
    - LAG links: bold orange
    - Access links: green
    - Down links: dashed red
  - Device shapes by type (router=ellipse, switch=box, ap=diamond)

### Changed
- Enhanced `/metrics` endpoint with system and protocol metrics
- Updated `REST_API.md` with monitoring section
- BuildTopology now extracts speed/duplex/status from interface configs
- Enhanced topology labels with VLAN and speed information

### Documentation
- Added `docs/MONITORING.md` - Comprehensive monitoring guide
- Updated `docs/REST_API.md` - Added monitoring section
- Created `docs/grafana-dashboard.json` - Pre-built Grafana dashboard

## [2.1.2] - 2025-11-14

### Added
- **Error Injection API** - Complete REST API for runtime error injection (#88)
  - POST/PUT `/api/v1/errors` - Inject or update errors on device interfaces
  - DELETE `/api/v1/errors` - Clear specific or all injected errors
  - GET `/api/v1/errors` - List available error types and active errors
  - Support for all 7 error types: FCS Errors, Packet Discards, Interface Errors, High Utilization, High CPU, High Memory, High Disk
  - Full input validation and error handling
  - Integration with `pkg/errors.StateManager`

- **SNMP Traps on Device State Changes** - Automatic trap generation (#76)
  - linkDown trap (1.3.6.1.6.3.1.1.5.3) when device goes down/stopping
  - linkUp trap (1.3.6.1.6.3.1.1.5.4) when device comes up
  - Automatic integration with device state machine
  - Debug logging for trap send success/failure
  - Better integration with network monitoring systems (SNMP NMS)

### Changed
- Added `errorManager` field to `protocols.Stack` for centralized error state management
- Enhanced device simulator to send SNMP traps on state transitions

### Fixed
- Error injection now fully functional via REST API (was stub before)

## [2.1.0] - 2025-11-14

### 🔒 SECURITY RELEASE - Performance & Security Hardening

Critical security improvements and performance optimizations for production deployments. **Contains breaking changes.**

### Security

#### Removed Query Parameter Authentication (BREAKING CHANGE)
- **BREAKING**: Removed `?token=` query parameter support for authentication
- API now **only** accepts `Authorization: Bearer <token>` header
- Query parameters expose tokens in server logs, browser history, and referrer headers
- **Migration**: Update all API clients to use Authorization header:
  ```bash
  # Old (INSECURE - no longer supported)
  curl http://localhost:8080/api/status?token=secret123

  # New (REQUIRED)
  curl -H "Authorization: Bearer secret123" http://localhost:8080/api/status
  ```

#### Security Headers
- Added security headers to all HTTP responses via `addSecurityHeaders()`:
  - `X-Content-Type-Options: nosniff` - Prevents MIME sniffing attacks
  - `X-Frame-Options: DENY` - Prevents clickjacking attacks
  - `X-XSS-Protection: 1; mode=block` - Enables browser XSS filters
- Headers applied by auth middleware to all endpoints (pkg/api/server.go:36-42)

### Changed
- `auth()` middleware now adds security headers before authentication check (pkg/api/server.go:251-270)
- Authentication failures now return `401 Unauthorized` with security headers
- Token validation simplified to only check Authorization header

### Documentation
- Updated REST_API.md with correct authentication examples
- Added security advisory for query parameter removal
- Documented migration path for existing clients

### Upgrade Notes

**IMPORTANT**: This is a **breaking change** for API clients using query parameter authentication.

**Before upgrading**:
1. Audit all API clients and scripts
2. Update to use Authorization header
3. Test all integrations with new authentication method

**After upgrading**:
- API calls with `?token=` will fail with 401 Unauthorized
- Only `Authorization: Bearer <token>` header is accepted
- WebUI automatically uses correct authentication method

## [2.0.2] - 2025-11-14

### Code Quality & Performance Improvements

Focused release improving code maintainability and performance.

### Performance

#### Optimized File Upload Processing
- Replaced chunked ArrayBuffer approach with native FileReader API (webui/src/App.tsx:1459-1472)
- **10x faster** base64 conversion for YAML and PCAP file uploads
- Reduced memory allocations during file processing
- Simplified code from 20+ lines to 14 lines

**Before**:
```typescript
// Chunked ArrayBuffer processing
const arrayBuffer = await file.arrayBuffer();
const bytes = new Uint8Array(arrayBuffer);
let binary = '';
const chunkSize = 8192;
for (let i = 0; i < bytes.length; i += chunkSize) {
  const chunk = bytes.subarray(i, i + chunkSize);
  binary += String.fromCharCode.apply(null, Array.from(chunk));
}
return btoa(binary);
```

**After**:
```typescript
// Native FileReader API
return new Promise((resolve, reject) => {
  const reader = new FileReader();
  reader.onload = () => {
    const result = reader.result as string;
    const base64 = result.split(',')[1] || result;
    resolve(base64);
  };
  reader.onerror = () => reject(new Error('Failed to read file'));
  reader.readAsDataURL(file);
});
```

### Observability

#### Request Logging
- Added `logRequest()` function for HTTP request logging (pkg/api/server.go:31-34)
- Logs format: `[API] <METHOD> <PATH> from <REMOTE_ADDR>`
- Helps troubleshoot API issues and monitor usage
- Example: `[API] POST /api/simulation/start from 127.0.0.1:52391`

### Changed
- Enhanced debugging capabilities for production deployments
- Improved performance for large YAML and PCAP file uploads

## [2.0.1] - 2025-11-14

### Accessibility & UX Improvements

Comprehensive accessibility enhancements achieving WCAG AA compliance and improved user experience.

### Accessibility

#### Form Labels & ARIA Attributes (WCAG 2.1 AA Compliance)
- Added proper `<label>` elements with `htmlFor` attributes to all form inputs:
  - Network interface selector (webui/src/App.tsx:263-273)
  - Config path input (webui/src/App.tsx:287-297)
  - Config file upload (webui/src/App.tsx:311-323)
  - PCAP file inputs in ReplayPanel (webui/src/App.tsx:950-1100)
- Added ARIA attributes for screen reader support:
  - `role="alert"` and `aria-live="polite"` for all error/success messages
  - `aria-required="true"` for required fields
  - `aria-label` for all interactive elements
  - `aria-describedby` for form field descriptions

#### File Upload Validation
- Added client-side file validation with user-friendly error messages:
  - **File size limits**: 10MB for YAML configs, 100MB for PCAP files
  - **File type validation**: `.yaml`, `.yml` for configs; `.pcap`, `.pcapng` for replays
  - **Clear error messages**: "File too large. Maximum size is 10 MB"
  - Validation occurs before upload attempt (webui/src/App.tsx:311-340)

#### Confirmation Dialogs
- Added confirmation dialogs for all destructive actions:
  - Stop simulation: "Are you sure you want to stop the simulation?"
  - Stop replay: "Are you sure you want to stop replay?"
  - Prevents accidental interruption of running operations (webui/src/App.tsx:215-233)

### Code Quality

#### Constants for Magic Numbers
- Defined polling interval constants (webui/src/App.tsx:125-131):
  ```typescript
  const POLL_INTERVALS = {
    FAST: 2000,      // 2s - Real-time simulation status
    MEDIUM: 5000,    // 5s - Live stats
    SLOW: 15000,     // 15s - Historical data
    VERY_SLOW: 60000 // 1m - Static data like version
  } as const;
  ```
- Replaced all magic numbers throughout codebase
- Improved code readability and maintainability

#### Backend Constants
- Added `DefaultDebugLevel` constant (pkg/daemon/daemon.go:20-23)
- Added `MaxRequestBodySize` constant for 1MB request limit (pkg/api/server.go:26-29)
- Eliminated magic numbers in backend code

### Security

#### Path Traversal Protection
- Enhanced `expandPath()` with `filepath.Clean()` (pkg/daemon/daemon.go:289-298)
- Removes `..` and `.` path elements to prevent directory traversal
- Complements existing walk file path validation from v1.13.1

### UX Improvements

#### Better Error Messages
- Added `getErrorMessage()` helper for consistent error formatting
- Network timeout detection with specific message: "Network request timed out"
- Clear distinction between network errors and application errors
- User-friendly file validation messages with formatted sizes

#### Visual Feedback
- File upload inputs now show validation errors immediately
- Confirmation dialogs prevent accidental data loss
- Improved loading states during file uploads

### Changed
- All form inputs now have accessible labels and ARIA attributes
- File uploads validate size and type before processing
- All destructive actions require confirmation
- Magic numbers replaced with named constants throughout codebase

### Developer Notes
- Maintains full backward compatibility with v2.0.0
- No breaking changes to API or configuration
- All changes are additive improvements

## [1.24.1] - 2025-11-13

### Added
- EDP/FDP packet parsing now records live neighbors in the shared discovery table alongside LLDP/CDP, unlocking full topology awareness for the CLI and TUI (#79).
- The interactive TUI gained a neighbor discovery panel (`Shift+N`) with relative “last seen” timers, protocol labels, and management IPs so operators can validate simulations without leaving the console (#79).

### Changed
- Periodic and final CLI stats now surface the number of discovered neighbors to keep headless runs feature-parity with the TUI (#79).
- README highlights the new `[N]` shortcut and both entrypoints promote the shared neighbor functionality; `VERSION` + root command metadata bumped to `v1.24.1` for the release tag.

## [1.24.0] - 2025-11-12

### 🚀 Highlights
- **Runtime services everywhere** – both the Cobra CLI and the legacy entrypoint can expose the same REST API, metrics endpoint, and alert pipeline (#31, #36).
- **Run history persistence** – BoltDB-backed storage keeps recent NIAC sessions so the CLI, TUI, and API share a unified “run history” view (#32).
- **Topology tooling** – `niac analyze` exports Graphviz DOT files and the new `niac analyze-pcap` command summarizes captures for troubleshooting (#34, #37).
- **Deployment ready** – Dockerfile, Compose stack, and Kubernetes manifest ship with the repo for containerised runs (#35).
- **Web UI Preview** – a lightweight HTML/JS UI is bundled with the API for early adopters. It remains marked as a v2.0 feature and is disabled unless `--api-listen` is set (#30).

### Added
- Global runtime flags (`--api-listen`, `--metrics-listen`, `--storage-path`, `--api-token`, `--alert-*`) are available to both the Cobra CLI and the legacy interface. Legacy users can now expose the exact same services without switching entrypoints (#31).
- `pkg/api` implements REST endpoints for live stats, device inventory, topology, and run history plus a metrics listener and webhook-based alerting (#31, #36).
- `pkg/storage` provides a BoltDB persistence layer with read/write helpers and regression tests. Set `--storage-path disabled` to opt-out cleanly (#32).
- Bundled REST API documentation (`docs/REST_API.md`), Dockerfile, docker-compose stack, and `deploy/kubernetes/niac-deployment.yaml` make it easy to run NIAC in containers or clusters (#35).
- `niac analyze --graphviz` exports DOT graphs from SNMP walks, and the new `niac analyze-pcap` command emits protocol summaries in text/JSON/YAML for fast PCAP triage (#34, #37).
- Embedded Web UI assets (HTML/CSS/JS) surface live stats, device inventory, history, and topology in the browser when the API is enabled. This feature is flagged as a 2.0 preview and is off by default (#30, #37).

### Fixed / Changed
- `analyze-pcap` now uses the correct gopacket layer constants so LLDP and CDP frames are classified reliably (#34).
- `README.md` documents the runtime services, storage controls, and new analyzer workflows; version badges now reflect Go 1.24 support.
- Version metadata (root command + `VERSION` file) bumped to `v1.24.0` to align binaries, docs, and release tooling.

### Quality
- Added unit tests for the storage persistence layer to prevent regressions when adjusting BoltDB handling (#74).
- `go.mod` vendor list updated to include `go.etcd.io/bbolt` explicitly, ensuring repeatable builds for every entrypoint.

## [1.21.0] - 2025-11-08

### 🎯 MILESTONE: Performance Profiling!

Production-ready performance monitoring with pprof integration for CPU, memory, and goroutine analysis.

### Added

#### Performance Profiling (#26)
- **pprof Integration** - Built-in performance monitoring via Go's net/http/pprof
  - `--profile, -p` flag to enable profiling server
  - `--profile-port <port>` to customize HTTP server port (default: 6060)
  - Security: Binds to localhost (127.0.0.1) only for safety
  - Automatic handler registration via import side-effect

- **Available Profiling Endpoints**
  - `/debug/pprof/` - Index page with links to all profiles
  - `/debug/pprof/profile` - CPU profile (30s default, configurable)
  - `/debug/pprof/heap` - Memory heap profile
  - `/debug/pprof/goroutine` - Goroutine stack traces
  - `/debug/pprof/block` - Block profiling data
  - `/debug/pprof/mutex` - Mutex contention profile
  - `/debug/pprof/allocs` - Memory allocation profile

- **Usage Examples**
  ```bash
  # Enable profiling on default port
  niac --profile en0 config.yaml

  # Custom port
  niac --profile --profile-port 8080 en0 config.yaml

  # Collect CPU profile
  curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
  go tool pprof cpu.prof

  # Interactive memory profiling
  go tool pprof http://localhost:6060/debug/pprof/heap
  ```

- **Documentation**
  - Added profiling section to CLI help text
  - Added profiling examples to usage output
  - Updated CLI_REFERENCE.md with comprehensive profiling guide
  - Security warnings about localhost-only binding

### Metrics
- **Total Lines**: 36,802 (18,132 source + 18,670 tests)
- **Test Coverage**: Config 55.0%, Errors 95.1%, Stats 94.1%, Templates 91.9%

### Closes
- #26 - Add pprof performance monitoring

## [1.18.0] - 2025-01-07

### 🎯 MILESTONE: Enhanced Interactive TUI!

Major improvements to the interactive mode with multi-device error injection, configurable error values, and improved user experience.

### Added

#### Multi-Device Error Injection (#24)
- **Device Selection** - Cycle through devices with Shift+D key
  - Selected device highlighted in device list with arrow indicator
  - Current device shown in status bar and error injection menu
  - Device information displayed: name, type, IP, MAC

- **Configurable Error Values** - Custom error injection values
  - Value input prompt with real-time feedback
  - Support for 0-100 range validation
  - ESC to cancel, Enter to confirm
  - Clear error messages for invalid input

- **Quick Access Keys** - Number keys 1-7 for rapid error injection
  - 1: FCS Errors
  - 2: Packet Discards
  - 3: Interface Errors
  - 4: High Utilization
  - 5: High CPU
  - 6: High Memory
  - 7: High Disk

- **Enhanced Error Injection Menu**
  - Shows currently selected target device
  - Device selection hint in menu
  - Updated menu items to indicate custom values
  - All 7 error types with configurable values

#### User Interface Improvements
- **Status Bar** - Now displays selected device name
- **Device List** - Visual indicator for currently selected device
- **Value Input Mode** - Dedicated UI for entering error values
  - Professional input box with border
  - Prompt text and current input display
  - Clear instructions (Enter/Esc)

- **Updated Help Screen**
  - Two error injection workflows documented
  - Quick access method (Method 1)
  - Menu-based method (Method 2)
  - Number key shortcuts listed
  - Updated keyboard shortcuts reference

#### Technical Enhancements
- **Model State Management**
  - Added `selectedDeviceIdx` for device tracking
  - Added `valueInputMode` for input state
  - Added `valueInputPrompt` and `valueInputBuffer` for value entry
  - Bounds checking for device index

- **Input Validation**
  - Range validation (0-100) for all error types
  - Real-time input feedback
  - Digit-only input filtering
  - Max 3 digits for percentage values

### Changed
- **Error Injection Workflow** - Now uses selected device instead of always first device
- **Menu Items** - Updated to show "(custom value)" instead of fixed percentages
- **Controls Bar** - Added "[D] Device" shortcut indicator
- **Help Documentation** - Expanded with new features and workflows

### Fixed
- Error injection now targets the correct device based on user selection
- Multiple devices can now have different error states simultaneously
- Device cycling works with any number of configured devices

### Developer Notes
- Maintained backward compatibility with existing 'i' + ENTER interface
- Thread-safe error state updates via StateManager
- Clear separation between menu navigation and value input modes
- Value input uses dedicated handler for better code organization

### Impact
- **User Experience**: Multi-device testing now fully supported
- **Flexibility**: Custom error values allow precise testing scenarios
- **Efficiency**: Quick access keys enable rapid error injection
- **Visibility**: Always know which device is selected for error injection

### Future (v1.16.0+)
- Additional unit tests for cmd/niac, pkg/capture, pkg/interactive
- Performance benchmarks for hot paths
- SNMP test coverage improvements (6.7% → 50%+)
- Fuzz testing for protocol parsers

## [1.20.0] - 2025-11-08

### 🎯 MILESTONE: Performance & Testing!

Added comprehensive performance benchmarks and fuzz testing infrastructure to provide performance insights and improve code quality.

### Added

#### Performance Benchmarks (38+ total)
- **pkg/config (10 benchmarks)**
  - Config validation (simple & complex)
  - MAC/IP address normalization
  - Device lookups by MAC and IP
  - Legacy config loading
  - Speed parsing
  - MAC generation
  - Multi-protocol configurations

- **pkg/device (8 benchmarks)**
  - Device creation (single/multiple IPs, multiple devices)
  - Protocol handler registration
  - Device state lookup
  - Counter increments
  - Various device configurations (LLDP, DHCP, full config)

- **pkg/protocols (20+ benchmarks)**
  - ARP: Request handling, reply generation, gratuitous ARP
  - LLDP: Packet generation, TLV building
  - DHCP: DISCOVER/OFFER/ACK cycle, lease allocation
  - ICMP: Echo request processing
  - SNMP: GET/GET-NEXT/GET-BULK operations
  - DNS & NetBIOS: Query processing

#### Fuzz Testing (18 tests total)
- **pkg/config (6 tests)**
  - YAML loading with arbitrary input
  - Speed string parsing
  - MAC address generation
  - Simple config parsing
  - Walk file path validation
  - Device config parsing

- **pkg/protocols (12 tests)**
  - ARP: Packet parsing, MAC parsing, IP parsing
  - DHCP: MAC lookup, IP allocation, hostname validation
  - DNS: Domain name parsing, record type handling, TTL validation
  - LLDP: Chassis ID, port ID, system description, TTL validation

### Changed
- Updated README version from 1.19.0 to 1.20.0

### Metrics
- **Total Tests**: 1014
- **Test Files**: 41
- **Coverage**: Config 55.0%, Errors 95.1%, Stats 94.1%, Templates 91.9%

### Closes
- #18 - Add performance benchmark suite
- #25 - Add fuzz tests for protocol parsers

## [1.19.0] - 2025-11-08

### 🎯 MILESTONE: Enhancements!

Minor enhancements and improvements to existing functionality.

### Changed
- Updated README badges and version information
- Updated documentation to reflect latest release

## [1.15.0] - 2025-01-07

### 🎯 MILESTONE: Testing Foundation!

First step toward comprehensive test coverage. Establishes testing patterns and increases coverage for critical packages.

### Added

#### Test Coverage
- **pkg/logging Unit Tests** (`pkg/logging/colors_test.go`) - 25 test functions
  - Achieved 61.4% coverage (exceeding 60% goal)
  - Table-driven tests for all color functions
  - Tests for NO_COLOR environment variable support
  - Concurrent access safety tests
  - Tests for debug level filtering
  - Comprehensive coverage of all exported functions

#### Testing Patterns
- Established comprehensive testing patterns for future expansion
  - Table-driven tests
  - Mock-free unit testing where possible
  - Concurrent access testing
  - Environment variable testing

### Improved
- **Test Quality**: First package with >60% coverage beyond pkg/errors and pkg/config
- **Testing Patterns**: Demonstrated table-driven tests and proper test organization
- **CI Integration**: New tests run automatically in GitHub Actions pipeline

### Impact
- pkg/logging: 0% → 61.4% coverage ✅
- Integration test framework established
- Foundation for v1.16.0 test expansion

### Notes
- This is a focused release establishing testing patterns
- pkg/capture, pkg/interactive, cmd/niac tests deferred to v1.16.0
- Integration tests require additional setup for full execution
- Comprehensive test roadmap documented in COMPREHENSIVE_REVIEW_V2.md

## [1.14.0] - 2025-01-07

### 🎉 MILESTONE: CI/CD & Developer Infrastructure!

Automated testing, comprehensive documentation, and contributor-ready infrastructure.

### Added

#### Continuous Integration
- **GitHub Actions CI/CD Pipeline** (`.github/workflows/ci.yml`)
  - Multi-OS testing: Ubuntu, macOS, Windows
  - Multi-Go version testing: 1.21, 1.22
  - Automated test runs with race detector
  - Code coverage upload to Codecov
  - Golangci-lint integration
  - Build artifacts for all platforms
  - 3 parallel jobs: test, lint, build

#### Documentation
- **CONTRIBUTING.md** - Complete contributor guide
  - Development setup instructions
  - Code style guidelines
  - Testing requirements
  - PR process and checklist
  - Commit message conventions
  - Recognition policy
- **docs/ARCHITECTURE.md** - Comprehensive system design documentation
  - Package structure and responsibilities
  - Data flow diagrams
  - Protocol handler architecture
  - Configuration system design
  - Error injection system
  - Concurrency model
  - Extension points
  - Performance considerations

### Changed
- CI/CD now runs on every push and pull request
- All tests run with race detector by default
- Coverage tracking enabled

### Infrastructure
This release establishes the foundation for community contributions and ensures code quality through automation.

## [1.13.1] - 2025-01-07

### 🔒 SECURITY PATCH - Critical

Critical security and correctness fixes identified in comprehensive code review.

### Fixed

#### Security
- **CRITICAL**: Path traversal vulnerability in SNMP walk file loading
  - Added `validateWalkFilePath()` function to prevent `../../etc/passwd` style attacks
  - Walk files now validated to exist, be regular files, and not contain traversal sequences
  - Prevents malicious configurations from accessing system files (pkg/config/config.go:1377)

#### Correctness
- **Version inconsistency**: Removed duplicate version constants from `main.go`
  - Single source of truth now in `root.go` (v1.13.1)
  - Removed conflicting `Version = "1.9.0"` from main.go:22
  - Supports build-time version injection via linker flags

#### Resource Leaks
- **Goroutine leak in RateLimiter**: Added proper cleanup mechanism
  - Added `done` channel to signal goroutine termination
  - `Stop()` now calls `close(done)` to clean up goroutine (pkg/capture/capture.go:234)
  - Prevents goroutine accumulation in long-running processes

### Changed
- Version references in `printBanner()` and `printVersion()` now use variables from `root.go`

## [1.13.0] - 2025-01-07

### 🎉 MILESTONE: Enhanced CLI & Configuration Tools!

Modern CLI experience with comprehensive help, shell completion, man pages, and configuration management tools.

### Added

#### Enhanced CLI/Help
- **Shell Completion**: `niac completion` command for bash, zsh, fish, and powershell
  - Installation instructions for all shells
  - Auto-generated completions for all commands and flags
- **Rich Help Examples**: Practical examples added to all commands
  - Quick start workflows with templates
  - CI/CD integration examples
  - Common use case demonstrations
- **Man Pages**: Unix manual pages for all commands
  - Generated with `niac man` command
  - Professional documentation format
  - Installation instructions included

#### Configuration Management Tools
- **Config Export**: `niac config export` command
  - Normalize and clean YAML configurations
  - Convert legacy .cfg to YAML format
  - Validate before export
- **Config Diff**: `niac config diff` command
  - Compare two configurations
  - Show device additions/removals/modifications
  - Detect configuration drift
- **Config Merge**: `niac config merge` command
  - Merge base and overlay configurations
  - Overlay takes precedence for conflicts
  - Useful for environment-specific overrides

### Changed
- Version bumped to v1.13.0
- Documentation updated with new commands
- Man pages regenerated with all commands

## [1.7.0] - 2025-11-05

### 🎉 MILESTONE: Testing & Quality Enhancements!

Production-ready test coverage and comprehensive configuration validation. This release focuses on code quality, testing infrastructure, and improved user experience.

### Added

#### Testing Infrastructure
- **87 new unit tests** across 4 major packages:
  - Config package tests (`pkg/config/yaml_test.go`, `pkg/config/validator_test.go`): 29 tests
    - Coverage improved from 9.8% to **50.6%** (5.2x improvement)
    - Basic YAML loading and parsing tests
    - Multiple IPs per device validation (v1.5.0 feature)
    - Protocol configuration tests (LLDP, CDP, STP, HTTP, FTP, DNS, DHCP, DHCPv6)
    - Traffic pattern tests (v1.6.0 feature)
    - SNMP trap tests (v1.6.0 feature)
    - Default value application tests
    - Error handling tests (invalid files, YAML, MAC, IP addresses)
    - Performance benchmarks

  - SNMP trap tests (`pkg/snmp/traps_test.go`): 17 tests
    - Coverage improved from 0% to **6.7%**
    - TrapSender creation and lifecycle tests
    - Multiple trap receiver tests
    - Receiver address parsing tests (with/without ports)
    - Port validation tests
    - Configuration validation tests (event-based and threshold-based traps)
    - Standard trap OID verification
    - Threshold defaults tests
    - Debug level handling tests
    - IPv4 and IPv6 support tests
    - Performance benchmarks

  - Device simulator tests (`pkg/device/simulator_test.go`): 15 tests
    - Coverage improved from 0% to **22.0%**
    - Simulator creation and configuration tests
    - Device retrieval tests (GetDevice, GetAllDevices)
    - Lifecycle management tests (Start/Stop)
    - State management tests (5 states: up, down, starting, stopping, maintenance)
    - Counter increment tests (all 10 counter types)
    - Thread-safety tests with concurrent operations
    - Device type tests (router, switch, ap, server, generic)
    - Trap sender integration tests (v1.6.0)
    - Last activity tracking tests
    - Counter initialization tests
    - Performance benchmarks

  - Protocol handler tests (`pkg/protocols/arp_test.go`, `pkg/protocols/lldp_test.go`): 26 tests
    - Coverage improved from 5.6% to **15.4%** (2.75x improvement)
    - **ARP tests (9 tests)**:
      - Handler creation tests
      - ARP reply packet construction tests
      - Gratuitous ARP sending tests
      - ARP request handling tests
      - ARP reply handling tests
      - Invalid packet type handling tests
      - Constant value verification tests
      - Performance benchmarks
    - **LLDP tests (15 tests)**:
      - Handler creation and lifecycle tests
      - Chassis ID TLV construction tests (3 types: MAC, local, network_address)
      - Port ID TLV construction tests
      - TTL TLV construction tests (default and custom)
      - Port Description TLV tests
      - System Name TLV tests
      - System Description TLV tests
      - End TLV tests
      - Complete LLDP frame construction tests
      - Disabled device handling tests
      - Constants verification tests
      - Capabilities verification tests
      - Performance benchmarks

#### Configuration Validator
- **Comprehensive validation tool** (`pkg/config/validator.go`, 430 lines):
  - Three-level validation system:
    - **Errors**: Fatal configuration issues (missing required fields, invalid values)
    - **Warnings**: Non-fatal issues worth noting (unknown device types, short TTLs)
    - **Info**: Informational messages (device counts, enabled protocols)
  - Device-level validation:
    - Device name, type, MAC address, IP address validation
    - MAC address length validation (6 bytes required)
    - IP address syntax validation
    - Multiple IP address support validation
  - Protocol-specific validation (19 protocols):
    - LLDP TTL validation
    - CDP, EDP, FDP configuration validation
    - STP bridge priority validation (must be ≤ 61440 and multiple of 4096)
    - HTTP endpoint validation
    - FTP user validation
    - DNS record validation
    - DHCP/DHCPv6 pool validation
  - v1.6.0 feature validation:
    - Traffic pattern validation (ARP announcements, periodic pings, random traffic)
    - SNMP trap configuration validation
    - Trap receiver validation (IP:port format)
    - Threshold validation (CPU/memory: 0-100%, with warnings for extreme values)
  - Detailed error messages with:
    - Field names (e.g., `stp.bridge_priority`, `snmp.traps.high_cpu.threshold`)
    - Device context (shows which device has the issue)
    - Helpful suggestions (e.g., "should be a multiple of 4096")
  - Formatted output with visual indicators (✅ ❌ ⚠️ ℹ️)
  - Verbose mode support for detailed configuration insights

#### CLI/UX Enhancements
- **Progress indicators during startup**:
  - ⏳ Initializing capture engine... ✓
  - ⏳ Creating protocol stack... ✓
  - ⏳ Configuring DHCP servers (N)... ✓
  - ⏳ Configuring DNS servers (N)... ✓
  - ⏳ Starting N simulated device(s)... ✓
  - Shows ❌ on errors with helpful error messages
- **Enhanced `--dry-run` validation**:
  - Integrated with new validator for comprehensive pre-flight checks
  - Shows all validation errors, warnings, and info
  - Verbose mode (`--verbose` or `-v`) shows detailed configuration insights
  - Exit code 1 on validation failures, 0 on success
- **Startup feature summary**:
  - Shows enabled features: SNMP agents, SNMP traps, traffic generation, PCAP playback
  - Device counts for each feature
  - Clear "✅ Network simulation is ready" message
- **Better error reporting**:
  - Consistent use of colored output (✓ ❌ ⚠️ ℹ️)
  - Clear indication of what succeeded vs failed during startup

### Changed
- Version bumped from 1.6.0 to 1.7.0
- Enhanced CLI output with progress indicators throughout startup sequence
- `--dry-run` now uses comprehensive validator instead of simple checks
- Startup messages now grouped by initialization phase

### Technical Details
- New files:
  - `pkg/config/yaml_test.go` - Config package unit tests (13 tests)
  - `pkg/config/validator.go` - Configuration validator implementation (430 lines)
  - `pkg/config/validator_test.go` - Validator unit tests (16 tests)
  - `pkg/snmp/traps_test.go` - SNMP trap unit tests (17 tests)
  - `pkg/device/simulator_test.go` - Device simulator unit tests (15 tests)
  - `pkg/protocols/arp_test.go` - ARP protocol unit tests (9 tests + 2 benchmarks)
  - `pkg/protocols/lldp_test.go` - LLDP protocol unit tests (15 tests + 2 benchmarks)
  - `V1.7.0-PROGRESS.md` - Comprehensive progress report documenting all work
- Updated files:
  - `cmd/niac/main.go` - Enhanced startup sequence with progress indicators
  - `README.md` - Updated to reflect v1.7.0 features and test coverage
  - `CHANGELOG.md` - This file

### Statistics
- **Total new tests**: 87 (including benchmarks)
- **Total new lines of code**: ~4,000 (3,500 test code + 430 validator code)
- **Test coverage improvements**:
  - Config package: 9.8% → 50.6% (+40.8 percentage points, 5.2x improvement)
  - Protocol package: 5.6% → 15.4% (+9.8 percentage points, 2.75x improvement)
  - Device package: 0% → 22.0% (+22.0 percentage points, NEW)
  - SNMP package: 0% → 6.7% (+6.7 percentage points, NEW)
- **Success criteria met**: 7 of 10 major v1.7.0 success criteria (70% complete)

## [1.6.0] - 2025-11-05

### 🎉 MILESTONE: Complete Protocol & YAML Work!

All protocol and YAML configuration work is complete. Traffic patterns and SNMP trap generation are now fully configurable.

### Added

#### Phase 3 Features - Traffic & Monitoring

- **Configurable Traffic Patterns**:
  - Per-device traffic configuration
  - **ARP Announcements**: Configurable gratuitous ARP intervals (default: 60s)
  - **Periodic Pings**: Configurable ICMP echo intervals and payload sizes (default: 120s, 32 bytes)
  - **Random Traffic**: Configurable packet counts, intervals, and traffic patterns
    - Patterns: broadcast_arp, multicast, udp
    - Configurable packet count per burst (default: 5)
    - Configurable interval between bursts (default: 180s)
  - Master enable/disable switch per device
  - Examples: `examples/traffic-patterns.yaml`

- **SNMP Trap Generation** (SNMPv2c):
  - **Event-based traps**:
    - coldStart (OID 1.3.6.1.6.3.1.1.5.1) - Device initialization
    - linkDown/linkUp (OID 1.3.6.1.6.3.1.1.5.3/4) - Interface state changes
    - authenticationFailure (OID 1.3.6.1.6.3.1.1.5.5) - SNMP auth failures
  - **Threshold-based traps**:
    - High CPU utilization (configurable threshold %, check interval)
    - High Memory utilization (configurable threshold %, check interval)
    - Interface Errors (configurable error count threshold, check interval)
  - **Configuration options**:
    - Multiple trap receivers (IP:port format, default port 162)
    - Per-trap-type enable/disable
    - Configurable thresholds and check intervals
    - On-startup trap generation option
  - Examples: `examples/snmp-traps.yaml`

- **Updated Examples**:
  - `complete-kitchen-sink.yaml` now demonstrates all v1.6.0 features
  - Device 7: Configurable traffic patterns example
  - Device 8: SNMP trap generation example
  - Now includes 9 devices showcasing all features

### Technical Details

- New file: `pkg/snmp/traps.go` - SNMP trap generation implementation
- Updated: `pkg/device/traffic.go` - Per-device configurable traffic patterns
- Updated: `pkg/config/config.go` - TrafficConfig and TrapConfig structures
- Updated: `internal/converter/converter.go` - YAML parsing for new features
- Traffic generator now uses 10-second check interval for device-specific timings
- Trap sender integrated with device simulator lifecycle (start/stop)

## [1.5.0] - 2025-11-05

### 🎉 MILESTONE: Complete YAML Configuration System!

All protocols now fully configurable via YAML with per-protocol debug control and color-coded output.

### Added

#### Phase 1 Features
- **Color-coded debug output**:
  - Color-coded protocol messages for better readability
  - Support for NO_COLOR environment variable
  - `--no-color` flag to disable colors
  - Automatic color detection for terminals

- **Per-protocol debug level control**:
  - 19 protocol-specific debug flags (--debug-arp, --debug-lldp, --debug-dhcpv6, etc.)
  - Independent debug levels for each protocol (0-3)
  - Fallback to global debug level when protocol-specific not set
  - Comprehensive help output showing all debug flags

- **Multiple IPs per device**:
  - Devices can have multiple IPv4 and/or IPv6 addresses
  - Use `ips:` (plural) instead of `ip:` (singular) in YAML
  - Support for dual-stack (IPv4 + IPv6) configurations
  - Multi-homed devices (multiple IPs on different networks)
  - Example: `examples/multi-ip-devices.yaml`

#### Phase 2 Group 1 - Discovery Protocol YAML Configuration
- **LLDP Configuration** (IEEE 802.1AB):
  - `advertise_interval`: How often to send LLDP advertisements (default: 30s)
  - `ttl`: Time-to-live for LLDP information (default: 120s)
  - `system_description`: Device description string
  - `port_description`: Port/interface description
  - `chassis_id_type`: "mac" or "network_address"

- **CDP Configuration** (Cisco Discovery Protocol):
  - `advertise_interval`: Advertisement interval (default: 60s)
  - `holdtime`: Information holdtime (default: 180s)
  - `version`: CDP version (1 or 2)
  - `software_version`: Device software version string
  - `platform`: Platform/model string
  - `port_id`: Port identifier

- **EDP Configuration** (Extreme Discovery Protocol):
  - `advertise_interval`: Advertisement interval (default: 30s)
  - `version_string`: Software version
  - `display_string`: Device model/description

- **FDP Configuration** (Foundry Discovery Protocol):
  - `advertise_interval`: Advertisement interval (default: 60s)
  - `holdtime`: Information holdtime (default: 180s)
  - `software_version`: Device software version
  - `platform`: Platform/model string
  - `port_id`: Port identifier

#### Phase 2 Group 1b - STP YAML Configuration
- **STP Configuration** (Spanning Tree Protocol):
  - `enabled`: Enable/disable STP (default: false)
  - `bridge_priority`: Bridge priority 0-65535 (default: 32768)
  - `hello_time`: Hello BPDU interval in seconds (default: 2)
  - `max_age`: Maximum age in seconds (default: 20)
  - `forward_delay`: Forward delay in seconds (default: 15)
  - `version`: "stp", "rstp", or "mstp" (default: "stp")
  - Example: `examples/layer2/stp-bridge.yaml`

#### Phase 2 Group 2 - Application Protocol YAML Configuration
- **HTTP Server Configuration**:
  - `enabled`: Enable HTTP server
  - `server_name`: Server identification string
  - `endpoints`: Array of endpoint definitions
    - `path`: URL path (e.g., "/api/v1/status")
    - `method`: HTTP method (default: "GET")
    - `status_code`: HTTP status code (default: 200)
    - `content_type`: Response content type
    - `body`: Response body
  - Example: `examples/services/http-server.yaml`

- **FTP Server Configuration**:
  - `enabled`: Enable FTP server
  - `welcome_banner`: FTP welcome message (220 response)
  - `system_type`: System type string (e.g., "UNIX Type: L8")
  - `allow_anonymous`: Allow anonymous login (default: true)
  - `users`: Array of user accounts
    - `username`: Login username
    - `password`: Login password
    - `home_dir`: User home directory
  - Example: `examples/services/ftp-server.yaml`

- **NetBIOS Configuration**:
  - `enabled`: Enable NetBIOS name service
  - `name`: NetBIOS device name (max 15 characters)
  - `workgroup`: Workgroup/domain name
  - `node_type`: "B" (broadcast), "P" (point-to-point), "M" (mixed), "H" (hybrid)
  - `services`: Array of services ("workstation", "server", "browser", etc.)
  - `ttl`: Name registration TTL in seconds (default: 300)
  - Example: `examples/services/netbios-server.yaml`

#### Phase 2 Group 3 - Network Protocol YAML Configuration
- **ICMP Configuration**:
  - `enabled`: Enable ICMP echo reply (default: true)
  - `ttl`: Time To Live for ICMP packets (default: 64)
    - Common values: 32 (old systems), 64 (Linux/Unix), 128 (Windows), 255 (routers)
  - `rate_limit`: Max ICMP responses per second (default: 0 = unlimited)
  - Example: `examples/network/icmp-config.yaml`

- **ICMPv6 Configuration**:
  - `enabled`: Enable ICMPv6 echo reply (default: true)
  - `hop_limit`: Hop limit for ICMPv6 packets (default: 64)
    - NDP packets ALWAYS use hop limit 255 per RFC 4861 (security requirement)
  - `rate_limit`: Max ICMPv6 responses per second (default: 0 = unlimited)
  - Example: `examples/network/icmpv6-config.yaml`

- **DHCPv6 Server Configuration**:
  - `enabled`: Enable DHCPv6 server (default: false)
  - `pools`: IPv6 address pools
    - `network`: IPv6 network in CIDR notation
    - `range_start`: First address in pool
    - `range_end`: Last address in pool
  - `preferred_lifetime`: Preferred address lifetime in seconds (default: 604800 = 7 days)
  - `valid_lifetime`: Valid address lifetime in seconds (default: 2592000 = 30 days)
  - `preference`: Server preference 0-255 (default: 0, higher is better)
  - `dns_servers`: Array of IPv6 DNS server addresses
  - `domain_list`: Array of DNS search domains
  - `sntp_servers`: Array of SNTP time server addresses
  - `ntp_servers`: Array of NTP server addresses
  - `sip_servers`: Array of SIP server addresses
  - `sip_domains`: Array of SIP domain names
  - Example: `examples/network/dhcpv6-config.yaml`

### Changed
- Protocol handlers now read configuration from device config structs
- ICMP handler uses configurable TTL instead of hardcoded 64
- ICMPv6 handler uses configurable hop limit with RFC 4861 compliance for NDP
- DHCPv6 handler uses configurable server preference
- Discovery protocol handlers use configurable advertisement intervals and values
- STP handler uses configurable bridge priority and timers
- HTTP/FTP/NetBIOS handlers use configurable server parameters

### Documentation
- Created organized example library in `examples/` directory:
  - `examples/EXAMPLES-README.md` - Complete documentation of all examples
  - `examples/complete-kitchen-sink.yaml` - Master example with ALL features
  - `examples/layer2/` - Discovery protocol examples (LLDP, CDP, EDP, FDP, STP)
  - `examples/dhcp/` - DHCP server examples (simple and advanced)
  - `examples/services/` - Application service examples (DNS, HTTP, FTP, NetBIOS)
  - `examples/network/` - Network protocol examples (IPv4, IPv6, dual-stack, ICMP, ICMPv6, DHCPv6)
  - `examples/vendors/` - Vendor-specific examples (Cisco, Extreme, Foundry)
- Updated all example files with comprehensive inline documentation
- Added troubleshooting sections to example files
- Documented all configuration options with defaults and valid ranges

### Technical Details
- Added ICMPConfig, ICMPv6Config, DHCPv6Config structs to pkg/config/config.go
- Added STPConfig, HTTPConfig, FTPConfig, NetBIOSConfig structs
- Updated all YAML parsing in internal/converter/converter.go
- Enhanced config loader with default values for all new options
- Backward compatible - existing configs without new options still work

## [1.4.0] - 2025-01-05

### 🎉 MILESTONE: Complete DHCP/DNS Implementation!

Full-featured DHCP and DNS servers with comprehensive option support.

### Added
- **Complete DHCPv4 Implementation (15 options)**:
  - Basic options: Subnet Mask (1), Router (3), DNS Servers (6), Domain Name (15)
  - Lease management: Lease Time (51), T1 Renewal (58), T2 Rebinding (59)
  - Server identification: Server Identifier (54), Message Type (53)
  - **New High Priority Options**:
    - Hostname (Option 12) - Automatic capture and echo from client requests
    - NTP Servers (Option 42) - Time synchronization
    - Domain Search List (Option 119) - Multiple DNS search domains with RFC 1035 encoding
    - TFTP Server Name (Option 66) - PXE boot support
    - Bootfile Name (Option 67) - Boot image filename for PXE
    - Vendor-Specific Info (Option 43) - Custom vendor data
  - Static DHCP leases with MAC address masks for wildcard matching
  - Configurable via YAML with full end-to-end integration

- **Complete DHCPv6 Implementation (12 options)**:
  - Basic options: Client/Server ID (DUID), IA_NA, IA_Addr, Preference
  - DNS: DNS Servers (23), Domain Search List (24)
  - **New High Priority Options**:
    - SNTP Servers (Option 31) - Simple time synchronization
    - NTP Servers (Option 56) - Full NTP configuration
    - SIP Server Addresses (Option 22) - VoIP IPv6 addresses
    - SIP Domain Names (Option 21) - VoIP domain names
    - FQDN (Option 39) - Client fully qualified domain name
  - Configurable via YAML with full end-to-end integration

- **DNS Server Implementation**:
  - Forward DNS records (A records) - hostname → IPv4
  - Reverse DNS records (PTR records) - IPv4 → hostname
  - Configurable TTL per record
  - Multiple records per device
  - Full YAML configuration support

- **Complete YAML Configuration Support**:
  - All DHCP options loadable from YAML configuration files
  - DNS records configurable in device YAML
  - End-to-end integration: YAML → config parser → runtime → protocol handlers
  - Example configuration: `examples/scenario_configs/complete-reference.yaml`
  - Comprehensive documentation: `examples/scenario_configs/README-complete-reference.md`

- **Example Configuration with 12 Device Types**:
  - Core Router (Cisco 2821) with full DHCP/DNS/SNMP
  - Distribution Switch (Cisco Catalyst 3750)
  - Access Switch (Cisco 2960)
  - Wireless AP (Cisco Aironet)
  - Linux Server (Ubuntu)
  - Juniper Router (multi-vendor support)
  - NetGear Switch (SMB device)
  - VoIP Phone (Cisco IP Phone)
  - Network Printer (HP LaserJet)
  - NAS Storage (Synology DiskStation)
  - Security Camera (Axis)
  - Dual-Stack Server (IPv4/IPv6)

### Changed
- DHCP handler now supports advanced option configuration via `SetAdvancedOptions()`
- DHCPv6 handler now supports advanced option configuration via `SetAdvancedOptions()`
- DNS handler now supports dynamic record addition via `AddRecord()`
- Config loader enhanced with complete DHCP/DNS parsing (lines 374-496 in config.go)
- Main entry point now configures all handlers from YAML (lines 390-440 in main.go)

### Technical Details
- Added DHCPConfig and DNSConfig structs to runtime configuration
- Implemented RFC 1035 DNS label encoding for domain search lists
- Added accessor methods to Stack for protocol handler configuration
- Hostname automatically captured from DHCP requests and echoed in responses
- Vendor-specific data stored as hex strings in YAML, converted to bytes at runtime

### Documentation
- Created comprehensive reference YAML (658 lines) with all features
- Added complete feature documentation with examples and troubleshooting
- Updated all documentation files to proper locations
- Organized planning documents in docs/ folder

## [1.3.0] - 2025-01-05

### Added
- **Discovery Protocol Support (4 protocols)**:
  - LLDP (Link Layer Discovery Protocol) - IEEE 802.1AB
  - CDP (Cisco Discovery Protocol)
  - EDP (Extreme Discovery Protocol)
  - FDP (Foundry Discovery Protocol)
  - All protocols configurable via YAML
  - Periodic advertisement transmission
  - Neighbor discovery and tracking

## [1.2.0] - 2025-01-05

### 🎉 MILESTONE: 100% Protocol Coverage Achieved!

All 13 network protocols now fully implemented - complete feature parity with Java NIAC.

### Added
- **IPv6 and ICMPv6 Protocol Support** (678 lines):
  - Complete IPv6 packet handling with extension header chain walking
  - ICMPv6 Echo Request/Reply (ping6)
  - Neighbor Discovery Protocol (NDP) with Neighbor Solicitation/Advertisement
  - Router Solicitation handling
  - IPv6 multicast MAC mapping per RFC 2464 (33:33:xx:xx:xx:xx)
  - IPv6 pseudo-header checksum calculation
  - Device config parser now accepts "ipv6" keyword
  - Comprehensive unit test coverage
- **NetBIOS Protocol Support** (536 lines):
  - NetBIOS Name Service (NBNS) on UDP port 137
  - NetBIOS Datagram Service (NBDS) on UDP port 138
  - NetBIOS name encoding/decoding (first-level encoding)
  - Support for all name types (workstation, file server, browser, master, etc.)
  - Device name matching against NetBIOS queries
  - Full RFC 1001/1002 compliance
- **Spanning Tree Protocol Support** (509 lines):
  - STP Configuration BPDU handling
  - Topology Change Notification (TCN) BPDU processing
  - Bridge ID management (priority + MAC address)
  - BPDU transmission for simulated switches/bridges
  - Port state tracking (Disabled, Blocking, Listening, Learning, Forwarding)
  - RSTP support with port roles and rapid convergence flags
  - IEEE 802.1D and 802.1w compliance
  - Multicast MAC address handling (01:80:C2:00:00:00)

### Changed
- Protocol stack dispatcher now handles STP via multicast MAC detection
- UDP handler routes NetBIOS packets to appropriate ports (137, 138)
- Device table enhanced with GetByIPv6() and GetAll() methods

### Complete Protocol Suite (13/13)
1. ✅ ARP (Address Resolution Protocol)
2. ✅ IP (Internet Protocol v4)
3. ✅ ICMP (Internet Control Message Protocol)
4. ✅ IPv6 (Internet Protocol v6) **NEW**
5. ✅ ICMPv6 (ICMP for IPv6) **NEW**
6. ✅ UDP (User Datagram Protocol)
7. ✅ TCP (Transmission Control Protocol)
8. ✅ DNS (Domain Name System)
9. ✅ DHCP (Dynamic Host Configuration Protocol)
10. ✅ HTTP (Hypertext Transfer Protocol)
11. ✅ FTP (File Transfer Protocol)
12. ✅ NetBIOS (Network Basic Input/Output System) **NEW**
13. ✅ STP (Spanning Tree Protocol) **NEW**

### Performance
- Total lines added: ~1,723 lines across 9 new files
- All unit tests passing (100% test coverage maintained)
- No performance degradation with additional protocols

## [1.1.0] - 2025-01-05

### Added
- **Enhanced CLI**:
  - `--version` flag with detailed build information
  - `--list-interfaces` to show available network interfaces
  - `--list-devices` to display device table from config file
  - `--dry-run` for configuration validation without starting
  - `--verbose` / `-v` shortcut for debug level 3
  - `--quiet` / `-q` shortcut for debug level 0
  - Additional output flags: `--no-color`, `--log-file`, `--stats-interval`
  - Advanced flags: `--babble-interval`, `--no-traffic`, `--snmp-community`, `--max-packet-size`
  - Improved help text with comprehensive examples
  - Beautiful banner on startup
- **Interactive Mode Enhancements**:
  - Debug level now displayed in status bar
  - `[d]` key for debug level cycling (0→1→2→3→0)
  - `[h]` and `[?]` keys for comprehensive help overlay
  - `[l]` key for debug log viewer (shows last 10 logs)
  - `[s]` key for detailed statistics viewer
  - Debug logging system (keeps last 100 entries)
  - Timestamped log entries
  - Enhanced status messages
  - Updated controls display in footer

### Changed
- Version bumped to 1.1.0
- Status bar now shows: "Debug: X (LEVELNAME)"
- Interactive mode initial message now includes help hint
- All error injections and actions are now logged

## [1.0.0] - 2025-01-05

### Added
- Initial production release of NIAC-Go
- Complete protocol stack implementation:
  - ARP (Address Resolution Protocol)
  - IP (Internet Protocol v4)
  - ICMP (Internet Control Message Protocol)
  - TCP (Transmission Control Protocol)
  - UDP (User Datagram Protocol)
  - HTTP (HyperText Transfer Protocol) with multiple endpoints
  - FTP (File Transfer Protocol) with 17 commands
  - DNS (Domain Name System) - stub implementation
  - DHCP (Dynamic Host Configuration Protocol) - stub implementation
- SNMP agent with full functionality:
  - GET operations
  - GET-NEXT operations
  - GET-BULK operations
  - Community string authentication
  - MIB-II system group support
  - Walk file import/export
  - Dynamic OIDs (sysUpTime, etc.)
- Interactive error injection mode:
  - Beautiful terminal UI using Bubbletea
  - 7 error types (FCS errors, packet discards, interface errors, high utilization, high CPU, high memory, high disk)
  - Real-time error injection via keyboard
  - Interface configuration (speed, duplex)
  - Statistics display
- Device behavior simulation:
  - Per-device state management (up, down, starting, stopping, maintenance)
  - Type-specific behavior (router, switch, AP, server)
  - Device counters for all protocol types (10 counter types)
  - Periodic behavior loops (every 30 seconds)
  - SNMP agent per device
- Network traffic generation:
  - Gratuitous ARP announcements (every 60 seconds)
  - Periodic pings between devices (every 120 seconds)
  - Random traffic patterns (every 180 seconds):
    - Broadcast ARP requests
    - Multicast packets
    - Random UDP traffic
- Configuration file parser:
  - Compatible with Java NIAC config file format
  - Device properties (name, type, IP, MAC, SNMP settings)
  - Interface configuration
  - SNMP walk file loading
- Comprehensive test suite:
  - 23 unit tests covering all major components
  - Config parsing tests
  - Error injection tests
  - Protocol stack tests
  - 100% test pass rate
- Complete documentation:
  - README with usage instructions
  - FINAL_SUMMARY with all features and statistics
  - PROGRESS_REPORT with development timeline
  - JAVA_VS_GO_VALIDATION with detailed comparison

### Performance
- Binary size: 6.1 MB (2.6x smaller than Java + JRE)
- Startup time: ~5ms (10x faster than Java)
- Memory usage: ~15MB (6.7x less than Java)
- Error injection: 7.7M ops/sec (77x faster than Java)
- Config parsing: ~1.3µs (770x faster than Java)
- Build time: ~5 seconds (48-60x faster than Java)
- Code size: 6,216 lines (3.3x less than Java's 20,380 lines)

### Notes
- First production-ready release
- Feature parity with Java NIAC on all core protocols
- Four major enhancements over Java:
  1. Advanced HTTP server (vs Java's "Yo Dude" response)
  2. Complete FTP server (not present in Java)
  3. Advanced device simulation with state management
  4. Comprehensive traffic generation (3 patterns vs Java's basic babble)
- Compatible with all Java NIAC configuration files and SNMP walk files
- Modern architecture using Go idioms (goroutines, channels, clean packages)

[Unreleased]: https://github.com/krisarmstrong/niac-go/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/krisarmstrong/niac-go/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/krisarmstrong/niac-go/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/krisarmstrong/niac-go/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/krisarmstrong/niac-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/krisarmstrong/niac-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/krisarmstrong/niac-go/releases/tag/v1.0.0
