# Plan: NIAC master closeout

**Status:** Active

**Date:** 2026-07-23

## Outcome

Finish every current NIAC plan without treating every historical idea as a
commitment to build. Each plan item must end in exactly one state:

1. **Delivered** with merged tests and the required browser, lab, or release
   evidence.
2. **Absorbed** into one newer authoritative issue or plan, with the older
   record closed as superseded.
3. **Retired** with an explicit reason when the item is stale, duplicated,
   outside the product boundary, or fails the feature or market gate.

The high/medium remediation epic is the implementation queue. The routed-lab
plan and pre-1.0 roadmap define acceptance. All other plans are inputs that
must be reconciled, not parallel backlogs.

## Source-of-truth hierarchy

| Rank | Source | Role |
| --- | --- | --- |
| 1 | This master closeout plan | Program order, dependencies, and final ledger |
| 2 | Epic #1053 and its remediation plan/execution guide | High/medium implementation |
| 3 | Routed virtual-lab plan | Phase 7 hardware acceptance and Phase 8 boundary |
| 4 | Pre-1.0 roadmap | Release boundary |
| 5 | Defect register and local plans | Findings to absorb, verify, or retire |
| 6 | `docs/archive/**` | Historical evidence only; never an active backlog |

## Stage 0: Establish one accurate ledger

Track current ownership and disposition in the
[master closeout ledger](2026-07-niac-master-closeout-ledger.md).

1. Finish and verify the v0.94.10 release from the merged Free-tier contract.
2. Reconcile the three existing worktrees before overlapping edits:
   `fix/https-only-contract`, `fix/fault-telemetry`, and
   `fix/free-tier-contract`.
3. Map every open GitHub issue, defect-register entry, and local-plan item to
   one owner and one disposition.
4. Reproduce all #1053 defects still labeled `status: needs-info`; accept,
   correct, or close them with evidence.
5. Create GitHub issues for unique unresolved high/medium defect-register
   findings that are not already represented by #1053.
6. Mark duplicate findings as absorbed instead of fixing the same contract
   twice.

Exit gate: no high or medium finding exists only in prose, and no work item has
two implementation owners.

## Stage 1: Repair the evidence gate

Complete #1057 first:

- Enforce the repository-pinned Node and npm versions.
- Propagate every failed child command.
- Print success only after a zero exit.
- Prove the hook blocks a deliberately failing frontend command.

Close #1068 before the next release. Then retain browser traces for Chromium,
WebKit, Firefox, installed Chrome, installed Edge, and actual Safari.

Exit gate: later work can rely on local and CI results without masked failures.

## Stage 2: Security and external boundaries

Treat #1037 as delivered by PR #1063, then complete remediation Wave 1 in
this order:

1. #1035 nested configuration containment.
2. #1041 bounded UDP proxy resources and shutdown.
3. #1032 secure browser authentication for SPA, REST, and SSE.

Run adversarial path, plaintext-listener, overload, cancellation,
credential-leak, authorization, and browser tests before merging.

Exit gate: every network entry point is authenticated as required, TLS-only,
resource-bounded, and unable to escape managed roots.

## Stage 3: Routed and authoritative-state correctness

Complete remediation Wave 2 after the fault-telemetry worktree lands or has an
explicit handoff:

- #1033, #1034, #1038, #1039, #1042, #1045, #1046, and #1047.
- Reconcile unresolved NIAC-DEF-008, 013, 015, 019, 021, 025, 027, 028, and
  029; create issues for unique contracts and absorb true duplicates.
- Keep the stack-owned state authoritative for CLI, forwarding, SNMP,
  topology, discovery, counters, and notifications.

Run focused tests before lab work. Then verify routed discovery, SNMP tables,
off-link notifications, TTL accounting, and isolation with CyberScope and
Link-Live.

Exit gate: configuration, runtime, CLI, API, SNMP, and packet behavior project
one consistent state.

## Stage 4: Streaming, lifecycle, browsers, and i18n

Complete remediation Waves 3 and 4:

- #1036 and #1040: production SSE envelopes and statistics publication.
- #1048 and #1049: offline PCAP access and capture-exit recovery.
- #1043 and #1044: transactional server startup and SSH expiry.
- #1051: Chrome, Edge, and Safari first-class release gates; Firefox
  compatibility and Brave best-effort smoke evidence.
