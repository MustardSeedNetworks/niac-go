# ADR 0009: Routed fabric separated from physical attachments

**Status:** Accepted (2026-07-20)

## Context

NIAC currently binds one protocol stack to one capture interface. ADR 0008 adds
independent physical VLAN engines, but those engines remain separate L2
networks. They do not model a router joining internal subnets, and their VLAN
identifiers are real 802.1Q tags.

The CyberScope demonstration needs a different capability: one tester-facing
network must expose a complete routed enterprise topology while every internal
subnet remains in memory. Only the tester-facing network may reach the physical
wire. The deployment currently uses VLAN 200, but another deployment could use
VLAN 2, VLAN 300, or any other isolated access VLAN.

Embedding a host interface or physical VLAN in a saved scenario would make the
scenario non-portable and could accidentally bind it to a production network.
Using one VLAN field for both internal topology and real tags would also make it
possible to leak a virtual network onto the wire.

## Decision

NIAC separates three concepts:

1. A **scenario** declares devices, virtual networks, routed interfaces, routes,
   DHCP ownership, and one logical external attachment.
2. A **deployment binding** maps that logical attachment to a host interface and
   an attachment policy when the operator preflights or starts the scenario.
3. A **physical attachment boundary** is the only component allowed to receive
   or serialize real Ethernet frames.

The scenario never stores a host interface name. Its virtual VLAN values are
topology metadata only and can never be passed to capture or packet
serialization.

The first deployment policies are:

- `direct`: a dedicated point-to-point interface. NIAC accepts and emits only
  untagged frames. The operator must affirm that the interface is dedicated.
- `access`: an externally enforced access VLAN. The binding requires any VLAN
  ID from 1 through 4094 for deployment identity, but NIAC still accepts and
  emits only untagged frames. The switch, hypervisor, or bridge owns tagging.

VLAN 200 is therefore CT 304 configuration, not product behavior. Changing the
binding to VLAN 2 does not change or regenerate the scenario.

`trunk` remains a future attachment policy. It will require an explicit physical
tag allowlist and is not part of the first CyberScope milestone.

The pure `internal/fabric` compiler consumes the parsed scenario plus a
deployment binding and produces immutable, canonical runtime topology. It owns
cross-network semantic validation and uses `net/netip` addresses and prefixes.
It performs no filesystem, capture, daemon, logging, or protocol work.

Preflight and start must use the same preparation and compiler path. Preflight
does not persist inline YAML, open capture, or change the active simulation.
Start recompiles server-side and never trusts a client-side preflight result.

The compiler and attachment boundary are separate. Compiler acceptance does not
enable routed packet forwarding; forwarding remains disabled until the shared
I/O boundary, network-scoped DHCP, and IPv4 forwarding phases pass their own
release gates.

## Consequences

**Positive**

- Saved scenarios are portable across hosts and isolated VLAN assignments.
- Physical exposure is explicit at start time and visible in the modern UI.
- Virtual topology identifiers cannot accidentally become 802.1Q tags.
- Preflight and start cannot drift onto different validation rules.
- The compiler can be tested without root access, libpcap, or network hardware.

**Negative**

- Routed scenarios require an explicit deployment binding before they can run.
- Existing wizard behavior must change because it currently starts the
  simulation before review.
- ADR 0008's incomplete capture ownership split must be completed before routed
  forwarding is enabled.

## Rejected alternatives

- **Store `eth0` and VLAN 200 in the scenario:** unsafe and non-portable.
- **Reuse `segments` as routed networks:** conflates independent physical VLAN
  engines with internally routed networks.
- **Represent internal networks with packet VLAN tags:** permits virtual state
  to escape through the physical serializer.
- **Validate only in the React UI:** the browser is not a safety boundary.
- **Create a separate routed-lab loader:** duplicates the canonical YAML loading
  and simulation preparation paths.

## References

- ADR 0003: dependency direction enforced by depguard.
- ADR 0008: multi-VLAN segment playback.
- `docs/design/2026-07-routed-virtual-lab-plan.md`.
