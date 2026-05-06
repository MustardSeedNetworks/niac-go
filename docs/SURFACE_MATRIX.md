# NIAC surface matrix — CLI / TUI / WebUI / API

Generated 2026-05-06 against `main` @ v0.66.36. This is the source of truth
for "which interface exposes which feature" and is the input to the
WebUI-parity follow-up.

## Frontend coverage at a glance

| Feature area                       | CLI          | TUI          | WebUI        | API          |
|------------------------------------|:------------:|:------------:|:------------:|:------------:|
| Run a simulation (legacy positional) | ✅ `niac <iface> <cfg>` | n/a          | n/a          | n/a          |
| Run a simulation (cobra subcommand)| ✅ `niac run`  | ✅ `niac interactive` | ✅ via `/runtime` | ✅ `/api/v1/simulation` (POST), `/api/v1/runtime` (GET) |
| Daemon mode (web UI host)          | ✅ `niac daemon` | n/a       | (this **is** the daemon) | n/a |
| List interfaces                    | ✅ `--list-interfaces` | ✅ stats panel | ✅ via `/runtime` | ✅ `/api/v1/interfaces` |
| Live device list                   | ❌           | ✅ device pane | ✅ `/devices` | ✅ `/api/v1/devices` |
| Device editor                      | ❌           | ✅ device_config.go | ✅ `/device-config/:hostname` | ✅ `/api/v1/config/devices/` |
| Topology view                      | ❌           | ✅ topology_view.go | ✅ `/topology` | ✅ `/api/v1/topology` |
| Topology export (graphviz/json)    | ❌           | ❌           | ✅ `/topology` (export button) | ✅ `/api/v1/topology/export` |
| Live stats                         | ✅ stdout stats ticker | ✅ stats.go | ✅ `/` dashboard + SSE | ✅ `/api/v1/stats`, `/api/v1/stream/stats` |
| Live packet inspector              | ❌           | ✅ hexdump.go | ✅ `/packets` (SSE) | ✅ `/api/v1/stream/packets` |
| Live log tail                      | ✅ `--log-file` + `tail -f` | ✅ logs.go | ✅ `/debug` (SSE) | ✅ `/api/v1/stream/logs` |
| PCAP playback                      | ✅ legacy `--pcap-file` | ✅ pcap_replay.go | ✅ `/analysis` | ✅ `/api/v1/replay` (under `/runtime`) |
| PCAP analyze                       | ✅ `niac analyze-pcap` | ❌ | ✅ `/pcap-analyzer` | ✅ `/api/v1/pcap/` |
| SNMP walk validation               | ✅ `niac analyze-walk` | ✅ snmp_walk.go + validation_panel.go | ❌ **gap (page)** | ✅ `/api/v1/walk/validate` |
| Walk auto-fix                      | ✅ `niac analyze-walk --fix` | ✅ TUI | ❌ **gap (page)** | ✅ `/api/v1/walk/fix` |
| Run history (storage-backed)       | ❌           | ✅ history.go | ✅ via dashboard | ✅ `/api/v1/history` |
| Templates: list                    | ❌           | ✅ template_browser.go | ✅ `/templates` | ✅ `/api/v1/templates` |
| Templates: get one                 | ❌           | ✅ TUI       | ✅ `/templates`  | ✅ `/api/v1/templates/{name}` |
| User configs CRUD                  | ❌           | ❌           | ✅ devices/config-builder | ✅ `/api/v1/configs[/{name}]` |
| Config diff                        | ✅ `niac config diff` | ✅ config_diff.go | ✅ `/config-diff` | ❌ (client-side) |
| Config merge                       | ✅ `niac config merge` | ❌ | ❌ **gap** | ❌ **gap** |
| Config export (Java DSL → YAML)    | ✅ `niac config export` | ❌ | ❌ **gap** | ❌ **gap** |
| Config init (template scaffold)    | ✅ `niac init` | ❌ | (close enough via Templates page) | ❌ |
| Config generate (interactive Q&A)  | ✅ `niac generate` | ✅ device_config.go | ✅ `/device-config/new` | ❌ (client-side) |
| Error / fault injection            | ✅ `niac inject` (`list`, `clear`) | ✅ error_injection.go | ✅ `/traffic` (Traffic Injection) | ✅ `/api/v1/errors` |
| Neighbor discovery view            | ✅ `niac neighbors`, `niac neighbors watch` | ❌ **gap** | ❌ **gap** (in topology?) | ✅ `/api/v1/neighbors` |
| Live monitor (TUI-lite over SSH)   | ✅ `niac monitor` | n/a       | (dashboard equivalent) | ✅ via streams |
| Service control (Windows)          | ✅ `niac service {install,uninstall,start,stop,status}` | ❌ | ❌ | ❌ |
| Shell completion                   | ✅ `niac completion {bash,zsh,fish,powershell}` | n/a | n/a       | n/a       |
| Man page                           | ✅ `niac man` | n/a          | n/a          | n/a       |
| Dump (config dump)                 | ✅ `niac dump` | ❌           | ❌ **gap** | ❌ **gap** |
| Logs subcmd (`logs`, `logs tail`)  | ✅           | ✅           | ✅           | ✅           |
| Alert config (admin)               | ❌ (config file only) | ✅ alert_config.go | ❌ **gap (page exists, not wired?)** | (via runtime config update) |
| Webhook host allowlist             | ✅ `--webhook-allowed-host` (daemon flag) | n/a       | n/a       | n/a       |
| Health endpoint                    | n/a       | n/a       | n/a       | ✅ `/__version` (no auth) |
| Prometheus metrics                 | n/a       | n/a       | n/a       | ✅ `/metrics` |
| CSRF token                         | n/a       | n/a       | n/a       | ✅ `/api/v1/csrf-token` |
| Files browse / serve               | n/a       | n/a       | (via templates) | ✅ `/api/v1/files` |
| Schema introspection               | n/a       | n/a       | (used by DeviceEditor) | ✅ `/api/v1/config/schema` |

