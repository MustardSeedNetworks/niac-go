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
| Fedora 44 (x86_64) | `x86_64.rpm` | **no — see below** | — | — | — |
| Windows 11 (x86_64) | `windows-amd64.zip` | yes | yes | yes | not taken — no Npcap |
| Docker | image | not taken | — | — | — |

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

## Fedora 44, x86_64 (`dev-srv-fedora`) — FAILS on this release

`v0.95.53` cannot start on Fedora. The binary carries Debian's libpcap SONAME,
which no Fedora libpcap provides (#1999):

```text
$ ldd ./niac | grep -i pcap
	libpcap.so.0.8 => /usr/lib/x86_64-linux-gnu/libpcap.so.0.8
```

The fix merged at `cbaad8d3`, **after** this tag, so `v0.95.54` is the first
release that can run here. The approach was proven before the pipeline changed:
a statically linked build depends on libc alone, runs on Fedora 44, enumerates
interfaces, and served a scenario over a veth pair in a throwaway namespace:

```text
$ ldd /tmp/niac-static
	linux-vdso.so.1
	libc.so.6 => /lib64/libc.so.6
	/lib64/ld-linux-x86-64.so.2

$ sudo ip netns exec pcaptest ping -c 3 -W 2 10.253.9.1
3 packets transmitted, 3 received, 0% packet loss
```

**This cell is not closed.** It must be retaken against the `v0.95.54` RPM
through `make deploy-validate HOST=dev-srv-fedora`, which is also the check
that found the defect.

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

## Docker — not taken

`deploy/docker` exists and the image needs `NET_RAW` + `NET_ADMIN`, but neither
dev server has a usable container runtime:

```text
$ ssh dev-srv-ubuntu 'docker info'
docker not usable
```

## Browsers

The browser matrix is exercised per-PR by the Playwright suite across Chromium,
WebKit, installed Chrome and installed Edge. Native Safari acceptance is
recorded separately in the v1 plan and is an owner-run check, not an automated
one.

## What closing this row still needs

1. Retake the Fedora cell against the `v0.95.54` RPM.
2. Npcap on a Windows host, then capture there.
3. A container runtime, then the Docker cell.
4. The `.pkg` cell, once the pipeline produces one.
