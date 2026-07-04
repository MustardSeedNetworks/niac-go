# ADR 0008: Multi-VLAN playback — N sim engines behind one L2 demux

**Status:** Proposed (revised 2026-07-04)

## Context

NIAC replays one config against one interface. VLAN handling is a **global
mode**, not a first-class dimension:

- Each device carries an optional `vlan:` field, and every reply echoes the
  VLAN the request arrived on (the "reply-VLAN pattern", niac-go #876/#883).
- VLAN confinement (#865) restricts a running sim to a single tagged VLAN and
  drops untagged frames, so a demo trunk sees no rogue replies on the native
  VLAN.

Two limits follow from the single-config assumption:

1. **Tagged _or_ untagged, never both.** Confinement is all-or-nothing for one
   VLAN. A real bench often has an untagged management segment _and_ tagged
   VLANs on the same wire.
2. **One flat IP namespace.** A single device table means two devices cannot
   share an IP even if they live on different VLANs. Real networks routinely
   reuse `10.0.0.1` across isolated VLANs.

**The behavior we want** (owner, 2026-07-04):

- **Direct-attached (access cable, no tags):** NIAC serves its config untagged;
  the tester (a NetAlly CyberScope) discovers everything in it. This is today's
  flat behavior — unchanged.
- **On an 802.1Q trunk (native + 200 + 300 …):** each VLAN tag is served by its
  **own completely independent config** — its own sites, topology, and IP space.
  Native (untagged-on-wire) → the management config; VLAN 200 → today's demo
  config, **unchanged**; VLAN 300 → a separate config that may reuse 200's
  addresses because the two never see each other. One NIAC, one port, N
  independent networks a CyberScope discovers by roaming VLANs.

Scope is **parity with today, per VLAN** — no new per-VLAN capability, and
**no QinQ / stacked tags**. "Run today's sim N times, one per tag."

## Decision

Model it as **one shared L2 I/O front-end plus N independent sim engines, one
per VLAN tag** (untagged/native is one more engine). Not "N NIAC servers": the
naïve reading — N copies of today's `Stack` — breaks, because a `Stack` owns its
capture engine and N of them would each try to bind the NIC. The split is
between shared network I/O and per-VLAN sim logic.

### Why it is even possible

NIAC **synthesizes every simulated service as raw packets and binds no OS
sockets** — verified: there is no `net.Listen*` in `internal/protocols`; the
only real socket is one outbound `net.DialUDP` for the MapToIP proxy. So N
engines can all "answer SNMP on 161 / DNS on 53 / DHCP on 67" because none of
them actually owns those ports. A service that bound a real listener would break
this on port contention; none does.

### Shared L2 front-end (one instance)

- **Capture** on the real interface (today's `capture.Engine`, lifted out of
  `Stack`).
- **Ingress demux:** read the frame's 802.1Q tag; route to the engine bound to
  that VLAN. VLAN 0 / priority-tagged → untagged. A tag with no engine is
  **dropped** — that is confinement (#865), generalized.
- **Egress:** one send path that stamps each outgoing frame with the **sending
  engine's** VLAN tag.

### N sim engines (one per tag)

- Each engine is today's single-network sim — device table, protocol handlers,
  discovery tickers, DHCP leases, neighbour/FDB caches, stats — running
  **unchanged**, bound to exactly one VLAN (or untagged).
- The engine **owns a fixed egress VLAN.** Consequences: a reply is tagged
  correctly *by construction* (the request could only have reached this engine
  on this VLAN), and device-initiated **emissions** (LLDP/CDP/babble) are tagged
  the same way for free. This **retires the reply-VLAN bug class** (#876/#883,
  and the IPv6 gap #884) instead of threading `pkt.VLAN` through every handler
  forever.
- L2/L3 identity is **engine-local:** IPs and MACs may repeat across engines;
  ARP and FDB resolve *within* an engine, never across.

### Config

- Top-level `segments:` list, `{ tag: untagged | <vlan-id>, config }`.
- A bare `devices:` config = one untagged engine (**backward-compatible**;
  today's flat / direct-attach behavior). Today's VLAN-200 multi-site config
  becomes the `tag: 200` engine, byte-for-byte unchanged.
- Untagged is **always one engine** (the native-bound one). It is never a
  flattened merge of the tagged VLANs — flattening would collide their reused
  IPs.
- Reload targets one engine or all.

## Consequences

**Positive**

- Isolation is **structural**, not policed: an engine physically cannot see
  another's devices, so IP/MAC reuse across VLANs is safe.
- Existing, battle-tested handler code runs **unchanged** inside an engine.
- Fixed-egress-VLAN **eliminates a whole bug class** (untagged replies) by
  construction rather than by per-handler patches.

**Negative / cost**

- The real lift is **splitting `Stack`'s I/O ownership from its sim logic** —
  `Stack` currently owns `capture.Engine`. This is a genuine refactor, not a
  loop around the current struct.
- N × discovery goroutines and timers. Fine for a handful of VLANs; cap N — do
  not scale to 4094.

**Sequencing**

- Build the front-end split + engine model as its own workstream, after the
  current UDP parity work (which has shipped).
- **Do not patch the IPv6 reply-VLAN gap (#884) standalone** if this lands soon
  — the engine's fixed egress VLAN subsumes it; a separate patch would be
  throwaway.
- No QinQ. If stacked tags are ever wanted, that is a separate ADR.

## References

- niac-go #865 (VLAN confinement), #876 (reply-VLAN + MAC echo), #883 (DNS
  reply-VLAN), #884 (IPv6 reply-VLAN gap — subsumed here)
- Supersedes this ADR's prior "segments / per-segment device table" framing with
  the sharper "N engines behind one L2 demux" model.
- Tracking epic: #882.
