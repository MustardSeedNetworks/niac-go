# REST API & Web UI

NIAC exposes a REST API, Prometheus metrics endpoint, and bundled Web UI over its HTTPS daemon listener.

## Enabling the API

```bash
export NIAC_API_TOKEN="$(openssl rand -base64 32)"
niac daemon
```

Flags:

| Flag | Description |
|------|-------------|
| `--listen` | HTTPS address for REST API & Web UI (default: `127.0.0.1:8445`) |
| `--api-token` | Daemon bearer token (prefer `NIAC_API_TOKEN`) |
| `--cert-dir` | Directory containing the generated HTTPS certificate and key |
| `--storage` | BoltDB location for run history (default: `~/.niac/niac.db`, set to `disabled` to opt out) |

Prometheus metrics share the daemon's HTTPS listener at `/metrics`; there is no separate metrics listener.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/stats` | Live packet counters, interface info, NIAC version |
| `GET` | `/api/v1/stream/stats` | Server-sent live statistics stream |
| `GET` | `/api/v1/devices` | Device inventory (type, IPs, enabled protocols) |
| `GET` | `/api/v1/history` | Recent runs persisted to BoltDB |
| `GET` | `/api/v1/config` | Active YAML config plus file metadata |
| `PUT` | `/api/v1/config` | Validate + persist new YAML config content |
| `GET` | `/api/v1/replay` | Current PCAP replay status |
| `POST`/`DELETE` | `/api/v1/replay` | Start or stop packet replay |
| `GET` | `/api/v1/alerts` | Current alert threshold + webhook |
| `PUT` | `/api/v1/alerts` | Update alert threshold/webhook |
| `GET` | `/api/v1/files?kind=walks|pcaps` | List available SNMP walk or PCAP files |
| `GET` | `/api/v1/topology` | Simple topology graph derived from configuration |
| `GET` | `/api/v1/version` | Version information |
| `GET` | `/api/v1/errors` | Available error types and active error injections |
| `POST` | `/api/v1/errors` | Inject network errors on device interfaces |
| `DELETE` | `/api/v1/errors` | Clear specific or all error injections |
| `GET` | `/metrics` | Prometheus metrics endpoint (see [Monitoring Guide](MONITORING.md)) |

Include `Authorization: Bearer <token>` or append `?token=<token>` when authentication is enabled.

The statistics stream publishes once per second while a simulation is active
and at least one client is subscribed. Each `stats` event carries the same
authoritative snapshot shape as `GET /api/v1/stats`.

## Web UI

Navigate to `https://localhost:8445/`. A non-loopback listener requires an API token. The interface displays:

- Live stats (packets, errors, device counts)
- Device inventory table
- Historical runs pulled from BoltDB
- YAML editor that reads/writes the same config file used by the CLI/TUI
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

`PUT /api/v1/config` expects JSON `{ "content": "<yaml here>" }`. NIAC runs the same validation pipeline as `niac validate` before swapping the on-disk file. On success the response mirrors the GET payload and the Web UI automatically refreshes. Validation errors (malformed YAML, missing fields, etc.) are surfaced with HTTP 400 and a descriptive message so editors can fix issues without leaving the browser.

Saving a config immediately reloads the running simulator—no CLI restart required. If the reload fails for any reason, the change is rejected and the previous configuration remains active.

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

The CLI's capture engine replays the PCAP immediately, optionally looping (`loop_ms`) or time-scaling (`scale`). When `data` is provided, NIAC stores the uploaded PCAP in a temporary directory so the server never needs direct access to the user's filesystem. If `data` is omitted, the `file` path must exist on the host running NIAC. `DELETE /api/v1/replay` stops the current playback and cleans up any uploaded file.

### File discovery

`GET /api/v1/files?kind=walks` returns `.walk` files located under the `include_path` defined in the YAML config. `kind=pcaps` scans the directory that contains the active config file for `.pcap`/`.pcapng` captures. Both responses include the absolute path, size, and timestamp so the Web UI (or operators) can copy/paste the correct paths into configs or replay requests without shelling into the host.

### Alerts

`GET /api/v1/alerts` exposes the current threshold + webhook:

```json
{
  "packets_threshold": 100000,
  "webhook_url": "https://hooks.example.com/niac"
}
```

`PUT /api/v1/alerts` expects the same payload to update the alert loop at runtime. Setting `packets_threshold` to `0` disables alerts.

### Error Injection

NIAC supports runtime error injection for testing and simulation scenarios. The Web UI provides a Traffic Injection page with controls for injecting errors on device interfaces.

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
  }
}
```

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

`DELETE /api/v1/errors?device=edge-switch&interface=GigabitEthernet0/1&errorType=FCS%20Errors`
clears one fault. Omitting `errorType` clears every fault on that interface.

`DELETE /api/v1/errors` (no query parameters) clears all active error injections.

Error injections persist until explicitly cleared or NIAC is restarted. The Web UI displays active errors in real-time and allows clearing individual interfaces or all errors at once.

FCS faults increment `dot3StatsFCSErrors` and `ifInErrors`; packet discards
increment `ifInDiscards` and `ifOutDiscards`; interface errors increment
`ifInErrors` and `ifOutErrors`; utilization advances the 32-bit and 64-bit
interface octet counters. All counters remain monotonic after a fault clears.

## Alerts

Add `--alert-packets-threshold <n>` and optional `--alert-webhook https://...` to receive webhook notifications when total packets exceed the threshold. Payload format:

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

NIAC-Go exposes comprehensive Prometheus-compatible metrics at `/metrics`. For complete monitoring setup instructions, see the [Monitoring Guide](MONITORING.md).

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
