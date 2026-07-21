# NIAC Defect-Hunting Plan

## Objective

Find and fix correctness, safety, interoperability, lifecycle, and usability defects across NIAC. The immediate acceptance target is a CyberScope discovery that reconstructs a routed multi-site network from NIAC's packet and management-plane responses without exposing the physical network.

Every defect must have reproducible evidence, a root-cause fix, and a regression test. A symptom-only workaround does not close a finding.

## Safety boundaries

- Keep CT304 attached only to host VLAN 200.
- Do not enable forwarding or NAT to the physical network.
- Preserve the currently deployed binary, service helper, and scenario before deployment changes.
- Use reserved lab ranges only: `10.240.0.0/16` through `10.243.0.0/16`, with `10.254.200.0/24` as the tester handoff.
- Do not use a live CyberScope discovery as the primary development loop. Reproduce captured traffic locally first.

## Finding record

Record each finding with:

- subsystem and severity;
- minimal reproduction;
- expected and actual behavior;
- packet capture, log, or test evidence;
- root cause;
- regression test;
- fix commit and pull request;
- CT304 and CyberScope validation result where applicable.

Severity is based on impact:

- **Critical:** physical-network exposure, remote compromise, data loss, or unusable installation.
- **High:** incorrect packet behavior, protocol interoperability failure, authorization bypass, or daemon failure.
- **Medium:** incorrect MIB/config/UI behavior with a contained workaround.
- **Low:** misleading diagnostics, minor usability defects, or maintainability hazards that can cause future defects.

## Phase 0: Freeze and baseline

1. Inventory all uncommitted changes and separate prior work from new audit fixes.
2. Capture the active CT304 binary version, service definitions, interfaces, VLAN restrictions, scenario checksum, and rollback files.
3. Run the full repository baseline: formatting, lint, unit tests, race tests, UI tests, end-to-end tests, build, vulnerability scan, and security checks.
4. Record existing failures without suppressing them.

Exit criterion: a reproducible baseline and a safe rollback point.

## Phase 1: Build an independent protocol harness

1. Convert representative CyberScope ARP, DHCP, ICMP, DNS, SNMP GET, GET-NEXT, and GET-BULK frames into deterministic test fixtures.
2. Cover untagged access/direct traffic, unexpected tagged traffic, malformed,
   truncated, duplicate, and out-of-order packets.
3. Add a test client outside the NIAC process that can run `snmpget` and `snmpwalk` against a five-device fixture.
4. Observe NIAC's transmit queue and final serialized frame independently so injected packets can be proven without relying on LXC `tcpdump` visibility.
5. Make request/response correlation report target device, community/user, request ID, VLAN, source/destination MAC, and send error.

Exit criterion: failures can be reproduced without CyberScope or Link-Live.

## Phase 2: Packet and attachment runtime

Audit:

- Ethernet, 802.1Q, and nested-tag parsing;
- access versus direct attachment semantics;
- ARP ownership and routed gateway identity;
- DHCP isolation and lease correctness;
- IPv4 checksums, UDP/TCP checksums, TTL, fragmentation, and broadcast handling;
- reply MAC, IP, VLAN, and interface selection;
- capture injection, queue shutdown, backpressure, and error reporting;
- interface restart, daemon restart, and stale runtime state.

Required matrix:

| Attachment | Wire frame | Expected behavior |
|---|---|---|
| Direct | Untagged | Accept and reply untagged |
| Direct | Any VLAN tag | Reject |
| Access VLAN | Untagged | Accept and reply untagged; external access port owns the tag |
| Access VLAN | Any VLAN tag | Reject |

Exit criterion: ARP, DHCP, ICMP, DNS, and SNMP work in every supported attachment mode with packet-level tests.

## Phase 3: Routed fabric correctness

Audit:

- network and interface uniqueness;
- overlapping networks and reserved endpoints;
- connected and static route compilation;
- longest-prefix selection and ambiguous routes;
- forward and return-path identity;
- next-hop/interface validity;
- missing routes and unreachable endpoints;
- route lifecycle during start, stop, import, and restart;
- isolation between virtual networks and the host attachment.

Add table-driven tests for every site and protocol, not only one generic endpoint.

Exit criterion: the tester can reach every intended endpoint, cannot reach undeclared endpoints, and cannot escape the lab boundary.

## Phase 4: SNMP and cross-MIB conformance

Audit SNMPv1, v2c, and v3 discovery, authentication, GET, GET-NEXT, GET-BULK, error responses, maximum PDU size, and concurrent walks.

