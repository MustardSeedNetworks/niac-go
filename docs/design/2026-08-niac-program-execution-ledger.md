# NIAC program execution ledger

**Status:** Pending approval
**Parent:** `2026-08-niac-product-architecture-program.md`

Every item uses tests first, lands as a focused PR, and preserves CSRF, API
scope, attachment isolation, output encoding, and offline operation.

## Milestone 0 — stabilize current work

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M0-1 | Inventory and separate the current 89-file diff into coherent commits | 4-6 | — | Diff map; no unrelated or lost changes |
| M0-2 | Review session registry, replacement, recovery, and trunk lifecycle | 6-10 | M0-1 | Race and lifecycle tests pass |
| M0-3 | Review scenario and Link-Live comparator changes | 5-8 | M0-1 | Authored-truth fixtures pass |
| M0-4 | Run local quality and full-build gates | 5-8 | M0-2, M0-3 | Recorded clean command output |

Checkpoint: merge a stable foundation before licensing or API surgery. Do not
combine the current dirty work with the later milestones.

## Milestone 1 — session-scoped control plane

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M1-1 | Define session route and event contracts in OpenAPI | 4-6 | M0 | Contract tests fail then pass |
| M1-2 | Replace `Daemon.simulation` selection with explicit registry lookup | 5-8 | M1-1 | Concurrent lifecycle tests |
| M1-3 | Make API state, topology, devices, replay, capture, stats, and events session-scoped | 8-12 | M1-2 | Cross-session isolation matrix |
| M1-4 | Move browser selection to client context and explicit request paths | 4-7 | M1-3 | Multi-session Playwright flow |
| M1-5 | Expose capture health and drop counts by reason, interface, VLAN, and session | 4-6 | M1-2 | Failure and overload tests |
| M1-6 | Fail/degrade dependent sessions when shared capture terminates | 3-5 | M1-5 | Injected capture-failure test |
| M1-7 | Enforce aggregate session, device, scheduled-action, and queue admission budgets; expose CPU/memory telemetry | 4-6 | M1-2 | Multi-session capacity tests |

Checkpoint: two sessions may run, fail, restart, and be inspected independently
without backend selection or data-plane leakage.

## Milestone 2 — remove runtime licensing

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M2-1 | Owner amends locked license strategy; approve legal-use model and counsel review | 4-8 | M0 | Updated source of truth and approved text |
| M2-2 | Move `ResolveConfigHome` to neutral ownership, then delete backend manager, activation CLI, trial, catalog, and routes | 8-12 | M2-1 | SSH key, backend, and help tests pass |
| M2-3 | Delete License page/context and feature-gated UI components | 5-8 | M2-2 | UI tests show all supported capabilities |
| M2-4 | Replace entitlement checks with aggregate technical safety policy | 5-8 | M1-7, M2-2 | Capacity and routed-lab tests |
| M2-5 | Remove keygen/catalog coupling and stale packaging assumptions | 3-5 | M2-2 | Build and package inspection |
| M2-6 | Update README, help, OpenAPI, roadmaps, ADRs, and plan supersession notes | 5-9 | M2-3, M2-4 | Repository search finds no tier claims |

Checkpoint: a fresh offline install exposes the full supported product without
activation while still enforcing security and technical safety limits.

## Milestone 3 — extend authoritative state and behavior

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M3-1 | Freeze gaps in `devicestate`, behavior, checkpoints, and required service outcomes; define the clock seam | 4-6 | M1 | Contract and illegal-state tests |
| M3-2 | Extend `devicestate.Snapshot` only for required service/outcome state and keep all projections consistent | 5-8 | M3-1 | SNMP/CLI/packet/topology parity tests |
| M3-3 | Make behavior replay clock-controlled, restartable, resettable, and session-owned | 5-8 | M3-1 | Same-seed/reset equivalence tests |
| M3-4 | Add outcome faults missing from the four existing interface fault rates | 7-12 | M3-2, M3-3 | DHCP/DNS/link/PoE/latency fault matrix |
| M3-5 | Include faults and required runtime state in checkpoints, recovery, and event assertions | 6-10 | M3-4 | Restart and checkpoint replay tests |

