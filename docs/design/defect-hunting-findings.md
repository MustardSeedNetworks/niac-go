# NIAC Defect-Hunting Findings

This is the working finding register for `defect-hunting-plan.md`. A finding is
not closed by code inspection alone. Closure requires a failing reproduction,
a root-cause fix, a regression test, and the applicable local/CT304 acceptance
evidence.

## Priority findings

| ID | Severity | Subsystem | Finding | State |
|---|---|---|---|---|
| NIAC-DEF-001 | High | YAML/SNMP | Authored SNMP community and system identity fields are silently discarded, so runtime falls back to `public` and generated identity | Root cause fixed; focused regression passes; full/live gates pending |
| NIAC-DEF-002 | Critical | Attachment safety | Direct mode trusts a request boolean instead of proving the selected host interface is isolated | Root cause fixed with operator-owned exact attachment policy; CT304 gate pending |
| NIAC-DEF-003 | High | VLAN runtime | Compiled attachment mode is discarded; routed access/direct mode can accept tagged frames instead of enforcing its untagged wire contract | Root cause fixed; tagged/untagged frame regressions pass; CT304 access-port gate passes |
| NIAC-DEF-004 | High | DHCP/routing | Routed DHCP indexes legacy `IPAddresses`; interface-only routed devices can time out or panic | Root cause fixed; full-frame regression and CT304 DHCP lease pass |
| NIAC-DEF-005 | High | Routed protocols | ICMP/SNMP use routed return identity, while DNS/TCP/generic UDP still use legacy ownership and endpoint MAC behavior | Root cause fixed through one response-identity path; protocol matrix passes |
| NIAC-DEF-006 | High | Routing | Routed TTL handling does not decrement through the compiled router or emit the correct gateway identity | First-hop forwarding and Time Exceeded identity fixed; a true multi-hop routing engine remains separate work |
| NIAC-DEF-007 | High | Runtime reload | `ReloadConfig` rebuilds device tables without atomically recompiling the fabric, leaving stale route/device pointers | Root cause fixed with transactional fabric reload; regression passes |
| NIAC-DEF-008 | High | Route compiler | Static routes validate a local interface name but not usable next-hop, egress, ambiguity, or return-path semantics | Explicit peer-owned on-link next hop implemented; broader loop/return-path checks pending |
| NIAC-DEF-009 | High | Validation | Daemon/preflight bypass the semantic config validator, allowing duplicate flat/segment identity to start | Root cause fixed; daemon and preflight share strict semantic preparation |
| NIAC-DEF-010 | High | Serialization | YAML export/round-trip omits routed topology and protocol fields and can destructively erase a scenario | Root cause fixed with canonical complete serializer and round-trip coverage |
| NIAC-DEF-011 | High | Preflight parity | Attachment-only scenarios compile in preflight but bypass the compiler at runtime | Root cause fixed; real-daemon E2E preflight/start parity passes |
| NIAC-DEF-012 | High | Daemon transaction | Rejected inline replacement persists rejected YAML before compilation, splitting reported config from running state | Root cause fixed; preparation completes before persistence/runtime replacement |
| NIAC-DEF-013 | High | Daemon transaction | Replacement stops the active simulation before the replacement stack is known to be startable | Code-confirmed; fault-injection seam needed |
| NIAC-DEF-014 | High | API filesystem | Read/write callers can select an arbitrary absolute YAML path readable by the daemon | Root cause fixed with managed-root path enforcement; security tests pass |
| NIAC-DEF-015 | High | UI lifecycle | Runtime Control duplicates and bypasses the wizard preflight/binding workflow | Code-confirmed; consolidate flow |
| NIAC-DEF-016 | High | UI lifecycle | Wizard Back/start can replay stale prepared YAML and erase device edits | Root cause fixed; wizard state and Back-path tests pass |
| NIAC-DEF-017 | High | Dependencies | Security gate fails on high-severity `brace-expansion` and `js-yaml` advisories | Lockfile patched; `npm audit --audit-level=high` passes; full security gate pending |

## Secondary findings

