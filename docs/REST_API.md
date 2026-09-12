# REST API & Web UI

NIAC exposes a REST API, Prometheus metrics endpoint, and bundled Web UI over its HTTPS daemon listener.

## Enabling the API

```bash
export NIAC_API_TOKEN="$(openssl rand -base64 32)"
niac daemon
```

Flags:

| Flag | Description |
| - | - |
| `--listen` | HTTPS address for REST API & Web UI (default: `127.0.0.1:8445`) |
| `--api-token` | Daemon bearer token (prefer `NIAC_API_TOKEN`) |
| `--cert-dir` | Directory containing the generated HTTPS certificate and key |
| `--storage` | BoltDB location for run history (default: `~/.niac/niac.db`, set to `disabled` to opt out) |

Prometheus metrics share the daemon's HTTPS listener at `/metrics`; there is no separate metrics listener.

## Endpoints

| Method | Path | Description |
| - | - | - |
| `GET` | `/api/v1/stats` | Live packet counters, interface info, NIAC version |
| `GET` | `/api/v1/devices` | Device inventory (type, IPs, enabled protocols) |
| `GET` | `/api/v1/history` | Recent runs persisted to BoltDB |
| `GET` | `/api/v1/config` | Active YAML config plus file metadata |
| `PUT` | `/api/v1/config` | Validate + persist new YAML config content |
| `GET` | `/api/v1/replay` | Current PCAP replay status |
| `POST`/`DELETE` | `/api/v1/replay` | Start or stop packet replay |
| `GET` | `/api/v1/alerts` | Current alert threshold + webhook |
| `PUT` | `/api/v1/alerts` | Update alert threshold/webhook |
| `GET` | `/api/v1/files?kind=walks\|pcaps` | List available SNMP walk or PCAP files |
| `GET` | `/api/v1/topology` | Simple topology graph derived from configuration |
| `GET` | `/api/v1/sessions` | Running simulation sessions a client can address |
| `GET` | `/api/v1/sessions/{id}/{resource}` | One session's runtime state — see below |
| `DELETE` | `/api/v1/sessions/{id}` | Stop that session |
| `GET` | `/api/v1/scenario/packs` | Versioned presentation presets (hospital, warehouse, manufacturing, campus, retail, service-provider) plus the `enterprise-scale` stress preset |
| `GET` | `/api/v1/scenario/profiles` | Reusable vendor, model, role, and discovery profiles |
| `POST` | `/api/v1/scenario/generate` | Generate deterministic validated YAML from sites and repeat controls |
| `PATCH` | `/api/v1/library/drafts/{name}/topology` | Apply one revision-safe topology edit to an isolated draft |
| `PUT` | `/api/v1/library/drafts/{name}/behaviors` | Replace a draft's deterministic behavior timelines |
| `GET` | `/api/v1/behaviors` | Current saved-timeline replay status |
| `GET` | `/api/v1/version` | Version information |
| `GET` | `/api/v1/errors` | Available error types and active error injections |
| `POST` | `/api/v1/errors` | Inject an interface fault or a device-service fault |
| `DELETE` | `/api/v1/errors` | Clear specific or all error injections |
| `GET` | `/metrics` | Prometheus metrics endpoint (see [Monitoring Guide](MONITORING.md)) |

Include `Authorization: Bearer <token>` or append `?token=<token>` when authentication is enabled.

### Scenario generation

`GET /api/v1/scenario/packs` returns the versioned composer presets with their
frozen manifests. The count is not restated here because it drifts: the packs
themselves are the list, and `/api/v1/scenario/packs` is how to read it. `GET /api/v1/scenario/profiles` returns the
role profiles used by the visual authoring flow. `POST
/api/v1/scenario/generate` accepts camelCase sites, infrastructure counts,
endpoint repeat counts, a domain, an SNMP community, and an attachment name. It
returns portable YAML plus a manifest containing device, network, and link
counts and deterministic SHA-256 fingerprints over the device names, the
networks, the links, and the ifTable truth a collector polls.

