# Design: Auto-generated baseline SNMP walks

**Issue:** #546 (part 2)
**Author:** Kris Armstrong
**Status:** Draft
**Date:** 2026-05-16

## Problem

A device the user adds in the Device editor (or via the API) starts
without an SNMP walk file attached. The simulated SNMP agent then
returns nothing useful — no `sysDescr`, no `ifTable`, no interface
counters — so any SNMP-aware client polling that device sees an
agent that's effectively dead.

The fix proposed in #546 part 2 is to let the user click **Generate
baseline walk** and have the daemon synthesise a sensible
walk file from the device's type + a vendor selector. The new file
then lands in the library and is auto-attached to the device.

## Goals

1. One click in the Device editor produces a walk that responds to
   the OIDs an operator would actually poll first: system group,
   `ifTable` basics, type-specific MIB tables (BRIDGE-MIB for
   switches, IP-MIB for routers, etc.).
2. The synthesised walk lands in `~/.niac/library/walks/` under a
   stable filename and the device's `walk_file` field is rewritten
   to point at it, so the next simulation run picks it up
   automatically.
3. Vendor selection produces plausible `sysDescr` + `sysObjectID`
   strings so the response matches what an operator would see from
   real gear of that type.
4. The endpoint is **idempotent** — running it twice on the same
   device overwrites the previous synthesised walk cleanly. A `.bak`
   is dropped next to the previous version so a regretted
   regeneration is recoverable.

## Non-goals

- Editing an existing walk line-by-line in the UI. That's
  `/walk-validator`'s job.
- Walk *capture* from real hardware. That's `snmpwalk` + the
  validator path.
- Generating walks for every MIB module the device might support.
  The point is a usable baseline, not exhaustive coverage. Operators
  who want vendor MIBs should still drop their own walk files.

## Endpoint

```
POST /api/v1/devices/{hostname}/synthesize-walk
Content-Type: application/json

{
  "vendor": "cisco-ios" | "junos" | "arista-eos" | "generic",
  "interfaceCount": 24,    // optional; defaults below
  "community": "public",   // optional; defaults to existing device community
  "force": false           // optional; overwrite existing walk without .bak prompt
}
```

Response:

```
201 Created
{
  "walkPath": "synthesized/router-3.walk",
  "oidCount": 142,
  "sizeBytes": 8420,
  "previousWalkBacked": "synthesized/router-3.walk.bak"  // omitted on first generation
}
```

The `walkPath` is library-relative so it slots straight into the
device's `walk_file` field (matching the picker integrations
from #556).

### Why a POST, not idempotent PUT

The endpoint mutates two things — writes a walk to the library
*and* rewrites the device's YAML to attach it. A `PUT` would imply
the body is the full desired state of an existing resource, which
doesn't model "go generate one for me." POST + 201 matches the
"create a derived resource" semantics already used elsewhere (e.g.
`POST /api/v1/devices/.../clone`).

## Vendor profiles

