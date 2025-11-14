# GitHub Issues Created from Comprehensive Review

**Date:** November 14, 2025
**Total Issues Created:** 39
**Issue Range:** #95 - #133

---

## Summary

All 39 actionable issues from the comprehensive code review, documentation review, and QA review have been successfully created as GitHub issues with full details including:

- Severity level (CRITICAL, HIGH, MEDIUM, LOW)
- File locations with line numbers
- Detailed descriptions
- Vulnerable code examples
- Recommended fixes with code snippets
- Estimated fix times
- References to review documents

---

## Issues by Severity

### 🔴 CRITICAL (3 issues) - Fix Immediately

| Issue | Title | File | Lines | Est. Time |
|-------|-------|------|-------|-----------|
| #95 | Path Traversal Vulnerability in File Listing API | pkg/api/server.go | 1196-1240 | 2-3 hours |
| #96 | Symlink Attack Risk in File Operations | pkg/api/server.go | 1036-1047 | 1-2 hours |
| #97 | Unbounded PCAP Upload Can Cause Memory Exhaustion | pkg/api/server.go | 1021-1033 | 1 hour |

**Total CRITICAL fix time:** 4-6 hours

---

### 🟠 HIGH (6 issues) - Fix This Sprint

| Issue | Title | File | Lines | Est. Time |
|-------|-------|------|-------|-----------|
| #98 | Goroutine Leak Risk in HTTP Server Startup | pkg/api/server.go | 179-183 | 2-4 hours |
| #99 | Missing HTTP Timeouts Allow Slowloris Attacks | pkg/api/server.go | Server config | 1-2 hours |
| #100 | API Token Comparison Vulnerable to Timing Attacks | pkg/api/server.go | ~270 | 30 minutes |
| #101 | API Token Exposed in CLI Arguments and Shell History | cmd/niac/root.go | 65 | 1 hour |
| #102 | Missing Security Headers (CSP, HSTS, Permissions-Policy) | pkg/api/server.go | 38-44 | 2-3 hours |
| #103 | Update Version Badge in README from 2.1.1 to 2.3.0 | README.md | 6 | 2 minutes |

**Total HIGH fix time:** 6.5-11 hours

---

### 🟡 MEDIUM (16 issues) - Fix Next 2 Sprints

| Issue | Title | Est. Time |
|-------|-------|-----------|
| #104 | Implement API Rate Limiting to Prevent Abuse | 4-6 hours |
| #105 | Standardize API Error Response Format | 2-3 hours |
| #106 | Silent Error Suppression in Critical Paths | 1 hour |
| #107 | Optional API Authentication Allows Unauthenticated Access | 2 hours |
| #108 | Config File Update Accepts Arbitrary YAML Paths | 2-3 hours |
| #109 | No CSRF Protection in API | 3-4 hours |
| #110 | SNMP Community Strings Stored in Plain Text | 2-3 hours |
| #111 | Request Body Size Limit Not Enforced | 1-2 hours |
| #112 | API Polling Intervals Should Be Configurable | 2-3 hours |
| #113 | Add Docker/Kubernetes Deployment Best Practices Documentation | 2-4 hours |
| #114 | Add Performance Tuning Guide | 3-4 hours |
| #115 | Update ARCHITECTURE.md Version Metadata | 2 minutes |
| #124 | Channel Buffer Sizes May Cause Blocking | 3-4 hours |
| #125 | WebUI Component Re-Render Optimization | 3-4 hours |
| #126 | Large Device Lists Need Virtual Scrolling | 4-6 hours |

**Total MEDIUM fix time:** 35-51 hours

---

### 🔵 LOW (14 issues) - Future Enhancements

| Issue | Title | Est. Time |
|-------|-------|-----------|
| #116 | Add Godoc Comments for Utility Functions | 1-2 hours |
| #117 | Add OpenAPI/Swagger Specification | 6-8 hours |
| #118 | Add Request Tracing IDs | 2-3 hours |
| #119 | Log Goroutine Count for Debugging | 30 minutes |
| #120 | Add FAQ Section to Documentation | 2-3 hours |
| #121 | Add WebUI Dedicated Documentation with Screenshots | 3-4 hours |
| #122 | Add More API Usage Examples (Python/curl) | 2-3 hours |
| #123 | Clarify Go Version Requirement (1.24.0 vs 1.24+) | 5 minutes |
| #127 | Add Documentation Index/Site Map to README | 30 minutes |
| #128 | Add SNMP Walk File Contribution Workflow Documentation | 1-2 hours |
| #129 | Some Variable Abbreviations Could Be More Explicit | 2-3 hours |
| #130 | Add CI/CD Integration Examples | 2-3 hours |
| #131 | Document Breaking Change Policy for API Versioning | 1 hour |
| #132 | Implement Graceful Degradation for Unavailable Replay Engine | 3-4 hours |
| #133 | Consider State Management Library for Complex WebUI State | 6-8 hours |