Generation is side-effect free: it does not replace the active configuration,
start a simulation, or save a draft. The returned `content` can be reviewed and
then stored through the draft API. The generate route requires a read-write
token and a CSRF token.

Scenario packs carry only the logical attachment name used by the composer.
They cannot select a host interface, attachment mode, or access VLAN. Starting
the generated draft still runs the normal preflight against the current
operator-approved attachment policy.

### Draft behavior timelines

`PUT /api/v1/library/drafts/{name}/behaviors` replaces the named draft's saved
traffic and fault phases. Send the current quoted draft revision in `If-Match`.
Each timeline supplies a name, `startOffsetMs`, `repeatCount`, and bounded
phases with `startOffsetMs`, `durationMs`, and reset behavior. The server
validates every device/interface target and rejects overlapping phases before
persisting a new revision.

When that draft starts, NIAC replays the same transitions through authoritative
device state on every simulation restart. `GET /api/v1/behaviors` reports the
current replay state, active phases, and applied/total transition counts.

### Draft topology mutations

`PATCH /api/v1/library/drafts/{name}/topology` applies one `add_device`,
`connect`, `disconnect`, `move_device`, or `update_link` operation. Send the
current quoted draft revision in `If-Match`; the response contains the updated
draft and its new `ETag`. A stale revision returns `412 Precondition Failed`.

Connections name both device/interface endpoints. NIAC refuses missing,
non-physical, or already occupied ports and writes reciprocal `trunk_ports`
records so the local and remote interface identities cannot drift. Link VLAN,
native VLAN, and FDB-only properties are updated on both endpoints together.

```json
{
  "operation": "connect",
  "link": {
    "source": { "device": "core-1", "interface": "Ethernet1/1" },
    "target": { "device": "dist-1", "interface": "Ethernet1/49" },
    "properties": { "vlans": [200, 210], "native_vlan": 200 }
  }
}
```

The route requires a read-write token and CSRF protection. It replaces only the
named draft; it does not apply or restart the running simulation.

## Session-scoped runtime

NIAC can run several scenarios at once, one per physical VLAN behind a shared
trunk. Each is a _session_ with its own ID. The unscoped runtime endpoints
(`/api/v1/topology`, `/api/v1/devices`, `/api/v1/stats`, …) report whichever
session is currently selected, which is ambiguous once more than one is
running. Address a session explicitly instead:

```text
GET /api/v1/sessions                        list running sessions
GET /api/v1/sessions/{id}/topology          that session's topology graph
GET /api/v1/sessions/{id}/devices           its device inventory
GET /api/v1/sessions/{id}/interfaces        its simulated devices' interfaces
GET /api/v1/sessions/{id}/segments          its VLAN segments
GET /api/v1/sessions/{id}/neighbors         its LLDP/CDP neighbours
GET /api/v1/sessions/{id}/stats             its live counters
GET /api/v1/sessions/{id}/runtime           its runtime summary
GET /api/v1/sessions/{id}/capture/export    its retained frames as pcapng
GET /api/v1/sessions/{id}/checkpoints       the checkpoints it can restore
POST /api/v1/sessions/{id}/checkpoints      save one under {"name": "..."}
POST /api/v1/sessions/{id}/checkpoints/restore  return to {"name": "..."}
DELETE /api/v1/sessions/{id}                stop that session
```

Naming a session that is not running returns `404 session_not_found` rather
than falling back to another session — a silent fallback is how one client
ends up driving a scenario it did not ask for.

`GET /api/v1/sessions/{id}/interfaces` lists the interfaces of the _simulated
devices_. The unscoped `/api/v1/interfaces` lists the **host's** capture NICs,
which is a different thing and is not session-scoped: it reports the same NICs
whatever session a client is looking at, and carries no `current_interface`.
To learn which NIC a session runs on, read that session's entry in
`GET /api/v1/sessions`.

Live streams take the session as a query parameter:
`/api/v1/stream/packets?sessionId={id}`.
A stream subscribed without `sessionId` receives the selected session only.

### Checkpointing a scenario

A checkpoint captures every device's running configuration and its active
faults, so an acceptance run can save a healthy scenario, inject a fault,
assert the consumer sees it, restore, and assert it is gone:

