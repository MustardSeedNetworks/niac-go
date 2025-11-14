# REST API & Web UI

NIAC now exposes a lightweight REST API, Prometheus metrics endpoint, and bundled Web UI for day-to-day monitoring.

## Enabling the API

```bash
niac --api-listen :8080 --api-token supersecret en0 config.yaml
```

Flags:

| Flag | Description |
|------|-------------|
| `--api-listen` | Address for REST API & Web UI (e.g., `:8080`) |
| `--api-token` | Optional bearer token required for requests |
| `--metrics-listen` | Optional dedicated metrics listener |
| `--storage-path` | BoltDB location for run history (default: `~/.niac/niac.db`, set to `disabled` to opt out) |

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/stats` | Live packet counters, interface info, NIAC version |
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

## Web UI

Navigate to `http://host:8080/` and supply the API token. The interface displays:

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
      "type": "fcs_errors",
      "description": "Frame Check Sequence errors (Layer 2 corruption)"
    },
    {
      "type": "packet_discards",
      "description": "Packets dropped due to buffer overflow"
    },
    {
      "type": "interface_errors",
      "description": "Generic interface input/output errors"
    },
    {
      "type": "high_utilization",
      "description": "Interface bandwidth saturation"
    },
    {
      "type": "high_cpu",
      "description": "Elevated CPU usage on device"
    },
    {
      "type": "high_memory",
      "description": "Memory pressure on device"
    },
    {
      "type": "high_disk",
      "description": "Disk space exhaustion"
    }
  ],
  "info": "Error injection allows testing monitoring and alerting systems",
  "active_errors": {
    "192.168.1.1": {
      "GigabitEthernet0/1": {
        "fcs_errors": 50,
        "packet_discards": 25
      }
    }
  }
}
```

`POST /api/v1/errors` injects an error on a specific device interface:

```json
{
  "device_ip": "192.168.1.1",
  "interface": "GigabitEthernet0/1",
  "error_type": "fcs_errors",
  "value": 50
}
```

The `value` field represents error severity (0-100), where:
- 0 = No errors
- 50 = Moderate error rate
- 100 = Maximum error injection

`DELETE /api/v1/errors?device_ip=192.168.1.1&interface=GigabitEthernet0/1` clears all errors on a specific interface.

`DELETE /api/v1/errors` (no query parameters) clears all active error injections.

Error injections persist until explicitly cleared or NIAC is restarted. The Web UI displays active errors in real-time and allows clearing individual interfaces or all errors at once.

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
# View raw metrics
curl http://localhost:8080/metrics

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
