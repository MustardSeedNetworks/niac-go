# Project audit — 2026-05-07

Sweep of the entire repo for best-practice gaps, defects, wiring issues,
and enhancement opportunities. Companion to
[SURFACE_MATRIX.md](./SURFACE_MATRIX.md) and
[CONFIG_COVERAGE.md](./CONFIG_COVERAGE.md).

This is a **point-in-time snapshot.** Items marked **fixed in this PR**
land alongside this doc; everything else is recorded as a follow-up so
nothing slips.

## Tier 1 — defects to fix soon

### A. Branch protection on `main` is **off**

`gh api repos/MustardSeedNetworks/niac-go/branches/main/protection` returns an
empty object. Anyone with push access can:

- push directly to `main` without a PR
- force-push to `main` (overwrites history of every collaborator's clone)
- skip required CI checks
- skip CODEOWNERS review

The CODEOWNERS file is purely advisory until protection is on.

**Recommended config** (Settings → Branches → main):

- ✅ Require a pull request before merging
- ✅ Require approvals: 1
- ✅ Require review from Code Owners
- ✅ Require status checks: `Backend (Go)`, `Frontend (TypeScript)`,
  `Security Scanning`, `License Compliance Check`, `Quality Checks`,
  `Build (linux-amd64)`, `E2E Browser Tests`, `Lighthouse Audit`,
  `CodeQL`, `C Lint (C23)`, `Trivy`, `gosec`, `Analyze (go)`,
  `Analyze (javascript-typescript)`
- ✅ Require branches to be up to date before merging
- ✅ Do not allow bypassing the above settings (admin too)

`gh api -X PUT` script provided in the follow-up issue.

### B. Newly-added Go code lacks unit tests

| File | Lines | Tests |
|---|---|---|
| `internal/api/config_ops.go` (POST /api/v1/config/{merge,import}) | 219 | 0 |
| `cmd/niac-schema/main.go` | 71 | 0 |

The merge endpoint in particular has subtle semantics worth testing
(overlay-replace on name, base-only kept, overlay-only appended) — and
the schema generator should round-trip with the parser to catch tag
mismatches.

### C. Newly-added UI components lack tests

| File | Lines | Tests |
|---|---|---|
| `ui/src/pages/NeighborsPage.tsx` | 175 | 0 |
| `ui/src/pages/WalkValidatorPage.tsx` | 237 | 0 |
| `ui/src/components/templates/JavaDslImportCard.tsx` | 131 | 0 |
| `ui/src/ui/Tooltip.tsx` | 64 | 0 |

A modest `vitest` suite per page covering the happy path + the empty /
error state would round out the existing coverage.

### D. Replay controller is duplicated across packages

`internal/daemon/daemon.go` and `cmd/niac/runtime_services.go` both
contain a `replayController` with the same logic. There's an explicit
`// TODO: Refactor to share code.` comment acknowledging it.

The cleanest fix: move the implementation to a new package
`internal/replay/` and have both call sites consume it. Today, fixing a
bug in one means remembering to mirror it to the other.

## Tier 2 — hygiene gaps

### E. Some Go dependencies are minor versions behind

`go list -m -u all` shows ~10 packages with available updates:

```
github.com/aymanbagabas/go-udiff v0.2.0 → v0.4.1
github.com/buger/jsonparser v1.1.1 → v1.2.0
github.com/charmbracelet/x/exp/golden  (newer commit)
github.com/google/pprof  (newer commit)
github.com/invopop/jsonschema v0.13.0 → v0.14.0
github.com/mailru/easyjson v0.7.7 → v0.9.2
github.com/vishvananda/netlink v1.1.0 → v1.3.1
github.com/vishvananda/netns  (newer commit)
github.com/yuin/goldmark v1.4.13 → v1.8.2
golang.org/x/telemetry  (newer commit)
```

Dependabot will roll most of these on its weekly cadence. Worth a
manual `go get -u ./... && go mod tidy` once the audit branch lands.

### F. `cmd/niac-schema` lacks a "schema is in sync" round-trip test

The CI guard added in this PR catches *commit-time* drift, but a Go
test that round-trips converter.Config → schema → validator → fixtures
would catch logic drift (e.g. accidental field rename in YAML tags).

### G. Help content is page-level, not per-control

The `?` icon on every page now opens a help slide-out. The next layer
down — "what does this specific button do" — is currently `title=` only.
That's fine for plain strings; for richer content we ship `Tooltip` from
the same PR. Low-priority follow-up: sweep secondary controls
(card-action chips, list-item action menus) and wrap with `Tooltip`.

### H. CI runs the schema check on `Backend (Go)` only

Dependabot opens PRs that may or may not include schema changes. If
struct tags drift on a dep update we'd catch it; if tags drift on a
non-Go-code-touching PR, the schema check still runs because Go is
needed to build. Acceptable for now.

## Tier 3 — enhancement ideas

### I. JSON Schema → schema selection in DeviceEditor

The WebUI's visual device editor consumes `/api/v1/config/schema`, which
is hand-written. With the new generator we could feed the same schema
to both the editor and external validators, eliminating the second
source of truth.

### J. ReplayPage download button next to "Choose file"

Once a user has run a replay, the Replay page could expose a
"Download captured packets" button (sourced from the in-memory ring
buffer or the SSE packet stream). Today they have to invoke
`niac analyze-pcap` on the CLI.

### K. Per-protocol debug levels also editable from /runtime header

The Protocol Debug page already has the per-protocol level UI. Adding a
single "Debug level" slider on the simulation header (0–3) would cover
the 90% case for users who don't care which protocol is loud.

### L. Supply-chain: vendor in-tree?

`go.sum` is committed but no `vendor/` directory. `go mod download`
runs in CI, which works as long as the proxy is up. For air-gapped
environments and reproducibility, `go mod vendor` and a `vendor/`
checkin is the standard play. No urgency unless someone actually
deploys NIAC behind a firewall.

### M. UI stack — `react-router-dom` v6 → v7 (long lead time)

We're on a recent v6 minor. v7 is out and changes a few APIs. Worth
queueing a v7 migration PR before the v6→v7 jump becomes painful.

## Tier 4 — confirmed clean

These were checked and are in good shape. Listed so future audits don't
re-investigate.

- ✅ **Open CodeQL alerts on `main`: 0** (was 16 before PR #483).
- ✅ **`npm audit --audit-level=moderate`: 0 vulnerabilities.**
- ✅ **`gosec`** runs on every PR via `ci.yml`'s Security Scanning job.
- ✅ **TODO/FIXME density: 8 occurrences** across the entire codebase,
  most documenting actual constraints (see `internal/converter/converter.go`
  field semantics).
- ✅ **No unsafe `panic()` in production code.** The single panic in
  `internal/protocols/dhcpv6.go` (entropy failure during DUID generation)
  is correct: a predictable DUID would create colliding identities across
  processes; crashing is the right answer.
- ✅ **Body-size guards everywhere.** All POST/PUT handlers wrap
  `r.Body = http.MaxBytesReader(...)`.
- ✅ **CSP, X-Frame-Options, X-Content-Type-Options** all set globally
  via `addSecurityHeaders` in `internal/api/middleware.go`.
- ✅ **Error-write coverage: 170 `writeError` / `http.Error` call sites**
  in 39 routes — handlers don't drop bad-input cases on the floor.
- ✅ **CSRF protection on every write route** via `csrfProtect`
  middleware in `routes.go`.
- ✅ **Rate limiting** on 15 of 39 routes — every write route, all walk
  endpoints (separate stricter limit), every error-injection endpoint.
  Read endpoints intentionally unrate-limited so dashboards don't page
  themselves.
- ✅ **React ErrorBoundary** wraps the whole app and each lazy page
  individually, so a render error on one page doesn't blank the whole UI.
- ✅ **All workflows have concurrency groups** (PR #486).
- ✅ **All `@actions/*` and third-party action versions standardised**
  to current latest majors (PR #486).
- ✅ **Manual release dispatch produces snapshot artifacts only**; publishing
  is reserved for CI runs triggered by version tags.
- ✅ **JSON Schema published + CI-guarded** (this PR).
- ✅ **Per-page help on all 15 pages** (this PR).
- ✅ **Tooltips on every interactive control on touched pages** (this
  PR).

## Process recommendations

1. **Open issues for Tier 1 items and assign target dates.** They're
   fixable in one PR each; making them visible prevents drift.
2. **Stop adding Go files to `internal/api/` without a sibling
   `_test.go`.** A Makefile / pre-commit guard would enforce this.
3. **Add a `make audit` target** that runs the same sweeps this audit
   used so it's repeatable, not a one-shot exercise.

EOF