Each vendor profile is a small Go struct keyed off the vendor name.
Defines the sysDescr template, sysObjectID prefix, and any
vendor-specific quirks (e.g. Cisco's `ciscoLwapp*` for APs).

| Vendor key      | sysDescr template                                          | sysObjectID prefix    | Notes |
|-----------------|------------------------------------------------------------|-----------------------|-------|
| `cisco-ios`     | `Cisco IOS Software, %s Software, Version 15.7(3)M3, ...`  | `.1.3.6.1.4.1.9`      | %s = platform string from type |
| `junos`         | `Juniper Networks, Inc. %s, kernel JUNOS 21.4R3.15, ...`   | `.1.3.6.1.4.1.2636`   |  |
| `arista-eos`    | `Arista Networks EOS version 4.30.4M`                      | `.1.3.6.1.4.1.30065`  |  |
| `generic`       | `Linux %s 6.1.0 #1 SMP x86_64 GNU/Linux`                   | `.1.3.6.1.4.1.8072`   | Net-SNMP enterprise; safe default |

Platform strings substituted into sysDescr come from the device
type → platform map:

| device.type     | Cisco platform | Juniper platform | Arista platform | Generic |
|-----------------|----------------|------------------|-----------------|---------|
| `switch`        | `Catalyst 3850` | `EX4300`         | `DCS-7050SX`    | `Linux switch` |
| `router`        | `ISR 4451-X`    | `MX204`          | `DCS-7280SR`    | `Linux router` |
| `firewall`      | `ASA 5525-X`    | `SRX340`         | `(none — n/a)`  | `Linux firewall` |
| `access_point`  | `Aironet 2802i` | `(none — n/a)`   | `(none — n/a)`  | `Linux AP` |
| `server` / `host` | `(skipped — use generic)` |              |                 | `Linux host` |
| `voip_phone`    | `IP Phone 7945` | `(none — n/a)`   | `(none — n/a)`  | `Generic phone` |
| `printer`       | `(skipped — use generic)` |              |                 | `Linux printer` |

When the operator picks a vendor that doesn't apply to the device's
type (e.g. Juniper + access_point), the daemon falls back to
`generic` with a flash message saying so. Doesn't error.

## OID coverage per type

All types include the system group (`1.3.6.1.2.1.1.*`) populated
from device fields. The rest is type-driven, mirroring
`internal/protocols/snmp/mib_*.go` (the responder already knows how
to *serve* these MIBs; this PR synthesizes the *table contents* the
responder would have read off a walk file).

| device.type     | System | IF-MIB | IP-MIB | BRIDGE-MIB | LLDP-MIB | ipForwarding |
|-----------------|:------:|:------:|:------:|:----------:|:--------:|:------------:|
| `switch`        | ✓      | ✓      |        | ✓          | ✓        |              |
| `router`        | ✓      | ✓      | ✓      |            | ✓        | 1 (enabled)  |
| `firewall`      | ✓      | ✓      | ✓      |            |          | 1            |
| `access_point`  | ✓      | ✓      |        |            | ✓        |              |
| `host` / `server` | ✓    | ✓ (single if) |  |            |          |              |
| `voip_phone`    | ✓      | ✓ (single if) |  |            | ✓        |              |
| `printer`       | ✓      | ✓ (single if) |  |            |          |              |

`ifTable` row count = `interfaceCount` request field, or the type
default:

| device.type     | Default interfaceCount |
|-----------------|-----------------------:|
| `switch`        |                     24 |
| `router`        |                      4 |
| `firewall`      |                      4 |
| `access_point`  |                      2 (radio0, radio1) |
| `host` / `server` |                    1 |
| `voip_phone`    |                      2 (eth0, voice) |
| `printer`       |                      1 |

Interface names follow vendor conventions where they're well-known
(Cisco → `GigabitEthernet0/0`, `Vlan1`; Juniper → `ge-0/0/0`; Arista
→ `Ethernet1`; Generic → `eth0`, `eth1`).

`ifSpeed` defaults to 1 Gbps; the operator can edit after generation.

`ipAddrTable` for routers / firewalls / hosts is populated from
the device's existing `ip` + `additional_ips` fields. For switches
it's omitted (L2 device).

## Storage

Generated walks land at:

```
~/.niac/library/walks/synthesized/{hostname}.walk
```

The `synthesized/` subdir makes operator-curated walks visible at a
glance vs auto-generated ones in the Walks library page (PR added
in #555 — the Source column will show `user` for both today; a
future enhancement could add a `synthesized` source badge).

On regeneration, the existing file is moved to `{hostname}.walk.bak`
before the new one is written. The `.bak` is removed on the *next*
regeneration so we never keep more than one rollback step.

## Frontend changes

Within the SNMP section (`ui/src/components/device-editor/SnmpSection.tsx`):

- New **Generate baseline walk** button, visible only when
  `device.walk_file` is empty.
- Click opens a small modal: vendor selector (4 options), optional
  interface count, "Generate" button.
- On success: walk is auto-attached (the response's `walkPath`
  becomes the device's `walk_file` via the existing `updateField`),
  modal closes, the section refreshes and now shows the walk like
  any other.

No changes needed on the walks-picker side: the synthesised file is
just another row in the library walks page.

## Backend implementation sketch

New file `internal/api/handlers_synthesize_walk.go`:

```go
func (s *Server) handleSynthesizeWalk(w http.ResponseWriter, r *http.Request) {
    if !s.libraryReady() { ... 503 ... }
    hostname := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/")
    hostname  = strings.TrimSuffix(hostname, "/synthesize-walk")
    // ... validate hostname (alpha+digits, no path traversal) ...

    var req SynthesizeWalkRequest
    if err := json.NewDecoder(r.Body).Decode(&req); ... { ... 400 ... }

    dev, err := s.config.FindDevice(hostname)
    if errors.Is(err, config.ErrDeviceNotFound) { ... 404 ... }

    profile := synth.PickProfile(req.Vendor, dev.Type)
    walk := synth.Build(profile, dev, req)

    relPath := filepath.Join("synthesized", hostname+".walk")
    if err := s.library.WriteFile(library.KindWalks, relPath, walk); err != nil { ... 500 ... }

    // Attach to device and persist YAML.
    dev.WalkFile = relPath
    if err := s.config.SaveDevice(dev); err != nil { ... 500 ... }

    s.writeJSON(w, SynthesizeWalkResponse{...})
}
```

New package `internal/protocols/snmp/synth/`:
- `profiles.go` — vendor + platform tables (the data in this doc)
- `build.go` — `Build(profile, device, req) []byte` that emits
  walk-format lines for system / ifTable / etc.
- `formats.go` — small helpers for the line format (`oid type value`).

Tests in `internal/protocols/snmp/synth/build_test.go`:
- Each (vendor, type) combination produces a parseable walk
- `interfaceCount` controls ifTable row count
- Switches don't get `ipAddrTable`; routers do
- `ipForwarding` is 1 for router/firewall, 0 otherwise
- Walk validator (existing `niac analyze-walk` code path) accepts
  every generated file

## Out of scope (filed as follow-ups when this lands)

- Per-vendor extended MIB tables (e.g. Cisco's `CISCO-MEMORY-POOL-MIB`)
  — needs more vendor MIB content; defer until operators ask
- Synthesizing port-channel / VLAN aware bridge tables on switches
  past the bare minimum
- A bulk "synthesize walks for every device missing one" endpoint
  — useful but easy to add once the single-device path exists

## Open questions for review

1. **`.bak` retention.** Keep one rollback step (current proposal)
   or keep N? One is simpler; N risks library bloat.
2. **Vendor `none → n/a` fallback.** Should picking a vendor that
   doesn't apply to the device type return a 422 instead of
   silently falling back to generic? 422 is more explicit but adds
   UI friction.
3. **`synthesized/` subdir vs flat library.** The subdir helps
   operators distinguish; a flat layout matches the existing
   library naming convention (everything goes at the root).
4. **Idempotency key.** Should the request body have a
   `clientRequestId` so a UI retry doesn't double-`.bak`? Today's
   POST-once semantics work; this is hardening for later.

Once these are resolved, the implementation is ~half a day of code
plus tests. Tracking continues on #546.
