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
| M4-2 | Finalize hospital guided baseline and fault story | 6-10 | M4-1 | **Done 2026-08-08.** Imaging congestion authored at 88/84 on both ends of both uplinks; Link-Live flagged exactly those four interfaces and nothing else. Analysis `6a77de929dc61ad4321d622d`, 75/88, **zero findings**. The error half is limited by the injection surface - see below. |
| M4-3 | Finalize warehouse and manufacturing stories, including distinct per-vertical topology shape | 8-14 | M4-2 | **Done 2026-08-08.** warehouse `6a77af229dc61ad432e2528a` 57/67 and manufacturing `6a77a37d9dc61ad432d7a746` 69/78, both **zero findings**, both on product-API-generated configs. Three closets versus a cell ring - the maps do not look alike. |
| M4-4 | Finalize campus, retail, and service-provider stories | 12-20 | M4-2 | **Done 2026-08-08.** campus `6a77bb6b9dc61ad432ed59f2` 147/186 collapsed core, retail `6a77c25f9dc61ad432f3908d` 95/112 lane chain, service provider `6a77c98b9dc61ad4320a6858` 117/146 POP ring - **zero findings each**. |
| M4-5 | Validate VLAN 299 scale workload and responsiveness | 4-7 | M1, M3 | **Done 2026-08-08.** Baseline published below. |
| M4-6 | Extend the runner to pin unit/analysis and record final-binary, pack, binding, and timestamp provenance | 5-8 | M4-1 | Repeatable read-only acceptance report |
| M4-7 | Run 24-hour isolation plus EtherScope and CyberScope discovery for all six packs | 12-20 | M4-2..M4-6 | **Done 2026-09-06.** Isolation proven 2026-08-08 (`scripts/lab/isolation.sh`, zero leaks). 24-hour soak ran 2026-08-19 04:12Z to 2026-08-20 04:20Z on v0.94.46 (see below). EtherScope nXG discovery of all six packs on v0.95.6: four packs zero findings, two carry only the NBSTAT source-MAC defect (#1842, fixed in #1843) — see the EtherScope section below. |

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

### Scale baseline, 2026-08-08

The 531-device workload on VLAN 299, alone on CT304 (v0.94.30-4-g66fd1b5,
2 vCPU / 24 GB container on pvm01):

| Measure | Result |
| --- | --- |
| Session start | 1.1 s for 531 devices |
| Resident memory | 147 MB (181 MB with the six presentation packs, 560 devices) |
| CPU, steady state | idle between polls; `top` reports 0.0% |
| `GET /api/v1/devices` | 0.02 s |
| `GET /api/v1/topology`, `/api/v1/stats` | 0.01 s |
| SNMP, far device | 20/20 sequential gets in 0.13 s — 6.5 ms each |

**The scale workload and the six presentation packs cannot run together.** The
daemon's aggregate budget is 1000 devices; the six packs hold 560 and the scale
workload needs 531, so starting it alongside them is refused with
`device_capacity_reached`. That is the safety budget doing its job, and it
matches the standing guidance to keep VLAN 299 off the trunk during a demo.
Stop the presentation packs to run it.

⚠️ `bridge vlan add dev vmbr0 vid <N> self` was missing on pvm01 for VLANs 203,
204, 205 and 299 — added 2026-08-08. Without it a VLAN looks dead from pvm01
exactly like a NIAC fault. Every capture so far ran on VLAN 200, which is why it
had not surfaced.

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
one finding per interface (110 on manufacturing, 120 on warehouse); a _Refresh
Discovery_ takes the second sample and both went to zero. **The tester only
sweeps subnets listed under Discovery Settings -> Extended Ranges**, so a pack's
client and server subnets must be added before its first capture - manufacturing
went 40 -> 14 -> 0 findings as `10.91.210.0/24` and then `10.91.240.0/24` were
added.

### VLAN isolation, 2026-08-08

`scripts/lab/isolation.sh` walks each pack's VLAN in turn and checks two things:
its own devices answer SNMP, and no other pack's device space answers at all.
Every pack shares the same gateway address, so a leak would otherwise be silent

- a device answering from the wrong VLAN looks exactly like a device that is
simply there.

```text
vlan 200  hospital          own device answers as "MED-ACC-SW01"
vlan 201  warehouse         own device answers as "FUL-ACC-SW01"
vlan 202  manufacturing     own device answers as "PLT-ACC-SW01"
vlan 203  campus            own device answers as "NTH-ACC-SW01"
vlan 204  retail            own device answers as "HQ-ACC-SW01"
vlan 205  service-provider  own device answers as "NYC-ACC-SW01"

isolation failures: 0
```

Thirty cross-VLAN probes, none answered. What remains under M4-7 is duration -
a 24-hour soak - and the EtherScope half of the discovery evidence.

### The hospital story, and what the error half still needs

The guided baseline is authored, not injected: `MED-ACC-SW02` saturates both
uplinks at 88%/84% and the `MED-DIST-SW01/02` ends match, deliberately above the
80% line where Link-Live raises an interface Warning. On the first capture
Link-Live flagged exactly those four interfaces and the three devices they roll
up to, and nothing else. The comparator expects a warning on an interface
authored at or above the line, so the run is clean at zero findings.

**The error half of the story cannot be told on those links today.** Runtime
error injection (`POST /api/v1/errors`) only reaches interfaces the stack
instantiates, which for an access switch is its SVI:

```text
GET /api/v1/errors -> targets
  {"device": "MED-ACC-SW01", "address": "10.51.200.21", "interfaces": ["Vlan200"]}

POST {"device":"MED-ACC-SW02","interface":"HundredGigabitEthernet1/0/49",
      "errorType":"FCS Errors","value":7}
  -> 400 interface_not_found

POST {"device":"MED-ACC-SW02","interface":"Vlan200", ...}
  -> {"success": true}
```

So FCS errors can be demonstrated on a switch's management interface but not on
the congested uplink an engineer would actually be looking at. Closing that
means exposing authored trunk ports as fault targets, which is engine work
rather than pack authoring; it belongs with the M3-4 fault matrix.

Checkpoint: all six presentation VLANs are demonstrable without regeneration,
YAML repair, or ambiguous Link-Live selection.

### 24-hour soak, 2026-08-19 (M4-7 duration evidence)

`scripts/lab/soak.sh` against CT304 on v0.94.46, report at
`pvm01:/root/soak-v0.94.46/report.txt`, samples in `samples.jsonl`.

| Field | Value |
| --- | --- |
| started | 2026-08-19T04:12:24Z |
| ended | 2026-08-20T04:20:43Z |
| rounds | 95 at 900 s |
| daemon restarts | 0 |
| goroutines | 78 -> 78 |
| heap | 58 MB -> 93 MB |
| errors / drops | 0 / 0 |
| failures | 0 |

Heap grew 35 MB over 24 hours with a flat goroutine count; nothing in the
samples restarts or drops. Worth a second soak on a 0.95.x build before v1
(P5 hardening), not a blocker.

### EtherScope acceptance, 2026-09-06 (v0.95.6, M4-7 EtherScope half)

Same bar as M4-1: fresh Link-Live analyses of product-API-generated configs,
compared with `tools/linklive-acceptance` against the generated YAML. CT304
was upgraded 0.94.67 -> 0.95.6 first (Proxmox snapshot `pre-0956`; all six
sessions recovered, zero restarts), then every pack was regenerated through
`POST /api/v1/scenario/generate` and restarted on its VLAN. Unit
`00C017-536204`, second-sample uploads (Refresh Discovery before upload).

| Pack | VLAN | Authored | Analysis | Findings |
| --- | --- | --- | --- | --- |
| hospital | 200 | 75 / 88 | `6a9cff5c9e0a52ab6110b536` | 5 `missing-link`, all NetBIOS hosts (#1842) |
| warehouse | 201 | 57 / 67 | `6a9cf8ad9e0a52ab610b1d6e` | **zero** |
| manufacturing | 202 | 69 / 78 | `6a9d03369e0a52ab612009c4` | **zero** |
| campus | 203 | 147 / 186 | `6a9d0c2d9e0a52ab6127fdaa` | 2 `missing-link`, both NetBIOS file servers (#1842) |
| retail | 204 | 95 / 112 | `6a9d0c309e0a52ab612801b2` | **zero** |
| service-provider | 205 | 117 / 146 | `6a9d10569e0a52ab612b6cfd` | **zero** |

**The one product defect.** Every surviving finding is a missing link between
an access switch and a device that has a `netbios:` block. The switch side is
correct on the wire (Q-BRIDGE FDB, bridge-port mapping, PVID, status learned
— all identical in shape to the neighbouring devices that resolve). A capture
on pvm01 showed the cause: NBSTAT replies to an off-subnet requester leave
with the endpoint's own MAC, while ICMP and SNMP replies from the same
endpoint leave with the gateway's MAC. The tester therefore sees the
endpoint's MAC on its own segment and Link-Live attaches the device to the
unmanaged bridge behind the lab switch instead of its access port. Fixed
in PR #1843 (frame source from `replySourceMAC`, unit ID unchanged); the
hospital and campus rows need a fresh run on a build carrying it.

**Not a product defect, and it cost the first hospital pass.** The
EtherScope's 35 stored Extended Ranges covered an old `10.240.x` scheme, so
its first scan found 49 of 75 hospital devices while all 75 answered SNMP
from pvm01. The 24 current endpoint and server subnets are on the unit now;
the runbook records the check.

`scripts/lab/acceptance.sh` could not run at all on a current build until this
run: every mutating route requires the CSRF token, and the script never sent
one. It fetches it now and stops a stale session of the same id first.

For F3, the whole run was captured on pvm01: `/root/cap/etherscope-discovery-20260905-2114.pcap`
(`vlan and udp port 161`, 255 MB, 979,935 packets) covers every discovery of all
six packs by the EtherScope; it is the tester-side input the consumer demand
matrix asks for.

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
