# Plan: Routed virtual labs behind one isolated attachment

**Status:** In implementation — core Phases 0-5 complete; Phase 6 next

**Date:** 2026-07-20

**Target:** Post-0.95 development

**Related:** ADR 0008 multi-VLAN segment playback

## Implementation progress

| Phase | Status | Evidence |
| --- | --- | --- |
| 0 — contract | Complete | ADR 0009, routed schema, isolation and packet fixtures |
| 1 — compiler | Complete | Immutable topology compiler and validation tests |
| 2-4 — fabric and IPv4 | Complete | Attachment-aware egress, scoped services, forwarding, fragmentation, ICMP, and routed response tests |
| 5 — management plane | Core complete | Shared device state, first IOS-like CLI profile, packet-backed SSH, SNMP projection, configuration lifecycle, SYSLOG/SNMP state notifications, and Pro gate |
| 6 — observability | Next | API/UI forwarding counters and packet-inspector route decisions remain |
| 7 — hardware acceptance | Pending | Reference eight-subnet lab and 24-hour isolation run remain |

## Outcome

NIAC should present a realistic routed, multi-subnet lab to a tester connected
through one explicitly allowed physical attachment. A CyberScope attached to
physical VLAN 200 should receive an address on `10.10.200.0/24`, use a simulated
gateway, and reach devices in the lab's internal subnets. NIAC must never emit
those internal VLANs, DHCP offers, ARP replies, or broadcasts onto the real
network.

The same lab must work when a tester is cabled directly to a dedicated NIAC
interface with no switch and no VLAN tag. Direct mode accepts untagged frames,
maps them to the lab-access network, and returns untagged frames. It does not
require or invent a VLAN.

The initial outcome is IPv4 static routing. OSPF, BGP, IPv6 forwarding, NAT,
firewalls, and arbitrary host bridging are later work.

## Goal statement

One NIAC process and one physical data interface should convincingly represent
an entire enterprise lab to an external tester. The tester interacts with normal
network protocols and device management surfaces; NIAC maintains the resulting
state and produces a coherent routed topology. The simulated internal links and
VLANs never need to exist on the physical wire.

This is a behavioral simulator, not an image emulator. It does not boot IOS,
JUNOS, or a virtual appliance. It implements the network and management
behaviors required by test, demonstration, reproduction, and training workflows.

## Feature and market gates

### Feature verdict: build a smaller foundation

- **User and pain:** network-tool developers, QA teams, field engineers, and
  trainers need one appliance to look like a routed enterprise network. The
  July 2026 CyberScope discovery showed that topology-only SNMP data does not
  reproduce discovery across routed subnets.
- **Product fit:** routed packet simulation is within NIAC's simulation charter.
- **Cost/value:** a complete router is disproportionate. Deterministic IPv4
  forwarding, static routes, DHCP relay, and router ICMP behavior solve the
  demonstrated workflow with a bounded implementation.
- **Cheapest useful result:** one external VLAN, an internal virtual fabric, and
  static routes. Do not begin with dynamic routing protocols.

### Market verdict: conditional market bet

This can differentiate NIAC Pro for network-tool QA, demos, and training, but
hardware proof must precede expansion. Stop after the IPv4 milestone unless a
CyberScope can discover the reference eight-subnet lab and the isolation test
records zero frames outside the allowed attachment for 24 hours.

### License placement

The locked NIAC matrix classifies advanced IPv4 stack behavior as Pro. Routed
virtual labs therefore require NIAC Pro. They use the same binary and local,
offline license validation as every other NIAC capability. The existing Free
tier keeps its basic responders and ten-device limit; this plan does not change
the published tier matrix.

## Current state and gap

ADR 0008 `segments:` binds independent simulations to physical 802.1Q tags.
`Stack.devicesFor(vlan)` selects a device table using the incoming wire tag.
This provides isolation and address reuse, but intentionally provides no path
between segments.

The current flat mode can answer for many IP subnets on one tag, but it does not
perform router forwarding. A request is dispatched directly to the destination
device, so it cannot model gateway MAC selection, TTL changes, route lookup,
router ICMP errors, or a routed return path.