Validate these cross-MIB chains:

```text
FDB MAC
  -> dot1dTpFdbPort
  -> dot1dBasePortIfIndex
  -> IF-MIB ifIndex
  -> ifName / ifDescr
```

```text
IP route
  -> ipRouteIfIndex
  -> IF-MIB ifIndex
  -> configured routed interface
```

```text
LLDP/CDP neighbor
  -> local port
  -> IF-MIB ifIndex
  -> reciprocal remote chassis and port
```

Check MIB-II system identity, uptime, interface counters, high-speed interfaces, bridge type, STP, ARP, IP address tables, route tables, FDB status, and VLAN-aware FDB data. Ensure captured walks cannot overwrite authored identity or topology.

Exit criterion: the five-device fixture passes automated MIB consistency checks and a direct `snmpwalk` produces a complete, self-consistent graph.

## Phase 5: Discovery protocol topology

1. Prove the five-device hierarchy: lab edge, site edge, core, distribution, access switch, and one AP.
2. Validate reciprocal LLDP/CDP relationships and unique local/remote port indexes.
3. Validate AP and endpoint placement from FDB data without inventing neighbor advertisements for devices that would not advertise them.
4. Add topology graph invariants: no self-edges, duplicate edges, accidental full mesh, orphan infrastructure, or multiple devices sharing one captured identity.
5. Scale the proven hierarchy to one complete site, then four sites.

Exit criterion: the generated graph is hierarchical before it is tested in Link-Live.

## Phase 6: Configuration, compiler, and catalog

Audit every YAML field for parse, validation, serialization, import/export, UI editing, and runtime use. Add referential-integrity checks for:

- device, network, attachment, interface, route, and neighbor names;
- duplicate IP and MAC addresses;
- walk file existence and containment;
- interface names missing from selected walks;
- FDB ports missing from IF-MIB;
- LLDP/CDP ports missing from IF-MIB;
- DHCP pools, routers, DNS servers, and collisions;
- VLAN and attachment inconsistencies.

Generated scenarios must validate with zero errors and zero actionable warnings.

Exit criterion: invalid topology cannot start, and the demo catalog is reproducible from its generator.

## Phase 7: Daemon, API, and modern UI

Audit:

- start, stop, restart, crash recovery, and boot activation;
- preflight parity with runtime behavior;
- stale asynchronous UI results and duplicate submissions;
- CSRF, authentication, scopes, rate limits, and error encoding;
- configuration import/export and path handling;
- version/build metadata;
- topology, statistics, packet-inspector, and diagnostic accuracy;
- accessibility and Playwright user flows.

Exit criterion: the modern UI cannot start an unsafe or compiler-invalid scenario and accurately reports runtime state.

## Phase 8: Reliability, performance, and security

Run:

- race detector and repeated tests;
- fuzzing for packet decoders, SNMP BER, YAML conversion, and walk parsing;
- load tests for 5, 135, 500, and supported-maximum device counts;
- concurrent CyberScope-style SNMP walks;
- memory, goroutine, queue, file-descriptor, and shutdown leak checks;
- malformed and hostile packet tests;
- `govulncheck`, `gosec`, Semgrep, dependency scanning, and secret scanning.

Exit criterion: no races, leaks, reachable vulnerabilities, parser panics, or unbounded resource growth.

## Phase 9: Acceptance ladder

Promote only after the previous stage passes:

1. Pure unit and property tests.
2. Captured-packet integration tests.
3. Independent `snmpget`/`snmpwalk` client.
4. Five-device local fixture.
5. Five-device CT304 fixture.
6. One-site CT304 fixture.
7. Four-site CT304 fixture.
8. Clean CyberScope discovery.
9. Link-Live topology and problem-log review.

Final acceptance requires:

- correct routed hierarchy;
- correct site and VLAN gateways;
- correct AP and endpoint switch-port placement;
- complete SNMP data for every managed device;
- bounded, intentional problems only;
- no physical-network addresses in the scenario;
- no DHCP, discovery advertisement, route, or packet leakage beyond VLAN 200;
- repeatable results after daemon and CT304 restart.

## Delivery sequence

Land small root-cause pull requests in dependency order:

1. protocol harness and observability;
2. attachment and packet-runtime correctness;
3. routed fabric correctness;
4. SNMP/MIB consistency;
5. topology and catalog correctness;
6. daemon/API/UI lifecycle correctness;
7. reliability and security hardening;
8. final CyberScope acceptance evidence.

Do not bundle unrelated findings into one change. Each fix must include its regression test and validation evidence.
