# ADR 0008: Multi-VLAN segment playback (tagged + untagged, per-segment config)

**Status:** Proposed (2026-07-04)

## Context

NIAC replays one config against one interface. VLAN handling is a **global
mode**, not a first-class dimension:

- Each device carries an optional `vlan:` field, and every reply echoes the
  VLAN the request arrived on (the "reply-VLAN pattern" established in
  niac-go #876). This makes a single device answer correctly whether it is
  reached tagged or untagged.
- VLAN confinement (niac-go #865) restricts a running sim to a single tagged
  VLAN and drops untagged frames, so a demo trunk does not see rogue replies on
  the native VLAN.

Two limitations follow from the single-config assumption:

1. **You pick tagged _or_ untagged, not both.** Confinement is all-or-nothing
   for one VLAN. A real bench often has an untagged management segment _and_
   tagged VLANs on the same wire; NIAC cannot present both at once.
2. **One flat IP namespace.** Because there is a single device table, two
   devices cannot share an IP even if they live on different VLANs. Real
   networks routinely reuse `10.0.0.1` across isolated VLANs.

The demo goal that motivates this: a NetAlly CyberScope roaming an 802.1Q trunk
should discover **several independent "sites"** — one per VLAN — from a single
NIAC instance, plus an untagged segment, each with its own topology and IP
space. The owner also wants this reachable as a UI toggle (play config A
untagged, config B on VLAN 200, config C on VLAN 300) without hand-editing YAML.

## Decision

Introduce a **Segment** as the top-level playback unit:

```
Segment = { tag: untagged | <vlan-id>, config: <device set> }
```

A running sim owns an ordered list of segments. The receive path **demuxes each
ingress frame by its VLAN tag** to the matching segment, and dispatches to that
segment's device table. Everything downstream (handlers, replies) operates
within the selected segment.

### Config-first, UI second

The binding lives in YAML as a top-level `segments:` list so it is reproducible
and CLI-testable; the Web UI toggle is a thin editor/enable layer over it, never
a UI-only concept:

```yaml
segments:
  - tag: untagged
    config: mgmt-segment.yaml      # or an inline `devices:` block
  - tag: 200
    config: site-evt.yaml
  - tag: 300
    config: site-cos.yaml
```

"Untagged" is simply a segment with no tag — this is how NIAC finally supports
tagged **and** untagged simultaneously rather than choosing one.

### Backward compatibility

A bare top-level `devices:` config (today's format) is treated as a single
implicit segment. Existing configs — including the current multi-site
`demo-multisite.yaml` running on VLAN 200 — keep working unchanged; VLAN 200
stays VLAN 200. A VLAN-300 sibling site is added by appending a second segment,
not by rewriting the first.

### Confinement generalizes

VLAN confinement (#865) becomes the degenerate one-segment case of a single
rule: **drop any ingress frame whose VLAN has no segment.** Same safety, now
multi-tenant.

### What is reused vs. new

- **Reused (already built):** the reply-VLAN echo (#876) already tags every
  reply to the segment's VLAN — the *tagging* half is free. Per-device `vlan:`
  and confinement (#865) are the seed of the model.
- **New:** (a) an ingress demux keyed on `pkt.VLAN` that selects a segment
  before handler dispatch; (b) per-segment device tables so IP/topology lookups
  are scoped to a segment instead of global; (c) the `segments:` config schema
  and its UI editor.

## Consequences

**Positive**

- One NIAC instance presents multiple isolated VLAN "sites" plus an untagged
  segment on a single trunk — the target CyberScope demo.
- Overlapping IP spaces across VLANs become legal (per-segment namespaces).
- Tagged and untagged environments are both first-class, addressing the
  standing gap where NIAC handled each poorly.

**Negative / cost**

- Touches the packet-ingress path and the single-config assumption baked into
  the stack — a focused refactor, not a drop-in. Device lookups move from global
  to per-segment; the stack holds N device tables instead of one.
- Reload, stats, topology, and the discovery caches all become
  segment-scoped and must be audited for the global-state assumption.
- The Web UI grows a segment manager (list, per-segment VLAN tag, config
  picker, enable toggle).

**Sequencing**

Land after the current UDP parity work (reflector #879, port-unreachable #878).
This is its own workstream with its own PR series; it is not folded into a
feature PR. The DHCP/service realism work (see the "beef up the services"
tracking issue) is independent and can proceed in parallel.

## References

- niac-go #865 — VLAN confinement (drop untagged on a tagged sim)
- niac-go #876 — reply-VLAN + source-MAC echo pattern
- `docs/TOPOLOGY_GUIDE.md` — trunk_ports / VLAN topology declarations
