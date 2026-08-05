# NIAC implementation map

**Parent:** `2026-08-niac-product-architecture-program.md`
**Ledger:** `2026-08-niac-program-execution-ledger.md`
**Purpose:** Minimize rediscovery and duplicate implementations during execution.

Before every PR, read `CODE_INDEX.md`, the neighboring tests, and the applicable
design document. Extend the canonical capability below, update `CODE_INDEX.md`
when ownership changes, and do not mix the current concurrent-session diff with
later architecture work.

## Reuse map

| Need | Extend these symbols | Start with these tests | Do not create |
| --- | --- | --- | --- |
| Session lookup/lifecycle | `daemon.sessionRegistry`, `Daemon.StartSimulation`, `StopSimulation`, recovery launch intents | `session_registry_test.go`, `concurrent_sessions_test.go`, `recovery_test.go` | Another runtime manager |
| Shared VLAN capture | `daemon.trunkCapture`, `fabric.PhysicalBinding`, final egress policy in `stack_threads.go` | `trunk_capture_test.go`, `egress_policy_test.go` | Per-session physical capture |
| Session API state | `api.simulationAPIState`, route registry in `api/route.go`, SSE message `SessionID` | `route_test.go`, SSE hub/serve tests | Query-parameter session selection |
| Device truth | `devicestate.Store`, `Snapshot`, events, faults; bindings in `stack_device_state.go` | `internal/devicestate/*_test.go`, SNMP device-state tests | A parallel health/state store |
| Behaviors | `behavior.Schedule`, `Runner`, `protocols/stack_behavior.go`, draft behavior composer | behavior, config behavior, stack behavior, UI composer tests | A second scheduler |
| Interface faults | `store_fault.go`, `stack_fault.go`, `snmp/fault_telemetry.go` | their existing table tests | Synthetic counter-only faults |
| Scenario fixtures | `scenario.Request`, `Manifest`, `Pack`, `Generate` | generator, pack count/truth tests | Separate demo and SEED generators |
| Discovery truth | SNMP MIB builders, `topology.Topology`, `acceptance/linklive.Compare` | authored IF-MIB, topology truth, comparator equivalence tests | A second comparator |
| UI flow | `NewSimulationWizardPage`, `RuntimeControlPage`, `api/client.ts` | component tests and `gui-daemon.spec.ts` | A separate demo application |

## Exact gaps to close

1. `Daemon.simulation`, `Server.selectedSimulation`, `SelectSimulation`, and
   helpers such as `currentStack()` still select global runtime truth.
2. The route registry supports exact and trailing-slash prefix handlers. Add
   `/api/v1/sessions` plus `/api/v1/sessions/`; parse and validate the ID and
   subresource once, then dispatch. Keep route security policy centralized.
3. Audit every `currentConfig()`, `currentStack()`, and `currentTopology()` caller.
   Keep draft/library authoring global; convert runtime topology, devices,
   interfaces, segments, errors, neighbors, behaviors, replay, capture/filter,
   stats, reads, walks, synthesis, and debug to explicit session lookup. Delete
   each old global runtime route with its replacement in the same pre-v1 PR.
4. `trunkCapture.drops` is only an aggregate counter. Add bounded reason/VLAN/
   session visibility and propagate a terminal capture error to dependent state.
5. Admission policy should bound active sessions, aggregate devices, scheduled
   actions, and queues. Report CPU/memory; do not promise hard OS quotas here.
6. `behavior.Runner.now` does not control `waitUntil`, which calls `time.Until`.
   Inject one session clock/timer seam and add reset/restart; avoid replacing
   unrelated wall-clock timestamps across the whole codebase.
7. `devicestate.Store` already owns running/startup/authored configuration,
   faults, events, and checkpoints. Checkpoints currently copy only
   `configuration`; include faults and only the service state required by an
   accepted scenario outcome.
8. Existing fault types cover FCS errors, discards, interface errors, and high
   utilization. Add link, DHCP, DNS, PoE, and latency only where the packet,
   SNMP, CLI, topology, and event observations are specified.
9. Scenario manifest version 3 has counts and three hashes only. Bump it once to
   include scenario/seed identity, interface truth, expected observations,
   behavior outcomes, and timing tolerances. Physical interface/VLAN is never
   pack content.
