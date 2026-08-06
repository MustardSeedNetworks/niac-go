# Concurrent VLAN scenario plan

**Status:** Implemented; live lab acceptance pending

## Outcome

NIAC runs several presentation scenarios at the same time behind one tagged
physical attachment. Each scenario owns one physical VLAN and one isolated
runtime. A tester selects a VLAN in its AutoTest profile and discovers only the
scenario assigned to that VLAN.

This keeps each Link-Live topology readable while allowing a lab operator to
switch demonstrations without stopping, regenerating, or redeploying NIAC.

## Replaced limitation

The daemon previously owned one `*Simulation`. A successful start published the
replacement and stopped the previous simulation. Physical attachment policies
supported only `direct` and `access`.

The implemented runtime uses one shared capture owner and a session registry;
running several daemon processes against subinterfaces remains unsupported.

## Lab VLAN assignments

| Physical VLAN | Scenario | Link-Live purpose |
|---:|---|---|
| 200 | Hospital | Presentation |
| 201 | Warehouse | Presentation |
| 202 | Manufacturing plant | Presentation |
| 203 | Campus | Presentation |
| 204 | Retail | Presentation |
| 205 | Service provider | Presentation |
| 299 | Enterprise scale | Stress testing only |

The physical VLAN is deployment identity and is not stored inside a portable
scenario pack. Internal virtual VLANs remain scenario-local metadata. Different
scenario runtimes may safely reuse the same IP prefixes and MAC-generation
rules because packets and lookup tables never cross runtime boundaries.

## Runtime architecture

One capture engine owns the physical interface. It validates the configured
tag allowlist and dispatches each received frame to the runtime mapped to that
tag. A runtime can emit only through its assigned tag. Untagged frames and tags
outside the allowlist are dropped and counted.

Each runtime owns its own:

- parsed configuration and compiled fabric;
- protocol stack, device state, counters, and behavior timelines;
- replay controller and lifecycle context;
- persisted launch intent and status;
- physical VLAN binding.

The API identifies runtimes with stable session IDs. Start, stop, status,
recovery, events, and UI operations are session-scoped. A failed start must not
alter any active session. A daemon restart restores every persisted session or
reports the failure for that session without affecting the others.

## UI contract

The runtime page shows a table of active scenarios with scenario name, session
ID, physical VLAN, device count, state, start time, and packet counters. Starting
a presentation pack requires selecting an operator-approved trunk attachment
and an unused allowed VLAN. Duplicate VLAN assignments are rejected before
start. Each row has its own stop and details actions.

The scenario picker continues to separate presentation maps from the enterprise
scale workload. The UI must not imply that combining every scenario into one
Link-Live discovery is a supported presentation mode.

## Safety invariants

- One process and one capture owner per physical interface.
- A physical VLAN can belong to at most one active session.
- A frame is delivered to exactly one scenario runtime.
- A runtime can transmit only on its assigned physical VLAN.
- The trunk allowlist is operator configuration, not browser input.
- Scenario YAML cannot name a host interface or physical VLAN.
- Mutating session routes retain CSRF and write-scope enforcement.
- Recovery files contain no credentials and are written atomically with mode
  `0600`.

## Delivery phases

### Phase 1: contracts and validation

- Add `trunk` physical attachment policy with an explicit tag allowlist.
- Add session IDs and VLAN bindings to preflight, start, status, and errors.
- Reject duplicate IDs, duplicate active VLANs, unapproved tags, and untagged
  trunk traffic.

### Phase 2: concurrent lifecycle

- Replace the daemon's single simulation pointer with a session registry.
- Make API publication, recovery, stop, and status session-scoped.
- Preserve atomic replacement semantics within one session.

### Phase 3: shared capture dispatch

- Give the physical interface one capture owner.
- Dispatch ingress by 802.1Q tag and tag egress at the boundary.
- Add packet, isolation, and shutdown tests before enabling live trunk mode.

### Phase 4: UI and operational setup

- Add concurrent-session controls and per-session telemetry.
- Configure CT304's Proxmox attachment for the approved VLAN allowlist.
- Create matching CyberScope AutoTest profiles.

### Phase 5: acceptance

For every presentation VLAN, run AutoTest, clear and rerun Discovery to 100%,
queue the discovery, switch to the management profile, upload, and compare the
fresh Link-Live analysis against that scenario's authored devices, links,
interfaces, traffic, errors, discards, and problems.

Every uploaded discovery must be identifiable without opening it:

- Name: `NIAC <Scenario> | VLAN <VID> | <NIAC version> | <YYYY-MM-DD>`
- Comment: state that it is UI-generated, include authored device/link counts,
  notable device profiles, the CyberScope identity, and the NIAC commit.
- Tags: `NIAC`, the scenario name, `VLAN-<VID>`, the NIAC version,
  `CyberScope`, and `Acceptance`.

Apply the name, comment, and tags in Link-Live immediately after upload and
verify them before selecting the analysis for comparison.

Completion requires readable visual maps and zero actionable comparator
findings for every presentation scenario. The scale workload is evaluated for
data fidelity and responsiveness, not presentation-map readability.
