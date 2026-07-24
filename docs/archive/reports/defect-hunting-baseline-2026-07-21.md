# NIAC Defect-Hunting Baseline — 2026-07-21

> Historical baseline captured for the archived defect-hunting plan. It is not
> a current repository-status report.

## Local NIAC worktree

- Path: `/Users/krisarmstrong/Developer/MustardSeedNetworks/niac-routed-plan`
- Branch: `docs/routed-virtual-lab-plan`
- Commit: `2bdcb91acbf1993a599a5096871a615da4ceaa48`
- Binary SHA-256: `5a59c1378de0f469b1514350b5832cc53b5beea4f10bfff5fcc3cbaa93e5b3e3`

Starting dirty files:

```text
 M internal/protocols/fabric_forwarding_test.go
 M internal/protocols/snmp.go
 M internal/protocols/snmp/agent.go
 M internal/protocols/snmp/constants.go
 M internal/protocols/snmp/mib_ip.go
 M internal/protocols/snmp/peer_topology.go
?? docs/design/defect-hunting-plan.md
?? internal/protocols/snmp/mib_ip_routes_test.go
```

Toolchain:

```text
go version go1.26.5 darwin/arm64
node v26.4.0
npm 11.18.0
golangci-lint 2.12.2
govulncheck 1.3.0
gitleaks 8.30.1
```

## Demo catalog worktree

- Path: `/Users/krisarmstrong/Developer/MustardSeedNetworks/niac-demo-catalog`
- Branch: `feat/segments-demo-scenario`
- Commit: `3609e1a5c740d8d947b0258029d545481e75b599`
- Generated scenario SHA-256:
  `5c35bf9685d72a1e87cb2f1f03eee135c806c4620045a06dfd35dee36d226520`

Starting dirty files:

```text
 M catalog.yaml
 M docs/catalog-scenarios-summary.md
 M manifests/catalog-scenarios.tsv
 M scenarios/README.md
 M scenarios/gen_multisite.py
 M scenarios/go-yaml/multisite/demo-multisite.yaml
?? scenarios/gen_linklive.py
?? scenarios/go-yaml/linklive/
?? scenarios/tests/
```

## CT304 deployment

- Container state: running
- `niac.service`: active
- `niac-demo-lab.service`: active
- Active service binary: `/usr/bin/niac`
- Deployed binary SHA-256:
  `46d491cb5ad2ffdc879f9819462d511a783e4557a541e235c68bdfe329912d69`
- Runtime scenario SHA-256:
  `d157dd77fc0ecb001b38468339804aa27ed8abc737fe800d73886391f1c17b27`
- Service helper SHA-256:
  `2be330ee7baa27e398bebf019b32b50775e8edc93188c2aa78d16873d6b6d159`
- Active scenario: `/var/lib/niac/scenarios/demo-multisite-modern.yaml`
- Capture interface: `eth0`

The runtime scenario checksum intentionally differs from the catalog artifact:
deployment rewrites catalog-relative SNMP walk paths to their installed CT304
locations. A future deployment test should verify a normalized representation
rather than accepting arbitrary checksum drift.

CT304 links at baseline:

```text
lo               UNKNOWN
eth0@if32        UP
eth1@if33        UP
eth0.200@eth0    UP
```

`eth0.200` is stale from an earlier tagged-interface experiment. The active
helper binds to `eth0`; this audit does not remove the stale link until its
lifecycle and rollback behavior are reproduced.

The Proxmox bridge restricts the CT304 lab-facing veth to VLAN 200:

```text
port              vlan-id
veth304i0         200
```

Rollback files present at baseline:

```text
/usr/local/bin/niac.pre-route-mib
/usr/local/libexec/niac-demo-lab.py.pre-vlan-subif
/var/lib/niac/scenarios/demo-linklive-routed.yaml.pre-modern
```

`/usr/local/bin/niac` also existed with SHA-256
`5a59c1378de0f469b1514350b5832cc53b5beea4f10bfff5fcc3cbaa93e5b3e3`,
but systemd did not execute that file.

## Baseline commands

The following gates are recorded in this file after they complete. Failures
remain failures; the audit will not add suppressions.

```text
make fmt-check
make lint
make test
go test -race ./...
make test-e2e
make security
make build
make schema
git diff --exit-code -- docs/schemas/niac.schema.json
```

## Baseline results

| Gate | Result | Evidence |
|---|---|---|
| `make fmt-check` | Pass | Go and 323 frontend files clean |
| `make lint` | Pass | 81 Go linters report 0 issues; Biome checked 323 files |
| `make test` | Pass | 39 Go packages, 58.6% statement coverage; 63 Vitest files |
| `go test -race ./...` | Pass with build warnings | No races; Darwin linker repeatedly warns about duplicate `-lpcap` |
| `make test-e2e` | **Fail** | 56 passed, 2 skipped, 70 failed; local harness emits repeated backend proxy failures and WebKit is not runnable in the current setup |
| `make security` | **Fail** | `npm audit` reports two high-severity advisories; secret scan is skipped by the target after the failure |
| `make security-secrets` | Pass | 1,009 commits and approximately 4.03 GB scanned; no leaks found |
| `make build` | Pass with warnings | Full UI/backend build succeeds; Vite chunk-size and outside-root output warnings plus duplicate `-lpcap` linker warning |
| schema drift | Pass | Regeneration leaves `docs/schemas/niac.schema.json` unchanged |

The security failure is release-blocking:

```text
brace-expansion: GHSA-3jxr-9vmj-r5cp (high)
js-yaml: GHSA-52cp-r559-cp3m (high)
```

The passing build is not warning-clean under the repository definition of done.
The Vite outside-root notice reflects the intentional UI embed contract, but
the duplicate libpcap linkage and oversized chunks require separate root-cause
classification rather than being silently accepted.
