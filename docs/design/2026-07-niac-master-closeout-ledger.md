# NIAC master closeout ledger

**Status:** Active

**Updated:** 2026-07-23

**Baseline:** `main` at `ee44d48c` after the #1074 merge

This ledger prevents duplicate ownership while the
[master closeout plan](2026-07-niac-master-closeout-plan.md) is executed.
Update it when work merges or a disposition changes.

## Release and publication

| Item | State | Next action |
| --- | --- | --- |
| v0.94.10 core artifacts | Verified | Release workflow, checksum, Cosign bundle, SLSA attestation, and macOS artifact passed |
| v0.94.10-v0.94.11 content packages | Integrity metadata missing | Tracked by accepted medium defect #1068 |
| v0.94.11 core artifacts | Verified | Workflow, checksum, Cosign bundle, SLSA attestation, and macOS artifact passed |
| Master plan PR #1064 | Merged | Execute and maintain this ledger from `main` |
| Next release after v0.94.11 | Blocked by #1068 | Add content-package integrity metadata before publication |

## Worktree ownership

| Worktree | State | Disposition |
| --- | --- | --- |
| Primary checkout | Stage 0 complete; Stage 1 next | Uses scoped branches for release-integrity work |
| Detached `niac-release-review` | Preserved at v0.94.11 | Separate release context; do not remove without owner confirmation |

## Epic #1053 implementation queue

