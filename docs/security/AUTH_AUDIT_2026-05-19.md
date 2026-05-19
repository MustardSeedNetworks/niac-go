# NIAC Auth Audit — 2026-05-19

Task: #77 — audit + modernize auth across all 3 repos.

Read-only audit of `cmd/niac/` and `internal/api/` covering the
existing bearer-token middleware. NIAC does not implement a login
flow today; auth is a single shared-secret bearer token resolved from
`NIAC_API_TOKEN`.

No severity ratings are assigned here.

---

## Summary

| Category                  | Result |
|---------------------------|--------|
| Token source              | PASS — env var preferred, CLI flag deprecated |
| Token comparison          | PASS — SHA-256 + constant-time compare |
| Warn-when-unset behavior  | PASS — startup logs scream when token missing |
| Default-on auth           | WARN — auth is OPTIONAL by default (no token => no auth) |
| Rate limiting             | PASS — per-IP token-bucket on every endpoint |
| Audit logging             | WARN — `unauthorized` warns logged with clientIP but not UA |
| Token rotation            | WARN — no rotation API; restart required |
| Password support          | N/A — no users, no passwords, no sessions |
| CSRF                      | N/A — bearer token only, no cookies |

Totals: 4 PASS, 3 WARN, 0 FAIL (2 N/A).

---

## Checklist

### Token source / handling
- **Result**: PASS
- **Where**:
  - `cmd/niac/runtime_services.go:34-48` — `resolveAPIToken` prefers
    `NIAC_API_TOKEN` over the deprecated `--api-token` CLI flag and
    prints a warning if the flag is used (process list exposure).
  - `cmd/niac/cmd_daemon.go:52-87` — CLI flag is marked deprecated.

### Token comparison
- **Result**: PASS
- **Where**: `internal/api/middleware.go:180-203`
- **Detail**: Strip `Bearer ` prefix, hash both sides with SHA-256 to
  equalize lengths, then `subtle.ConstantTimeCompare`. Avoids
  length-leak timing attack.

### Warn-when-unset
- **Result**: PASS
- **Where**: `internal/api/server.go:291-297`
- **Detail**: On startup when `s.cfg.Token == ""` the server logs two
  WARN lines plus an INFO with the recommended
  `openssl rand -base64 32` command. Loud enough to notice in a
  systemd journal.

### Auth-on by default
- **Result**: WARN
- **Where**: `internal/api/middleware.go:174-178`
- **Detail**: When `s.cfg.Token == ""` the middleware short-circuits
  with `next(w, r)` — i.e. every endpoint is publicly accessible. A
  warn on startup is not a substitute for a default-secure posture.
- **Remediation note**: Making auth mandatory by default is a breaking
  change for operators who currently bind to `127.0.0.1` and rely on
  network ACLs. It would break:
  - `make dev-run` and the local Playwright suite (they hit the API
    without a token).
  - Any local installer that runs niac on `127.0.0.1` without
    setting `NIAC_API_TOKEN`.
  - `cmd_service_windows.go:82` which calls `resolveAPIToken("")`
    with no env-var fallback.
  Decision is product, not security: should `127.0.0.1`-only binds
  bypass the requirement while `0.0.0.0`/external binds require a
  token? Filed as followup.

### Rate limiting
- **Result**: PASS
- **Where**: `internal/api/middleware.go:161-172`
- **Detail**: Per-IP token-bucket via `rateLimiter.GetLimiter(clientIP)`
  before auth check. 429 response on overage. Memory-bounded via
  visitor cap (same pattern as stem/seed).

### Audit logging
- **Result**: WARN
- **Where**: `internal/api/middleware.go:194-200`
- **Detail**: Failed auth attempts are logged with `requestID` and
  `clientIP` but no `User-Agent`, no event-type tag (e.g.
  `event=auth.unauthorized`), and the rate-limit event is also a
  generic WARN. Makes SIEM ingestion harder.
- **Remediation note**: Add a `userAgent` field + `event` tag to both
  the unauthorized warn and the rate-limit warn. <10-line change.
  See Small fixes below.

### Token rotation
- **Result**: WARN
- **Where**: token loaded once at startup via
  `resolveAPIToken` then captured in `ServerConfig.Token`
  (`cmd/niac/runtime_services.go:63`).
- **Detail**: No way to rotate the token without restarting the
  process. Acceptable for a low-volume admin tool but worth flagging.
- **Remediation note**: Add a `SIGHUP` handler that re-reads
  `NIAC_API_TOKEN`. Filed as followup.

### Password / CSRF / sessions
- **Result**: N/A
- **Detail**: NIAC has no user accounts, no passwords, no cookies.
  A shared static bearer token is the whole auth model.

---

## Small fixes shipped in this PR

### Structured event tag + User-Agent on auth warns
- **File**: `internal/api/middleware.go`
- **Change**: Add `event=auth.unauthorized` and `userAgent` to the
  unauthorized log line; add `event=api.rate_limited` and `userAgent`
  to the rate-limit log line. ~6 lines. Behavior-preserving — only
  log fields change.

Other items (mandatory-auth default, SIGHUP rotation) are deferred
because they cross the "no behavior change in this PR" line.

---

## Followup tickets (deferred work)

1. **Default-secure auth posture** — require a token unless bound to
   `127.0.0.1`/`::1`, OR a new `--auth=optional` flag is explicitly
   passed. Touches `cmd_service_windows.go`, the systemd unit, and
   docs. Proposed task:
   `feat(api): require API token by default on non-loopback binds`.
2. **Token rotation via SIGHUP** — re-read `NIAC_API_TOKEN` and swap
   atomically without restarting the daemon. Proposed task:
   `feat(api): hot-reload NIAC_API_TOKEN on SIGHUP`.
3. **Multi-token + scopes** — future, if NIAC ever needs different
   tokens for read-only vs. write access (CI vs. operator). Proposed
   task: `feat(api): scoped API tokens (read-only vs admin)`.
4. **mTLS option** — for deployments where the bearer token is too
   weak. Proposed task: `feat(api): mutual-TLS authentication option`.
