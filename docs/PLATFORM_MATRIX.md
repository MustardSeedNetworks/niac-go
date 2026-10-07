# Platform matrix

Per-platform evidence for the v1 exit criterion "the platform matrix is
recorded with per-cell command output" (plan row P5-4).

Every cell below was taken against a **release artifact downloaded from the
GitHub release and checked against the signed checksum manifest** — not a local
build. A cell with no recorded output is not a cell that passed; it says
`not taken` and why.

**Release under test:** `v0.95.53`, commit `45bd8e14`.

## Summary

| Platform | Artifact | Runs | Serves HTTPS | Embedded UI | Capture |
| --- | --- | --- | --- | --- | --- |
| macOS 26 (Apple Silicon) | `darwin-arm64.tar.gz` | yes | yes | yes | not taken |
| Ubuntu 24.04 (x86_64) | `linux-amd64.tar.gz`, `.deb` | yes | yes | yes | yes |
| Fedora 44 (`v0.112.1`) | `x86_64.rpm` | yes | yes | yes | simulation on a dummy link |
| Windows 11 (x86_64) | `windows-amd64.zip` | yes | yes | yes | not taken — no Npcap |
| Docker (`v0.111.0`) | `deploy/docker` image | yes | yes | yes | simulation on a dummy link |

"Embedded UI" is a non-empty `uiBuildHash` in `/__version`, which is what
proves the artifact came through the make pipeline rather than a bare
`go build`.

## macOS 26, Apple Silicon

```text
$ sha256sum niac-0.95.53-darwin-arm64.tar.gz
0d62781c9a876785ebfa526447dd08cc440cd4104326c2edcfa4b0c0873701e5
$ grep darwin-arm64.tar.gz checksums.txt
0d62781c9a876785ebfa526447dd08cc440cd4104326c2edcfa4b0c0873701e5  niac-0.95.53-darwin-arm64.tar.gz

$ ./niac version
niac 0.95.53 (commit: 45bd8e140896d50db31fa85b206f9b9bd5d382f4, built: 2026-09-11T16:54:53Z)

$ curl -sk https://127.0.0.1:18448/__version
{
    "buildTime": "2026-09-11T16:54:53Z",
    "commit": "45bd8e1",
    "goVersion": "go1.27.0",
    "platform": "darwin/arm64",
    "uiBuildHash": "c5f0f6cb7ab77b7366b26d01c3f27997",
    "version": "0.95.53"
}
```

No `.pkg` is produced by the release pipeline, so macOS is an archive install
today. Capture was not exercised here.

## Ubuntu 24.04, x86_64 (`dev-srv-ubuntu`)

```text
$ sha256sum niac-0.95.53-linux-amd64.tar.gz
500a9c426ab4c863b80149387922c84ea4f4dc1eeca226bb77d8b3734d65c7a2
$ grep linux-amd64.tar.gz checksums.txt
500a9c426ab4c863b80149387922c84ea4f4dc1eeca226bb77d8b3734d65c7a2  niac-0.95.53-linux-amd64.tar.gz

$ ./niac version
niac 0.95.53 (commit: 45bd8e140896d50db31fa85b206f9b9bd5d382f4, built: 2026-09-11T16:54:53Z)
```

The acceptance harness drove this artifact, which covers HTTPS, the embedded
UI, the bearer and CSRF middleware, and capture through a live scenario:

```text
$ NIAC_ACCEPTANCE_BINARY=/tmp/relcheck/niac go test -tags acceptance ./internal/acceptance/harness/...
--- PASS: TestReleasedBinaryReportsItsBuild (0.60s)
--- PASS: TestReleasedBinaryRefusesAnUnauthenticatedMutation (1.97s)
--- PASS: TestReleasedBinaryRefusesACheckpointOnAnAbsentSession (1.36s)
--- PASS: TestReleasedBinaryPreflightsWithoutStarting (4.57s)
ok  	internal/acceptance/harness	8.502s

$ sudo NIAC_ACCEPTANCE_BINARY=/tmp/relcheck/niac \
    go test -tags integration ./internal/wiretest/ -run TestReleasedBinaryCheckpointsMutatesAndResets
--- PASS: TestReleasedBinaryCheckpointsMutatesAndResets (1.46s)
```

### Package install and upgrade

`make deploy-validate` installed the `.deb` over an existing **0.95.38**, so
this is a real cross-version upgrade rather than the same-version no-op that
cannot prove an upgrade path:

```text
$ make deploy-validate HOST=dev-srv-ubuntu RELEASE=v0.95.53
{
  "commitFull": "45bd8e140896d50db31fa85b206f9b9bd5d382f4",
  "platform": "linux/amd64",
  "uiBuildHash": "c5f0f6cb7ab77b7366b26d01c3f27997",
  "version": "0.95.53"
}
PASS /__version still correct after installing over an existing configuration

==> Watching for a restart loop for 30s
PASS niac.service active, NRestarts unchanged at 0
```

That covers the "install must not crash-loop an existing config" clause
directly — the failure seed#377 taught us to check for.

## Fedora 44, x86_64 (`dev-srv-fedora`)