| ID | Severity | Subsystem | Finding | State |
|---|---|---|---|---|
| NIAC-DEF-018 | Medium | Packet parser | Truncated Ethernet/VLAN input is not rejected and nested tags lack an explicit policy | Root cause fixed; truncated, priority-tagged, and nested-tag regressions pass |
| NIAC-DEF-019 | Medium | IPv4 transport | Inbound checksums, fragments, and routed source-network validity are not enforced | Code-confirmed |
| NIAC-DEF-020 | Medium | TX observability | Observers see pre-tag bytes and queue/injection errors cannot be correlated to the final serialized frame | Root cause fixed; observers and counters consume the successfully injected final frame |
| NIAC-DEF-021 | Medium | Stack lifecycle | Stack channels are one-shot and sends remain accepted after stop without delivery status | Code-confirmed |
| NIAC-DEF-022 | Medium | Catalog validation | Forward neighbor references generate order-dependent false warnings | Root cause fixed; order-independent catalog validation passes |
| NIAC-DEF-023 | Medium | YAML parser | Unknown YAML fields are accepted, which hid the dropped SNMP community defect | Root cause fixed; strict unknown-field tests pass across API/CLI |
| NIAC-DEF-024 | Medium | Fabric compiler | Attachment names and virtual VLAN identities are not uniquely enforced | Root cause fixed; compiler uniqueness regressions pass |
| NIAC-DEF-025 | Medium | DHCP compiler | Static leases, DNS options, server ID, next-server, and remaining collision rules lack validation | Code-confirmed |
| NIAC-DEF-026 | Medium | Walk integration | Startup does not prove authored IF-MIB, bridge, FDB, LLDP, and CDP indexes agree with the selected walk | Root cause fixed; all synthesized topology tables resolve through the walk's real ifIndex |
| NIAC-DEF-027 | Medium | Catalog sync | Sync is copy/diff against mutable `main`, without parse/semantic/walk validation or a recorded immutable source revision | Code-confirmed |
| NIAC-DEF-028 | Medium | Recovery | Daemon restart has no persisted active-simulation recovery contract | Code-confirmed |
| NIAC-DEF-029 | Medium | Preflight | Preflight can report safe for an interface runtime will reject as nonexistent | Code-confirmed |
| NIAC-DEF-030 | Medium | E2E harness | Default E2E run claims a backend requirement but starts only Vite and floods proxy connection failures | Root cause fixed; real HTTPS daemon harness passes Chromium and WebKit (124 pass, 4 intentional skips) |
| NIAC-DEF-031 | Low | Build | Darwin race/build links libpcap twice and emits repeated linker warnings | Baseline reproduced |
| NIAC-DEF-032 | Low | UI build | Production build exceeds the configured 350 kB chunk warning threshold | Baseline reproduced |
| NIAC-DEF-033 | Low | UI recovery | Connection-step preflight failure has no Back path to correct source/interface selection | Root cause fixed; modern wizard exposes tested correction path |
| NIAC-DEF-034 | Low | CT lifecycle | Stale `eth0.200` remains after returning CT304 to access-interface mode | Fixed; exact stale subinterface removed and absent after CT304 reboot |
| NIAC-DEF-035 | High | Route MIB | Route synthesis invents next hops as `destination[0:2].200.1`, ignoring the actual fabric and hard-coding the demo VLAN octet | Root cause fixed; MIB-II next-hop regression passes; live gate pending |
| NIAC-DEF-036 | High | Demo routing | Demo static routes point through the tester-facing interface while WAN/site devices lack a complete routed interface and return-route model; runtime ignores the selected egress | Catalog rebuilt with nine /30 links and peer-owned routes; NIAC/live gates pending |
| NIAC-DEF-037 | Medium | Frontend tooling | `npm ci` warns that deprecated `i18next-parser` and `mktemp` do not support required Node 26 and pulls deprecated transitive packages | Baseline reproduced; migrate tooling |
| NIAC-DEF-038 | High | License foundation | Hardware fingerprint generation is not actually cached; parallel full-suite load can make consecutive managers derive different fallback identifiers, lose persisted activation, and panic the NIAC lifecycle test after an unchecked nil state | Full-suite reproduced; isolated test passes 10/10; fix belongs in `foundation` |
| NIAC-DEF-039 | High | DHCP configuration | Omitting DHCP materializes a default scope and can unexpectedly answer clients | Root cause fixed; absent DHCP remains disabled and only one demo scope is compiled |
| NIAC-DEF-040 | Medium | Walk loading | A repeated walk path is parsed once per community/agent, multiplying startup work and warnings | Root cause fixed; walk files are loaded once and shared deterministically |
| NIAC-DEF-041 | Medium | Walk parser | Unsigned walk values can overflow Go integer conversions and corrupt counters/indexes | Root cause fixed with bounded 32-bit parsing tests |
| NIAC-DEF-042 | Low | Walk parser | Valid net-snmp end-of-MIB markers are logged as malformed walk lines | Root cause fixed; terminal markers are recognized without warnings |
| NIAC-DEF-043 | High | SNMP authorization | Missing v2c community silently exposes `public`, including on v3-only devices | Root cause fixed; v2c requires explicit communities and v3-only devices expose no v2c agent |
| NIAC-DEF-044 | High | Built-in templates | Permissive YAML let shipped templates drift from the actual schema | Root cause fixed; all built-ins and integration fixtures validate under strict parsing |
| NIAC-DEF-045 | High | Discovery topology | Passive endpoints are advertised as LLDP/CDP neighbors, producing invented topology edges | Root cause fixed with `fdb_only`; endpoints remain in bridge FDB without neighbor advertisements |
| NIAC-DEF-046 | High | MIB-II | Native RFC 1213 coverage omits mandatory IP, ICMP, TCP, UDP, EGP, and SNMP objects or serves static counters | TCP lifecycle scalars and dynamic tcpConnTable, per-VLAN fragment reassembly, packet-backed IF/bridge counters, and authored link-problem MIB state implemented; CT304/CyberScope acceptance remains open |
| NIAC-DEF-047 | Medium | Device identity | Demo MACs use local-admin prefixes despite an embedded IEEE assignment registry | Root cause fixed with `vendor`/`mac_suffix` allocation and canonical round-trip support |
| NIAC-DEF-048 | Medium | Demo persona | Access points replay a Meraki switch walk while claiming to be APs | Root cause fixed using the only genuine AP walk; modern AP captures remain a catalog-data gap |
| NIAC-DEF-049 | Low | E2E tooling | `make test-e2e-install` installs only Chromium although the default suite also requires WebKit | Root cause fixed; install target now matches the two-browser default suite |
| NIAC-DEF-050 | Low | Markdown tooling | `lint-md` scans `ui/node_modules` and exposes a large pre-existing documentation-lint backlog | Reproduced; tooling/backlog fix pending |
| NIAC-DEF-051 | High | Routed VLAN runtime | Virtual device VLAN metadata enabled the legacy untagged-frame drop before routed-fabric DHCP/SNMP dispatch | Root cause fixed by scoping the legacy guard to non-fabric playback; regression and live DHCP/SNMP proof pass |