Physical tags and simulated broadcast domains must not remain the same concept.
Using production VLAN tags to represent the internal lab would recreate the
leakage risk this design is intended to remove.

## Target architecture

```text
CyberScope
  |
  | physical VLAN 200 only
  v
External attachment
  |
  v
Virtual interface: LAB-EDGE / 10.10.200.1
  |
  v
Internal forwarding plane
  |-- COS data       10.10.210.0/24
  |-- COS corporate  10.10.220.0/24
  |-- COS guest      10.10.230.0/24
  |-- EVT management 10.20.200.0/24
  |-- EVT data       10.20.210.0/24
  |-- EVT corporate  10.20.220.0/24
  `-- EVT guest      10.20.230.0/24
```

Only the first link is physical. Every network below the forwarding plane is
an in-memory broadcast domain. Its VLAN identifier is scenario metadata and is
never serialized as an 802.1Q tag unless a separate external attachment
explicitly allows that tag.

### Components

1. **External attachment**
   - Owns the capture interface and an explicit `direct`, `access`, or `trunk`
     mode.
   - In direct mode, maps untagged frames from a dedicated physical cable to one
     virtual network and emits untagged frames.
   - In access mode, maps untagged guest frames to one externally enforced VLAN
     identity. In trunk mode, maps only allowlisted on-wire tags.
   - Centralizes egress validation; a packet with an unapproved physical tag is
     rejected before serialization.

2. **Virtual fabric**
   - Stores internal broadcast domains, interfaces, links, ARP/neighbor state,
     and MTUs.
   - Delivers frames between simulated interfaces without using libpcap.
   - Keeps internal VLAN identifiers separate from physical tags at the type
     level so accidental egress is difficult to express.

3. **IPv4 forwarding plane**
   - Performs longest-prefix route lookup.
   - Decrements TTL and updates the IPv4 checksum.
   - Selects an egress interface and next hop.
   - Rewrites source and destination Ethernet addresses at each virtual hop.
   - Generates ICMP Time Exceeded, Network/Host Unreachable, and Fragmentation
     Needed where required.

4. **Service dispatch**
   - Delivers the forwarded request to the destination device's existing
     protocol handlers.
   - Routes the generated response back through the virtual fabric.
   - Preserves the destination device's IP identity while presenting the
     external gateway's MAC to the tester.

5. **Scoped network services**
   - Stores DHCP leases and server configuration per virtual network.
   - Supports an explicit DHCP relay on a router interface.
   - Keeps DNS and other broadcast-sensitive state scoped to a virtual network.

## Proposed configuration model

The names are provisional and require an ADR before schema work. The important
rule is that `attachments` describe the real wire while `networks` describe only
the simulation.

```yaml
attachments:
  - name: tester
    connect: lab-access

networks:
  - name: lab-access
    subnet: 10.10.200.0/24
    virtual_vlan: 200
  - name: cos-data
    subnet: 10.10.210.0/24
    virtual_vlan: 210
  - name: evt-data
    subnet: 10.20.210.0/24
    virtual_vlan: 210

devices:
  - name: LAB-EDGE-R1
    type: router
    interfaces:
      - name: outside
        network: lab-access
        address: 10.10.200.1/24
      - name: cos
        network: cos-data
        address: 10.10.210.1/24
      - name: evt
        network: evt-data
        address: 10.20.210.1/24
    routes:
      - destination: 10.10.0.0/16
        via: cos
      - destination: 10.20.0.0/16
        via: evt

  - name: LAB-DHCP01
    type: server
    interfaces:
      - name: eth0
        network: lab-access
        address: 10.10.200.2/24
    dhcp:
      pools:
        - network: lab-access
          start: 10.10.200.200
          end: 10.10.200.220
          router: 10.10.200.1