```bash
curl -sk -X POST -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-Csrf-Token: $CSRF" -H 'Content-Type: application/json' \
  -d '{"name":"healthy"}' \
  https://localhost:8445/api/v1/sessions/hospital/checkpoints
```

A save covers the whole session, and a restore refuses unless every device
holds that name — a half-restored scenario is a state that never existed.
Restoring an unknown name answers `404 checkpoint_not_found`.

Restoring returns device state, not the clock: a timeline action the scenario
has already consumed stays consumed, and the device's uptime keeps running.
Rewinding a scenario completely means stopping the session and starting it
again.

### Exporting a capture

`GET /api/v1/sessions/{id}/capture/export` answers with a pcapng file of the
frames that session has handled recently. The packet stream broadcasts and
keeps nothing, and truncates each frame to 256 bytes for the browser's hex
view; the export reads a bounded per-session ring of whole frames, so an
exchange that happened before anyone connected is still there.

Each frame carries the fabric decision as a pcapng packet comment
(`direction`, `vlan`, `ingress`, `egress`, `route`, `hop`, `rejection`) —
the part of a replay that no sniffer on the wire could reconstruct. Wireshark
shows it in the packet detail pane.

| Parameter | Meaning |
| --- | --- |
| `filter` | libpcap BPF expression applied to the retained frames. An expression that does not compile is a `400`, not a truncated file. |
| `last` | Keep only the newest N frames. |

`last` is applied first and `filter` second, so `?filter=arp&last=200` means
"the ARP frames among the newest 200", not "the newest 200 ARP frames".

The ring is bounded by frame count and by bytes, so the export is a recent
window rather than the whole run. `X-Niac-Frame-Count` reports how many frames
the file holds.

```bash
curl -sk -H "Authorization: Bearer $TOKEN" \
  "https://localhost:8445/api/v1/sessions/hospital/capture/export?filter=arp&last=500" \
  -o hospital.pcapng
```

`niac dump --session hospital --pcap hospital.pcapng` is the same read from
the CLI.

## Web UI

Navigate to `https://localhost:8445/`. A non-loopback listener requires an API token. The interface displays:

- Live stats (packets, errors, device counts)
- Device inventory table
- Historical runs pulled from BoltDB
- YAML editor that reads/writes the same config file used by the CLI
- An interactive topology graph (ForceGraph)
- Traffic injection controls for error injection and PCAP replay

### Configuration management

`GET /api/v1/config` returns:

```json
{
  "path": "/Users/alice/projects/niac/config.yaml",
  "filename": "config.yaml",
  "modified_at": "2025-01-07T22:18:24Z",
  "size_bytes": 18432,
  "device_count": 42,
  "content": "include_path: walks/\ndevices:\n  - name: core1\n    ..."
}
```

`PUT /api/v1/config` expects JSON `{ "content": "<yaml here>" }`. NIAC runs the same
validation pipeline as `niac validate` before swapping the on-disk file. On success the
response mirrors the GET payload and the Web UI automatically refreshes. Validation errors
(malformed YAML, missing fields, etc.) are surfaced with HTTP 400 and a descriptive message
so editors can fix issues without leaving the browser.

Saving a config immediately reloads the running simulator—no CLI restart required. If the
reload fails for any reason, the change is rejected and the previous configuration remains
active.

### Packet replay

`GET /api/v1/replay` returns:

```json
{
  "running": true,
  "file": "/captures/bgp-demo.pcap",
  "loop_ms": 0,
  "scale": 1.0,
  "started_at": "2025-01-07T22:45:00Z"
}
```

`POST /api/v1/replay` accepts:

```json
{
  "file": "/captures/bgp-demo.pcap",
  "loop_ms": 10000,
  "scale": 1.0,
  "data": "BASE64_ENCODED_PCAP"
}
```

The CLI's capture engine replays the PCAP immediately, optionally looping (`loop_ms`) or
time-scaling (`scale`). When `data` is provided, NIAC stores the uploaded PCAP in a temporary
directory so the server never needs direct access to the user's filesystem. If `data` is
omitted, the `file` path must exist on the host running NIAC. `DELETE /api/v1/replay` stops
the current playback and cleans up any uploaded file.

