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

**Sequencing decision (owner, 2026-08-05): all scenario-shape work happens here,
last — not earlier.** Presentation packs share one infrastructure skeleton today,
so several verticals render as the same Link-Live map (warehouse and
manufacturing were byte-identical at 69 devices / 82 links). Giving each vertical
a realistic shape is real work, but doing it before M1-M3 means re-freezing golden
manifests and re-running live acceptance after every intervening milestone, since
any generator, MIB, packet or comparator change invalidates earlier live results.
Shape the scenarios once, against the finished engine, then validate once.

A prepared warehouse divergence (3 closets x 9 radios, 57 devices / 67 links) and
a validated 6-node ring spike for manufacturing are parked on
`park/m4-vertical-topology-differentiation` — rebase and finish them under M4-3
rather than re-deriving them. Hospital's 75-device / 88-link shape is already
Link-Live-validated and should stay unchanged if at all possible.

**Settled 2026-08-08: Link-Live keeps the ring.** A hand-closed 6-node access
ring (`PLT-ACC-SW01..06`, east and west on `Te1/0/47` and `Te1/0/48`) was
discovered by the CyberScope on analysis `6a774b2f9dc61ad4327b182a`, and
Link-Live reported all six adjacencies including the closing `SW06 - SW01` edge,
each with its own utilization sample. Its topology model carries the cycle rather
than reducing it to a tree, so ring generator work is worth doing and the cheaper
"fewer, larger cells" fallback is not needed. That spike config was hand-edited
from a generated one, which makes it a spike and never an acceptance artifact.

What the generator still needs is the ability to close a loop. `addEdge` already
expresses an arbitrary adjacency; `addSiteLAN` only ever walks tiers downward, so
nothing today emits an edge between two peers within one tier.

| ID | Work item | Hours | Depends on | Acceptance evidence |
| --- | --- | ---: | --- | --- |
| M4-1 | Freeze authored-truth manifest and comparator rules | 5-8 | M3 | **Done and signed off 2026-08-08** (v0.94.30). Hospital pack: 158 -> 1, then a confirming capture on the release binary at 152 -> 2 once APs joined the leaf rule. Both remaining findings are the bare-IP discovery-timing artifact, each wire-confirmed by an NBSTAT probe. Analysis `6a7740009dc61ad43270d928`. |
| M4-2 | Finalize hospital guided baseline and fault story | 6-10 | M4-1 | EtherScope/CyberScope/Link-Live evidence |
| M4-3 | Finalize warehouse and manufacturing stories, including distinct per-vertical topology shape | 8-14 | M4-2 | **Done 2026-08-08.** warehouse `6a77af229dc61ad432e2528a` 57/67 and manufacturing `6a77a37d9dc61ad432d7a746` 69/78, both **zero findings**, both on product-API-generated configs. Three closets versus a cell ring - the maps do not look alike. |
| M4-4 | Finalize campus, retail, and service-provider stories | 12-20 | M4-2 | **Done 2026-08-08.** campus `6a77bb6b9dc61ad432ed59f2` 147/186 collapsed core, retail `6a77c25f9dc61ad432f3908d` 95/112 lane chain, service provider `6a77c98b9dc61ad4320a6858` 117/146 POP ring - **zero findings each**. |
| M4-5 | Validate VLAN 299 scale workload and responsiveness | 4-7 | M1, M3 | Published resource baseline |
| M4-6 | Extend the runner to pin unit/analysis and record final-binary, pack, binding, and timestamp provenance | 5-8 | M4-1 | Repeatable read-only acceptance report |
| M4-7 | Run 24-hour isolation plus EtherScope and CyberScope discovery for all six packs | 12-20 | M4-2..M4-6 | 12 clean unit-pinned Link-Live comparisons |

### M4-1 outcome (2026-08-08, v0.94.29)

Findings on the hospital pack, one live CyberScope capture per stage, analysis
`6a767a6f9dc61ad432a4616f`:

| Stage | Findings |
| --- | ---: |
| Before | 158 |
| After the endpoint work (#1213-#1218) | 37 |
| After the comparator rules (#1219) | **1** |

Two long-standing finding classes turned out to be the comparator being wrong
about Link-Live rather than NIAC being wrong about the network. Both are now
documented at their implementation and guarded by tests that keep the real
checks alive:

- Link-Live measures interface utilization on **switch and router ports only**.
  It takes no sample from a leaf node even though every one of ours serves
  `ifHCInOctets`. This was the "zero utilization on WLCs and servers" gap.
- An endpoint that answers SNMP is filed as `SNMP Agent`, which is the correct
  reading of a clinical appliance.

Endpoint identity was reshaped to match how discovery actually names things:
personal computers carry no SNMP agent (so they file as Host/Client) and take
NetBIOS-legal names of 15 characters or fewer, since NetBIOS is now their only
name source. Appliances keep their agent and their readable asset names. mDNS
and NetBIOS node-status both work on the wire.

The single remaining finding is one nurse station rendered as a bare address.
It is discovery timing, not a defect — that host answers an NBSTAT probe with
its full name. **A second confirming capture was not obtained**: CyberScope
Discovery stalled twice at ~36 devices without walking past the first hop while
the simulation answered SNMP normally. Re-run before treating the pack as
finally signed off.

### Acceptance results, 2026-08-08

Every presentation pack, generated through `POST /api/v1/scenario/generate` and
discovered by the CyberScope on physical VLAN 200:

| Pack | Shape | Devices / links | Analysis | Findings |
| --- | --- | --- | --- | ---: |
| hospital | dual-homed access | 75 / 88 | `6a7740009dc61ad43270d928` | 2 |
| warehouse | three closets, wireless-dominant | 57 / 67 | `6a77af229dc61ad432e2528a` | 0 |
| manufacturing | cell ring | 69 / 78 | `6a77a37d9dc61ad432d7a746` | 0 |
| campus | collapsed core | 147 / 186 | `6a77bb6b9dc61ad432ed59f2` | 0 |
| retail | lane chain | 95 / 112 | `6a77c25f9dc61ad432f3908d` | 0 |
| service provider | POP ring | 117 / 146 | `6a77c98b9dc61ad4320a6858` | 0 |

Hospital's two are the bare-IP discovery-timing artifact, both wire-confirmed by
an NBSTAT node-status probe; every other pack is exactly clean.

Two tester behaviors decide whether a capture is usable, and neither is a NIAC
fault. **Link-Live computes utilization from two SNMP polls**, so uploading
straight after a clear-and-rerun reports `util 0` on every interface and files
one finding per interface (110 on manufacturing, 120 on warehouse); a *Refresh
Discovery* takes the second sample and both went to zero. **The tester only
sweeps subnets listed under Discovery Settings -> Extended Ranges**, so a pack's
client and server subnets must be added before its first capture - manufacturing
went 40 -> 14 -> 0 findings as `10.91.210.0/24` and then `10.91.240.0/24` were
added.

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
