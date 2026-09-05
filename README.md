# NIAC — Network In A Can

> Single-binary network device simulator for protocol testing, packet capture analysis, and topology modelling.

[![CI](https://github.com/MustardSeedNetworks/niac-go/actions/workflows/ci.yml/badge.svg)](https://github.com/MustardSeedNetworks/niac-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/MustardSeedNetworks/niac-go?logo=github)](https://github.com/MustardSeedNetworks/niac-go/releases/latest)
[![CodeQL](https://github.com/MustardSeedNetworks/niac-go/actions/workflows/codeql.yml/badge.svg)](https://github.com/MustardSeedNetworks/niac-go/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/MustardSeedNetworks/niac-go/badge)](https://scorecard.dev/viewer/?uri=github.com/MustardSeedNetworks/niac-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/MustardSeedNetworks/niac-go.svg)](https://pkg.go.dev/github.com/MustardSeedNetworks/niac-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/MustardSeedNetworks/niac-go)](https://goreportcard.com/report/github.com/MustardSeedNetworks/niac-go)
[![License: BSL 1.1](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)

NIAC is a network device simulator from **Mustard Seed Networks**. It stands up
configurable layer-2/3 endpoints — routers, switches, servers, workstations,
APs — that respond to ARP, ICMP, DHCP, DNS, LLDP/CDP, SNMP, HTTP, and other
protocols against real interfaces. Use it to exercise discovery tools, generate
test traffic, validate monitoring pipelines, or reproduce field issues in
the lab.

## Features

- **Configurable topology** — declare devices, interfaces, VLANs, and
  neighbours in YAML; load templates or generate interactively
- **Multi-IP per device** — each simulated endpoint can carry multiple v4/v6 addresses
- **Protocol coverage** — ARP, ICMPv4/v6, DHCPv4/v6, DNS, LLDP, CDP, SNMP
  (v1/v2c/v3, walks + traps), TCP, UDP (incl. NetAlly reflector), HTTP, iperf3
- **Per-protocol debug levels** — turn verbose logging on/off at the protocol layer without restarting
- **PCAP analysis** — `niac analyze-pcap` summarises captures by protocol;
  `niac analyze-walk` extracts topology from SNMP walks
- **Error injection** — inject latency, loss, jitter, or protocol-specific faults on a running simulation
- **Web UI** — daemon mode exposes a React/TypeScript control plane over HTTPS
  on port 8445
- **Templates** — ship YAML scenarios (`niac template`) and run them anywhere

## Quick Start

```bash
# Install (Linux/macOS, requires Go 1.27+)
git clone https://github.com/MustardSeedNetworks/niac-go
cd niac-go
make build

# Generate a starter config interactively
./niac init my-lab.yaml

# Validate
./niac validate my-lab.yaml

# Run on eth0 (needs CAP_NET_RAW or sudo)
sudo ./niac run eth0 my-lab.yaml

# Or start the daemon + web UI
sudo ./niac daemon
# → open https://localhost:8445
```

## Commands

<!-- BEGIN GENERATED COMMANDS -->

| Command | Description |
| --- | --- |
| `niac analyze-pcap <pcap-file>` | Summarise a packet capture by protocol |
| `niac analyze-walk <walk-file>` | Analyze an SNMP walk file: device, interfaces, and LLDP/CDP neighbors |
| `niac completion [bash\|zsh\|fish\|powershell]` | Generate completion script |
| `niac config` | Configuration management tools |
| `niac content` | Install and inspect the on-disk content library |
| `niac daemon` | Run NIAC in daemon mode with web UI control |
| `niac dump` | Dump captured packets from a running NIAC simulation |
| `niac init [output-file]` | Interactive template wizard for quick configuration setup |
| `niac install-ca` | Install NIAC's self-signed root certificate into the OS trust store |
| `niac list` | List interfaces and demo content |
| `niac logs` | View and stream simulation logs |
| `niac man` | Generate man pages |
| `niac mibzip` | Convert SNMP walk files to and from MibZip format |
| `niac monitor` | Stream real-time statistics from a running NIAC simulation |
| `niac neighbors [watch]` | Display neighbor discovery table from LLDP/CDP protocols |
| `niac run <interface> <config-file>` | Run network simulation |
| `niac sanitize <input-walk> <output-walk>` | Sanitize SNMP walk files with NIAC branding |
| `niac status` | Query the status of a running NIAC simulation |
| `niac template` | Manage configuration templates |
| `niac topology` | Network topology management commands |
| `niac validate <config-file>` | Validate a NIAC configuration file |
| `niac version` | Print the version, commit and build metadata |

<!-- END GENERATED COMMANDS -->

The table is generated from the command tree by `make cli-docs`. On Windows the
binary also carries `niac service` for Windows service management. Run
`niac <command> --help` for flags, or see [docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md).

## Architecture

```text
ui/src/             → React/TypeScript control plane (Vite)
                          ↓ npm run build
internal/api/ui/    → Built assets (embedded via go:embed)
                          ↓
cmd/niac/           → Cobra-based CLI (subcommands above)
internal/
├── api/            → HTTP/WebSocket handlers
├── protocols/      → Per-protocol simulators (arp, icmp, dhcp, snmp, …)
├── device/         → Device model + simulator loop
├── topology/       → Topology graph + neighbours
├── ipc/            → daemon ↔ runner socket
├── converter/      → YAML ↔ runtime config
└── version/        → Build metadata (injected via ldflags)
```

The frontend builds **directly into `internal/api/ui/`** and is embedded at
compile time — no copy step, no file syncing. One binary, no runtime
dependencies.

Architecture decisions live in [`docs/adr/`](docs/adr/). The
schema-generation pattern used to keep YAML schemas, Go structs, and
(soon) TypeScript types in sync is documented in
[ADR 0001](docs/adr/0001-schema-generation-from-go-structs.md).

## Configuration

YAML topology + per-device behaviour. Generate a starter with `niac init`
or use a template (`niac template list`). Schema is documented in
[`docs/schemas/niac.schema.json`](docs/schemas/niac.schema.json) and
regenerated by `make schema`.

```yaml
# minimal example
devices:
  - name: switch-1
    type: switch
    interfaces:
      - name: eth0
        ipv4: 192.168.1.10/24
        protocols: [arp, icmp, lldp, snmp]
```

## Demo Assets

Large example scenarios, walks, and captures are generated from the shared
NIAC demo catalog instead of being committed to this repo. Generation records
the immutable source commit and validates scenarios and walks before updating
the local examples:

```bash
./scripts/sync-demo-catalog.sh --sync
```

Windows:

```powershell
.\scripts\sync-demo-catalog.ps1 -Mode Sync
```

See [`docs/SHARED_DEMO_CATALOG.md`](docs/SHARED_DEMO_CATALOG.md).

## Build

| Command | Purpose |
| --------- | --------- |
| `make build` | Full build (frontend + backend) |
| `make quick` | Backend-only (dev iteration; do **not** ship) |
| `make test` | Go unit + integration tests |
| `make test-e2e` | Playwright UI tests |
| `make lint` | golangci-lint + Biome |
| `make fmt-check` | Format check (Go + TS) |
| `make fmt-all` | Auto-format everything |
| `make schema` | Regenerate JSON schema from `Config` struct |

Verified versions: **Go 1.27.0**, Node.js 26.8.1, golangci-lint v2.13.2.
All release artifacts are built in GitHub Actions by the pinned `release.yml`
pipeline after release-please creates a `v*` tag. Linux and Apple Silicon macOS
use GoReleaser Cross; Windows uses native GitHub runners with CGO and the Npcap
SDK. Intel macOS is not a supported release target.

## Versioning & Releases

Conventional commits drive [release-please](https://github.com/googleapis/release-please).
`feat:` → minor bump, `fix:` → patch, `refactor:`/`chore:`/`ci:` →
no bump (use `Release-As:` footer to force). Tags trigger `release.yml`
which builds Linux, macOS, and Windows archives and attaches checksums plus
SLSA provenance to the GitHub release.

## Limits

NIAC ships as one unrestricted binary — no runtime tier, no activation, no
phone-home. One configuration may carry up to 1,000 devices, and the daemon
bounds concurrent sessions and total devices across everything running at once.
These are technical safety limits, not entitlements.

## License

[Business Source License 1.1](LICENSE) — free for non-commercial use;
commercial use requires a license. Converts to Apache-2.0 on the change
date stated in the LICENSE file. Matches the licensing on seed and stem.

For commercial licensing inquiries: `kris.armstrong@gmail.com`.

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability-disclosure policy.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Related projects

NIAC is the simulator. Two sibling tools complete the Mustard Seed Networks
testing toolkit:

- **[seed](https://github.com/MustardSeedNetworks/seed)** — portable network diagnostic appliance
- **[stem](https://github.com/MustardSeedNetworks/stem)** — RFC-compliant network performance testing