- #1052: fixture-first migration from `i18next-parser` to `i18next-cli`
  without catalog loss.

Exit gate: the critical browser journey passes, lifecycle cleanup is
deterministic, and translation validation is maintained and non-mutating.

## Stage 5: Finish routed-lab Phase 7

Treat already-recorded access-mode work as evidence, not work to repeat.
Complete and retain the missing acceptance:

1. Run the reference scenario through the dedicated direct cable.
2. Confirm DHCP, routing, SNMP, SSH, topology, and discovery match access mode.
3. Verify nearest-switch switch/port/VLAN placement; the NIAC synthesis appears
   implemented, so finish catalog generation and live proof rather than
   rebuilding it.
4. Capture CT `eth0` and the Proxmox bridge for 24 hours and prove isolation.
5. Run a fresh CyberScope discovery and compare Link-Live with NIAC authored
   truth.
6. Record discrepancies as defects and return them to the appropriate earlier
   stage.

Exit gate: every routed-plan completion criterion has attached evidence.

## Stage 6: Close the pre-1.0 roadmap

- Confirm injected faults through IF-MIB and EtherLike-MIB counters.
- Confirm the ten-device Free-tier limit across CLI, API, UI, import, template,
  and runtime entry paths.
- Remove stale compatibility, licensing, deployment, and roadmap claims.
- Pass formatting, lint, unit, race, integration, browser, security, package,
  install, deployment, and version validation from release-built artifacts.
- Run the final touched-area defect review and file every new high/medium
  finding before release.

Exit gate: all five roadmap checkboxes are complete and the release candidate
is reproducible.

## Stage 7: Close or replace every remaining plan

| Plan or issue | Required disposition |
| --- | --- |
| Defect-hunting plan/register | Close delivered findings; move unique debt to issues; archive the working register |
| Nearest-switch plan | Close after catalog and CyberScope evidence |
| Multi-VLAN epic #882 | Reconcile delivered architecture; issue only the actual remaining UI/catalog/acceptance work |
| Fleet-drift epic #1004 | Record delivered Renovate/foundation work; finish or separately track shared CI and doc alignment |
| Library issue #905 | Fix the broken/phone-home default; gate content curation as a separate product decision |
| License issue #911 | Obtain counsel decision; merge approved wording or close with the decision |
| UI architecture plan | Re-audit correctness items; absorb defects; retire low-value parity and speculative UI work |
| Modernization plan | Re-baseline landed `os.Root` work; approve only changes with measurable security, reliability, or maintenance value |

Exit gate: every non-archived plan has a final status and no local-only active
work remains.

## Stage 8: Post-stabilization expansion gates

Do not automatically implement Phase 8 or discretionary modernization merely
because it appears in an old plan. Run the feature and market gates separately
for:

1. OSPF, using the authoritative forwarding table first.
2. BGP only if a distinct accepted workflow remains after OSPF.
3. IPv6 forwarding and DHCPv6 relay.
4. Route policy and simulated ACLs.
5. Multiple external attachments with a separate threat model.
6. Replay expansion, React Compiler, TanStack Query, and remaining UI parity.

For each, record **build**, **reshape smaller**, or **retire**, then create a
bounded implementation plan only for approved work.

## Program controls

- One root-cause contract per PR where practical.
- Shared-state changes land before projections and UI.
- Security changes remain isolated from broad refactors.
- Stacked PRs land one at a time and rebase after each merge.
- Browser and lab evidence is attached to the issue that owns the contract.
- The master ledger is updated when work merges, not at the end of the program.
- A stage cannot close with an untracked high or medium defect.

## Definition of complete

- Epic #1053 is closed with every child delivered or invalidated by evidence.
- Routed Phase 7 and all pre-1.0 exit criteria pass.
- Every active plan and planning issue has a delivered, absorbed, or retired
  disposition.
- Optional work has an explicit owner decision rather than an indefinite
  backlog state.
- Main, local branches, remote branches, and worktrees contain no abandoned or
  ambiguous NIAC work.
