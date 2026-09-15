# Architecture Decision Records

Durable record of significant architectural decisions for NIAC. Each ADR
captures Context, Decision, and Consequences so the reasoning survives the
people and the diffs. Format matches the sibling repos (seed/stem).

| ADR | Title | Status |
| ----- | ------- | -------- |
| [0001](0001-schema-generation-from-go-structs.md) | Schema generation from Go structs | Amended |
| [0002](0002-capability-registry.md) | Capability registry for route policy | Amended |
| [0003](0003-dependency-direction-depguard.md) | Dependency direction enforced by depguard | Amended |
| [0004](0004-scope-based-auth-posture.md) | Scope-based bearer-token auth + CSRF posture | Amended |
| [0005](0005-ed25519-signed-licenses.md) | Ed25519-signed license tokens | Superseded |
| [0006](0006-internal-api-sub-package-decomposition.md) | Decompose `internal/api` into isolated sub-packages | Amended |
| [0007](0007-json-wire-casing-convention.md) | JSON wire-casing convention (camelCase API, no exceptions) | Accepted |
| [0008](0008-multi-vlan-segment-playback.md) | Multi-VLAN segment playback (tagged + untagged, per-segment config) | Amended |
| [0009](0009-routed-fabric-and-physical-attachments.md) | Routed fabric separated from physical attachments | Amended |

Status values: Proposed · Accepted · Amended · Superseded.

Decisions recorded outside this directory: the replay-fidelity substitution
contract (`docs/design/2026-09-replay-fidelity-contract.md`, owner-signed
2026-09-05 and 2026-09-08) is an accepted decision record in everything but
its path.
