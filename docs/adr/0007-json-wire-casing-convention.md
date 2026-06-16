# ADR 0007: JSON wire-casing convention (camelCase API, no exceptions)

**Status:** Accepted (2026-06-16)

## Context

The Mustard Seed Networks fleet standardizes JSON API casing on seed's ADR-0010
(revised 2026-06-14), which establishes a pure boundary-mapping model: the JSON
wire layer speaks camelCase exclusively; internal Go and config-file layers keep
their own idiomatic casing (snake\_case struct tags are a Go convention in Go
source, snake\_case in YAML config files).

niac currently **diverges** from this standard. A codebase audit on 2026-06-14
found **74 snake\_case `json:"..."` tags** in `internal/api` (non-test files):

```
grep -rnoE 'json:"[a-z][a-z0-9]*_[a-z0-9_]+[^"]*"' internal/api \
  --include='*.go' | grep -v _test | wc -l
# → 74
```

These tags span wire-facing response structs, request-body structs, and the
device-editor schema descriptors (`schema*.go`).

### The YAML config layer is decoupled from the API json tags

niac's simulation config files (YAML) are built by a **separate serialization
path** that does not touch Go struct tags at all. `serializeDeviceToYAML` in
`internal/api/devices_helpers.go` constructs a plain `map[string]any` with
literal string keys (`"name"`, `"type"`, `"mac"`, `"snmp_agent"`,
`"walk_file"`, …) sourced directly from `config.Device` field values. The YAML
output is therefore determined by those literal strings, not by any `json:` tag.

Consequence: **camelCasing the API wire structs has zero effect on the YAML
simulation config format.** The two layers are independent.

### SNMP and protocol field data niac simulates

niac *generates* SNMP/protocol data on the wire — it does not parse an external
SNMP agent's JSON output. The simulated data is therefore niac's own
representation, and the same camelCase rule applies to it on the wire. Contrast
this with, e.g., iperf3 (`internal/protocols`), which parses an external tool's
`-json` output that arrives with its own casing — those internal parsing structs
may retain snake\_case *internally* and must map to camelCase before emission.

## Decision

1. **Every field niac's API emits or accepts is camelCase.** There are no
   wire-level snake\_case exceptions and no allow-list or grandfather baseline.
   This applies to all routes, all response bodies, all request bodies, and all
   SSE event payloads.

2. **snake\_case is only used off the wire:**
   - YAML simulation config files (unchanged — decoupled, see above).
   - SQL column names and migration files.
   - Internal Go struct fields (idiomatic Go).
   - Internal adapter structs that parse external tool output (e.g., iperf3
     `-json`), which must map to camelCase before the data reaches a wire
     response.

3. **SNMP and protocol data niac simulates** is niac's own wire representation
   and must be camelCase.

4. **Go source files** remain idiomatic Go (snake\_case is not used for Go
   identifiers, which follow PascalCase/camelCase; this rule only concerns
   `json:` struct tags and map keys in wire-serialization paths).

5. Once the migration described in **Consequences** is complete, a
   `scripts/check-json-casing.sh` gate with an **empty baseline** (zero
   allowed snake\_case wire tags in `internal/api` non-test files) will be
   added to CI. Status flips to **Accepted** at that point.

## Consequences

The tracked migration — the **W7 workstream** — has **landed** (PR #816: all 74
`internal/api` wire tags + the device-editor schema descriptors + the affected
frontend files + Go fixtures migrated to camelCase), and the
`scripts/check-json-casing.sh` gate is wired into CI with a **verified-empty
baseline** (0 non-comment entries). With both conditions met this ADR is now
Accepted. The scope that was migrated:

| Area | Scope |
|---|---|
| `internal/api` wire structs | Rename all 74 snake\_case `json:"..."` tags to camelCase |
| `internal/config` wire structs | Fields of `config.Device` and siblings that appear on the wire (not in YAML marshalling) |
| `schema*.go` (device-editor schema descriptors) | Update field name literals to camelCase to match renamed wire fields |
| Frontend (`ui/src`) | ~6 files that read or write the affected API fields |
| Go test fixtures | JSON literals in `*_test.go` files that assert on wire shape |
| CI gate | Add `scripts/check-json-casing.sh` with empty baseline after above changes land |

**YAML simulation configs are untouched** — the YAML serialization path
(`serializeDeviceToYAML`, `config.MarshalConfigYAML`) uses literal string keys
and is independent of `json:` struct tags.

**No API version bump is required before v1.0.0** — niac is pre-alpha with no
backwards-compatibility obligation (see project CLAUDE.md: "Until v1.0.0 there
is NO backwards compatibility … Delete and replace; migrate every caller in the
same change").

## References

- seed ADR-0010 — JSON wire-casing convention (revised 2026-06-14, pure
  boundary-mapping model; seed's authoritative doc for the fleet standard)
- niac ADR-0006 — `internal/api` sub-package decomposition (2026-06-08);
  the decomposition boundaries inform which sub-packages own wire structs
- `msn-docs-internal/05-Engineering/JSON_WIRE_CASING.md` — fleet-wide
  engineering standard
- `internal/api/devices_helpers.go:serializeDeviceToYAML` — confirms YAML
  decoupling (literal map keys, no struct-tag dependency)
