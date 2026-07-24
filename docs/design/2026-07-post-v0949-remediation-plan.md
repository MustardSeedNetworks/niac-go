# Plan: v0.94.9 high/medium defect and browser remediation

**Status:** Implementation complete; live acceptance pending

**Date:** 2026-07-23

**Baseline:** v0.94.9 (`4057bcf9d36944c0df26105a2db84dd76d21f9ed`)

**Tracking epic:** [#1053](https://github.com/MustardSeedNetworks/niac-go/issues/1053)

## Outcome

Close every high- and medium-priority defect found by the repository review
and confirmed against v0.94.9, and make Chrome, Microsoft Edge, and Safari
first-class NIAC web clients. Replace deprecated translation extraction
tooling and restore a trustworthy local quality gate. Preserve the HTTPS-only,
authorization, release, and routed-isolation contracts while the work lands.

This is remediation, not a feature expansion. Each confirmed issue must be
fixed at its root cause, receive regression coverage, and pass the repository's
full quality and security gates. A finding invalidated by current evidence is
closed with that evidence instead of receiving unnecessary implementation.

## Browser support contract

| Tier | Browsers | Required evidence |
| --- | --- | --- |
| First-class | Chrome stable, Edge stable, Safari current | Critical journeys pass in the installed browser; failures block release |
| Engine CI | Playwright Chromium, WebKit, Firefox | Critical journeys pass on every PR that changes the UI or API contract |
| Compatibility | Firefox current | Engine suite passes; browser-specific defects are fixed when reproducible |
| Best-effort | Brave current | Pre-release smoke test with shields/default privacy behavior; not release-blocking |

WebKit in Playwright is necessary but is not accepted as proof of Safari
support. The release candidate must also run in actual Safari on the current
supported macOS version. Chrome and Edge evidence must use their installed
stable channels rather than treating bundled Chromium as a substitute.

Brave is useful because its privacy defaults can expose storage, cookie,
clipboard, download, and streaming assumptions. It does not add another
browser engine, so it remains a focused smoke test. Firefox stays in the
automated matrix because it provides independent engine coverage.

## Scope

### High priority

| Issue | Contract to restore | Completion evidence |
| --- | --- | --- |
| [#1032](https://github.com/MustardSeedNetworks/niac-go/issues/1032) | The web UI can authenticate to a token-protected daemon without exposing credentials | Authenticated browser journeys and authorization-denial tests |
| [#1033](https://github.com/MustardSeedNetworks/niac-go/issues/1033) | Device CLI rejects static routes whose next hop is unusable | Table-driven CLI and forwarding tests |
| [#1034](https://github.com/MustardSeedNetworks/niac-go/issues/1034) | Runtime Control requires attachment selection, direct/access mode, VLAN policy, and operator-approved preflight before starting a routed template | UI/API full-stack routed preflight and start test |
| [#1035](https://github.com/MustardSeedNetworks/niac-go/issues/1035) | Nested segment paths cannot escape managed configuration roots | Adversarial path tests on every load path |
| [#1036](https://github.com/MustardSeedNetworks/niac-go/issues/1036) | Packet Inspector consumes the documented SSE envelope | Producer/consumer contract test plus browser journey |
| [#1037](https://github.com/MustardSeedNetworks/niac-go/issues/1037) | Every API and metrics listener honors the HTTPS-only contract | Listener registration and TLS integration tests |
| [#1041](https://github.com/MustardSeedNetworks/niac-go/issues/1041) | UDP `map_to_ip` fan-out is bounded and lifecycle-owned | Load, cancellation, and leak tests |
| [#1042](https://github.com/MustardSeedNetworks/niac-go/issues/1042) | Off-link notifications traverse the simulated routed path | Routed SYSLOG/SNMP integration test and lab capture |
| [#1051](https://github.com/MustardSeedNetworks/niac-go/issues/1051) | Chrome, Edge, and Safari are enforced first-class clients | Automated matrix and actual-browser evidence |

### Medium priority

| Issue | Contract to restore | Completion evidence |
| --- | --- | --- |
| [#1038](https://github.com/MustardSeedNetworks/niac-go/issues/1038) | Duplicate segment tags are rejected, never silently overwritten | Parser and compiler validation tests |
| [#1039](https://github.com/MustardSeedNetworks/niac-go/issues/1039) | API topology reflects CLI mutation and simulation stop | State-to-topology lifecycle tests |
| [#1040](https://github.com/MustardSeedNetworks/niac-go/issues/1040) | Stats SSE publishes current authoritative counters | Hub publication and browser rendering tests |
| [#1044](https://github.com/MustardSeedNetworks/niac-go/issues/1044) | SSH idle timeout applies to established sessions | Deterministic connection timeout tests |
| [#1045](https://github.com/MustardSeedNetworks/niac-go/issues/1045) | SNMP transport tables project authoritative runtime state | State/SNMP parity tests and CyberScope walk |
| [#1046](https://github.com/MustardSeedNetworks/niac-go/issues/1046) | Programmatic segment paths fail explicitly when unresolved | Loader error propagation tests |
| [#1047](https://github.com/MustardSeedNetworks/niac-go/issues/1047) | TTL-expired packets are not counted or traced as forwarded | Forwarding counter and trace tests |
| [#1048](https://github.com/MustardSeedNetworks/niac-go/issues/1048) | Offline PCAP analysis does not require live capture | Browser journey with no active capture |
| [#1049](https://github.com/MustardSeedNetworks/niac-go/issues/1049) | Unexpected capture exit clears daemon running state | Process-exit lifecycle tests |
| [#1052](https://github.com/MustardSeedNetworks/niac-go/issues/1052) | Translation extraction uses the maintained `i18next-cli` on the pinned Node/npm toolchain | Catalog fixtures, migration parity, and a non-mutating CI gate |
| [#1057](https://github.com/MustardSeedNetworks/niac-go/issues/1057) | Local hooks cannot report success after a failed frontend command or under the wrong Node/npm toolchain | Hook regression tests and exact toolchain validation |

Issue #1043 is closed without implementation because the single-listener HTTPS
architecture removed the reported partial-start failure mode.

## Cross-cutting release work

Three required workstreams are not product defects, but the epic is incomplete
without them:

1. [#1051](https://github.com/MustardSeedNetworks/niac-go/issues/1051) changes
   the support contract and requires actual Chrome, Edge, and Safari evidence;
   engine-only CI is insufficient.
2. [#1052](https://github.com/MustardSeedNetworks/niac-go/issues/1052) removes
   deprecated extraction tooling and its JavaScript configuration without
   changing valid catalogs.
3. [#1057](https://github.com/MustardSeedNetworks/niac-go/issues/1057) repairs
   the local evidence gate before later fixes rely on it.

The final release also requires coordination with the active HTTPS and
fault-telemetry worktrees, documentation that matches the enforced browser
matrix, retained browser/lab artifacts, and a fresh defect review of every
touched area.

## Delivery order

Detailed sequencing, worktree boundaries, lab acceptance, i18n migration
steps, and PR strategy are in the
[remediation execution guide](2026-07-post-v0949-remediation-execution.md).

| Wave | Scope |
| --- | --- |
| 0 | Repair #1057, reproduce each defect, and baseline every browser |
| 1 | Security boundaries, HTTPS-only enforcement, UDP bounds, and UI authentication |
| 2 | Routed behavior, authoritative state, SNMP, topology, and notifications |
| 3 | SSE, packet capture, offline PCAP, and first-class browser journeys |
| 4 | SSH lifecycle and the `i18next-cli` migration |
| 5 | Integrated review, lab/browser acceptance, and release candidate |

## Estimates

These are active engineering-time estimates. They exclude time waiting for
GitHub runners, hardware availability, and an operator to move cables or
approve browser prompts.

| Work | Active engineering time | External elapsed time |
| --- | ---: | ---: |
| Wave 0 quality gate, reproduction, and browser baseline | 4-7 hours | 1-2 hours of CI/browser runs |
| Wave 1 security and authentication | 6-9 hours | 1-2 hours of CI |
| Wave 2 routed/state correctness | 10-16 hours | 3-6 hours of Link-Live/CyberScope lab time |
| Wave 3 streaming/capture/browser support | 10-15 hours | 3-6 hours of browser and CI runs |
| Wave 4 lifecycle and i18next-cli migration | 5-9 hours | 1-2 hours of CI |
| Wave 5 integrated review | 4-6 hours | 3-5 hours of CI and lab runs |
| **Total** | **39-62 hours** | **12-23 hours**, partly parallel |

With stable lab and browser access, plan on roughly five to eight working days
of elapsed engineering time. Security findings may increase that range if their
reproductions expose a broader shared-state or transport correction.

## Definition of complete

- All issues linked from [#1053](https://github.com/MustardSeedNetworks/niac-go/issues/1053)
  are closed with merged regression-tested fixes or an evidence-backed
  disposition showing that the reported failure mode no longer exists.
- Chrome, Edge, and Safari evidence is release-blocking and retained.
- Firefox compatibility and Brave smoke evidence are recorded.
- Link-Live/CyberScope routed, SNMP, and notification acceptance passes.
- `i18next-parser` and its JavaScript configuration are removed; the
  `i18next-cli` gate is non-mutating and release-blocking.
- Local hooks enforce the repository-pinned Node/npm versions and cannot mask a
  failed command.
- Security, lint, format, unit, E2E, build, and vulnerability gates are green
  with zero warnings.
- The final defect review has no untracked high or medium findings.