10. The Link-Live runner already selects and compares analyses. Extend its
    report with NIAC version/commit/UI hash, pack/version/manifest hash,
    session/VLAN, unit MAC, and analysis timestamp. Do not duplicate comparison
    logic or mutate/upload Link-Live data without a proven supported API and
    explicit authorization.
11. `protocols/ssh_host_key.go` imports `internal/license` only for
    `ResolveConfigHome`. Move that helper and its tests to neutral ownership
    before deleting licensing. Keep `foundation` because CSRF still uses it.
12. Runtime-license removal contradicts the currently locked company strategy.
    Do not start Milestone 2 until the owner updates `LICENSE_STRATEGY.md`.

## Target session API

| Method/path | Ownership |
| --- | --- |
| `POST /api/v1/sessions/preflight` | Compile and authorize an intended ID/binding without mutation |
| `GET /api/v1/sessions` | Collection status and per-session summaries |
| `POST /api/v1/sessions` | Preflight-validated start with explicit ID/binding |
| `GET /api/v1/sessions/{id}` | One lifecycle/health/resource snapshot |
| `DELETE /api/v1/sessions/{id}` | Stop one session; rate limit plus CSRF |
| `GET /api/v1/sessions/{id}/stream/{kind}` | Path-scoped packet/stats/status SSE; no `?sessionId` selection |
| `/sessions/{id}/{resource}` | Topology, devices, interfaces, segments, errors, neighbors, behaviors, replay, capture, stats, read, debug |

The browser may keep a selected session ID in React context, but every request
passes it. Keep process-wide logs/metrics explicitly separate from session
streams, and never label global metrics with unbounded user-supplied session
IDs. OpenAPI, route manifest tests, handwritten TypeScript response types, and
Playwright flows change in the same PR as each route family.

## SEED compatibility contract

SEED registers exactly ten collectors in
`internal/polling/snmp/orchestrator/orchestrator.go` despite comments saying
eleven. Do not invent an eleventh.

| Collector | Required root | NIAC path |
| --- | --- | --- |
| `sys_info` | `1.3.6.1.2.1.1` | Native MIB synthesis |
| `if_table` | IF-MIB and IF-X | Native state/fault telemetry |
| `lldp`, `cdp` | LLDP remote and CDP cache | Native peer topology |
| `arp` | `1.3.6.1.2.1.4.22.1` | Native ARP topology |
| `fdb` | bridge port and Q-BRIDGE FDB | Native peer/Q-BRIDGE topology |
| `routing` | `1.3.6.1.2.1.4.24.4.1` | Captured walk today; decide whether native output is required |
| `host_resources` | `1.3.6.1.2.1.25` | Captured walk today |
| `bgp4_mib` | `1.3.6.1.2.1.15.3.1` | One captured-walk source today |
| `fdp` | Foundry enterprise cache | No bundled match found; author or explicitly exclude with evidence |

The harness uses released NIAC and SEED binaries over an isolated network. It
creates a SEED `/api/v1/polling-targets` resource pointing at a NIAC device,
waits for `lastStatus`, and verifies persisted `snmp_observations` through public
SEED behavior or a purpose-built test adapter. It must first replace
`poller.credentialsForTarget`, which currently returns empty credentials even
when `CredentialsID` is present. Never import either product's `internal`
packages from the other repository.

## Fast PR cuts

1. Stabilize and merge the existing session/trunk/scenario/comparator work.
2. Add session route contract and registry lookup; delete selected backend state.
3. Migrate runtime API families and UI one family at a time; finish with SSE.
4. Add capture health/drop reasons and aggregate admission policy.
5. Extract config-home ownership, then remove backend and UI licensing.
6. Fix behavior clock/reset and checkpoint completeness using existing stores.
7. Add only scenario-required outcome faults and cross-surface tests.
8. Bump manifest v3 once; extend the existing Link-Live report.
9. Land SEED credential resolution in a separate SEED PR.
10. Add cross-binary collector fixtures, then live NetAlly acceptance evidence.

## Per-PR execution checklist

- State one invariant and one smallest failing behavior test before editing.
- Name the canonical capability and rejected duplicate in the PR description.
- Search every caller before changing a public interface; pre-v1 replaces all
  callers without aliases or fallbacks.
- Keep security wrappers at route registration and physical bindings outside
  portable scenario data.
- Run focused tests, then lint, format, full tests, security, E2E when applicable,
  and full build. Record exact binary metadata and invalidate affected live
  acceptance whenever generator, MIB, packet, or comparator output changes.