Legend: ✅ supported · ❌ missing · **gap** = WebUI is meant to have this and doesn't (priority follow-up).

## Per-interface inventories

### CLI (cobra subcommands + legacy positional mode)

```
niac                          # legacy: <interface> <config> with all the --debug-* flags
niac run <iface> <cfg>        # cobra alias for the same
niac interactive <iface> <cfg># launches the TUI

niac daemon                   # web UI host (--listen, --token, --webhook-allowed-host, --storage)
niac monitor                  # remote-stats TUI-lite that talks to a running daemon

niac analyze-pcap <pcap>      # parse a PCAP and dump structured info
niac analyze-walk <walkfile>  # validate a walk file (--fix to auto-repair)

niac config export <in> <out> # Java DSL → YAML
niac config diff <a> <b>      # YAML diff
niac config merge <base> <overlay> <out>

niac init [out.yaml]          # scaffold a starter config
niac generate [out.yaml]      # interactive Q&A

niac inject <device> <error-type> <value>
niac inject list
niac inject clear

niac neighbors                # one-shot
niac neighbors watch          # live

niac dump                     # dump runtime config
niac logs
niac logs tail
niac man
niac completion {bash,zsh,fish,powershell}
niac service ...              # Windows only
```

### TUI screens (`internal/interactive/`)

| File                     | Screen / responsibility                                  |
|--------------------------|----------------------------------------------------------|
| `interactive.go`         | Top-level loop, screen routing, status bar                |
| `stats.go`               | Live counters per device + protocol                       |
| `device_config.go`       | Edit a device in-place                                    |
| `topology_view.go`       | ASCII topology                                            |
| `template_browser.go`    | List + preview built-in templates                         |
| `pcap_replay.go`         | PCAP playback control                                     |
| `snmp_walk.go`           | Walk file viewer / validator                              |
| `validation_panel.go`    | Show walk-validation issues + fixes                       |
| `hexdump.go`             | Live packet hex view                                      |
| `logs.go`                | Live log tail                                             |
| `error_injection.go`     | Set/clear faults                                          |
| `alert_config.go`        | Threshold + webhook URL editor                            |
| `config_diff.go`         | YAML diff viewer                                          |
| `history.go`             | Past run history (from storage.db)                        |
| `search.go`              | Cross-screen find                                         |
| `export.go`              | Topology / report export                                  |
| `keyboard.go`, `handlers.go`, `display_utils.go`, `helpers_test.go`, `types.go` | infrastructure |

### WebUI pages (`ui/src/pages/`, routed in `ui/src/App.tsx`)

