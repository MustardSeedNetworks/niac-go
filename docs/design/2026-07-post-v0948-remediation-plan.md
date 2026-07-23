# Plan: Post-v0.94.8 defect and browser remediation

**Status:** Ready for implementation

**Date:** 2026-07-23

**Baseline:** v0.94.8

**Tracking epic:** [#1053](https://github.com/MustardSeedNetworks/niac-go/issues/1053)

## Outcome

Close every high- and medium-priority defect found by the v0.94.8 repository
review and make Chrome, Microsoft Edge, and Safari first-class NIAC web
clients. Preserve the HTTPS-only, authorization, release, and routed-isolation
contracts while the work lands.

This is remediation, not a feature expansion. Each issue must be fixed at its
root cause, receive regression coverage, and pass the repository's full quality
and security gates.

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
| [#1043](https://github.com/MustardSeedNetworks/niac-go/issues/1043) | Partial server startup rolls back every listener | Failure-injection lifecycle tests |
| [#1044](https://github.com/MustardSeedNetworks/niac-go/issues/1044) | SSH idle timeout applies to established sessions | Deterministic connection timeout tests |
| [#1045](https://github.com/MustardSeedNetworks/niac-go/issues/1045) | SNMP transport tables project authoritative runtime state | State/SNMP parity tests and CyberScope walk |
| [#1046](https://github.com/MustardSeedNetworks/niac-go/issues/1046) | Programmatic segment paths fail explicitly when unresolved | Loader error propagation tests |
| [#1047](https://github.com/MustardSeedNetworks/niac-go/issues/1047) | TTL-expired packets are not counted or traced as forwarded | Forwarding counter and trace tests |
| [#1048](https://github.com/MustardSeedNetworks/niac-go/issues/1048) | Offline PCAP analysis does not require live capture | Browser journey with no active capture |
| [#1049](https://github.com/MustardSeedNetworks/niac-go/issues/1049) | Unexpected capture exit clears daemon running state | Process-exit lifecycle tests |
| [#1052](https://github.com/MustardSeedNetworks/niac-go/issues/1052) | Translation extraction uses maintained Node 26 tooling | Catalog fixtures and a non-mutating CI gate |

## Delivery order

The waves are ordered by security boundary and shared-state dependency. PRs
within a wave may run in parallel only when they do not touch the same state
owner or test surface.

### Wave 0: Reproduce and lock the contracts

1. Move each accepted issue from `status: needs-info` to `status: accepted`
   after confirming its reproduction on v0.94.8.
2. Add a failing focused test before changing behavior.
3. Record the authoritative state owner for each fix. Do not introduce a
   second device, topology, capture, or counter store.
4. Add browser project names and trace retention without making the new matrix
   release-blocking until the baseline is measured.

Exit criteria:

- Every issue has a confirmed reproduction or is corrected/closed with
  evidence that the original finding is invalid.
- No implementation issue remains `needs-info`.
- Baseline results exist for Chromium, WebKit, Firefox, Chrome, Edge, and
  Safari.

### Wave 1: Security and external entry points

Fix [#1035](https://github.com/MustardSeedNetworks/niac-go/issues/1035),
[#1037](https://github.com/MustardSeedNetworks/niac-go/issues/1037),
[#1041](https://github.com/MustardSeedNetworks/niac-go/issues/1041), and
[#1032](https://github.com/MustardSeedNetworks/niac-go/issues/1032).

Required order:

1. Contain configuration paths and listener transport.
2. Bound UDP resources and prove shutdown.
3. Add the UI authentication flow against the corrected HTTPS boundary.
4. Run the authenticated critical journey in Chrome, Edge, and Safari.

Exit criteria:

- No plaintext API/metrics listener is reachable.
- Authentication works without writing the bearer token to logs, URLs, or
  persistent browser storage.
- Path escape and UDP exhaustion tests pass under the race detector where
  applicable.

### Wave 2: Routed and authoritative-state correctness

Fix [#1033](https://github.com/MustardSeedNetworks/niac-go/issues/1033),
[#1034](https://github.com/MustardSeedNetworks/niac-go/issues/1034),
[#1042](https://github.com/MustardSeedNetworks/niac-go/issues/1042),
[#1038](https://github.com/MustardSeedNetworks/niac-go/issues/1038),
[#1039](https://github.com/MustardSeedNetworks/niac-go/issues/1039),
[#1045](https://github.com/MustardSeedNetworks/niac-go/issues/1045),
[#1046](https://github.com/MustardSeedNetworks/niac-go/issues/1046), and
[#1047](https://github.com/MustardSeedNetworks/niac-go/issues/1047).

Before Wave 2 starts, the active `fix/fault-telemetry` worktree must either
land or reach an explicit handoff. Rebase Wave 2 from that result before
touching device state, `stack.go`, or SNMP state projection. Do not duplicate
or overwrite its uncommitted fault and telemetry work.

The stack-owned device and fabric state remains authoritative for CLI,
forwarding, topology, SNMP, discovery, and notifications. Fix projections and
event wiring rather than copying state into subsystem-local caches.

Lab acceptance:

- Run the eight-subnet VLAN 200 scenario with CyberScope.
- Verify routed discovery and SNMP transport tables with Link-Live evidence.
- Verify off-link SYSLOG/SNMP notification delivery.
- Capture TTL expiry and confirm it is neither counted nor displayed as
  forwarded.
- Confirm zero internal VLAN leakage to non-allowlisted attachments.

### Wave 3: Streaming, packet capture, and browsers

Fix [#1036](https://github.com/MustardSeedNetworks/niac-go/issues/1036),
[#1040](https://github.com/MustardSeedNetworks/niac-go/issues/1040),
[#1048](https://github.com/MustardSeedNetworks/niac-go/issues/1048), and
[#1049](https://github.com/MustardSeedNetworks/niac-go/issues/1049), then
complete [#1051](https://github.com/MustardSeedNetworks/niac-go/issues/1051).

Critical browser journey:

1. Complete first-run/authentication against HTTPS.
2. Load, edit, validate, start, stop, and restart a routed template.
3. Observe topology mutation and live statistics.
4. Inspect packets across SSE reconnects.
5. Stop live capture and analyze an offline PCAP.
6. Download/export data and exercise clipboard/dialog interactions.
7. Repeat at desktop and narrow responsive widths.

CI policy:

- Chromium, WebKit, and Firefox run the critical suite on relevant PRs.
- Installed Chrome and Edge stable run on a scheduled workflow and every
  release candidate.
- Actual Safari runs on macOS for every release candidate.
- Brave runs the critical smoke subset before release; failures create issues
  but do not block unless the same failure reproduces in Chrome or Edge.

### Wave 4: Lifecycle and maintenance cleanup

Fix [#1043](https://github.com/MustardSeedNetworks/niac-go/issues/1043),
[#1044](https://github.com/MustardSeedNetworks/niac-go/issues/1044), and
[#1052](https://github.com/MustardSeedNetworks/niac-go/issues/1052).

The i18n migration starts with catalog fixtures for plural, context, namespace,
and dynamic keys. A tool replacement that rewrites valid catalogs is rejected.
The final `i18n:check` command must be non-mutating and release-blocking.

### Wave 5: Integrated review and release candidate

1. Run `make fmt-check`, `make lint`, `make test`, `make security`,
   `make test-e2e`, and `make build`.
2. Run the complete browser evidence matrix.
3. Repeat the routed CyberScope/Link-Live acceptance suite.
4. Perform an adversarial review of the security and lifecycle changes.
5. Re-run the repository defect review in every touched area.
6. File any new high or medium finding before the release candidate can ship.

## Pull request strategy

- Use one root-cause issue per PR when practical.
- Bundle producer/consumer changes only when separating them would leave a
  broken intermediate contract.
- Keep security fixes isolated from broad UI refactors.
- Land shared-state changes before their API, SNMP, or UI projections.
- Rebase each dependent branch after the base PR merges; do not enable
  cascading auto-merge on a stack.
- Every PR links its issue, includes test output, and identifies browser or lab
  evidence still required.

## Estimates

These are active engineering-time estimates. They exclude time waiting for
GitHub runners, hardware availability, and an operator to move cables or
approve browser prompts.

| Work | Active Codex time | External elapsed time |
| --- | ---: | ---: |
| Wave 0 reproduction and browser baseline | 3-5 hours | 1-2 hours of CI/browser runs |
| Wave 1 security and authentication | 6-9 hours | 1-2 hours of CI |
| Wave 2 routed/state correctness | 10-16 hours | 3-6 hours of Link-Live/CyberScope lab time |
| Wave 3 streaming/capture/browser support | 10-15 hours | 3-6 hours of browser and CI runs |
| Wave 4 lifecycle/i18n | 5-8 hours | 1-2 hours of CI |
| Wave 5 integrated review | 4-6 hours | 3-5 hours of CI and lab runs |
| **Total** | **38-59 hours** | **12-23 hours**, partly parallel |

With stable lab and browser access, plan on roughly five to eight working days
of elapsed agent time. Security findings may increase that range if their
reproductions expose a broader shared-state or transport correction.

## Definition of complete

- All issues linked from [#1053](https://github.com/MustardSeedNetworks/niac-go/issues/1053)
  are closed with merged regression-tested fixes.
- Chrome, Edge, and Safari evidence is release-blocking and retained.
- Firefox compatibility and Brave smoke evidence are recorded.
- Link-Live/CyberScope routed, SNMP, and notification acceptance passes.
- Security, lint, format, unit, E2E, build, and vulnerability gates are green
  with zero warnings.
- The final defect review has no untracked high or medium findings.