### File discovery

`GET /api/v1/files?kind=walks` returns `.walk` files located under the `include_path`
defined in the YAML config. `kind=pcaps` scans the directory that contains the active config
file for `.pcap`/`.pcapng` captures. Both responses include the absolute path, size, and
timestamp so the Web UI (or operators) can copy/paste the correct paths into configs or
replay requests without shelling into the host.

### Alerts

`GET /api/v1/alerts` exposes the current threshold + webhook:

```json
{
  "packets_threshold": 100000,
  "webhook_url": "https://hooks.example.com/niac"
}
```

`PUT /api/v1/alerts` expects the same payload to update the alert loop at runtime. Setting
`packets_threshold` to `0` disables alerts.

### Error Injection

NIAC supports runtime error injection for testing and simulation scenarios. The Web UI
provides a Traffic Injection page with controls for injecting errors on device interfaces.

`GET /api/v1/errors` returns available error types and currently active injections:

```json
{
  "available_types": [
    {
      "type": "FCS Errors",
      "description": "Frame Check Sequence errors (0-100)"
    },
    {
      "type": "Packet Discards",
      "description": "Dropped packets (0-100)"
    },
    {
      "type": "Interface Errors",
      "description": "Generic interface errors (0-100)"
    },
    {
      "type": "High Utilization",
      "description": "Interface bandwidth saturation (0-100%)"
    },
    {
      "type": "Link Down",
      "description": "Drop the link (non-zero takes the interface down)"
    }
  ],
  "info": "Fault injection updates SNMP interface counters",
  "targets": [
    {
      "device": "edge-switch",
      "address": "192.168.1.1",
      "interfaces": ["GigabitEthernet0/1"]
    }
  ],
  "active_errors": {
    "edge-switch": {
      "GigabitEthernet0/1": {
        "FCS Errors": 50,
        "Packet Discards": 25
      }
    }
  },
  "available_device_types": [
    {
      "type": "DHCP No Offer",
      "description": "DHCP server consumes the Discover and sends no Offer"
    },
    {
      "type": "DNS NXDOMAIN",
      "description": "DNS server answers every query with NXDOMAIN"
    },
    {
      "type": "DNS Timeout",
      "description": "DNS server answers nothing at all"
    },
    {
      "type": "Latency",
      "description": "Delay every ICMP echo reply (0-60000 ms)"
    }
  ],
  "device_targets": [
    {
      "device": "site-gateway",
      "address": "192.168.1.1",
      "errorTypes": ["DHCP No Offer", "DNS NXDOMAIN", "DNS Timeout", "Latency"]
    }
  ],
  "active_device_errors": {
    "site-gateway": {
      "DNS NXDOMAIN": 1
    }
  }
}
```

### Two fault axes

Faults come in two scopes and the error type decides which one a request lands
on. **Interface faults** perturb one port's SNMP telemetry and name an
interface. **Device faults** are service outcomes — the device's DHCP or DNS
server stops behaving — and name no interface, because the outage belongs to
the service rather than to a port. A request that mixes the two (an interface
with a device fault type, or a device fault type with no interface) is refused
with `validation_failed` rather than reinterpreted.

`device_targets` lists only the devices that actually run the affected
service, with the fault types each can serve; arming a fault on a device that
does not run that service is refused with `fault_service_absent`. Device
faults change no MIB object: a faulted server keeps its inventory, interfaces
and counters, and only its answers change.

`POST /api/v1/errors` injects an error on a specific device interface:

```json
{
  "device": "edge-switch",
  "interface": "GigabitEthernet0/1",
  "errorType": "FCS Errors",
  "value": 50
}
```

For FCS, discard, and interface errors, `value` is the counter increment rate
per second. For utilization, it is the percentage of the authored interface
speed applied to both input and output octet counters. Setting a fault to `0`
clears only that fault type.

`POST /api/v1/errors` with no `interface` arms a device fault:

