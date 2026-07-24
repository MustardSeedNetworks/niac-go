# NIAC local-plan intake

**Status:** Reconciled; live acceptance remains

**Date:** 2026-07-23

This tracked record preserves and reconciles five formerly gitignored plans.
The [master closeout plan](2026-07-niac-master-closeout-plan.md) is their only
remaining owner.

## Nearest-switch plan

### Delivered

- PR #869 implemented NIAC-side peer topology synthesis.
- The stack builds the device MAC roster, synthesizes learned remote-MAC FDB
  entries and LLDP/CDP chassis identifiers, and maps bridge ports to
  walk-backed interface indexes.
- Regression tests cover peer FDB and bridge-port behavior.

### Remaining acceptance

- Attach endpoints to realistic access-switch interfaces in the demo catalog.
- Emit real CDP/LLDP port identifiers and regenerate the catalog manifest.
- Verify the FDB and bridge-port-to-interface chain by SNMP.
- Confirm CyberScope reports the expected nearest switch, port, and VLAN.

Stage 5 owns this catalog and live-lab evidence.

## UI architecture and completeness plan

### Delivered

- PRs #812-#813 closed the shortcut, theme ownership, dead template page, and
  Vite output-contract findings.
- PRs #921-#925 closed the recorded correctness batch: dead UI, PCAP size
  handling, auto-scroll, filtered export, decoded-layer boundaries, surfaced
  failures, corrected copy, password masking, mutation confirmation, and
  unsaved-change protection.
- PRs #926-#929, #931-#932, #936-#939, #951-#953, and #955-#957 delivered
  the naming/IA cleanup, dashboard and simulation status, confirmation and
  toast consolidation, accessibility disclosures, i18n expansion,
  template-preview clarification, and shared table behavior.
- PRs #940, #943-#948, #954, #960-#963, and #965-#967 delivered batch walk
  validation and the recorded backend/UI enablers, including structured
  errors, error-injection interfaces, replay progress, batch deletion, guided
  configuration, sanitize, content management, and upload progress. Commit
  `abbd929e` carries the BPF error work from closed PR #942.

### Final disposition

- Removed stale compatibility routes and breadcrumbs for `/pcap-analyzer`,
  `/analysis`, `/templates`, and `/neighbors`.
- Retired the botanical module-color references and stale feature-number
  comments.
- Topology page extraction and spacing cleanup remain low, opportunistic
  maintenance debt.
- Missing MIB archive, template-apply, dump, and packet-analysis parity are not
  commitments. They require a concrete user need and feature gate.

## Modernization plan

### Delivered

- PRs #980, #982, and #983 completed the accepted grouped Go modernization.
- PR #985 moved content extraction containment to `os.Root`.
- PRs #987 and #993 made replay opening root-contained and pure Go.
- PRs #997-#1000 delivered streaming replay, rate modes, bounded loop count,
  and replay filtering; PRs #1001-#1002 exposed and localized the controls.

### Retired

- Issues #988 and #991 record decisions not to adopt React Compiler or TanStack
  Query. PRs #989 and #992 removed the unused dependencies.
- `React.FC`, router, memoization, and sorting-style sweeps remain retired
  churn unless a new measured defect changes the decision.

### Final disposition

- Regression coverage proves a truncated PCAP tail does not inflate replay
  packet accounting.
- Endpoint rewrite, VLAN rewrite, expanded throughput reporting, and Inspector
  export/dissection remain optional expansion ideas and are not active plans.

## Cruft and duplication plan

### Delivered or absorbed

- PR #1012 removed the dead MIB database package and other unused subsystems.
- PR #921 removed the recorded dead UI components and storage-key leak.
- The UI cleanup and parity work listed above absorbed the toast, confirmation,
  polling, i18n, shared-table, and orphan-route findings.
- Path containment findings are absorbed into #1035 and the delivered
  `os.Root` work; do not create a parallel security owner.

### Final disposition

- Removed the dead history-rewrite and token-migration scripts, the empty
  generated-types placeholder, and stale comments.
- Reverified the recorded helper and API duplication findings against the
  current capability index and lint results. No unique high or medium defect
  remains; broad rename or abstraction sweeps are retired.
- ADR-0001 now records that the YAML authoring schema is not a REST response
  schema. REST TypeScript generation needs a real wire-contract source and a
  separate accepted decision.
- The file-size check now has a reviewed baseline and blocks new or growing
  red-flag files. Its regression test proves both rejection paths.

## Replay positioning plan

### Delivered

- Replay streaming, topspeed/pps/Mbps modes, loop count, and replay filtering
  satisfy four of the five original credibility gates.
- The catalog, live responders, UI controls, and root-contained replay path are
  real product capabilities and may be described accurately.

### Final disposition

- Active documentation now describes NIAC as a simulator and test target.
  Replay claims are limited to delivered timing, rate, loop, filter, catalog,
  and interactive behavior.
- Protocol and operator-surface claims were corrected against the shipping
  product.
- Packet rewrite, VLAN rewrite, completion statistics, explicit
  pcapng/nanosecond coverage, Inspector expansion, and protocol-specific
  analysis require separate feature gates.
- Dual-interface bridging, line-rate acceleration, true TCP reassembly, expert
  analysis, and broad Wireshark parity are retired absent a concrete user need.

The only remaining item from these local plans is nearest-switch live
acceptance in routed-lab Phase 7.
