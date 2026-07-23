# NIAC master closeout ledger

**Status:** Active

**Updated:** 2026-07-23

**Baseline:** `main` at `b2201f3d` after the v0.94.10 release PR merge

This ledger prevents duplicate ownership while the
[master closeout plan](2026-07-niac-master-closeout-plan.md) is executed.
Update it when work merges or a disposition changes.

## Release and publication

| Item | State | Next action |
| --- | --- | --- |
| v0.94.10 release PR #1061 | Merged; all PR checks passed | Verify tag, release workflow, assets, attestations, and signatures |
| Master plan PR #1064 | Draft; CI running | Complete review, mark ready, merge, and update `main` |
| Latest published release | v0.94.9 | Replace only after v0.94.10 publication is verified |

## Worktree ownership

| Worktree | State | Disposition |
| --- | --- | --- |
| Primary `docs/master-plan-closeout` | Clean; PR #1064 | Owns plan and ledger only |
| `fix/https-only-contract` | Clean; PR #1063 | Owns #1037; do not duplicate listener work |
| `fix/fault-telemetry` | Clean; base landed as #1059 | Preserve follow-ups `72d2aca5` and `55aae24f`; rebase into a focused follow-up PR |
| `fix/free-tier-contract` | Clean; base landed as #1062 | Preserve `e1dc2b2c`; rebase into a focused wording PR |

## Epic #1053 implementation queue

| Issue | Priority | Current state | Stage or owner |
| --- | --- | --- | --- |
| #1032 browser authentication | High | Needs reproduction | Stage 2 after HTTPS |
| #1033 live static-route validation | High | Needs reproduction | Stage 3 |
| #1034 routed Runtime Control | High | Needs reproduction | Stage 3 |
| #1035 nested path containment | High | Needs reproduction | Stage 2 first |
| #1036 Packet Inspector SSE envelope | High | Needs reproduction | Stage 4 |
| #1037 HTTPS-only listeners | High | PR #1063 open | HTTPS worktree |
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
| #1057 fail-closed hooks | Medium | Accepted | Stage 1 first |

## Defect-register reconciliation

These entries still require a current-v0.94.10 disposition. Do not create a
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
| Nearest-switch local plan | NIAC synthesis delivered; catalog/live evidence remains |
| UI architecture local plan | Re-audit after #1053; absorb or retire each item |
| Modernization local plan | Deferred to Stage 8 gates |
| Multi-VLAN epic #882 | Reconcile delivered architecture in Stage 7 |
| Library issue #905 | Fix unsafe default; gate curation separately |
| License issue #911 | External counsel decision required |
| Fleet-drift epic #1004 | Reconcile Renovate/foundation delivery and remaining shared CI |

## Stage 0 exit checklist

- [x] One master plan and ledger exist.
- [x] Current worktrees have explicit owners.
- [x] Hidden follow-up commits are identified and preserved.
- [ ] v0.94.10 release artifacts and provenance are verified.
- [ ] PR #1064 is merged.
- [ ] PR #1063 is merged or explicitly handed off.
- [ ] Every #1053 `needs-info` issue has a current reproduction.
- [ ] Unique high/medium prose-only findings have accepted GitHub issues.
- [ ] Duplicate findings are marked absorbed.
