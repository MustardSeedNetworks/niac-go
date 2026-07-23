# NIAC master closeout ledger

**Status:** Active

**Updated:** 2026-07-23

**Baseline:** `main` at `e8c91940` after the #1071 merge

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
| Primary checkout | Stage 0 coordinator | Uses scoped branches for ledger and reproduction work |
| Detached `niac-release-review` | Preserved at v0.94.11 | Separate release context; do not remove without owner confirmation |

## Epic #1053 implementation queue

| Issue | Priority | Current state | Stage or owner |
| --- | --- | --- | --- |
| #1032 browser authentication | High | Needs reproduction | Stage 2 after HTTPS |
| #1033 live static-route validation | High | Needs reproduction | Stage 3 |
| #1034 routed Runtime Control | High | Needs reproduction | Stage 3 |
| #1035 nested path containment | High | Delivered in #1071 | Stage 2 evidence retained |
| #1036 Packet Inspector SSE envelope | High | Needs reproduction | Stage 4 |
| #1037 HTTPS-only listeners | High | Delivered in #1063 | Stage 2 evidence retained |
| #1041 bounded UDP proxy | High | Needs reproduction | Stage 2 |
| #1042 off-link notifications | High | Needs reproduction | Stage 3 after fault handoff |
| #1051 first-class browsers | High | Accepted | Stages 1 and 4 |
| #1038 duplicate segment tags | Medium | Needs reproduction | Stage 3 |
| #1039 stale API topology | Medium | Needs reproduction | Stage 3 |
| #1040 missing stats publisher | Medium | Needs reproduction | Stage 4 |
| #1043 partial server startup | Medium | Needs reproduction | Stage 4 |
| #1044 SSH idle expiry | Medium | Needs reproduction | Stage 4 |
| #1045 SNMP transport state | Medium | Needs reproduction | Stage 3 |
| #1046 unresolved ConfigPath | Medium | Needs reproduction | Stage 3 |
| #1047 TTL accounting | Medium | Needs reproduction | Stage 3 |
| #1048 offline PCAP access | Medium | Needs reproduction | Stage 4 |
| #1049 capture-exit recovery | Medium | Needs reproduction | Stage 4 |
| #1052 i18next migration | Medium | Accepted | Stage 4 |
| #1057 fail-closed hooks | Medium | Delivered in #1067 | Stage 0 prerequisite complete |
| #1068 content-package integrity | Medium | Accepted | Stage 1 before release |

## Defect-register reconciliation

These entries still require a current-v0.94.11 disposition. Do not create a
new issue until reproduction confirms that the contract remains open.

| Finding | Current evidence | Candidate disposition |
| --- | --- | --- |
| NIAC-DEF-001 | Fix landed; full/live gate pending | Absorb into Stage 5/6 acceptance |
| NIAC-DEF-002 | Fix landed; direct-mode gate pending | Absorb into Stage 5 |
| NIAC-DEF-008 | Partial next-hop fix | Reproduce with #1033 |
| NIAC-DEF-013 | Transactional replacement code-confirmed | Create unique high issue if current |
| NIAC-DEF-015 | Runtime Control code-confirmed | Absorb into #1034 |
| NIAC-DEF-017 | Lockfile fixed; full gate pending | Absorb into Stage 6 |
| NIAC-DEF-019 | IPv4 input validation code-confirmed | Create unique medium issue if current |
| NIAC-DEF-021 | Stack channel lifecycle code-confirmed | Create unique medium issue if current |
| NIAC-DEF-025 | DHCP validation code-confirmed | Create unique medium issue if current |
| NIAC-DEF-027 | Catalog sync integrity code-confirmed | Create unique medium issue if current |
| NIAC-DEF-028 | Restart recovery code-confirmed | Create unique medium issue if current |
| NIAC-DEF-029 | Preflight/interface parity code-confirmed | Reproduce with #1034 |
| NIAC-DEF-031 | Darwin link warning reproduced | Retain as low maintenance debt |
| NIAC-DEF-032 | UI chunk warning reproduced | Retain as low maintenance debt |
| NIAC-DEF-035 | Fix landed; live gate pending | Absorb into Stage 5 |
| NIAC-DEF-036 | Catalog rebuilt; live gate pending | Absorb into Stage 5 |
| NIAC-DEF-037 | Deprecated extractor reproduced | Absorb into #1052 |
| NIAC-DEF-038 | Foundation fingerprint race reproduced | Verify foundation status; create high issue if current |
| NIAC-DEF-046 | MIB-II work landed; live gate pending | Absorb into Stage 5/6 acceptance |
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
- [ ] Every #1053 `needs-info` issue has a current reproduction.
- [ ] Unique high/medium prose-only findings have accepted GitHub issues.
- [ ] Duplicate findings are marked absorbed.
