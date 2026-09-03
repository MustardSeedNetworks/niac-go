# Plan: Customer scenario authoring

**Status:** Approved for phased delivery

**Date:** 2026-07-28

## Outcome

Let a network engineer build a credible, discoverable multisite simulation
without hand-writing YAML. The workflow must preserve NIAC's authored truth:
device identity, interfaces, peer ports, VLANs, routing, services, telemetry,
and faults must agree in the config, runtime, SNMP, and topology projections.

This plan starts after the released v0.94.22 enterprise scenario passed the
CyberScope-to-Link-Live acceptance path. It extends the existing configuration,
device editor, topology, library, and preflight capabilities rather than adding
a second builder.

## Product and market gate

**Verdict: build the authoring workflow; reshape the recorder; retire broad
protocol imitation.**

> **Amended 2026-09-02 — the buyer named below no longer exists.** This gate was
> written on 2026-07-28 against a paid NIAC Pro tier. The 2026-08-05 licensing
> amendment removed NIAC's runtime tier entirely: it ships as one unrestricted
> binary, and `internal/license/` has since been deleted from the repository, so
> Phase 1's "enforce license entitlements before persistence" is unimplementable
> by design and was not implemented.
>
> That does not invalidate the work, which is materially complete and whose
> engineering rationale — composable, verifiable scenarios against MIMIC's
> breadth — stands on its own. It reclassifies it. Under the fleet's
> marketability gate this is now a **deliberate non-market build**: the return is
> the CyberScope/Link-Live demo path and the lab lead-magnets, not a tier unlock.
> Revenue attaches to releases, support and custom modeling per
> LICENSE_STRATEGY.md 2.3.
>
> The original text is kept below rather than rewritten, so the decision that was
> actually made at the time stays legible.

- **User and pain:** network-management QA, support, demo, and lab engineers
  need realistic networks, but assembling hundreds of consistent device,
  interface, and topology records manually is slow and error-prone.
- **Buyer:** the engineering or lab owner who can purchase NIAC Pro at
  $599/year. Faster scenario creation is directly connected to product value.
- **Differentiation:** NIAC should make packet-backed DHCP, DNS, routing,
  discovery, SSH, SNMP, topology, and telemetry easy to compose and verify.
  MIMIC remains stronger at very large agent counts and broad protocol modules.
- **Distribution:** the workflow is demonstrable with CyberScope and Link-Live,
  while saved scenarios remain portable and usable by any monitoring product.
- **Kill criterion:** if three representative scenarios still require YAML
  repair after the composer ships, stop adding scenario types and repair the
  authoring model.
  - **Run 2026-09-02: not tripped.** All seven packs generate through the
    composer, load, and validate strict-clean — no errors, no warnings, no hand
    editing — and each still matches its frozen parity manifest. Executed as
    `TestEveryPackGeneratesWithoutYAMLRepair` in `internal/scenario/`, so it is a
    standing check rather than a one-off measurement.

## Competitive evidence

Gambit documents the following MIMIC strengths:

- A Recorder and Discovery Wizard can capture one device or a network and turn
  SNMP walks into reusable simulations.
- The Snapshot Wizard records changes over time and creates timed behavior.
- The Simulation and Topology wizards build devices, scenarios, and large labs.
- Running agents can be copied, paused, changed in groups, and modified through
  a MIB-oriented value browser.
- Optional modules cover NetFlow/IPFIX, sFlow, CLI, web services, MQTT, IPMI,
  Redfish, gNMI, and other interfaces.
- Published scale reaches 100,000 agents on supported 64-bit systems.

