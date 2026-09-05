# NIAC Documentation

NIAC is a source-available network device simulator distributed under the
[Business Source License 1.1](../LICENSE).

## Start Here

- [Project overview and quick start](../README.md)
- [Deployment](DEPLOYMENT.md)
- [Distribution and release validation](DISTRIBUTION.md)
- [CLI reference](CLI_REFERENCE.md)
- [REST API](REST_API.md)
- [Web UI](WEBUI.md)
- [Monitoring](MONITORING.md)
- [Configuration schema](schemas/niac.schema.json)
- [Pre-1.0 roadmap](ROADMAP.md)
- [Master closeout plan](design/2026-07-niac-master-closeout-plan.md)
- [Compatibility policy](BREAKING_CHANGES.md)

## API Modes

The installed daemon is HTTPS-only and listens on `127.0.0.1:8445` by default.
Its API base URL is `https://localhost:8445/api/v1/`; use `-k` only when testing
with the generated self-signed development certificate.

NIAC exposes the API, Web UI, and Prometheus metrics only through the daemon's
HTTPS listener. A `niac daemon --once` run is foreground-only and opens no
network service.

Authenticated requests use a bearer token. Mutating requests also require the
CSRF token returned by `/api/v1/csrf-token`.

## Development

- [Architecture](ARCHITECTURE.md)
- [Architecture decisions](adr/)
- [Contributing](../CONTRIBUTING.md)
- [Performance](PERFORMANCE.md)
- [SNMP walk files](SNMP_WALKS.md)

Run `niac <command> --help` for the command surface shipped by the current
binary. Generated help is authoritative when an older example disagrees.