```

A tester plugged directly into a dedicated NIAC port uses:

```yaml
attachment: tester
interface: eth0
mode: direct
```

The scenario contains the logical attachment only. Preflight and start supply
the physical interface and attachment policy. An access deployment adds an
`accessVlan` from 1 through 4094; VLAN 200 is CT 304's current binding, not a
hardcoded product or scenario value.

Configuration validation must reject duplicate interface addresses within one
network, missing connected routes, routes with unreachable next hops, DHCP pools
outside their network, overlapping pools, attachments to unknown networks, and
physical tags outside the operator allowlist.

`access_vlan` is deployment identity, not an on-wire tag visible inside NIAC.
For CT 304, Proxmox removes VLAN 200 before delivering the frame to `eth0`, so
NIAC accepts only untagged frames and emits only untagged frames in access mode.
The Proxmox bridge and persistent hook enforce that those frames belong to VLAN
200. Trunk mode is different and must declare `allowed_tags`; the two modes are
never inferred from observed traffic.

Direct and access modes deliberately have the same untagged frame shape inside
NIAC. Their operational contracts differ: direct mode requires a dedicated
physical interface and cable, while access mode requires an external switch or
hypervisor to enforce the declared access VLAN.

## Packet flow

For a CyberScope request from `10.10.200.200` to `10.20.210.31`:

1. CyberScope ARPs for `10.10.200.1`; only `LAB-EDGE-R1` answers.
2. The external attachment accepts the VLAN-200 frame and maps it to
   `lab-access`.
3. The forwarding plane verifies the destination MAC is the router interface,
   decrements TTL, and performs longest-prefix lookup.
4. The virtual fabric forwards the packet internally toward `evt-data`; no
   physical VLAN-210 frame is created.
5. The destination device's existing ICMP, SNMP, TCP, or UDP handler produces a
   response.
6. The response traverses the reverse route and exits the one external
   attachment tagged VLAN 200, sourced from the gateway MAC at layer 2 and the
   simulated endpoint IP at layer 3.

In direct mode, steps 2 and 6 use untagged frames. The virtual forwarding path
and destination behavior are otherwise identical.

## Stateful management plane

MIMIC Virtual Lab demonstrates an important behavior NIAC should adopt: users
can reach a simulated device through SSH, Telnet, or SNMP, and a configuration
change made through one interface is reflected through the others. MIMIC does
this with protocol and command simulation rather than vendor images. That model
fits NIAC better than introducing GNS3-style appliances.

NIAC should add the following after the forwarding foundation:

1. **One authoritative device state**
   - Interfaces, addresses, routes, VLAN membership, hostname, administrative
     state, counters, neighbors, running configuration, and startup
     configuration live in one typed state model.
   - SNMP, CLI, routing, LLDP/CDP, DHCP, and the UI read from that state.
   - Mutations are atomic and emit structured events.

2. **SSH device access**
   - An SSH connection to a simulated device IP terminates in NIAC's virtual TCP
     stack and is bridged to a vendor command profile.
   - The initial profile covers Cisco IOS-like behavior. Vendor-specific
     NX-OS-like and JUNOS-like profiles remain post-acceptance expansions.
   - Profiles describe supported commands and rendering; they do not claim to
     contain vendor software.

3. **Stateful command engine**
   - Support user, privileged, global configuration, interface, VLAN, and router
     modes where applicable.
   - Implement command help and completion from the same command tree.
   - Derive `show` output from authoritative state instead of replaying canned
     transcripts.
   - Make configuration commands change forwarding and management behavior. For
     example, `shutdown` changes link and SNMP state; an address or route change
     changes packet forwarding; a hostname change updates SNMP and discovery.

4. **Configuration lifecycle**
   - Maintain running and startup configurations.
   - Support copy/save, reload, scenario reset, checkpoints, and deterministic
     rollback.
   - Add TFTP import/export only after the in-memory lifecycle is correct.

5. **Event output**
   - Configuration and link changes can generate SYSLOG and SNMP notifications.
   - Event values and SNMP counters come from the same state transition.

6. **Guided exercises**
   - A scenario can define an initial state, allowed learner actions, expected
     end state, hints, and reset behavior.
   - Exercises remain a layer over the normal simulator; they do not introduce
     a second device model.

Telnet can be offered later as an explicitly enabled, lab-only compatibility
surface. SSH is the default. NETCONF, RESTCONF, gNMI, flow export, MQTT, and
recorded command-session import are useful extensions, but they require a named
workflow before implementation.

### MIMIC capability selection

| Capability | NIAC decision | Reason |
| --- | --- | --- |
| SSH and first IOS-like CLI profile | Built | High-value device interaction without images |
| Additional NX-OS-like and JUNOS-like profiles | Evaluate after operator acceptance | Expand only when a named workflow needs vendor-specific commands |
| Cross-protocol shared state | Build as foundation | Prevents contradictory CLI, SNMP, and forwarding views |
| Running/startup config and reset | Build | Required for reproducible labs and training |
| SYSLOG and SNMP notifications | Build | Supports monitoring and failure demonstrations |
| Guided exercises | Build after stateful CLI | Reuses scenarios without changing the packet core |
| TFTP config transfer | Build after config lifecycle | Useful but not foundational |
| Telnet | Optional, disabled by default | Compatibility surface with weak security |
| CLI/session recorder | Evaluate later | Valuable only when authored profiles are insufficient |
| NETCONF, RESTCONF, and gNMI | Demand-gated | Separate protocol surfaces and test burden |
| NetFlow, IPFIX, and sFlow | Demand-gated | Useful for collector testing, not routed-lab correctness |
| 100,000-device target | Do not copy as a goal | Optimize against measured NIAC customer scenarios |

Official MIMIC documentation describes simulated IOS/JUNOS command access over
SSH and Telnet, cross-protocol state, SYSLOG, TFTP, configurable command rules,
and guided virtual labs. NIAC should implement the coherent behavior, not copy
MIMIC's product breadth or training catalog.

## Isolation and safety invariants

These are release blockers, not optional hardening:

1. The default configuration has no external attachment.
2. Every attachment requires an explicit interface. Trunk and access modes
   require physical-VLAN policy; direct mode requires an explicit
   dedicated-interface declaration and accepts no tagged traffic.
3. Virtual VLAN identifiers cannot be passed to the capture send API.
4. Direct and access modes drop every tagged frame; trunk mode drops untagged,
   priority-tagged, and unapproved tagged traffic before protocol dispatch.
5. DHCP broadcast processing is limited to the ingress virtual network unless
   an explicit simulated relay forwards it internally.
6. NIAC never enables host IP forwarding, creates host bridges, changes host
   routes, or depends on kernel network namespaces for the simulated fabric.
7. Internal broadcasts, multicast, ARP, and discovery advertisements cannot
   cross an external attachment unless that network is explicitly attached.
8. A centralized egress guard records and rejects every attempted tag outside
   the attachment allowlist.
9. Reload is atomic: an invalid new topology never partially replaces a running
   safe topology.
10. Resource limits cap virtual networks, routes, devices, queued frames, and
    broadcast replication.
11. Direct mode startup displays the selected interface and requires an
    operator-owned deployment policy to mark it dedicated; NIAC never silently
    treats the management interface as a direct lab port.

## Delivery plan

### First implementation milestone: CyberScope routed discovery

Begin with one end-to-end workflow in the modern React UI, not with the
general-purpose router CLI or routing-protocol backlog. The operator must be
able to select the shipped `demo-linklive-routed` scenario, bind it to either a
dedicated untagged cable or the VLAN-200 access port, review exactly what can
reach the wire, start it, and verify what CyberScope and Link-Live observe.

The first milestone has this fixed scope:

- One external attachment and one simulated router.
- External subnet `10.10.200.0/24`, with DHCP only on that network.
- Eight authored internal IPv4 subnets behind the router.
- Connected and static routes only.
- ICMP, traceroute/path behavior, ARP, DHCP, DNS, SNMP, LLDP, CDP, and FDB
  behavior required by the CyberScope discovery workflow.
- Both `direct` untagged mode and `access` mode backed by VLAN 200. Trunk mode
  remains in the architecture but is not part of the first hardware release.
- An authored-truth view in NIAC showing the devices, subnets, and links that
  CyberScope should discover and subsequently upload to Link-Live.

Do not put SSH, vendor command profiles, OSPF, BGP, NETCONF, gNMI, NAT, ACLs,
or multiple physical attachments on this milestone's critical path. They build
on the same authoritative state later, but none is needed to prove the stated
CyberScope problem.

#### Modern UI workflow

Extend the existing New Simulation wizard rather than creating a second lab
builder. Routed scenarios must use a draft-first flow; the current behavior of
starting the simulation at the end of source selection is not safe enough for
an externally attached routed lab.

1. **Scenario:** choose `CyberScope Routed Discovery` from the existing
   template/library picker.
2. **Attachment:** choose the interface and either `Dedicated cable (untagged)`
   or `Access port (VLAN 200)`. The UI must explain that VLAN 200 is enforced by
   the switch in access mode and that NIAC sends and accepts untagged frames.
3. **Networks:** show the external network, all internal subnets, router
   interfaces, routes, and the single DHCP scope. Internal networks must be
   visually distinct from the physical attachment; a VLAN number alone must
   never imply physical exposure.
4. **Safety review:** call the compiler/preflight API without opening capture.
   Show the selected interface, attachment mode, physical VLAN policy, DHCP
   network, reachable simulated subnets, and every validation error. Start is
   disabled until this report is safe.
5. **Run and observe:** reuse the current Simulation status, Topology, Segments,
   and Packets surfaces, adding virtual-network, route-decision, DHCP-scope, and
   egress-drop data rather than inventing parallel pages.
6. **Verify discovery:** show NIAC's authored device, subnet, and link truth next
   to a concise CyberScope test checklist. The operator runs discovery on
   CyberScope, lets CyberScope upload its result to Link-Live, and compares that
   result with the NIAC view during troubleshooting. NIAC does not ingest an
   export, authenticate to Link-Live, or call a Link-Live API in the shipped
   product workflow. Development and hardware-acceptance tooling may call the
   Link-Live API to automate this comparison.

#### API seams required by the UI

- `POST /api/v1/simulation/preflight` compiles supplied draft YAML and an
  attachment selection into a typed, read-only report. It must not open an
  interface or alter the running simulation.
- `POST /api/v1/simulation` accepts the same already validated attachment
  model. The server recompiles and revalidates it; client-side state is never a
  safety boundary.
- `GET /api/v1/fabric` exposes attachments, virtual networks, router
  interfaces, routes, DHCP scopes, and drop counters for the running lab.
- The existing topology response and `GET /api/v1/fabric` together provide the
  authored device, subnet, route, and link truth rendered by the verification
  view. Do not create a Link-Live-specific product API or import model.

#### Development-only Link-Live verification

An external acceptance harness may use the Link-Live API after CyberScope has
uploaded a discovery. This harness lives outside the NIAC daemon and React UI,
uses developer-supplied credentials from the approved credential store, and is
never embedded in a scenario or product binary.

The harness should normalize the NIAC authored-truth API and the corresponding
Link-Live discovery into stable test records, then compare:

- Devices by MAC address first, with management IP and system name as diagnostic
  fields.
- Routed subnets and default gateways.
- LLDP/CDP neighbor relationships and advertised port identities.
- Switch FDB observations needed for CyberScope's topology inference.
- SNMP identity, interface, IP-address, route, and bridge-table data.

It must report missing, unexpected, and conflicting observations without
changing the NIAC scenario or Link-Live data. API responses used as test
evidence must be sanitized before they are stored; tokens, tenant identifiers,
customer names, and unrelated discoveries must never enter the repository.
Manual comparison remains available when API access is unavailable.

#### Ordered implementation slices

Each slice is independently reviewable and keeps unsafe packet I/O disabled
until its exit gate passes:

1. **Draft compiler and preflight contract:** add the routed schema, immutable
   topology compiler, attachment types, validation errors, preflight endpoint,
   generated TypeScript types, and a read-only Safety Review step in the wizard.
   This is the correct first code change.
2. **Guarded attachment boundary:** centralize capture receive/send behind
   `direct` and `access` policies, reject tagged frames in both modes, and expose
   drop reasons. Prove with fake capture tests that no virtual identifier or
   unapproved frame reaches the host interface.
3. **Minimal virtual fabric and network-scoped DHCP:** deliver internal frames,
   ARP, and existing protocol handlers by virtual network; make DHCP ownership
   per-network before forwarding is enabled. The UI shows the one eligible DHCP
   scope and live lease/counter state.
4. **IPv4 forwarding and response return path:** implement connected/static
   routing, TTL/checksum/MTU/ICMP behavior, and route every existing handler's
   response back through the fabric. Add golden PCAP tests before enabling the
   routed scenario in production builds.
5. **CyberScope discovery fidelity:** populate shared topology state for SNMP,
   LLDP, CDP, and FDB views; enhance Topology and Segments to display the
   authored routed networks and links accurately. This UI is NIAC's source of
   truth when examining the discovery CyberScope uploaded to Link-Live. Add one
   vendor-identity allocator backed by a pinned `oui.txt` dataset so generated
   Cisco, Juniper, Aruba, and other personas use the correct prefix with a
   deterministic suffix. Preserve explicitly authored MAC addresses, reject
   duplicates during preflight, and never copy OUI-selection logic into
   templates or the React UI.
6. **Hardware acceptance:** run the same scenario first through a dedicated
   direct cable, then through CT 304's VLAN-200 access port. Packet captures on
   the NIAC host and Proxmox bridge are release evidence, not an optional manual
   check. Compare the resulting Link-Live topology with NIAC's authored topology
   using the development harness when API access is available, otherwise compare
   it manually. Record discrepancies as protocol-simulation defects.

**First milestone exit gate:** from the modern UI, a user can preflight and run
the reference lab in direct or VLAN-200 access mode; CyberScope receives an
address only from `10.10.200.0/24`, reaches all seven internal routed subnets, discovers
the authored devices and links through SNMP/LLDP/CDP/FDB, and uploads a
Link-Live topology that matches NIAC's authored-truth view. A simultaneous
physical capture proves NIAC emitted no tagged traffic and no DHCP response
outside the dedicated lab attachment.

### Phase 0: ADR and executable contract

- Write ADR 0009 defining physical attachments versus virtual networks.
- Freeze the smallest IPv4 schema and JSON Schema representation.
- Add packet-flow fixtures for one routed request and its response.
- Add negative fixtures for DHCP leakage and unapproved physical tags.
- Specify IPv4 field preservation, fragmentation when DF is clear,
  Fragmentation Needed when DF is set, and RFC ICMP-error suppression for
  broadcasts, multicast, non-initial fragments, and ICMP error responses.

**Exit gate:** architecture review accepts the type boundary and packet-flow
contract before production code changes.

### Phase 1: topology compiler

- Parse networks, interfaces, connected routes, and static routes into immutable
  runtime topology.
- Add validation and deterministic longest-prefix route tables.
- Keep this phase pure: no capture or packet-handler changes.

**Tests:** table-driven schema, overlap, next-hop, route-selection, and atomic
reload tests with full coverage of the compiler core.

### Phase 2: virtual fabric

- Complete ADR 0008's I/O boundary first: capture and serialization move out of
  `Stack`, and no protocol handler retains direct capture-send authority.
- Introduce distinct physical-tag and virtual-network identifier types.
- Implement internal frame delivery and per-network neighbor tables.
- Move the capture send boundary behind an attachment-aware egress guard.
- Build the routed path inactive while it is incomplete. When it lands, migrate
  every scenario and caller in the same change and delete the superseded path;
  pre-1.0 NIAC does not carry a compatibility architecture.

**Tests:** no internal frame reaches the fake capture engine; unknown physical
tags are dropped; repeated addresses on different networks remain isolated.

### Phase 3: network-scoped services

- Replace the single stack-wide DHCP server state with per-network server state
  before routed forwarding can be enabled.
- Scope DNS, NetBIOS, ARP, IPv6 neighbor discovery, and periodic discovery
  emissions to their virtual network.
- Implement the DHCP relay contract: set `giaddr` to the ingress router
  interface, increment and cap hops, unicast the relayed request internally to
  the selected server, accept server replies addressed to `giaddr`, and honor
  the client broadcast flag on delivery to the client network.
- Reject Option 82 in the first release with a clear validation error; support
  it only when a demonstrated workflow requires it.
- Make duplicate DHCP servers on one network a validation error unless a later
  high-availability design explicitly supports them.

**Tests:** DHCP pools never answer outside their network; relay `giaddr`, hops,
UDP endpoints, L2 addressing, and broadcast handling match the contract;
multiple networks can reuse pool ranges without state collision; unapproved
physical VLANs receive no response.

### Phase 4: IPv4 forwarding

- Forward packets addressed to router interfaces.
- Implement TTL, checksum, DSCP/ECN preservation, IP ID/options and fragment
  preservation, next-hop, and Ethernet rewrite behavior.
- Fragment oversized IPv4 packets when DF is clear. Send Fragmentation Needed
  only when DF is set, and implement the Phase 0 ICMP suppression rules.
- Route responses from existing handlers through the fabric rather than sending
  them directly to capture.
- Keep routed mode disabled until Phases 2-4 pass together; there is no release
  in which forwarding uses the old global DHCP or service state.

**Tests:** golden PCAP tests for one, two, and three hops; TTL expiration;
unreachable routes; MTU failure; asymmetric routing; and concurrent flows.

### Phase 5: authoritative state and management plane

- Introduce the shared device-state model before implementing CLI commands.
- Adapt SNMP, discovery, forwarding, and configuration rendering to consume the
  shared state.
- Implement virtual TCP sessions and SSH transport for simulated device IPs.
- Add the first IOS-like command profile with operational, privileged, global
  configuration, interface, VLAN, and router modes.
- Make supported configuration commands change forwarding and management state.
- Add running/startup configuration, save, reload, checkpoint, and reset.
- Emit matching SYSLOG and SNMP notifications for state changes.

Additional NX-OS-like and JUNOS-like profiles are post-acceptance expansions,
not part of the Phase 5 exit gate. Add one only for a named workflow that the
IOS-like profile cannot represent.

**Tests:** command parser and mode transitions, SSH authentication and session
isolation, concurrent configuration transactions, CLI-to-SNMP consistency,
CLI-to-forwarding consistency, save/reload/reset, event output, and malformed or
unsupported command handling.

### Phase 6: observability and operator controls

- Expose attachments, virtual networks, routes, drops, and forwarding counters
  through the existing API and UI.
- Distinguish `physicalVlan` from `virtualVlan` in every response and label.
- Add packet-inspector fields for ingress network, route decision, hop, egress
  network, and egress rejection reason.
- Require a prominent unsafe-attachment validation error rather than silently
  widening access.

### Phase 7: reference scenario and hardware acceptance

- Convert `demo-linklive-flat` into `demo-linklive-routed` with the external
  `10.10.200.0/24` access network and seven internal site subnets.
- Keep CT 304 `eth0` physically confined to VLAN 200 and `eth1` on management
  VLAN 40. Do not configure a production trunk.
- Run DHCP, ICMP, SNMP, LLDP/CDP, FDB, DNS, TCP, and path-analysis tests from an
  isolated Linux namespace before connecting CyberScope.
- Run the same lab through a dedicated direct cable with `mode: direct`, no
  switch, and no VLAN tags. DHCP, routing, SNMP, SSH, and topology discovery must
  match access mode.
- Capture both CT `eth0` and the Proxmox bridge for 24 hours. Assert that CT
  `eth0` contains only untagged access-mode traffic and the bridge contains no
  NIAC traffic outside VLAN 200.
- Run a new CyberScope discovery, let CyberScope upload it to Link-Live, and
  compare the displayed Link-Live result with NIAC's authored topology and
  subnet view. Automate this with the development-only Link-Live harness when
  credentials and API access are available. This is acceptance tooling, not
  NIAC product data ingestion.

**Release gate:** all eight subnets are reachable through the simulated gateway,
the topology relationships are correct, DHCP only serves `10.10.200.0/24`, and
the physical capture contains only allowed VLAN-200 lab traffic.

### Phase 8: controlled expansion

Only after Phase 7 passes:

- Add IPv6 forwarding and DHCPv6 relay.
- Add route policy and simulated ACL behavior.
- Add OSPF/BGP control-plane sessions that populate the same forwarding table.
- Consider multiple external attachments only with an equally strict allowlist
  and a separate threat model.

NAT, VPN data planes, MPLS, kernel bridging, and distributed simulation remain
out of scope until a demonstrated workflow requires them.

## Test matrix

| Layer | Required proof |
| --- | --- |
| Config | Invalid networks, routes, pools, and attachments fail closed |
| Route core | Longest prefix, connected routes, next hops, ECMP rejection |
| L2 | ARP and MAC rewrite occur only inside the selected network |
| IPv4 | TTL, checksum, MTU, ICMP errors, forward and return paths |
| Services | ICMP, SNMP, DNS, TCP, and UDP survive routed dispatch |
| DHCP | Server and relay state is per network with no cross-network offers |
| Management | SSH commands, SNMP, discovery, config, and forwarding share state |
| Isolation | Capture sees no unapproved tag, native response, or internal broadcast |
| Direct cable | Untagged DHCP, routing, SNMP, SSH, and discovery require no VLAN |
| Reload | Concurrent traffic sees either the old or new topology, never a partial one |
| Scale | Reference 70-device/eight-subnet lab meets latency and loss budgets |
| Hardware | CyberScope and Link-Live reproduce the authored routed topology |

Run race tests and fuzz the topology compiler, Ethernet/IP decoder, route lookup,
DHCP relay, and egress guard. Integration tests should use an in-memory capture
first, then a Linux namespace and veth pair. Hardware validation is the final
gate, not a substitute for automated tests.

## Operational deployment for CT 304

- `eth0`: no host IP; Proxmox bridge membership restricted to VLAN 200.
- `eth1`: management only on VLAN 40; never part of simulation capture.
- NIAC service account retains only the raw-network capabilities already needed.
- Scenario configuration declares exactly one external attachment:
  `eth0` in access mode with deployment identity VLAN 200.
- Proxmox hook verification remains in place as defense in depth.
- Startup fails if attachment mode and the operator-owned deployment allowlist
  disagree. NIAC does not claim it can inspect a hypervisor's access-VLAN
  assignment from inside the guest.

## Completion criteria

- The identical scenario runs in direct mode over an untagged dedicated cable
  and in access mode behind physical VLAN 200.
- A CyberScope lease is issued only from `10.10.200.200-220` by `10.10.200.2`.
- CyberScope reaches and polls devices across the access network and all seven
  internal `/24` networks.
- Traceroute/path analysis shows the authored virtual router hops.
- SNMP/LLDP/CDP/FDB data matches the scenario manifest.
- SSH reaches the addressed simulated device, supported configuration changes
  alter forwarding and SNMP state, and scenario reset restores the baseline.
- A 24-hour bridge capture contains no DHCP response outside the lab access
  network and no NIAC traffic on a physical VLAN other than 200. CT `eth0`
  contains only the untagged frames expected in access mode.
- `make lint`, `make fmt-check`, `make test`, `make test-e2e`, race tests,
  fuzz smoke tests, `make security`, and the full build pass without warnings.
- The deployed `/__version` reports the expected build and non-empty UI hash.

## Kill criteria

Stop before dynamic routing work if any of these remain true after Phase 7:

- CyberScope cannot discover the reference routed topology reliably.
- Correct forwarding requires exposing internal simulated VLANs on the wire.
- Isolation cannot be enforced in one centralized egress boundary.
- The 70-device reference lab cannot meet an agreed response-latency budget on
  CT 304 hardware.
- The configuration model requires users to understand both physical and
  virtual VLAN concepts without clear validation and UI separation.

## References

- `docs/adr/0008-multi-vlan-segment-playback.md`
- Legacy Java NIAC: `src/fluke/niac/protocols/Stack.java` and
  `src/fluke/niac/devices/Device.java` in the `niac-java` reference repository
- [MIMIC Virtual Lab Cloud](https://www.gambitcomm.com/cloud_vlab/)
- [MIMIC IOS Simulator](https://www.gambitcomm.com/site/ios-simulator.php)
- [MIMIC Virtual Lab CLI commands](https://doc.gambitcomm.com/mimic/vlabappendixa.htm)
- [MIMIC Quick Start Guide](https://doc.gambitcomm.com/mimic/quick.htm)
