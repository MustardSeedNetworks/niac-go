# ADR 0006: Decompose `internal/api` into isolated sub-packages

**Status:** Accepted (2026-06-08)

## Context

`internal/api` had grown into a ~13k-LOC monolith mixing HTTP transport
(handlers, routing, the capability registry) with several self-contained
infrastructure concerns: the bearer-token store and scope model
(`token_store.go`), the per-client rate limiter (`rate_limiter.go`), the
per-session CSRF manager (`csrf_manager.go`), the SSE hub/streaming engine
(`sse*.go`), and the auth/admin middleware (`middleware.go`).

The dependency-direction depguard rule (ADR-0003) protects the _domain core_
(protocols/converter/device/mibdb) from importing the API layer, but it cannot
express boundaries _within_ `internal/api` — everything is one package, so any
file can reach any other. That makes the cohesive concerns above implicit and
lets future code couple, say, the SSE engine to an auth handler with no signal.

## Decision

Extract the cohesive concerns into sibling leaf packages under
`internal/api/<concern>/`, one at a time, lowest-coupling first
(`tokenstore` → `ratelimit` → `csrf` → `sse` → `auth`). Each leaf:

- owns its type(s) and any unexported helpers and tests;
- imports **only** the standard library and inward domain packages — never the
  `internal/api` transport layer;
- is composed inward by the `Server` (and, where relevant, `internal/daemon`),
  which holds the concrete manager and wires it into the declarative route
  registry (`register()`/`registerAll()` in `route.go`/`routes.go`). The
  registry, `/__capabilities`, and the middleware composition order are
  unchanged — only _where the building blocks live_ changes.

Each extraction adds a depguard `api-<concern>-isolated` rule
(modelled on `domain-core-inward-only`) that denies the sub-package importing
`github.com/MustardSeedNetworks/niac-go/internal/api`, so the leaf boundary is
CI-enforced rather than convention.

The first extraction, **`tokenstore`**, is the cleanest case: `token_store.go`
imported nothing from `api`. `TokenStore`, `ScopedToken`, `TokenScope`
(+`ScopeReadOnly`/`ScopeReadWrite`/`ScopeAdmin`), `NewTokenStore`,
`LoadTokenFile`, `RequiredScopeForMethod`, and the `Err*` values now live in
`internal/api/tokenstore`; `Server.tokens` and the daemon's token-seeding use
the package directly.

## Consequences

- **Enforced cohesion.** A leaf cannot grow a back-edge into transport without
  failing `golangci-lint`. The token store, in particular, is now a reusable
  building block (the daemon imports it directly).
- **No behavioural change.** Tests that exercised unexported internals move with
  their package (`package tokenstore`); `Server`-level tests stay in `package
  api` and reference the leaf through its exported surface. The race suite,
  route-policy gate, and `/__capabilities` output are unchanged.
- **Incremental, reviewable PRs.** One concern per PR keeps each diff a moves +
  qualifier change plus one depguard rule, instead of a single 13k-LOC churn.
- The remaining four extractions (ratelimit, csrf, sse, auth) follow in their own
  PRs; the auth extraction is last because it depends on tokenstore/ratelimit/csrf.