| Issue | Priority | Current state | Stage or owner |
| --- | --- | --- | --- |
| #1032 browser authentication | High | Reproduced and accepted | Stage 2 after #1041 |
| #1033 live static-route validation | High | Reproduced and accepted | Stage 3 |
| #1034 routed Runtime Control | High | Reproduced and accepted | Stage 3 |
| #1035 nested path containment | High | Delivered in #1071 | Stage 2 evidence retained |
| #1036 Packet Inspector SSE envelope | High | Reproduced and accepted | Stage 4 |
| #1037 HTTPS-only listeners | High | Delivered in #1063 | Stage 2 evidence retained |
| #1041 bounded UDP proxy | High | Reproduced and accepted | Stage 2 next |
| #1042 off-link notifications | High | Reproduced and accepted | Stage 3 after fault handoff |
| #1051 first-class browsers | High | Accepted | Stages 1 and 4 |
| #1075 transactional simulation replacement | High | Reproduced and accepted | Stage 3 |
| [foundation#2](https://github.com/MustardSeedNetworks/foundation/issues/2) fingerprint stability | High | Reproduced and tracked in foundation | Stage 2 shared dependency |
| #1038 duplicate segment tags | Medium | Reproduced and accepted | Stage 3 |
| #1039 stale API topology | Medium | Reproduced and accepted | Stage 3 |
| #1040 missing stats publisher | Medium | Reproduced and accepted | Stage 4 |
| #1043 partial server startup | Medium | Closed; obsolete after single-listener migration | No implementation required |
| #1044 SSH idle expiry | Medium | Reproduced and accepted | Stage 4 |
| #1045 SNMP transport state | Medium | Reproduced and accepted | Stage 3 |
| #1046 unresolved ConfigPath | Medium | Reproduced and accepted | Stage 3 |
| #1047 TTL accounting | Medium | Reproduced and accepted | Stage 3 |
| #1048 offline PCAP access | Medium | Reproduced and accepted | Stage 4 |
| #1049 capture-exit recovery | Medium | Reproduced and accepted | Stage 4 |
| #1052 i18next migration | Medium | Accepted | Stage 4 |
| #1057 fail-closed hooks | Medium | Delivered in #1067 | Stage 0 prerequisite complete |
| #1068 content-package integrity | Medium | Accepted | Stage 1 before release |
| #1076 IPv4 checksum/source validation | Medium | Reproduced and accepted | Stage 3 |
| #1077 stack lifecycle contract | Medium | Reproduced and accepted | Stage 3 |
| #1078 DHCP compiler validation | Medium | Reproduced and accepted | Stage 3 |
| #1079 immutable catalog sync | Medium | Reproduced and accepted | Stage 3 |
| #1080 active-simulation recovery | Medium | Reproduced and accepted | Stage 3 |
| #1081 preflight interface parity | Medium | Reproduced and accepted | Stage 3 |

## Defect-register reconciliation

Every high or medium entry now has a current disposition. Findings that share
one root cause and acceptance contract are absorbed into the linked issue.

| Finding | Current evidence | Disposition |
| --- | --- | --- |
| NIAC-DEF-001 | Root cause fixed | Absorbed into Stage 5/6 full and live acceptance |
| NIAC-DEF-002 | Root cause fixed | Absorbed into Stage 5 direct-mode hardware acceptance |
| NIAC-DEF-008 | Partial next-hop fix | Absorbed into accepted high #1033 |
| NIAC-DEF-013 | Transactional replacement reproduced | Accepted high #1075 |
| NIAC-DEF-015 | Runtime Control bypass reproduced | Absorbed into accepted high #1034 |
| NIAC-DEF-017 | Lockfile fixed | Absorbed into Stage 6 full security acceptance |
| NIAC-DEF-019 | Checksum/source validation reproduced; fragments fixed | Accepted medium #1076 |
| NIAC-DEF-021 | One-shot stack lifecycle reproduced | Accepted medium #1077 |
| NIAC-DEF-025 | Remaining DHCP validation gaps reproduced | Accepted medium #1078 |
| NIAC-DEF-027 | Mutable catalog sync reproduced | Accepted medium #1079 |
| NIAC-DEF-028 | Missing restart recovery reproduced | Accepted medium #1080 |
| NIAC-DEF-029 | Preflight/interface parity reproduced | Accepted medium #1081 |
| NIAC-DEF-031 | Darwin link warning reproduced | Retain as low maintenance debt |
| NIAC-DEF-032 | UI chunk warning reproduced | Retain as low maintenance debt |
| NIAC-DEF-035 | Root cause fixed | Absorbed into Stage 5 route-MIB live acceptance |
| NIAC-DEF-036 | Catalog rebuilt | Absorbed into Stage 5 routed live acceptance |
| NIAC-DEF-037 | Deprecated extractor reproduced | Absorbed into accepted medium #1052 |
| NIAC-DEF-038 | Foundation fingerprint race reproduced | Tracked as high in [foundation#2](https://github.com/MustardSeedNetworks/foundation/issues/2) |
| NIAC-DEF-046 | MIB-II work landed | Absorbed into Stage 5/6 CyberScope acceptance |
| NIAC-DEF-050 | Markdown backlog reproduced | Retain as low maintenance debt |

## Remaining plan registry

| Plan or issue | Current disposition |
| --- | --- |
| Pre-1.0 roadmap | Active acceptance boundary; Stage 6 |
| Routed virtual-lab plan | Phases 0-6 delivered; Phase 7 acceptance active |
| Remediation plan/execution guide | Active implementation queue under #1053 |
| Defect-hunting plan/register | Active evidence source; close after reconciliation |
| [Local-plan intake](2026-07-local-plan-intake.md) | Five local plans reconciled; catalog, stale UI references, replay accounting/positioning, cruft, and optional gates remain |
| Multi-VLAN epic #882 | Reconcile delivered architecture in Stage 7 |
| Fleet-drift epic #1004 | Reconcile Renovate/foundation delivery and remaining shared CI |

## Stage 0 exit checklist

- [x] One master plan and ledger exist.
- [x] Current worktrees have explicit owners.
- [x] Former follow-up commits are confirmed present in merged squash commits.
- [x] v0.94.10 core artifacts are verified; content integrity is tracked in #1068.
- [x] v0.94.11 publication is complete and its core artifacts are verified.
- [x] PR #1064 is merged.
- [x] PR #1063 is merged or explicitly handed off.
- [x] PR #1067 is merged before defect reproductions begin.
- [x] Every open #1053 `needs-info` issue has a current reproduction and accepted disposition.
- [x] Unique high/medium prose-only findings have accepted GitHub issues.
- [x] Duplicate findings are marked absorbed.
