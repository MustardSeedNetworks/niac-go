# Execution: v0.94.9 remediation waves

This execution guide implements the
[v0.94.9 remediation plan](2026-07-post-v0949-remediation-plan.md).
The waves are ordered by security boundary and shared-state dependency. PRs
within a wave may run in parallel only when they do not touch the same state
owner or test surface.

## Wave 0: Reproduce and lock the contracts

1. Fix [#1057](https://github.com/MustardSeedNetworks/niac-go/issues/1057)
   first. The hook must select or reject anything other than Node 26.5.0 and
   npm 12.0.1, propagate every child-process exit status, and print success
   only after the command exits zero.
2. Move each accepted issue from `status: needs-info` to `status: accepted`
   after confirming its reproduction on v0.94.9.
3. Add a failing focused test before changing behavior.
4. Record the authoritative state owner for each fix. Do not introduce a
   second device, topology, capture, or counter store.
5. Add browser project names and trace retention without making the new matrix
   release-blocking until the baseline is measured.

Exit criteria:

- Every issue has a confirmed reproduction or is corrected/closed with
  evidence that the original finding is invalid.
- No implementation issue remains `needs-info`.
- Local hooks fail closed under an incorrect toolchain or failed test command.
- Baseline results exist for Chromium, WebKit, Firefox, Chrome, Edge, and
  Safari.

## Wave 1: Security and external entry points

Fix [#1035](https://github.com/MustardSeedNetworks/niac-go/issues/1035),
[#1037](https://github.com/MustardSeedNetworks/niac-go/issues/1037),
[#1041](https://github.com/MustardSeedNetworks/niac-go/issues/1041), and
[#1032](https://github.com/MustardSeedNetworks/niac-go/issues/1032).

Before implementing #1037, review and either land or explicitly hand off the
active `fix/https-only-contract` worktree. It already changes listener
registration, daemon startup, packaging, smoke tests, CI, and public
documentation. Reimplementing #1037 independently would create conflicting
transport contracts and discard validated work.

Before implementing #1032, land or explicitly hand off
`fix/free-tier-contract` and verify that browser authentication does not alter
the Free-tier device contract.

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

## Wave 2: Routed and authoritative-state correctness

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
or overwrite its fault and telemetry work.

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

## Wave 3: Streaming, packet capture, and browsers

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

## Wave 4: Lifecycle and maintenance cleanup

Fix [#1044](https://github.com/MustardSeedNetworks/niac-go/issues/1044) and
[#1052](https://github.com/MustardSeedNetworks/niac-go/issues/1052).

#1043 requires no implementation because the single-listener HTTPS
architecture removed the reported partial-start failure mode.

The i18n migration is an intentional contract migration:

1. Capture catalog fixtures for plurals, contexts, namespaces, interpolation,
   aliases, and every dynamic-key prefix.
2. Run `i18next-cli migrate-config` only as a draft conversion. Review every
   option before it can read or write the real catalogs.
3. Replace `ui/i18next-parser.config.js` with typed
   `ui/i18next.config.ts`, pin `i18next-cli` exactly, and remove
   `i18next-parser` plus its deprecated transitive tree.
4. Configure extraction paths, primary language, namespace/key separators,
   plural/context behavior, and preserved dynamic patterns explicitly.
   Disable unused-key deletion until parity is proven.
5. Compare the old and new extracted English catalogs byte-for-byte or through
   a reviewed semantic diff. Valid English and Spanish translations must not
   be deleted or reset.
6. Make `npm run i18n:check` non-mutating and release-blocking in CI. Keep
   glossary, banned-vocabulary, locale parity, and interpolation validation.
7. Remove custom extraction checks only where `i18next-cli` proves equivalent
   coverage; retain repository-specific validation that the replacement does
   not provide.

Type generation is not required for this migration. Add it only if it replaces
existing type machinery with less code and no catalog or runtime behavior
change.

## Wave 5: Integrated review and release candidate

1. Run `make fmt-check`, `make lint`, `make test`, `make security`,
   `make test-e2e`, and `make build`.
2. Verify a clean `npm ci` under the exact Node/npm pins and run the
   non-mutating i18n gate.
3. Run the complete browser evidence matrix.
4. Repeat the routed CyberScope/Link-Live acceptance suite.
5. Perform an adversarial review of the security and lifecycle changes.
6. Re-run the repository defect review in every touched area.
7. File any new high or medium finding before the release candidate can ship.

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
