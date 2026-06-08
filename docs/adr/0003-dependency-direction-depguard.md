# ADR 0003: Dependency direction enforced by depguard

| Status  | Date       | Deciders         |
|---------|------------|------------------|
| Accepted | 2026-06-07 | @krisarmstrong   |

## Context

NIAC is already capability-first/flat (no hexagon rings). The simulation/domain
core — `internal/protocols`, `internal/converter`, `internal/device`,
`internal/mibdb` — should not depend on the web/API layer (`internal/api`); the
transport is composed at `internal/api` and `internal/daemon`. This direction
was clean **by convention** but unenforced, so a future upward import would
compile and ship silently. (Unlike stem, niac's `internal/api` is legitimately
imported by orchestration layers — daemon/ipc/replay — so the rule targets the
genuinely-inward domain packages, not a blanket "nothing imports api".)

## Decision

Add a `depguard` rule (`domain-core-inward-only`) to `.golangci.yml` barring
`internal/{protocols,converter,device,mibdb}` from importing `internal/api`.
golangci-lint runs the strict golden config, so the direction becomes a hard CI
gate.

## Consequences

- An upward import from the domain core now fails CI with an actionable message
  instead of silently coupling core to transport — enforcement by construction.
- Clean on the current tree (`depguard: 0 issues`).

## Alternatives considered

- **Convention + code review** (status quo): left the direction unenforced.
  Rejected.

## Related issues and PRs

- #799 (this rule)
