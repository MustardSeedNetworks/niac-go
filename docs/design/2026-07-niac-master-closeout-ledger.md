# NIAC master closeout ledger

**Status:** Active

**Updated:** 2026-07-24

**Baseline:** `main` at `f95a4d7b` after the #1113 merge

This ledger prevents duplicate ownership while the
[master closeout plan](2026-07-niac-master-closeout-plan.md) is executed.
Update it when work merges or a disposition changes.

## Release and publication

| Item | State | Next action |
| --- | --- | --- |
| v0.94.10 core artifacts | Verified | Release workflow, checksum, Cosign bundle, SLSA attestation, and macOS artifact passed |
| v0.94.10-v0.94.11 content packages | Integrity metadata missing | Tracked by accepted medium defect #1068 |
| v0.94.11 core artifacts | Verified | Workflow, checksum, Cosign bundle, SLSA attestation, and macOS artifact passed |
| v0.94.12 | Failed closed | Superseded without publication after the release-integrity gate rejected incomplete content metadata |
| v0.94.13 | Verified | Core and content assets, checksums, Cosign bundles, SLSA attestations, and macOS artifact passed |
| Master plan PR #1064 | Merged | Execute and maintain this ledger from `main` |
| Content-package integrity #1068 | Delivered by #1083 and #1084 | Content artifacts now participate in the release handoff and integrity contract |
| Release PR #1087 | Open and held | Accumulate remediation, acceptance, and final review before publishing v0.94.14 |

## Worktree ownership

| Worktree | State | Disposition |
| --- | --- | --- |
| Primary checkout | Stages 0-4 complete; Stages 5-7 active | Uses one scoped branch per root-cause contract |
| Detached `niac-release-review` | Preserved at v0.94.11 | Separate release context; do not remove without owner confirmation |

## Epic #1053 implementation queue

| Issue | Priority | Current state | Stage or owner |
| --- | --- | --- | --- |
| #1032 browser authentication | High | Delivered in #1090 | Stage 2 evidence retained |
| #1033 live static-route validation | High | Delivered in #1092 | Stage 3 evidence retained |
| #1034 routed Runtime Control | High | Delivered in #1098 | Stage 3 evidence retained |
| #1035 nested path containment | High | Delivered in #1071 | Stage 2 evidence retained |
| #1036 Packet Inspector SSE envelope | High | Delivered in #1108 | Stage 4 evidence retained |
| #1037 HTTPS-only listeners | High | Delivered in #1063 | Stage 2 evidence retained |
| #1041 bounded UDP proxy | High | Delivered in #1088 | Stage 2 evidence retained |
| #1042 off-link notifications | High | Delivered in #1101 | Stage 3 evidence retained |
| #1051 first-class browsers | High | CI contract delivered in #1109; installed-browser evidence pending | Stage 5 acceptance |
| #1075 transactional simulation replacement | High | Delivered in #1100 | Stage 3 evidence retained |
| [foundation#2](https://github.com/MustardSeedNetworks/foundation/issues/2) fingerprint stability | High | Delivered in foundation#3 and NIAC #1086 | Foundation v0.2.1 |
| #1038 duplicate segment tags | Medium | Delivered in #1093 | Stage 3 evidence retained |
| #1039 stale API topology | Medium | Delivered in #1102 | Stage 3 evidence retained |
| #1040 missing stats publisher | Medium | Delivered in #1111 | Stage 4 evidence retained |
| #1043 partial server startup | Medium | Closed; obsolete after single-listener migration | No implementation required |
| #1044 SSH idle expiry | Medium | Delivered in #1112 | Stage 4 evidence retained |
| #1045 SNMP transport state | Medium | Delivered in #1103 | Stage 3 evidence retained |
| #1046 unresolved ConfigPath | Medium | Delivered in #1095 | Stage 3 evidence retained |
| #1047 TTL accounting | Medium | Delivered in #1094 | Stage 3 evidence retained |
| #1048 offline PCAP access | Medium | Delivered in #1110 | Stage 4 evidence retained |
| #1049 capture-exit recovery | Medium | Delivered in #1113 | Stage 4 evidence retained |
| #1052 i18next migration | Medium | Delivered in #1089 | Stage 4 evidence retained |
| #1057 fail-closed hooks | Medium | Delivered in #1067 | Stage 0 prerequisite complete |
| #1068 content-package integrity | Medium | Delivered in #1083 and #1084 | Verified by v0.94.13 |
| #1076 IPv4 checksum/source validation | Medium | Delivered in #1104 | Stage 3 evidence retained |
| #1077 stack lifecycle contract | Medium | Delivered in #1105 | Stage 3 evidence retained |
| #1078 DHCP compiler validation | Medium | Delivered in #1097 | Stage 3 evidence retained |
| #1079 immutable catalog sync | Medium | Delivered in #1107 | Stage 3 evidence retained |
| #1080 active-simulation recovery | Medium | Delivered in #1106 | Stage 3 evidence retained |
| #1081 preflight interface parity | Medium | Delivered in #1096 | Stage 3 evidence retained |

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
| NIAC-DEF-037 | Deprecated extractor replaced | Delivered in #1089 |
| NIAC-DEF-038 | Foundation fingerprint race fixed | Delivered in foundation#3, v0.2.1, and NIAC #1086 |
| NIAC-DEF-046 | MIB-II work landed | Absorbed into Stage 5/6 CyberScope acceptance |
| NIAC-DEF-050 | Markdown backlog reproduced | Retain as low maintenance debt |

## Remaining plan registry

| Plan or issue | Current disposition |
| --- | --- |
| Pre-1.0 roadmap | Entitlement implementation complete in #1114 when merged; live and release acceptance remain |
| Routed virtual-lab plan | Phases 0-6 delivered; Phase 7 acceptance active |
| Remediation plan/execution guide | Active implementation queue under #1053 |
| Defect-hunting plan/register | Archived; high/medium findings reconciled into this ledger and #1053 |
| [Local-plan intake](2026-07-local-plan-intake.md) | Maintenance and positioning reconciled; nearest-switch live acceptance remains |
| Multi-VLAN acceptance #882 | Implementation delivered; tagged live evidence remains |
| Fleet-drift epic #1004 | Closed; shared Renovate, foundation, and security gates delivered; product release pipelines remain intentionally local |

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
