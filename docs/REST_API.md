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
| `GET` | `/metrics` | Prometheus metrics (`niac_packets_sent_total`, etc.) |

Include `Authorization: Bearer <token>` or append `?token=<token>` when authentication is enabled.

## Web UI

Navigate to `http://host:8080/` and supply the API token. The interface displays:

- Live stats (packets, errors, device counts)
- Device inventory table
- Historical runs pulled from BoltDB
- YAML editor that reads/writes the same config file used by the CLI/TUI
- An interactive topology graph (ForceGraph)

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