**Total LOW fix time:** 32-48 hours

---

## Total Estimated Fix Time

| Priority | Count | Time Range |
|----------|-------|------------|
| CRITICAL | 3 | 4-6 hours |
| HIGH | 6 | 6.5-11 hours |
| MEDIUM | 16 | 35-51 hours |
| LOW | 14 | 32-48 hours |
| **TOTAL** | **39** | **77.5-116 hours** |

---

## Issues by Category

### Security (11 issues)
- #95, #96, #97 (CRITICAL path traversal, symlink, upload)
- #98, #99, #100, #101, #102 (HIGH goroutine, timeouts, timing, token, headers)
- #104, #107, #109 (MEDIUM rate limiting, auth, CSRF)

### Code Quality (9 issues)
- #106, #111, #124 (MEDIUM errors, limits, buffers)
- #116, #118, #119, #129, #132, #133 (LOW godoc, tracing, logging, naming, degradation, state)

### Documentation (12 issues)
- #103 (HIGH version badge)
- #110, #113, #114, #115 (MEDIUM SNMP, Docker/K8s, performance, architecture)
- #120, #121, #122, #123, #127, #128, #130, #131 (LOW FAQ, WebUI, examples, version, index, walks, CI/CD, versioning)

### Performance (4 issues)
- #98 (HIGH goroutines)
- #112, #125, #126 (MEDIUM polling, re-renders, virtualization)

### API Design (3 issues)
- #105, #108 (MEDIUM error format, config validation)
- #117 (LOW OpenAPI)

---

## Recommended Action Plan

### Phase 1: Critical Security Fixes (4-6 hours)
**Timeline:** This week (BLOCKING PRODUCTION)

1. Fix path traversal vulnerability (#95) - 2-3 hrs
2. Fix symlink attack risk (#96) - 1-2 hrs
3. Add PCAP upload size limit (#97) - 1 hr

**Deploy to staging after Phase 1**

### Phase 2: High Priority Fixes (6.5-11 hours)
**Timeline:** Next sprint (REQUIRED FOR PRODUCTION)

1. Fix goroutine leak risk (#98) - 2-4 hrs
2. Add HTTP timeouts (#99) - 1-2 hrs
3. Fix token timing attack (#100) - 30 min
4. Move API token to env var (#101) - 1 hr
5. Add security headers (#102) - 2-3 hrs
6. Update version badge (#103) - 2 min

**Deploy to production after Phase 2**

### Phase 3: Medium Priority (35-51 hours)
**Timeline:** Next 2 sprints

Address all 16 medium priority issues in order of impact:
- Security enhancements (rate limiting, auth)
- Code quality improvements
- Documentation updates
- Performance optimizations

**Complete production system after Phase 3**

### Phase 4: Low Priority (32-48 hours)
**Timeline:** Future sprints

Enhancement and polish work as time permits.

---

## Issue Labels Applied

- **bug** (13 issues) - Functionality issues
- **enhancement** (14 issues) - New features/improvements
- **documentation** (12 issues) - Documentation updates

Note: Some labels ('security', 'performance', 'accessibility') were attempted but not available in the repository.

---

## Links

- **Review Documents:**
  - [COMPREHENSIVE_REVIEW_INDEX.md](COMPREHENSIVE_REVIEW_INDEX.md)
  - [CODE_REVIEW_COMPREHENSIVE.md](CODE_REVIEW_COMPREHENSIVE.md)
  - [DOCUMENTATION_REVIEW.md](DOCUMENTATION_REVIEW.md)
  - [QA_REVIEW_COMPREHENSIVE.md](QA_REVIEW_COMPREHENSIVE.md)

- **GitHub Issues:**
  - View all: https://github.com/krisarmstrong/niac-go/issues
  - Filter by label: Use labels `bug`, `enhancement`, `documentation`
  - Sort by creation date to see all 39 issues (#95-#133)

---

## How to Use These Issues

### For Project Managers
1. Review Phase 1 (CRITICAL) issues immediately
2. Schedule Phase 1 fixes this week
3. Plan Phase 2 (HIGH) for next sprint
4. Budget ~8-17 hours for Phases 1+2

### For Developers
1. Start with CRITICAL issues (#95-#97)
2. Each issue includes:
   - Exact file and line numbers
   - Vulnerable code examples
   - Recommended fix with code
   - Testing instructions
3. Reference SECURITY_FIXES_CHECKLIST.md for implementation details

### For QA Engineers
1. Use issues for test planning
2. Each issue describes the risk/bug
3. Testing instructions provided where applicable
4. Track progress using issue status

---

**Generated:** November 14, 2025
**Source:** Comprehensive code/doc/QA review of NIAC-Go v2.3.0
**Automation:** GitHub CLI + Claude Code
