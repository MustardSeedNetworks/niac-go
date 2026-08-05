# NIAC product and architecture program

**Status:** Proposed; Milestone 2 requires owner amendment of locked `LICENSE_STRATEGY.md`
**Date:** 2026-08-04
**Execution ledger:** `2026-08-niac-program-execution-ledger.md`
**Code map:** `2026-08-niac-implementation-map.md`

## Decision

Build one NIAC simulation engine and one complete binary. Remove runtime
activation, Free/Pro feature gates, trials, fingerprints, and payment-derived
device limits. Retain explicit legal distribution and commercial-use terms.
Technical resource limits remain safety controls and never represent a tier.

Use that engine for three outcomes:

1. Curated EtherScope, CyberScope, and Link-Live demonstrations.
2. Deterministic SEED development, regression, and performance fixtures.
3. A supported standalone network-simulation product.

Goals 1 and 2 share protocol fidelity, state, faults, and scenarios. They differ
at delivery: demonstrations optimize for a guided, readable result; SEED needs
stable machine contracts, reset, seeded time, and assertions.

## Current implementation

The current branch already adds the correct data-plane foundation:

- `internal/daemon/session_registry.go` owns multiple simulation runtimes.
- `internal/daemon/trunk_capture.go` owns shared tagged ingress and enforced
  per-session egress.
- `internal/daemon/recovery.go` persists several launch intents.
- the runtime UI can list, select, and stop concurrent sessions;
- scenario packs and the Link-Live comparator have substantial in-flight work.

The remaining architecture gaps are concrete:

- `Daemon.simulation` and `Server.SelectSimulation` still create a mutable,
  process-wide selected session for most runtime APIs;
- capture termination is logged but does not fail or degrade affected sessions;
- trunk drops are counted internally but not identified by reason and exposed;
- license limits are evaluated per scenario and can be multiplied by sessions;
- `internal/devicestate` already supplies shared mutable state, checkpoints,
  events, and interface faults, but checkpoints omit faults and behavior replay
  is start-once, system-clock driven, and limited to four interface fault rates;
- Link-Live acceptance is strong but is not yet a repeatable release gate for
  every presentation pack;
- SEED lacks a versioned fixture contract, and its live SNMP poller currently
  resolves every target to empty credentials.

## Target architecture

```text
UI / CLI / SEED harness / acceptance runner
                    |
       session-scoped API and event streams
                    |
              Runtime manager
        +-----------+-----------+
        |           |           |
     Session A   Session B   Session C
   config/state config/state config/state
   fabric/clock fabric/clock fabric/clock
   faults/trace faults/trace faults/trace
        |           |           |
        +-----------+-----------+
                    |
    Capture broker per physical interface
      allowlist / dispatch / health / drops
                    |
          isolated lab attachment
```

### Runtime ownership

Each session owns configuration, compiled fabric, authoritative device state,
protocol stack, deterministic clock and seed, behavior controller, replay,
telemetry, lifecycle, and one physical binding. The runtime manager owns the
session registry and aggregate resource policy. There is no backend selected
session.

Every runtime route includes a session ID, for example
`/api/v1/sessions/{id}/topology`, `/devices`, `/stream/packets`, `/replay`, and
`/capture`. The browser may remember a selected row, but each request remains
explicit. Pre-v1 means old global routes are replaced in the same change rather
than retained as compatibility shims.

### Physical boundary

One capture broker owns each physical interface. It validates operator-owned
attachment policy, dispatches one frame to at most one session, enforces egress
VLAN identity, reports drops by reason and VLAN, and propagates terminal capture
failure to every dependent session. Scenario content cannot select an interface
or physical VLAN.

### Authoritative truth

One typed device-state model supplies packet handlers, SNMP, CLI, topology,
counters, events, and faults. A state transition is atomic and observable. The
same authored manifest is compared with runtime truth, tester discovery, and
Link-Live output so a demo cannot pass with contradictory interfaces or links.

### Scenario contract

A portable pack contains topology, profiles, initial state, deterministic seed,
behaviors, faults, expected observations, reset rules, and a versioned manifest.
Deployment supplies the session ID and physical binding. Presentation packs and
scale workloads remain visibly separate.

## Product experiences

### NetAlly demonstrations