Checkpoint: a named fault changes packet behavior, SNMP, topology, counters,
events, and recovery state consistently and repeatably.

## Milestone 4 — NetAlly acceptance

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M4-1 | Freeze authored-truth manifest and comparator rules | 5-8 | M3 | Versioned golden manifests |
| M4-2 | Finalize hospital guided baseline and fault story | 6-10 | M4-1 | EtherScope/CyberScope/Link-Live evidence |
| M4-3 | Finalize warehouse and manufacturing stories | 8-14 | M4-2 | Two clean comparisons |
| M4-4 | Finalize campus, retail, and service-provider stories | 12-20 | M4-2 | Three clean comparisons |
| M4-5 | Validate VLAN 299 scale workload and responsiveness | 4-7 | M1, M3 | Published resource baseline |
| M4-6 | Extend the runner to pin unit/analysis and record final-binary, pack, binding, and timestamp provenance | 5-8 | M4-1 | Repeatable read-only acceptance report |
| M4-7 | Run 24-hour isolation plus EtherScope and CyberScope discovery for all six packs | 12-20 | M4-2..M4-6 | 12 clean unit-pinned Link-Live comparisons |

Checkpoint: all six presentation VLANs are demonstrable without regeneration,
YAML repair, or ambiguous Link-Live selection.

## Milestone 5 — SEED development contract

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M5-1 | Extend scenario manifest v3 with versioned expected observations and compatibility data | 5-8 | M3 | Schema and consumer tests |
| M5-2 | In SEED, implement secure per-target SNMP credential resolution; remove the empty-credential stub | 8-14 | M5-1 | Real v2c/v3 poll tests |
| M5-3 | Build an authenticated shipped-binary start/reset/mutate/checkpoint/stop harness | 7-11 | M1, M3 | End-to-end harness test |
| M5-4 | Cover all ten SEED collectors and current topology/alert consumers with native or captured-walk fixtures | 8-14 | M5-2, M5-3 | Cross-repository observation assertions |
| M5-5 | Add CI orchestration, timing tolerances, and performance budgets | 4-8 | M5-4 | Stable repeated CI runs |

Checkpoint: SEED can reproduce a failed diagnostic from a fixture identifier and
seed, then assert the complete observation path without manual NIAC operation.

## Milestone 6 — standalone product hardening

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M6-1 | Complete non-technical install, first-run, pack launch, health, and recovery journey | 8-12 | M2, M4 | New-operator usability run |
| M6-2 | Complete scenario authoring timelines and validation report | 6-10 | M3 | Pack built with no YAML edits |
| M6-3 | Harden backup, restore, export, diagnostics, and support bundle | 5-8 | M1 | Restore and redaction tests |
| M6-4 | Validate macOS, Ubuntu, Fedora, Chrome, Edge, and Safari | 5-8 | M6-1..M6-3 | Platform matrix evidence |
| M6-5 | Finalize support policy, update delivery, commercial terms, and pilot materials | 4-6 | M2, M6-1 | Pilot-ready package |

Checkpoint: a supported release can be installed, operated, diagnosed, upgraded,
and restored by someone who did not build NIAC.

## Pull-request order

1. Current branch stabilization and concurrent-runtime tests.
2. Session API contract and registry-only control plane.
3. Session-scoped runtime surfaces and UI.
4. Capture health, drop telemetry, and aggregate safety budgets.
5. Legal text approval, then backend license deletion.
6. UI/documentation license deletion.
7. Existing state/behavior gap contract, then deterministic clock and reset.
8. Missing outcome faults, complete checkpoints, and cross-surface tests.
9. Acceptance manifest and runner.
10. Presentation packs and live lab gate.
11. SEED credential resolution, fixture contract, binary harness, and CI.
12. Customer workflow, platform matrix, and pilot release.

## Program evidence

Each checkpoint records branch and commit, test commands, package version and UI
hash, deployed host, session/VLAN binding, tester identity, Link-Live analysis,
comparator output, packet capture, isolation result, and unresolved findings.
Any generator, emitted MIB, packet, or comparator change invalidates earlier
affected live acceptance and requires a fresh final-binary run.