`v0.95.53` could not start here: the binary carried Debian's libpcap SONAME
(#1999), fixed at `cbaad8d3` after that tag. The cell was retaken on
2026-10-07 against `v0.112.1`. The RPM matches the signed manifest, and the
package installed on the host has the same header digest as the verified file:

```text
$ cosign verify-blob --bundle checksums.txt.cosign.bundle \
    --certificate-identity-regexp '^https://github.com/MustardSeedNetworks/niac-go/\.github/workflows/release\.yml@' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt
Verified OK
$ sha256sum niac-0.112.1-1.x86_64.rpm
b0b12cfdd4b56ea29a5b9f56296f6e494afe0520758247bef20fcf2fd9c6e7b4  niac-0.112.1-1.x86_64.rpm
$ grep 'x86_64.rpm$' checksums.txt
b0b12cfdd4b56ea29a5b9f56296f6e494afe0520758247bef20fcf2fd9c6e7b4  niac-0.112.1-1.x86_64.rpm
$ rpm -qp --qf '%{SHA256HEADER}\n' niac-0.112.1-1.x86_64.rpm
d5adcbf75a328dad01194de097251d32e98129cf5ea101769d158b36fcfaba54
dev-srv-fedora$ rpm -q --qf '%{SHA256HEADER}\n' niac
d5adcbf75a328dad01194de097251d32e98129cf5ea101769d158b36fcfaba54
```

The binary links libc alone, so no distribution libpcap is involved:

```text
dev-srv-fedora$ ldd /usr/bin/niac
	linux-vdso.so.1
	libc.so.6 => /lib64/libc.so.6
	/lib64/ld-linux-x86-64.so.2
dev-srv-fedora$ niac version
niac 0.112.1 (commit: 9838004238814f59bfb753c76380b6db533cecf9, built: 2026-10-05T23:50:28Z)
```

`make deploy-validate` installed `v0.112.0` over the host's existing `0.109.0`
configuration, started a simulation on a dummy link, then upgraded to
`v0.112.1` with that simulation running:

```text
$ make deploy-validate HOST=dev-srv-fedora RELEASE=v0.112.1
==> Install 1 of 2: v0.112.0 (this host may or may not already carry a configuration)
PASS /__version reports 0.112.0 with a non-empty uiBuildHash
==> Start a simulation on v0.112.0, so the upgrade runs against recovery state
PASS basic-network runs as deploy-validate-pre on niac-dv-pre
==> Install 2 of 2: v0.112.1, over the configuration the first one created
{
  "buildTime": "2026-10-05T23:50:28Z",
  "commit": "9838004",
  "platform": "linux/amd64",
  "uiBuildHash": "f1d267a8a4102d89b122a382451f8b57",
  "version": "0.112.1"
}
PASS /__version reports 0.112.1 after installing over an existing configuration
==> Watching for a restart loop for 30s
PASS niac.service active, NRestarts unchanged at 0
PASS a simulation is running after the upgrade
PASS deploy-validate-pre recovered across the upgrade
PASS basic-network started as deploy-validate-post after the upgrade
deploy-validate: v0.112.1 on dev-srv-fedora
```

The RPM upgrade runs the new `%post` before the old `%preun`; the
scriptlet-ordering defect that stopped the service there (#2085) does not
recur: the service stays active with no restarts.

## Windows 11, x86_64 (`dev-win11-02`)

```text
Microsoft Windows [Version 10.0.26200.9445]

PS> Get-FileHash niac.zip -Algorithm SHA256
3ED245969D1A6081EB65FD6BD365795C98D2C80B5B44720E7C5B6B33931911EB
$ grep windows-amd64.zip checksums.txt
3ed245969d1a6081eb65fd6bd365795c98d2c80b5b44720e7c5b6b33931911eb  niac-0.95.53-windows-amd64.zip

PS> niac.exe version
niac 0.95.53 (commit: 45bd8e140896d50db31fa85b206f9b9bd5d382f4, built: 2026-09-11T17:28:09Z)

PS> curl.exe -sk https://127.0.0.1:18447/__version
{
  "buildTime": "2026-09-11T17:28:09Z",
  "commit": "45bd8e1",
  "goVersion": "go1.27.0",
  "platform": "windows/amd64",
  "uiBuildHash": "c5f0f6cb7ab77b7366b26d01c3f27997",
  "version": "0.95.53"
}
```

Note the Windows artifact has its own `buildTime` — it is built separately from
the goreleaser-cross job, with the Npcap SDK.

**Capture is not verified on Windows.** `dev-win11-02` has no Npcap installed,
and installing it is an operator action on a shared host rather than a driver
one. Until that happens this cell covers install, start and serve only.

`dev-win11` (10.44.30.70) did not answer SSH when this was taken.

## Docker

Taken 2026-10-05 on `dev-srv-ubuntu` (Docker 29.1.3) against `v0.111.0`,
upgrading from `v0.110.0` on the same volume. Each image installs its
release's `linux-amd64.tar.gz` after checking it against that release's
`checksums.txt`. This host also runs a package install on 8445, hence the
port:

```text
$ scripts/lab/docker-validate.sh --port 8449
==> Validating the NIAC image for v0.111.0 (first run: v0.110.0)
PASS built niac:0.110.0 and niac:0.111.0 from their release archives
==> Run 1 of 2: v0.110.0 on a new volume
PASS /__version reports 0.110.0 with a non-empty uiBuildHash
PASS basic-network runs as docker-validate-pre on niac-dk-pre
==> Run 2 of 2: v0.111.0 over the volume run 1 left
PASS /__version reports 0.111.0 over the existing volume
==> Watching for a restart loop for 30s
PASS container running, RestartCount 0
PASS docker-validate-pre recovered across the upgrade
PASS a new simulation starts after the upgrade
==> Docker deployment validation PASSED for v0.111.0
```

## Browsers

The browser matrix is exercised per-PR by the Playwright suite on Chromium
and WebKit, the fleet policy. Native Safari and Edge acceptance is recorded
separately in the v1 plan and is an owner-run check, not an automated one.

## What closing this row still needs

1. Npcap on a Windows host, then capture there.
2. The `.pkg` cell, once the pipeline produces one.
