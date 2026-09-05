# NIAC FAQ

## What is NIAC?

NIAC is a source-available network-device simulator for lab, monitoring,
discovery, and troubleshooting workflows. It responds as configurable devices
over real network interfaces and supports routed labs, SNMP projections,
device CLI/SSH behavior, packet capture analysis, and replay.

## What are the system requirements?

- A supported Linux, macOS, or Windows host.
- A packet-capture driver: libpcap on Linux/macOS or Npcap on Windows.
- Permission to capture and inject packets on the selected interface.
- A current supported browser for the web UI.

See [Platform Support](PLATFORM.md) and [Deployment](DEPLOYMENT.md).

## Why are elevated packet permissions required?

NIAC reads and writes Ethernet frames. Linux can grant the installed binary
the required capabilities; macOS commonly requires elevated execution for
capture; Windows uses Npcap permissions. Use the narrowest permissions that
work for the lab host.

## How do I create and validate a configuration?

Start from a shipped template or a YAML example, then validate it:

```bash
niac template list
niac template use minimal lab.yaml
niac validate lab.yaml
```

The committed schema at `docs/schemas/niac.schema.json` provides editor
completion and structural validation. Runtime validation remains authoritative.

## How do I start a simulation?

For a headless run:

```bash
sudo niac daemon --once eth0 lab.yaml
```

For the HTTPS API and web UI:

```bash
niac daemon
```

Then open `https://localhost:8445`. Non-loopback listeners require an API
token. Routed labs also require an operator-approved attachment policy; see
[Deployment](DEPLOYMENT.md).

## How do I stop a simulation?

Use Ctrl+C for a foreground CLI run. In daemon mode, stop the simulation from
the web UI or API; stopping the daemon also releases active runtime resources.

## Can I run more than one NIAC instance?

Yes, when each process uses a different capture/interface boundary, HTTPS
listener, storage path, and managed content location. Avoid attaching two
simulators to the same production-facing interface.

## How many devices can I simulate?

One configuration may carry up to 1,000 devices, enforced across CLI, API, UI,
import, template, configuration mutation, and runtime-start paths. The daemon
additionally bounds concurrent sessions and total devices across everything
running at once.

Practical capacity depends on enabled protocols, traffic rate, walk size,
host resources, and the observer polling NIAC.

## Which protocols are supported?

Run `niac help` and inspect the current schema/help drawer for the shipping
surface. NIAC includes common discovery, addressing, management, service, and
routing behavior. Documentation does not use a fixed protocol count because
the set changes as implementations are added or retired.

## How do SNMP walk files work?

A sanitized walk can provide the baseline MIB identity and tables for a
simulated device. Authoritative runtime state overlays dynamic values such as
interface status, counters, addresses, routes, and topology. See
[SNMP Walks](SNMP_WALKS.md).

## Are interface faults visible to monitoring tools?

Yes. Supported injected faults advance the corresponding IF-MIB, IF-X, and
EtherLike-MIB counters while active. Clearing a fault stops new increments
without resetting the monotonic counter value.

## Can NIAC send notifications?

NIAC can emit configured SYSLOG and SNMP notifications for supported
authoritative state transitions. Routed notification delivery follows the
simulated forwarding path and is part of routed-lab acceptance.

## How is the API authenticated?

Loopback-only development can run without a token. A non-loopback listener
requires a bearer token or scoped token file. Mutating browser requests also
require a per-session CSRF token. Destructive whole-configuration operations
require admin authorization.

See [REST API](REST_API.md) and [API Reference](API_REFERENCE.md).

## Is the API rate-limited?

Yes. Authentication, write, file, and other sensitive route classes have
declared limits. A rate-limit response uses HTTP 429 and includes the
appropriate retry metadata.

## Why is a device not responding?

Check, in order:

1. the selected physical interface and link state;
2. the attachment policy and VLAN mode for routed labs;
3. configuration validation and preflight diagnostics;
4. host firewall and packet-capture permissions;
5. address, route, and protocol configuration;
6. daemon logs and a packet capture on the attachment.

## Why is PCAP replay not starting?

Confirm the file is a valid classic PCAP or pcapng file inside an allowed
managed root, the selected BPF filter compiles, and a capture engine is active.
Malformed or truncated trailing records are excluded from replay accounting;
the valid prefix is replayed.

## How do I inspect performance?

Use `/metrics`, `/api/v1/stats`, and the benchmark suites in the repository.
Legacy CLI profiling is available only when explicitly enabled and binds to
loopback. See [Performance](PERFORMANCE.md), [Benchmarking](BENCHMARKING.md),
and [Monitoring](MONITORING.md).

## Which browsers are supported?

Chrome stable, Edge stable, and current Safari are first-class. Firefox is an
automated independent-engine compatibility target. Brave receives a focused
pre-release smoke test with default Shields. See [Web UI](WEBUI.md).

## Why is the UI not updating?

Check `/__version`; `uiBuildHash` must be non-empty. Confirm the simulation is
still running and inspect the affected SSE or WebSocket request. The UI should
reconnect and refresh authoritative state after an interruption.

## Can I customize the UI?

The source lives in `ui/src`, but release binaries embed one supported UI and
backend build. Make changes on a feature branch and use `make build`; do not
copy a separate frontend distribution into the binary.

## How do I deploy NIAC?

Use the signed, checksummed artifacts from GitHub Releases and follow
[Deployment](DEPLOYMENT.md). GitHub Actions is the release build environment;
local packages are development aids, not canonical release artifacts.

## How do I report a defect?

Open a GitHub issue with:

- NIAC version and `/__version` output;
- operating system, architecture, and browser when relevant;
- the smallest sanitized configuration or capture that reproduces it;
- exact steps and expected/actual behavior;
- relevant logs without credentials or customer identifiers.

## Where is the current roadmap?

The active pre-1.0 exit criteria are in [ROADMAP.md](ROADMAP.md). Historical
plans and reviews live under `docs/archive` and are not current commitments.
