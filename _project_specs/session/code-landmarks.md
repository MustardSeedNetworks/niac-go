# Code Landmarks

Quick reference for important code locations. Update when discovering key areas.

---

## Entry Points

| Location | Purpose |
|----------|---------|
| `cmd/niac/main.go` | CLI entry point |
| `internal/api/` | HTTP handlers |
| `ui/src/main.tsx` | Frontend entry |

## Core Modules

| Module | Path | Purpose |
|--------|------|---------|
| SNMP | `internal/protocols/snmp/` | SNMP operations |
| Network | `internal/network/` | Network scanning |
| Capture | `internal/capture/` | Packet capture |
| Device | `internal/device/` | Device management |

## Key Interfaces

| Interface | Location | Implementers |
|-----------|----------|--------------|
| Protocol | `internal/protocols/` | SNMP, etc. |

## Configuration

| File | Purpose |
|------|---------|
| `configs/` | Configuration files |
| `.golangci.yml` | Go linter config |
| `ui/biome.json` | Frontend linter |
| `pyproject.toml` | Python tooling |

## Patterns to Follow

- Protocol implementations: See existing in `internal/protocols/`
- API handlers: See existing handlers in `internal/api/`
- Frontend components: Check `ui/src/components/`