### Required MIB-II completion work

The implementation now supplies stateful behavior for the corresponding
objects and problem signals. The remaining acceptance work is live validation
against CyberScope and explicit coverage of event sources that the simulator
does not currently generate:

- TCP listener and connection lifecycle, input/output segments, resets, and
  state transitions are supplied by the bounded per-device flow tracker and
  dynamic tcpConnTable publication. Retransmission counters remain zero unless
  the simulator emits a retransmission event.
- IPv4 fragmentation/reassembly and associated request, success, and failure
  counters are supplied by the bounded per-VLAN reassembly engine.
- Per-interface and bridge-port octet/packet counters are packet-authoritative
  and share the forwarding attribution path; authored speed, duplex, admin,
  oper, and alias state is reflected consistently across IF-MIB, IF-X, and
  EtherLike groups.
- Error/discard counters, dynamic FDB learning, and STP topology-change
  counters remain truthful zero because NIAC has no packet-loss/error injector,
  physical ingress-port event source, or BPDU event source yet. These are
  explicit capability gaps, not fabricated telemetry.
- CyberScope/Link-Live acceptance must still confirm the resulting topology,
  counters, and authored problem conditions on the GitHub-built CT304 artifact.

## CT304 live acceptance — 2026-07-21

The deployed Linux build is the GitHub Actions `Build (linux-amd64)` artifact
from workflow run `29870817687`, built from the pull-request merge commit
reported by the binary as `8304494`.
The deployed binary SHA-256 is
`7d82d0e02b9d2f82d6f0af8eed373104f4f80e990c0557b68fb9fa8742c11cb0` and
`/__version` reports commit `8304494`, Go `1.26.5`, and a non-empty UI hash.
CT304 runs `/usr/bin/niac daemon --attachment-policy eth0=access:200` with
`demo-multisite-modern.yaml` from the managed `/var/lib/niac/configs` root;
the simulation was started through the daemon API with attachment `cyberscope`
and reports 135 devices.

- Proxmox `net0` is `tag=200`; the bridge reports `200 PVID Egress Untagged`.
- The container has no IPv4 address or VLAN subinterface on `eth0`.
- Both systemd units are active after a CT reboot and the simulation reports 135 devices.
- An isolated VLAN-200 test endpoint received DHCP `10.254.200.100` from the sole
  responder `10.254.200.1`; no other VLAN was attached.
- Independent Net-SNMP v2c MIB-II queries succeeded for COS, EVT, EHV, and LON;
  v3 authPriv succeeded for EVT. `ifNumber=53`, system identity, IP, TCP, and
  UDP counters returned. NIAC reported 62 SNMP queries, 2 DHCP requests, and
  zero errors during the test.
- A VLAN-200 namespace probe queried `LAB-EDGE-R1`, `COS-CORE-R1`,
  `EVT-CORE-R1`, `EHV-CORE-R1`, and `LON-CORE-R1` with the authored
  `NetAllyDemo` community. The probe was removed after the test.
- The temporary endpoint namespace and its bridge port were removed after testing.

## First implementation slice

The first fix sequence is intentionally dependency-ordered:

1. Add reproductions for NIAC-DEF-001, NIAC-DEF-003, and NIAC-DEF-004.
2. Fix strict YAML-to-runtime SNMP preservation before removing the temporary
   community fallback.
3. Carry attachment mode/VLAN into runtime and enforce the wire-frame matrix.
4. Make DHCP and every routed protocol consume one compiled response context.
5. Add final-frame TX observation and an external SNMP client fixture.

That sequence attacks the known CyberScope failures while also creating the
test harness needed to diagnose later topology defects without another blind
live-discovery loop.
