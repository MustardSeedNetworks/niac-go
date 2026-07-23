# Code capability index

This index records shared capabilities that must be checked before adding a
parallel implementation. It is intentionally organized by purpose.

## Configuration loading and validation

| Capability | Canonical location | Notes |
| --- | --- | --- |
| YAML authoring DTO | `internal/converter/types.go` | Source for generated YAML schema |
| YAML field validation | `internal/converter/validate.go` | Shape validation using YAML field names |
| Runtime config loading | `internal/config/yaml_load.go` | One file/bytes conversion pipeline |
| Vendor-authored MAC identity | `internal/converter/types.go` + `internal/config/yaml_device.go` | `vendor` plus optional `mac_suffix` is resolved through the embedded IEEE registry while preserving the authored form on export |
| Routed YAML adapters | `internal/config/yaml_fabric.go` | Converts authoring DTOs into runtime config |
| Complete-config validation | `internal/config/validator.go` | Existing device validation; not routed semantics |

## Routed fabric and topology

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Routed semantic compiler | `internal/fabric/compiler.go` | Pure config plus binding to immutable topology |
| Routed device compilation | `internal/fabric/compiler_devices.go` | Interfaces, routes, and DHCP ownership |
| Fabric domain and diagnostics | `internal/fabric/types.go` | Physical binding is distinct from virtual networks |
| Device/link UI graph | `internal/topology/topology.go` | Projection only; not a forwarding compiler |
| Physical VLAN engines | `internal/protocols/stack_init.go` | ADR 0008 segments; not routed virtual networks |
| Routed reply Ethernet identity | `internal/protocols/stack.go` | One source for gateway/device source MAC, requester destination MAC, and ingress VLAN |
| Final wire egress policy | `internal/protocols/stack_threads.go` | Last enforcement point for direct/access untagged frames and observer-visible bytes |
| Operator attachment authorization | `internal/fabric/types.go` + `internal/daemon/daemon.go` | Exact interface/mode/access-VLAN policy; browser input cannot grant approval |
| IEEE vendor registry | `internal/oui/registry.go` | Embedded IEEE assignment lookup and deterministic isolated-lab MAC allocation; do not add vendor-prefix tables elsewhere |

## SNMP discovery and topology

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Walk parsing and validation | `internal/protocols/snmp/walk.go` + `walk_validator.go` | Shared net-snmp syntax, numeric bounds, continuation, and terminal-marker handling |
| IF-MIB identity resolution | `internal/protocols/snmp/mib_if.go` + `peer_topology.go` | Authored interface metadata and every topology table resolve through the walk's real ifIndex |
| LLDP/CDP synthesis | `internal/protocols/snmp/mib_discovery.go` | Infrastructure neighbor rows; `fdb_only` attachments are intentionally excluded |
| Bridge/FDB synthesis | `internal/protocols/snmp/peer_topology.go` | MAC to bridge-port to ifIndex chain used for endpoint placement |
| IP and route MIB synthesis | `internal/protocols/snmp/mib_ip.go` | MIB-II addresses, routes, ARP, and explicit next-hop identity |
| Per-device MIB-II and interface telemetry | `internal/protocols/stack_protocol_telemetry.go` + `internal/protocols/snmp/protocol_telemetry.go` | One atomic event source per simulated device and authored interface, shared by its SNMP agents; protocol, IF-MIB, IF-X, and bridge counters advance from packet events |

## Stateful device management

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Authoritative mutable device state | `internal/devicestate` + `internal/protocols/stack_device_state.go` | The stack owns one concurrency-safe store per simulated device; management protocols consume that shared store rather than owning mutable copies |
| Observable interface faults | `internal/devicestate/store_fault.go` + `internal/protocols/stack_fault.go` + `internal/protocols/snmp/fault_telemetry.go` | One stack-owned fault catalog and state source drives API/TUI controls plus monotonic IF-MIB, IF-X, and EtherLike-MIB counters |
| IOS-like command profile | `internal/devicecli` | Stateful command modes, help, operational rendering, configuration mutations, running/startup/checkpoint lifecycle, and explicit configuration events |
| Virtual TCP byte streams | `internal/virtualtcp` | Buffered in-memory and packet-backed `net.Conn` implementations used by simulated stream protocols |
| Simulated SSH transport | `internal/devicecli/ssh_server.go` + `internal/protocols/tcp_ssh.go` | Explicit per-device credentials, isolated command sessions, and SSH termination through the virtual IPv4/TCP packet path |
| Shared-state SNMP projection | `internal/protocols/snmp/device_state*.go` | Dynamic hostname, discovery identity, interface status/alias, IP address, and route values derived from authoritative state |
| State notification output | `internal/protocols/state_notifications.go` | Authoritative transitions drive RFC 5424 SYSLOG plus SNMPv2c coldStart/linkUp/linkDown notifications; nonfunctional synthetic threshold traps are not part of the schema |

## Simulation lifecycle API

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Simulation request contract | `internal/api/server.go` | Shared start and preflight request fields |
| Simulation request validation | `internal/api/validation.go` | Interface/config source boundary validation |
| Simulation lifecycle handlers | `internal/api/handlers_simulation.go` | Strict decoder and standard error envelope |
| Route security policy | `internal/api/routes.go` | Methods, rate limits, and CSRF are registered here |
| Config preparation and start | `internal/daemon/daemon.go` | Preflight must not persist or open capture |

## Modern UI reuse points

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Guided simulation flow | `ui/src/pages/NewSimulationWizardPage.tsx` | Extend instead of creating another lab builder |
| Physical binding preflight | `ui/src/components/wizard/PreflightStep.tsx` | Review direct/access attachment and gate start on server diagnostics |
| Config source picker | `ui/src/components/simulation/ConfigPicker.tsx` | Templates, saved networks, and uploads |
| API request boundary | `ui/src/api/client.ts` | Add typed sibling functions here |
| API response types | `ui/src/api/api-response-types.ts` | Handwritten; generated types do not yet exist |
| Runtime topology view | `ui/src/pages/TopologyPage.tsx` | Authored/observed device and link projection |