Sources: [Quick Start](https://doc.gambitcomm.com/mimic/quick.htm),
[Wizards Guide](https://doc.gambitcomm.com/mimic/wizards.htm),
[Simulator Guide](https://doc.gambitcomm.com/mimic/simulator.htm), and
[Simulator Suite](https://gambitcomm.com/site/mimic-simulator.php).

## Capability decisions

| Capability | Decision | NIAC direction |
| --- | --- | --- |
| Draft-first visual authoring | Build | Edit and validate without starting a simulation |
| Device profiles and cloning | Build | Vendor/model/role profiles with deterministic identity |
| Site and fleet generation | Build | Generate layers, counts, addressing, and connections |
| Visual topology editing | Build | Add/connect/move devices and edit both link endpoints |
| Walk-based device capture | Reshape | Use sanitized net-snmp walks; do not build a MIB compiler |
| Timed behavior and faults | Build | Saved traffic/fault phases over authoritative state |
| Runtime value editing | Build smaller | Typed device/interface controls, not arbitrary MIB cells |
| Scenario library | Build | Versioned vertical scenarios with validation manifests |
| 100,000-agent parity | Defer | Benchmark customer-relevant sizes before scale work |
| NetFlow/IPFIX and sFlow | Gate later | Add only for an accepted monitoring workflow |
| MQTT, IPMI, Redfish, cable | Retire | Outside NIAC's network-simulation product boundary |
| Arbitrary web/CLI recording | Retire | Prefer typed, stateful protocol implementations |
| Vertical protocol identity | Build | Discovery surface only — see below (2026-08-07) |

**Vertical application protocols (amended 2026-08-07).** DICOM, HL7,
EtherNet/IP, Modbus, ONVIF and SIP sit in the same class as the retired MQTT and
Redfish entries, so they are bounded rather than adopted wholesale: NIAC
implements each protocol's **identity and discovery surface only** — enough that
a scanner fingerprints the device correctly — and none of the application
workflow behind it. A real DICOM C-STORE or CIP connection will fail by design.
Full scope in [vertical device realism](2026-08-vertical-device-realism.md).

## Authoring journey

1. Choose a starter scenario, import a saved network, capture a walk, or start
   empty.
2. Define sites, network layers, naming, address pools, VLANs, and defaults.
3. Add a device profile once, then multiply it by site, floor, or role.
4. Connect devices visually; NIAC assigns compatible local and remote ports.
5. Add services, traffic, faults, and optional timed phases.
6. Review an authored-truth report for identities, links, interfaces, routing,
   services, telemetry, warnings, and license limits.
7. Save a portable scenario, run preflight, bind the physical interface, and
   start only after validation passes and operator-approved attachment policy is
   revalidated.

## Delivery phases

### Phase 1: Draft scenario contract

**Status: Complete**

- Introduce a saved draft that is independent of the running simulation.
- Reuse the canonical converter/config validation pipeline.
- Add create, read, replace, and delete operations under one authenticated,
  CSRF-protected library boundary.
- Register mutations through `internal/api/routes.go` with the write rate limit;
  require `ScopeReadWrite` for ordinary draft changes and `ScopeAdmin` for
  destructive whole-library operations.
- Persist drafts atomically and require revision/ETag matching on replacement
  so concurrent browser sessions cannot silently overwrite each other.
- Enforce license entitlements before persistence; draft operations must never
  change `Server.configPath`, call `ApplyConfig`, or restart runtime resources.
- Make the wizard prepare a draft instead of starting the runtime before edits.
- Acceptance: editing a draft cannot change or restart the running simulation.

Estimate: 8-12 engineering hours.

### Phase 2: Profiles and deterministic fleet generation

**Status: Complete**

- Define device profiles from existing templates and synthesized walk models.
- Add repeat controls for site, role, count, hostname, IP, MAC, and VLAN rules.
- Generate vendor-correct identity and occupied interfaces only.
- Acceptance: the same request produces byte-equivalent validated YAML.

Estimate: 10-16 engineering hours.

### Phase 3: Visual topology composer

**Status: Complete**

- Extend the existing React Flow topology page into an explicit edit mode.
- Support add, connect, disconnect, move, and link-property editing.
- Select peer ports as a pair and prevent duplicate or impossible occupancy.
- Acceptance: the four-site reference can be rebuilt without YAML edits.

Estimate: 16-24 engineering hours.

### Phase 4: Walk capture and profile creation

**Status: Complete**

- Import or capture a net-snmp walk through a bounded, cancelable operation.
- Accept credentials only for the capture request and never persist or return
  them in the profile, logs, draft, history, or generated content.
- Sanitize secrets and identity before saving reusable content.
- Infer device class, vendor/model, interfaces, and supported SNMP data, then
  require review before creating a profile.
- Acceptance: a supported switch walk becomes a reusable profile with no
  manual file movement.

Estimate: 10-16 engineering hours.

### Phase 5: Saved behavior timelines

**Status: Complete** — landed in `072af571 feat: add saved behavior timelines
(#1163)`. `internal/api/handlers_draft_behaviors.go`,
`ui/src/components/wizard/DraftBehaviorComposer.tsx`, and
`ui/e2e/behavior-timeline.spec.ts` cover the model, the composer and the
acceptance.

- Compose named phases from existing traffic and fault capabilities.
- Support deterministic start offsets, duration, repetition, and reset.
- Keep every transition in authoritative device state and emit observable
  counters, notifications, and topology status.
- Acceptance: a link-degradation exercise replays identically after restart.

Estimate: 12-20 engineering hours.

### Phase 6: Scenario packs and release acceptance

**Status: Complete** — landed in `b9c91267 feat: add versioned scenario packs
(#1165)`, retiered by `12566d00`, with Link-Live comparison in `99b7bd1d`.
Seven packs ship against the six this phase asked for: hospital, warehouse,
campus, enterprise-scale, retail, manufacturing and service-provider.

Browser acceptance was the last gap and is now closed: the `edge` Playwright
project drives the real installed Edge via `channel: 'msedge'` over the
authoring journey (scenario pack, device editor, behavior timeline).

- Build hospital, warehouse, campus, retail, industrial, and service-provider
  packs through the composer, not bespoke generators.
- Validate Chrome, Edge, and Safari as first-class authoring browsers.
- Run CyberScope/Link-Live acceptance for topology-facing scenarios.
- Drafts cannot grant or widen interface/VLAN attachment permissions; final
  egress must revalidate the operator-approved attachment policy.
- Acceptance: each pack passes config, runtime, browser, discovery, and authored
  truth checks and includes a versioned manifest.

Estimate: 8-14 engineering hours per scenario family, plus 8-12 hours for
cross-browser and lab release acceptance.

## Pull-request sequence

1. Draft storage and API contract.
2. Draft-first wizard migration.
3. Profile and deterministic generator core.
4. Fleet-generation UI.
5. Topology edit contract and backend mutations.
6. Topology composer UI.
7. Walk capture and profile review.
8. Behavior timeline model and runtime.
9. Behavior timeline UI.
10. Scenario packs, browser gates, hardware acceptance, and release.

Each pull request must remain independently reviewable, use tests first, retain
the route security invariants, and update the capability index when it creates a
new canonical implementation location.

## Definition of complete

- A customer can build, save, validate, and run the four-site enterprise
  scenario without editing YAML.
- Draft edits cannot mutate a running simulation.
- Generated device identity, interfaces, peer links, VLANs, routing, services,
  counters, and faults agree across config, runtime, SNMP, and topology.
- The composer produces reusable scenario packs rather than one-off fixtures.
- Chrome, Edge, and Safari pass the critical authoring journey.
- A fresh CyberScope discovery renders the intended topology in Link-Live.