| Route                          | Page                       | Purpose                                          |
|--------------------------------|----------------------------|--------------------------------------------------|
| `/`                            | DashboardPage              | live stats, recent runs                          |
| `/runtime`                     | RuntimeControlPage         | start/stop simulation, choose interface + config |
| `/devices`                     | DevicesPage / DeviceListPage | live device table                              |
| `/device-config`               | DeviceEditorPage           | full device editor                              |
| `/device-config/new`           | DeviceEditorPage (new mode)| create device                                    |
| `/device-config/:hostname`     | DeviceEditorPage (edit)    | edit device                                      |
| `/topology`                    | TopologyPage               | interactive graph + export                      |
| `/analysis`                    | AnalysisPage               | replay control                                   |
| `/automation`                  | AutomationPage             | alerts & workflows (beta)                        |
| `/traffic`                     | TrafficInjectionPage       | error / fault injection                          |
| `/debug`                       | DebugConsolePage           | log tail, raw API console                        |
| `/packets`                     | PacketInspectorPage        | live packet stream                              |
| `/templates`                   | TemplatesPage              | template browser                                 |
| `/config-diff`                 | ConfigDiffPage             | YAML diff                                        |
| `/pcap-analyzer`               | PcapAnalyzerPage           | upload + analyze a PCAP                          |

### API endpoints (`internal/api/routes.go`)

```
/__version                               # health (no auth)
/metrics                                 # prometheus

/api/v1/csrf-token
/api/v1/version
/api/v1/stats
/api/v1/devices
/api/v1/history
/api/v1/interfaces
/api/v1/runtime                         # POST start, DELETE stop, GET status
/api/v1/topology                        # GET tree
/api/v1/topology/export                 # GET ?format=graphviz|json
/api/v1/config/schema                   # JSON schema (powers DeviceEditor)
/api/v1/config/devices                  # CRUD on the running config
/api/v1/config/devices/{name}           # CRUD per device
/api/v1/configs                         # named user configs CRUD
/api/v1/configs/{name}
/api/v1/templates                       # built-in template list
/api/v1/templates/{name}
/api/v1/files                           # file browse (sandboxed)
/api/v1/pcap/                           # pcap analyzer
/api/v1/errors                          # error/fault injection
/api/v1/neighbors                       # neighbor table

/api/v1/replay                          # PCAP replay control (under runtime)
/api/v1/capture/filter                  # BPF filter for live packets

/api/v1/stream/logs                     # SSE
/api/v1/stream/packets                  # SSE
/api/v1/stream/stats                    # SSE
/api/v1/stream/status                   # SSE
```

## Protocol coverage (config schema → runtime)

Generated from `internal/config/config.go` (struct types) cross-referenced
against `internal/protocols/*.go` (implementations). All present and
implemented:

ARP · CDP · DHCP (v4) · DHCPv6 · DNS · EDP · FDP · FTP · HTTP · ICMP ·
ICMPv6 · IP (v4 / v6) · iperf3 · LLDP · NetBIOS · OS-fingerprint ·
SNMP (agent + walks + traps) · STP · TCP · UDP · plus traffic
generation (ARP announce, periodic ping, random traffic).

## Identified gaps (priority for follow-up PR)

These are real WebUI / API gaps where the CLI exposes something the WebUI
doesn't:

1. **SNMP walk validation + auto-fix** — API exists
   (`POST /api/v1/walk/{validate,fix}`), but no WebUI page consumes it.
   _Suggested:_ a "Validate Walk" button in the device editor + a
   standalone `/walk` page.
2. **Config merge** — only in CLI. _Suggested:_ `POST /api/v1/config/merge`
   plus a "Merge into base" action in `/config-diff`.
3. **Config export (Java DSL → YAML)** — only in CLI. _Suggested:_
   `POST /api/v1/config/import?format=java-dsl` + a "Import Java DSL"
   button on the templates page.
4. **`niac dump`** — runtime config dump in CLI; nothing in WebUI.
   _Suggested:_ `GET /api/v1/runtime/dump` + a "Dump current config"
   button on the runtime page.
5. **Alert config (admin-side)** — TUI has a full editor; WebUI's
   automation page exists but doesn't appear to wire it. _Suggested:_
   `PUT /api/v1/alerts/config` + an alert editor on `/automation`.
   (The data path already lives in `internal/api/alerts.go`.)
6. **Neighbor watch view** — CLI has it, WebUI doesn't surface it
   distinctly from the topology view. _Suggested:_ a `Neighbors` panel
   on the dashboard or a dedicated `/neighbors` page.

These are the "WebUI behind CLI" items called out in the original ask.

## Coverage of features by example config

See [examples/README.md] (existing) for the per-file purpose summary. The
audit of "which examples cover which features" is in
[CONFIG_COVERAGE.md](./CONFIG_COVERAGE.md) (next file).
