# NIAC local-plan intake

**Status:** Reconciled intake

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

### Remaining disposition

- Remove the stale `/pcap-analyzer` breadcrumb.
- Replace the retired botanical reference in `themeModuleColors.ts`.
- Retain Topology page extraction and spacing cleanup as low, opportunistic
  maintenance debt unless touched by accepted work.
- Feature-gate any still-missing MIB archive, template-apply, dump, or
  packet-analysis parity instead of assuming CLI symmetry is valuable.

Stage 7 owns these final dispositions.

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

### Remaining disposition

- Reverify malformed or truncated replay-packet accounting.
- Feature-gate endpoint rewrite, VLAN rewrite, expanded throughput reporting,
  and Inspector export/dissection work.

Stage 7 owns the reliability check. Stage 8 owns feature and market gates for
optional replay and Inspector expansion.

## Cruft and duplication plan

### Delivered or absorbed

- PR #1012 removed the dead MIB database package and other unused subsystems.
- PR #921 removed the recorded dead UI components and storage-key leak.
- The UI cleanup and parity work listed above absorbed the toast, confirmation,
  polling, i18n, shared-table, and orphan-route findings.
- Path containment findings are absorbed into #1035 and the delivered
  `os.Root` work; do not create a parallel security owner.

### Remaining reconciliation

- Reverify and remove dead scripts, config parsing exports, placeholders, and
  stale comments, including `apply_history_changes.py` and the generated-types
  placeholder.
- Decide whether each unowned helper script is a supported tool or dead
  migration residue, then wire or remove it.
- Reverify duplicated OID, safe-conversion, packet-serialization, MAC-format,
  and hex-format helpers; refactor only current duplication.
- Reverify API dispatch, error-envelope, JSON-writing, and path-field
  duplication without weakening the existing security wrappers.
- Decide whether generated Go-to-TypeScript device types warrant a separate
  epic; retire the empty placeholder if not.
- Fold file naming and package extraction into accepted internal API work
  instead of a rename-only sweep.
- Make the file-size gate honest after triaging current red flags, or remove
  it; do not preserve a non-blocking gate as false assurance.

Stage 7 owns this maintenance reconciliation. Any unique high or medium defect
must become an accepted issue before implementation.

## Replay positioning plan

### Delivered

- Replay streaming, topspeed/pps/Mbps modes, loop count, and replay filtering
  satisfy four of the five original credibility gates.
- The catalog, live responders, UI controls, and root-contained replay path are
  real product capabilities and may be described accurately.

### Remaining positioning and capability work

- Close the malformed/truncated replay accounting gate before making parity
  claims.
- Reconcile internal strategy and marketing documents so NIAC is described as
  a test target, not a test instrument, and replay is represented without
  displacing NIAC's simulator charter.
- Correct protocol-count and CLI-surface claims against the shipping product.
- Keep comparisons to tcpreplay bounded to delivered rate, loop, filtering,
  timing, catalog, and interactivity behavior.
- Feature-gate packet rewrite, VLAN rewrite, completion statistics, explicit
  pcapng/nanosecond coverage, Inspector export, protocol-specific filters and
  dissection, and conversation statistics.
- Retire dual-interface bridging, line-rate acceleration, true TCP reassembly,
  expert analysis, and broad Wireshark parity absent a concrete buyer need.

Stage 7 owns truthful documentation alignment. Stage 8 owns optional capability
and market decisions.
