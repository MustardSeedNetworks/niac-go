# ADR 0004: Scope-based bearer-token auth + CSRF posture

| Status  | Date       | Deciders         |
|---------|------------|------------------|
| Accepted | 2026-06-07 | @krisarmstrong   |

## Context

NIAC is a network-device simulator driven by operators and automation. It has no
interactive login or password store; access is via operator-provisioned bearer
tokens. The requirement is to distinguish read-only consumers (dashboards,
monitoring) from read-write operators and from administrative whole-topology
operations, without a full user/role system. This ADR records the security model
already implemented (and hardened in this pass) so it is a deliberate, durable
decision rather than incidental.

## Decision

- **Scope-based bearer tokens.** A `TokenStore` issues tokens at one of three
  scopes — `ReadOnly` < `ReadWrite` < `Admin`. The `auth()` middleware enforces
  **scope-by-method** (`RequiredScopeForMethod`): safe methods need ReadOnly,
  mutating methods need ReadWrite. `adminProtect` gates whole-config replacement
  (`POST /api/v1/config/import`) at Admin and fails closed.
- **Per-session CSRF**, keyed by `sha256(bearer)` (`CSRFManager`), constant-time
  compared, fail-closed for mutating requests.
- **No usable default credentials.** A non-loopback bind refuses to start without
  a token (re-checked at request time as defense in depth); empty token values
  and token files wider than `0600` are rejected at load.
- **Authorization denials emit `event=auth.forbidden`** with `reason=scope` for
  SIEM filtering.
- **CORS is same-origin only** (reflects the origin only when it equals the
  request host); there is no arbitrary-origin reflection.

## Consequences

- Automation gets least-privilege tokens (read-only monitors can't mutate);
  whole-topology replacement requires an explicit admin token.
- The route-policy registry (ADR-0002) structurally enforces that mutating
  routes carry `csrfProtect` + the write rate limiter, so the scope/CSRF wrapping
  can no longer be forgotten on a new route.
- No login surface means no dedicated login rate limiter; the generic per-IP
  limiter throttles token-guessing, which is acceptable given high-entropy tokens
  + constant-time lookup. A future interactive-login requirement would revisit.

## Alternatives considered

- **Full user accounts + roles**: heavier than an operator/automation appliance
  needs; the scope model covers the privilege tiers that matter. Deferred.

## Related issues and PRs

- #798 (govulncheck pin), #800 (route-policy registry that locks in the wrapping)
