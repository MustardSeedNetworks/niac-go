# NIAC Architecture

NIAC is a network-device simulator built as one Go binary with an embedded
React/TypeScript web UI. The same runtime state feeds the CLI, daemon API,
browser UI, protocol responders, SNMP projections, discovery output, and
notifications.

## Runtime flow

```text
YAML or saved configuration
        |
        v
configuration loader and validation
        |
        +--> routed fabric compiler
        |
        v
daemon simulation transaction
        |
        +--> capture engine --> protocol stack --> wire responses
        |                         |
        |                         +--> authoritative device state
        |                         +--> SNMP and discovery projections
        |                         +--> SYSLOG and SNMP notifications
        |
        +--> HTTPS API and embedded web UI
```

Simulation replacement is transactional: NIAC prepares and validates the next
configuration before replacing the active runtime. Preflight compiles the same
configuration and physical attachment policy without opening capture or
persisting state.

## Package boundaries

| Area | Canonical package | Responsibility |
| --- | --- | --- |
| CLI | `cmd/niac` | Commands, automation output, daemon entry point |
| YAML authoring | `internal/converter` | Authoring DTO and field validation |
| Runtime config | `internal/config` | YAML loading and runtime structures |
| Capture and replay | `internal/capture` | Packet I/O, filtering, bounded playback |
| Routed topology | `internal/fabric` | Routed semantic compiler and physical bindings |
| Protocol runtime | `internal/protocols` | Packet handlers, forwarding, telemetry, notifications |
| SNMP | `internal/protocols/snmp` | MIB synthesis, walk projection, runtime tables |
| Device state | `internal/devicestate` | Concurrency-safe mutable state and injected faults |
| Device CLI | `internal/devicecli` | Stateful IOS-like command and SSH sessions |
| Daemon | `internal/daemon` | Simulation lifecycle and recovery |
| HTTPS API | `internal/api` | Routes, authorization, handlers, SSE, embedded UI |
| Content library | `internal/library`, `internal/content` | Managed configs, walks, captures, and bundles |
| Web UI | `ui/src` | Guided browser workflows over the API |

Before adding a helper or subsystem, check [`CODE_INDEX.md`](../CODE_INDEX.md).
It identifies the canonical implementation for shared capabilities.

## Configuration and routed labs

`internal/converter` defines the YAML authoring contract. The loader in
`internal/config` converts that document into runtime configuration. Routed
networks, interfaces, routes, DHCP ownership, and physical attachment policy
are compiled by `internal/fabric`.

Physical attachment approval is explicit. A browser request cannot grant
itself permission to bind an arbitrary interface or VLAN. Direct and access
mode behavior is enforced again at final wire egress so observer-visible
frames match the approved policy.

The committed schema at `docs/schemas/niac.schema.json` is generated from the
Go authoring type. CI regenerates it and fails on drift.

## Stateful simulation

Each simulated device has one authoritative state store. CLI mutations,
interface faults, routes, aliases, addresses, and lifecycle changes update that
store. SNMP, topology, forwarding, SSH, and notification code read the same
state rather than keeping independent mutable copies.

Interface fault telemetry is monotonic. Injected interface, discard, FCS, and
utilization faults advance IF-MIB, IF-X, and EtherLike-MIB counters while the
fault is active. Clearing a fault stops new increments without resetting the
counter baseline.

## API and browser UI

The daemon exposes one TLS listener on port 8445 by default. It serves:

- `/api/v1/` for the versioned API;
- `/ws` and SSE endpoints for live state;
- `/metrics` for Prometheus output;
- `/__version` for unauthenticated build verification;
- the embedded web UI for other browser paths.

There is no HTTP redirect listener. The web UI is built from `ui/src` directly
into `internal/api/ui`, embedded with `go:embed`, and released in the same
binary as the backend.

## Security boundaries

- Mutating routes require CSRF protection and the route’s declared scope.
- Authentication surfaces are rate-limited.
- Whole-configuration replacement and other destructive operations require
  admin authorization.
- Bearer credentials remain in browser memory and are never placed in URLs or
  persistent browser storage.
- Managed file operations are root-contained and reject traversal or symlink
  escape.
- CORS wildcard origins cannot be combined with credentials.
- The daemon has no usable default credential and requires operator setup.
- Output encoding, secret scanning, dependency scanning, and reachable Go
  vulnerability checks are release gates.

Route policy is declared in `internal/api/routes.go`; new handlers must enter
through that registry.

## Concurrency and lifecycle

Long-running components are owned by the simulation transaction and stop
through cancellation or explicit `Close`/`Stop` methods. Capture exit clears
running state, active resources are released once, and a later start creates a
fresh stack. Shared mutable state uses mutexes or atomics according to its
access pattern; tests run under the race detector.

## Extending NIAC

1. Add authoring fields in `internal/converter` and regenerate the schema.
2. Convert them once in `internal/config` or `internal/fabric`.
3. Extend the canonical protocol or state owner named in `CODE_INDEX.md`.
4. Add table-driven unit tests and routed integration coverage where relevant.
5. Expose the capability through the existing API/CLI/UI boundary only when a
   supported operator workflow requires it.
6. Run formatting, lint, race tests, browser tests, security scans, and the
   full build before release.

Architecture decisions and their current status are recorded in
[`docs/adr/`](adr/).
