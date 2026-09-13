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

The released Linux binaries and packages link libpcap statically, so there is
no capture library to install: they depend on the C library alone and run on
both Debian and Red Hat families. Building from source still needs the
development headers (`libpcap-dev` / `libpcap-devel`).

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

Recovery also restores running/startup device configuration, active interface
and device faults, named checkpoints, and retained device event history from
the session's runtime record. Restored historical events are not retransmitted
as new traps or syslog messages. Protocol timers and authored behavior timelines
start again; this is not a suspended-process image.

An orderly daemon shutdown stops producers and flushes final device state.
While running, changed state is saved periodically (every two seconds when
the writer is scheduled). A process crash recovers the last completed save, so recent
changes can be lost; slow or failing storage can extend that window. A write
failure is logged rather than silently treated as successful persistence.

Each replacement uses a distinct generation of runtime and inline configuration
files. The launch record selects the committed generation, so an interrupted
replacement cannot combine a new launch with an old run's saved faults.

An explicit simulation Stop removes that session's recovery intent and runtime
record. A subsequent Start begins from the authored configuration with no
previous runtime faults. If recovery fails, NIAC keeps
the API available, leaves the simulation stopped, and reports an actionable
`recovery` object from `GET /api/v1/simulation`. Correct the reported condition
or remove the named state file before restarting the daemon.

Writes use a synchronized temporary file followed by an atomic replacement.
An interrupted temporary write is ignored on the next start.
This covers daemon process failure, not host power loss or storage failure;
directory entries are not synchronized for a power-loss durability guarantee.

Recovery files use a validated private schema. Before upgrading from an older
pre-1.0 build that saved only launch intent, stop its simulations and start them
again after the upgrade. Launch-only records cannot recover runtime data that
was never saved and are rejected rather than reported as successful recovery.

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
  serves no UI),
- upgrading over the configuration, database and recovery state an **older**
  release left behind does not leave the unit restarting, and
- the daemon still starts a simulation afterwards, and the one that was running
  before the upgrade survived it.

That last pair is why the first install is the previous release rather than the
same package twice: a reinstall of identical bytes crosses no version boundary,
so nothing an older build wrote is ever carried forward. The simulations run on
throwaway `dummy` interfaces the script creates and deletes, so nothing it
starts reaches the host's network.

```bash
make deploy-validate HOST=dev-srv-ubuntu
make deploy-validate HOST=dev-srv-fedora RELEASE=v0.95.38
make deploy-validate HOST=dev-srv-ubuntu FROM_RELEASE=v0.95.57
make deploy-validate HOST=dev-srv-ubuntu FROM_RELEASE=none   # same-version reinstall
make deploy-validate HOST=dev-srv-ubuntu PACKAGE=./dist/niac_0.95.38_amd64.deb
```

`HOST` is an ssh target with passwordless sudo and is required; deployment
hosts are passed in rather than hardcoded. `FROM_RELEASE` names the build
installed first and defaults to the release immediately before `RELEASE`. The assertions run on the host
against the loopback listener, so a closed firewall does not read as a broken
deployment. Packages themselves are built by goreleaser in CI — there is no
local packaging target, and `deploy-validate` deliberately does not add one.
