# ADR 0002: Capability registry for route policy

| Status  | Date       | Deciders         |
|---------|------------|------------------|
| Accepted | 2026-06-07 | @krisarmstrong   |

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
  while each repo keeps its own implementation (niac's policy model is
  scope-based with multiple rate limiters + a license feature).

## Alternatives considered

- **A grep-only CI gate** without the registry: catches a direct registration
  but does not prevent it. Rejected as a band-aid (kept only as a complement).
- **Status quo (hand-nested middleware)**: the source of the foot-gun.

## Related issues and PRs

- #800 (this registry: `route.go`, `/__capabilities`, `check-route-policy.sh`)