Provide hospital, warehouse, manufacturing, campus, retail, and service-provider
packs on VLANs 200-205. Each pack has a short guided story, baseline discovery,
one or more controlled faults, expected EtherScope/CyberScope observations, and
a readable Link-Live topology. VLAN 299 is scale-only.

### SEED development

Provide versioned fixtures with a fixed seed and controllable clock. Tests can
start, reset, checkpoint, mutate, and stop a named session through an authenticated
API or CLI. Each fixture publishes expected OIDs, packets, topology, counters,
events, faults, and timing tolerances. SEED tests assert behavior rather than MIB
catalog counts.

### Standalone product

The customer UI composes and runs the same packs and engine. Distribution is one
full-function binary. Revenue should initially attach to maintained releases,
validated scenario packs, support, deployment, and custom modeling. A future
organization subscription may control downloads and support, but must not make
an installed air-gapped runtime stop working.

## Licensing migration

Remove `internal/license`, `niac license`, license API/UI surfaces, `TierGate`,
trial state, feature middleware, and `SimulationEntitlements`. Replace
`FreeTierDeviceCount` with aggregate resource validation plus the existing
absolute safety ceiling. Keep the shared foundation dependency where used for
CSRF. First move `ResolveConfigHome`, which SSH host-key storage also uses, to a
neutral operator-config package so license deletion cannot break key persistence.

Retain a legal license file. Owner approval must first replace the locked NIAC
Free/Pro strategy; counsel must then review the NIAC Additional Use Grant,
particularly its anti-circumvention language, before release. Update every plan,
README, roadmap, schema description, and customer-facing reference that still
claims routed labs or capacity are Pro features. This decision supersedes the
license placement sections in the July routed-lab and authoring plans.

## Delivery sequence and estimate

| Milestone | Outcome | Hours | Cumulative |
| --- | --- | ---: | ---: |
| 0. Stabilize current branch | Reviewable concurrent runtime foundation | 20-32 | 20-32 |
| 1. Session control plane | Explicit routes, events, health, aggregate limits | 32-50 | 52-82 |
| 2. Licensing simplification | One complete binary, legal terms retained | 30-50 | 82-132 |
| 3. Behavior/state extension | Deterministic transitions, reset, faults | 27-44 | 109-176 |
| 4. NetAlly release gate | Six packs accepted on both testers | 52-87 | 161-263 |
| 5. SEED contract | Credentials, fixtures, harness, assertions, baselines | 32-55 | 193-318 |
| 6. Commercial hardening | UX, packaging, docs, support and release evidence | 28-44 | 221-362 |

The complete program is **221-362 engineering hours**: about **6-9 focused
weeks for one engineer**. NetAlly-ready is approximately 161-263 hours depending
on how many current changes survive review. Two engineers can shorten elapsed
time only after Milestone 1; the state model and live hardware acceptance remain
critical-path work.

## Release gates

- No session can read, mutate, stream, replay, or stop another session without
  naming it explicitly.
- A 24-hour multi-VLAN isolation run emits zero frames outside approved tags;
  all capture errors and drop reasons are visible.
- The same seeded behavior produces equivalent state, packets, counters, and
  events after reset and daemon restart.
- Every presentation pack passes fresh final-binary EtherScope and CyberScope
  discoveries and unit-pinned Link-Live comparisons with zero actionable findings.
- SEED consumes shipped fixtures through public contracts and validates queried
  OIDs through collectors and consumers.
- `make lint`, `make fmt-check`, `make test`, `make test-e2e`, `make security`,
  and `make build` pass; CI packages contain the same embedded UI and metadata.
- A new operator can install NIAC, launch a supported pack, and complete its
  documented workflow without YAML repair or an activation step.

## Scope controls

Do not build a generic NMS, vendor image emulator, cloud control plane, license
server, dynamic routing suite, arbitrary MIB editor, or combined full-hospital
map. New protocols require an observed NetAlly, SEED, or paying-customer
workflow and an acceptance oracle.

## Market checkpoint

Verdict: Goals 1 and 2 are a deliberate non-market strategic build, capped by
Milestones 0-5 at 193-318 hours. Goal 3 is a conditional market bet: interview
at least five lab, QA, training, or support owners and obtain three concrete
pilot commitments. If that evidence is absent after accepted demo packs ship,
stop commercial expansion and continue NIAC as internal tooling, a demo
platform, and a paid-services accelerator.
