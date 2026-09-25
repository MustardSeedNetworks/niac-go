# ADR 0002: Capability registry for route policy

| Status | Date | Deciders |
| --------- | ------------ | ------------------ |
| Accepted | 2026-06-07 | @krisarmstrong |

## Context

API routes were registered imperatively in `internal/api/routes.go`, each
hand-nested with its middleware across multiple lines, e.g.
`mux.HandleFunc(path, s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(h)))))`.
Because each call site re-applied the chain by hand, the wrapping could be
applied inconsistently or forgotten. In particular `registerReadOnlyRoutes`
mixed genuinely-mutating routes (templates, configs, library CRUD, device
subpaths) in with reads, each relying on the author remembering to add
`csrfProtect` + `writeRateLimit` — a foot-gun the security audit flagged.

## Decision

Routes are declared as **data** and a single `register()` / `registerAll()`
composes their per-route policy in **one canonical order**:
`recover → auth → rateLimit → csrf → admin → feature → handler`. A route is an
`apiRoute{path, handler, rl, csrf, admin, feature}` value (`internal/api/route.go`).
`register()` records each route in a manifest and installs it on the mux.

Supporting mechanisms:

- `GET /__capabilities` serves the route-policy manifest (rate-limit / CSRF /
  admin / feature per route) for deployment and audit.
- `scripts/check-route-policy.sh` (a CI gate) fails if any `/api/` route is
  registered directly via `mux.HandleFunc` instead of through `register()`.

## Consequences

- A route cannot ship without its policy; the manifest makes the policy visible
  in one place, and the previously-hidden mutating routes are now explicitly
  tagged `rl: rlWrite, csrf: true`.
- Mirrors stem's capability registry and seed's ADR-0002, harmonizing the fleet
  while each repo keeps its own implementation (NIAC's policy model is
  scope-based with multiple rate limiters + a license feature).

## Alternatives considered

- **A grep-only CI gate** without the registry: catches a direct registration
  but does not prevent it. Rejected as a band-aid (kept only as a complement).
- **Status quo (hand-nested middleware)**: the source of the foot-gun.

## Related issues and PRs

- #800 (this registry: `route.go`, `/__capabilities`, `check-route-policy.sh`)

## Amendment 2026-09-14 — no `feature` step

The canonical chain above names a `feature` step and the struct a `feature`
field for the license gate. Both went with runtime licensing (#1203, ADR 0005).
The chain today is `recover → auth → rateLimit → csrf → admin → methodGate →
bodyLimit → handler` and `apiRoute` is `{path, handler, methods, maxBodyBytes,
rl, csrf, admin}` (`internal/api/route.go`). Everything else stands.

## Amendment 2026-09-24 — the registry is foundation's

The registrar now lives once for the fleet in foundation
`pkg/httpserver/route` (#2257). NIAC declares `route.Route{Path, Handler,
Methods, MaxBodyBytes, Auth, CSRF, Scope, Limiter, Hidden}` values in
`internal/api/routes.go`; `internal/api/route.go` supplies only NIAC's error
envelope, auth middleware, the meaning of the `admin` scope and its four
limiters. The canonical order is the shared one: `limiter → auth →
methodGate → csrf → scope → bodyLimit → handler`, inside a request ID, an
access log line and panic recovery around the whole mux. Three behaviours move
with it: a per-route limiter now refuses before the token is checked (a flood
gets 429, not 401), a wrong method answers 405 before CSRF or scope is
demanded, and every request gets one INFO access log line.

Every route goes through the registrar, including `/__version`,
`/__capabilities` (`Auth: false`) and the SPA shell (`Auth: false`,
`Hidden: true`). `/__capabilities` serves foundation's `route.Policy`, so an
admin route reports `"scope": "admin"` rather than `"admin": true`, and `auth`
and `hidden` are new keys. `cmd/niac-openapi` renders through foundation's
emitter, and `docs/openapi.yaml` is byte-identical to before.
`scripts/check-route-policy.sh` now fails on any ServeMux or literal-pattern
registration under `internal/api`, not only `/api/` ones.

**Amended 2026-09-25 (#2257):** NIAC's copy of the gate is deleted. CI
runs foundation's `pkg/httpserver/route/check-route-policy.sh` from the pinned
module copy (foundation v0.6.1, which anchors the registration pattern to a
string literal so `slog.Handler.Handle` no longer matches, foundation#70).
