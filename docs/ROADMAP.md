# NIAC Roadmap

NIAC is in pre-1.0 stabilization. Product features are frozen until every exit
criterion below is complete and verified from a release-built artifact.

## What v1.0 Is

An authored or captured network, replayed from the shipped binary on a
supported platform, produces the authored truth in the consuming NMS,
including injected faults, repeatably and from CI.

Every exit criterion below is that sentence made checkable. A criterion is met
only when its verification has actually run and its output is recorded --
merged code is not evidence.

## Approved Delivery Order

Complete and verify a release at each boundary before advancing to the next
phase. Fix defects found during implementation and acceptance; a passing retry
does not close a flaky test.

The closeout also covers all other remaining pre-v1 plan items: routing MIB
content, consumer-demand coverage, capture-backed packs after their prerequisites,
and the final P6 release gate. Recheck recorded completions against their evidence;
do not silently omit a prerequisite because it belongs to an earlier phase.

| Phase | Remaining scope | Completion evidence |
| --- | --- | --- |
| P2 — Faults | Runtime recovery, link/fault syslog, and the remaining device-outcome faults. | Fault/state recovery tests, real-wire assertions, and a verified phase release. |
| P3 — Second consumer | Authenticated release-binary harness, six-pack consumer comparisons, and CI orchestration. | Zero findings plus three consecutive green scheduled runs, followed by a verified phase release. |
| P5 — Product hardening | Install/upgrade paths, authoring and routing content, platform/browser matrix, support materials, and flake closure. | Per-platform evidence, ten consecutive qualifying green merge-queue runs, required owner acceptance, and a verified phase release. |

The v1 readiness plan tracks individual tasks and acceptance evidence. Preserve
human sign-off and new-operator usability requirements; automated checks cannot
stand in for those observations. Deferred v1.1 work remains outside this sequence.
The final v1.0 release remains gated by every exit criterion below.

## Pre-1.0 Exit Criteria

### Replay fidelity

- [x] A walk replayed through the SNMP agent is byte-identical to its source
      except for a signed, documented list of substitutions, and every
      unclassified row fails the build.
- [x] The same comparison holds over the wire, not only in process, so
      datagram-level effects such as the response-size budget cannot hide.
- [ ] The wire suite runs nightly against the merged tree and is green three
      consecutive times.

### Faults

Implementation checkpoint (2026-09-10): resource telemetry is merged in
v0.95.45. Captive-portal authoring and live device-fault controls merged in
PR #2021, with real TCP isolation/clearing and saved-configuration browser
checks. Reboot/STP implementation passes isolated Linux packet tests for
notifications, telemetry, no replay of actions recorded in recovered state,
and scenario VLAN tags;
UI parity and final integration review remain in progress. DHCP isolation
and explicit address faults remain uncompleted.
The phase boxes below track acceptance, not individual implementation merges.

- [x] Injected interface faults are observable through IF-MIB and
      EtherLike-MIB counters while preserving monotonic counter behavior.
- [ ] Every fault type is asserted on the wire, and its MIB effect is named as
      an expected substitution so a fault cannot silently move a row it does
      not own.
- [ ] Fault and runtime state survive checkpoint, restart and recovery with an
      identical event sequence.
- [ ] Link and fault events are emitted as RFC 5424 syslog from the same seam
      as the existing traps.

### Second consumer

- [x] A network analyzer discovers all six scenario packs with zero findings,
      recorded with per-pack analysis identifiers.
- [ ] An authenticated harness drives start, reset, mutate, checkpoint and
      stop over the API against a release-built binary.
- [ ] A second product's collectors, topology and alert consumers run against
      all six packs with zero findings, orchestrated from CI on three
      consecutive green scheduled runs.

### Release candidate

- [x] Route, schema, output-encoding, token-discipline and i18n gates are
      enforced in CI rather than asserted in review.
- [ ] Lint, formatting, unit, integration, browser, security, package, install
      and deployment validation all pass for the release candidate.
- [ ] Install and first-run succeed on deb, rpm, pkg, Windows and a container
      image, without crash-looping an existing configuration.
- [ ] The platform matrix is recorded with per-cell command output.
- [ ] The browser suite holds a zero flake budget across ten consecutive
      merge-queue runs.

### Documentation and authoring

- [x] Stale roadmap, compatibility, licensing and deployment claims are
      removed from active documentation and superseded issues are closed.
- [x] The OpenAPI description is generated from the route registry and drift
      fails the build.
- [x] An authoring guide carries one complete annotated scenario, and the
      schema descriptions carry the rules an author needs.
- [ ] Routing MIB content is generated from the authored routes and networks,
      so path analysis in the consuming NMS resolves hops through a replayed
      multi-site pack.

No new product capability enters the pre-1.0 line until these criteria pass.

## v1 Product Boundary

NIAC simulates configurable network devices and protocol behavior for lab,
monitoring, discovery, and troubleshooting workflows. The supported surface
includes the CLI, embedded web UI, API, topology and scenario configuration,
packet capture analysis and replay, and the protocol implementations shipped in
the current binary.

Wired identity, discovery and monitoring surfaces are in v1: SNMP walk replay,
ARP, DHCP, DNS, LLDP and CDP, LLDP-MED with vendor TLVs, POWER-ETHERNET-MIB
with a matching power-loss fault, SNMPv3 traps and informs, and routing MIB
content.

## After v1.0

Selected through the project feature-scope and marketability gates, not
precommitted. The following are already scoped and deliberately held back:

| Capability | Why it is not v1 |
| --- | --- |
| Wi-Fi device model: SSIDs, BSSIDs, client tables, authored roams, IEEE802dot11-MIB | The v1 sentence above does not require it, its acceptance spans more than one product, and it is the model the real-radio work is built on -- so the two belong together |
| 802.11 implemented over virtual and then physical radios | Needs hardware and a nightly radio harness; sits directly on the Wi-Fi model above |
| NetFlow v9, IPFIX and sFlow export | Additive to a v1 that already satisfies its own definition |
| Endpoint application banners for the vertical packs | As above |
| 802.1X authenticator and a RADIUS device | As above |
| Streaming telemetry, NETCONF, gNMI | Held until a buyer names them |

NIAC ships as one unrestricted binary. There is no runtime tier, no activation,
and no phone-home. Resource ceilings are technical safety limits rather than
entitlements: 1,000 devices for one configuration, plus daemon-wide budgets
bounding concurrent sessions and total devices.

## Release Process

Release Please owns version selection and creates the release PR. Merging that
PR creates the tag; the release workflow then builds, signs, attests, and
publishes the platform artifacts. See [Distribution](DISTRIBUTION.md) for the
artifact matrix and validation contract.