```json
{
  "device": "site-gateway",
  "errorType": "DNS NXDOMAIN",
  "value": 1
}
```

A device fault is an outcome rather than a rate: any non-zero `value` arms it
and `0` clears it, the same way `Link Down` behaves on the interface axis.

`DELETE /api/v1/errors?device=edge-switch&interface=GigabitEthernet0/1&errorType=FCS%20Errors`
clears one fault. Omitting `errorType` clears every fault on that interface.
`DELETE /api/v1/errors?device=site-gateway` (no `interface`) clears that
device's service faults; adding `errorType` clears one of them.

`DELETE /api/v1/errors` (no query parameters) clears all active error injections.

Error injections persist until explicitly cleared or the simulation is explicitly
stopped and started again. Daemon restart recovery restores the last saved fault
state; orderly shutdown flushes final state, while a daemon process crash can lose changes since
the last completed periodic save. See [Daemon Simulation Recovery](DEPLOYMENT.md#daemon-simulation-recovery).
The Web UI displays active errors in real-time and allows clearing individual
interfaces or all errors at once.

FCS faults increment `dot3StatsFCSErrors` and `ifInErrors`; packet discards
increment `ifInDiscards` and `ifOutDiscards`; interface errors increment
`ifInErrors` and `ifOutErrors`; utilization advances the 32-bit and 64-bit
interface octet counters. All counters remain monotonic after a fault clears.

### Link and Fault Syslog

A device's enabled `syslog` configuration sends link transitions and fault
updates/clears to its configured UDP collectors. These are
[RFC 5424](https://www.rfc-editor.org/rfc/rfc5424.html) messages using the
`local0` facility and the device's authored hostname, not the daemon host name.

| Event | Severity | Message ID |
| --- | --- | --- |
| Operational link down | Warning | `LINK_DOWN` |
| Operational link up | Notice | `LINK_UP` |
| Interface or device fault set/changed | Warning | `FAULT_UPDATED` |
| Interface or device fault cleared | Notice | `FAULT_CLEARED` |

Description-only edits and other configuration events do not emit syslog.
The message includes the authoritative event version, kind and quoted target;
it does not infer an old event's fault value from today's state. Timestamps use
UTC with at most six fractional digits. Structured data is `-`; target text is
ASCII-escaped to preserve one message per event. Restored event history is not
resent after recovery, while new transitions continue normally.

## Alerts

Add `--alert-packets-threshold <n>` and optional `--alert-webhook https://...` to receive
webhook notifications when total packets exceed the threshold. Payload format:

```json
{
  "type": "packet_threshold",
  "threshold": 100000,
  "total": 152300,
  "interface": "en0",
  "triggeredAt": "2025-11-13T01:33:00Z"
}
```

## Monitoring & Metrics

NIAC-Go exposes comprehensive Prometheus-compatible metrics at `/metrics`. For complete
monitoring setup instructions, see the [Monitoring Guide](MONITORING.md).

### Quick Start

```bash
# Use -k only until the generated certificate is trusted.
curl -k -H "Authorization: Bearer $NIAC_API_TOKEN" https://localhost:8445/metrics

# Example metrics:
# niac_packets_sent_total 15234
# niac_packets_received_total 12890
# niac_devices_total 10
# niac_uptime_seconds 3600
# niac_memory_usage_bytes 45678912
# niac_goroutines_total 42
# ...
```

### Available Metric Categories

1. **Traffic Metrics**: Packet counts, device counts, error counts
2. **Protocol Metrics**: ARP, ICMP, DNS, DHCP, SNMP activity
3. **System Metrics**: Memory, goroutines, GC runs, uptime

### Grafana Dashboard

A pre-built Grafana dashboard is available at `docs/grafana-dashboard.json` with panels for:

- Overview (devices, packets, errors)
- System health (memory, goroutines, uptime)
- Protocol breakdown (traffic by protocol type)
- Runtime metrics (GC, memory trends)

Import the dashboard into Grafana after configuring Prometheus as a data source.

For detailed setup instructions, metric descriptions, and alert configuration, see the [Monitoring Guide](MONITORING.md).
