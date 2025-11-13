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
| `GET` | `/api/v1/topology` | Simple topology graph derived from configuration |
| `GET` | `/api/v1/version` | Version information |
| `GET` | `/metrics` | Prometheus metrics (`niac_packets_sent_total`, etc.) |

Include `Authorization: Bearer <token>` or append `?token=<token>` when authentication is enabled.

## Web UI

Navigate to `http://host:8080/` and supply the API token. The interface displays:

- Live stats (packets, errors, device counts)
- Device inventory table
- Historical runs pulled from BoltDB
- An interactive topology graph (ForceGraph)

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
