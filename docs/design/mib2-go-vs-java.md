# MIB-II implementation comparison

## Summary

`niac-go` implements MIB-II semantics and derives supported telemetry from its
simulated protocol and forwarding state. `niac-java` is a walk replay engine:
it can expose any OIDs present in a captured walk or added with `AddMib`, but it
does not calculate those values from packet processing.

| Capability | niac-go | niac-java |
|---|---|---|
| RFC 1213 groups | Explicit system, IF/IF-X, IP, ICMP, TCP, UDP, EGP, and SNMP MIB implementations | `Mib.Constants` and `OidMap` names; values come from walks or injected entries |
| Interface and bridge counters | Packet-authoritative shared counters for IF-MIB, IF-X, and bridge ports | No packet-derived counters |
| Routing and topology | Route, ARP/IP address, bridge/FDB, STP, LLDP/CDP tables are synthesized from configured topology | No route/FDB/STP synthesis; DHCP can add static FDB rows |
| TCP | Stateful lifecycle scalars and dynamic `tcpConnTable` | No lifecycle/table generation |
| IPv4 reassembly | Bounded per-VLAN reassembly with request/success/failure counters | No reassembly telemetry |
| SNMP protocol | v1, v2c, and v3 USM | Community-based GET/GETNEXT/GETBULK; no v3/USM and no meaningful version validation |
| Error injection | Authored link state is reflected consistently; unsupported physical event sources remain zero | Walk/`AddMib` injection can replay error and topology counters without packet causality |

Go evidence is in `internal/protocols/snmp/mib_*.go`,
`internal/protocols/snmp/tcp_flow_tracker.go`, `internal/protocols/ip.go`, and
the corresponding conformance tests. Java evidence is in
`/Users/krisarmstrong/Developer/NetAlly/niac-java/src/fluke/niac/snmp/Agent.java`,
`Mib.java`, `Handler.java`, and `DhcpServer.java`.

The practical result is that Go is the correct implementation for a routed,
CyberScope-facing simulation with truthful dynamic telemetry. Java remains
useful for replaying vendor-specific captured walks, including values for
physical error conditions that Go does not yet generate as simulation events.
