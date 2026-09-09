# Deployment Guide

NIAC is distributed as native binaries. Container deployment is not the primary
deployment model because packet capture and network simulation need direct host
interface access.

## Linux

Download the Linux archive for your architecture from the GitHub release,
extract it, and install the binary:

```bash
tar -xzf niac-linux-amd64.tar.gz
sudo install -m 0755 niac-linux-amd64/niac /usr/local/bin/niac
```

Install packet capture dependencies:

```bash
sudo apt-get install libpcap0.8
```

For service operation, use the bundled `systemd/niac.service` as a starting
point and adjust paths, user, interface, and config locations for the host.

The DEB and RPM packages do not open TCP 8445 by default. Set
`NIAC_OPEN_FIREWALL=1` during installation to add a UFW or firewalld rule. The
package records ownership only when it creates that rule; purge removes a
package-created rule but preserves any matching rule that existed beforehand.

## macOS

Download the macOS archive for your architecture from the GitHub release:

```bash
tar -xzf niac-darwin-arm64.tar.gz
sudo install -m 0755 niac-darwin-arm64/niac /usr/local/bin/niac
```

Install libpcap through the operating system or Homebrew if your environment
requires a newer package:

```bash
brew install libpcap
```

The archive includes launchd helper files under `launchd/` for users who want
NIAC to run as a service.

## Daemon Simulation Recovery

Daemon mode records the active simulation launch intent in
`<data-root>/state/active-simulation.json`. A graceful service shutdown keeps
this record, and the next daemon start restores the simulation only after the
attachment policy, host interface, configuration, runtime requirements, and
routed preflight all pass.

An explicit simulation stop removes the record. If recovery fails, NIAC keeps
the API available, leaves the simulation stopped, and reports an actionable
`recovery` object from `GET /api/v1/simulation`. Correct the reported condition
or remove the named state file before restarting the daemon.

Writes use a synchronized temporary file followed by an atomic replacement.
An interrupted temporary write is ignored on the next start.

## Signal Handling

`SIGHUP` means different things depending on how NIAC was started, because
the two modes have different things worth reloading without a restart:

- **Daemon mode** (`niac daemon`, including the systemd unit) — `SIGHUP`
  rotates the API bearer-token set: it re-reads the configured token file (or
  `NIAC_API_TOKEN`) and swaps in the new tokens without dropping connections
  or restarting the simulation. See `cmd/niac/cmd_daemon.go`'s `handleSIGHUP`
  and `internal/daemon.Daemon.ReloadTokens`.
- **Single-shot mode** (`niac daemon --once <interface> <config-file>`) — the
  same runtime with no listener, so there is nothing to rotate; it runs for
  `--duration` or until interrupted, then prints a JSON summary.

`SIGTERM`/`SIGINT` mean the same thing in both modes: stop the simulation and
exit cleanly.

## Windows

Download and extract the Windows zip for your architecture from the GitHub
release. From an elevated PowerShell session, run the bundled installer:

```powershell
.\install.ps1
```

The installer validates that the archive-local `niac.exe` is present, copies it
under Program Files, adds NIAC to the machine `PATH`, installs the service, and
starts it. You can instead run `niac.exe` directly from the extracted folder.

Npcap is required for packet capture and injection features:

```text
https://npcap.com/
```

Install Npcap in WinPcap-compatible mode when using tools or workflows that
expect the WinPcap API.

## Validation

After installation:

```bash
niac version
niac --help
```

For API deployments, set `NIAC_API_TOKEN` and validate the health endpoint from
the same host or trusted management network.

### Automated deployment validation

An install is not finished until the service answering on the host is provably
the artifact the release published. `make deploy-validate` installs a released
package over ssh and asserts that:

- `/__version` reports the installed version,
- `uiBuildHash` is non-empty, which is the only signal that the web UI was
  embedded (a binary built outside the make pipeline reports an empty hash and
  serves no UI), and
- installing a second time, over the configuration and database the first
  install left behind, does not leave the unit restarting.

```bash
make deploy-validate HOST=dev-srv-ubuntu
make deploy-validate HOST=dev-srv-fedora RELEASE=v0.95.38
make deploy-validate HOST=dev-srv-ubuntu PACKAGE=./dist/niac_0.95.38_amd64.deb
```

`HOST` is an ssh target with passwordless sudo and is required; deployment
hosts are passed in rather than hardcoded. The assertions run on the host
against the loopback listener, so a closed firewall does not read as a broken
deployment. Packages themselves are built by goreleaser in CI — there is no
local packaging target, and `deploy-validate` deliberately does not add one.
