# Config coverage audit

Generated 2026-05-06 against `examples/` (45 YAML files). Companion to
[SURFACE_MATRIX.md](./SURFACE_MATRIX.md).

## Method

Each file scanned for substring tokens that map to schema features, then
features aggregated. Source: `internal/config/config.go` struct types
cross-referenced against `internal/protocols/*.go` implementations.

## Feature → coverage

| Feature                  | Covered by                                     |
|--------------------------|------------------------------------------------|
| ARP                      | most configs (implicit in any device with ips) |
| CDP                      | layer2/cdp-only.yaml, layer2/all-discovery-protocols.yaml, vendors/cisco-network.yaml, complete-kitchen-sink.yaml, … |
| LLDP                     | layer2/lldp-only.yaml, layer2/all-discovery-protocols.yaml, multi-vendor configs |
| EDP                      | layer2/edp-only.yaml, layer2/all-discovery-protocols.yaml |
| FDP                      | layer2/fdp-only.yaml, layer2/all-discovery-protocols.yaml |
| STP                      | layer2/stp-bridge.yaml, vendors/dell-network.yaml, complete-kitchen-sink.yaml |
| DHCPv4                   | dhcp/dhcpv4-{simple,advanced}.yaml, network/dual-stack.yaml |
| DHCPv6                   | dhcp/dhcpv4-advanced.yaml (mixed), network/dhcpv6-config.yaml, network/dual-stack.yaml |
| DNS                      | services/dns-server.yaml, complete-kitchen-sink.yaml |
| HTTP                     | services/http-server.yaml, complete-kitchen-sink.yaml |
| FTP                      | services/ftp-server.yaml, complete-kitchen-sink.yaml |
| NetBIOS                  | services/netbios-server.yaml, complete-kitchen-sink.yaml |
| ICMP                     | network/icmp-config.yaml, complete-kitchen-sink.yaml |
| ICMPv6                   | network/icmpv6-config.yaml, complete-kitchen-sink.yaml |
| IPv6 (dual / single)     | network/ipv6-only.yaml, network/dual-stack.yaml |
| iperf3                   | traffic-generation.yaml, complete-kitchen-sink.yaml |
| Traffic gen (announce/ping/random) | traffic-generation.yaml |
| OS fingerprint           | health-check-network.yaml |
| Fault injection          | fault-injection.yaml, snmp/snmp-complete-network.yaml |
| SNMP agent               | snmp/snmp-agent-basic.yaml, snmp/snmp-multiple-communities.yaml, vendors/* |
| SNMP traps               | snmp/snmp-traps-all.yaml |
| **SNMP FDB injection**   | **complete-kitchen-sink.yaml (added 2026-05-06)** |
| Capture playback         | complete-kitchen-sink.yaml |
| VLANs / trunks           | topology/* + complete-kitchen-sink.yaml |
| Vendor-flavored          | vendors/{cisco,arista,juniper,aruba,extreme,foundry,dell,fortinet,paloalto,meraki,multi-vendor-campus}.yaml |

## Walks

`examples/device_walks_sanitized/` ships **575 walk files** across 15
vendor folders (3com, arista, aruba, brocade, cisco, dell, extreme,
fortinet, hp, hpe, huawei, juniper, meraki, mikrotik, plus `misc`).
Each major vendor folder has multiple model-specific walks suitable for
SNMP agent simulation.

`mapping.json` maps walk filenames to the original device metadata
(model, sysDescr, etc.).

The kitchen-sink config exercises walks from cisco, extreme, brocade,
and misc; the `enterprise-campus-walks.yaml` config does broader
multi-vendor walk coverage.

## What changed in this audit

Before: 21 of 25 feature-tokens covered by the kitchen-sink (FDB,
non-OID-walk-warnings, and a couple false positives).

After: kitchen-sink now also exercises `dot1d_fdb_table` and
`dot1q_fdb_table` (SNMP bridge MIB FDB injection). The remaining
"missing" tokens from my heuristic scan are false positives — see
SURFACE_MATRIX for the actual schema.

## Known behavior nuance

`capture_playbacks:` declared in a config file auto-starts on the legacy
CLI path (`niac <iface> <cfg>`). On the daemon path, playback is on-demand:
the YAML field is parsed but the playback engine has to be started
explicitly via `POST /api/v1/replay`. This is by design — daemon mode is
intended for interactive control — but may surprise users coming from
the CLI flow. Document in the daemon section of CONTRIBUTING.md if it
keeps tripping people up.

## Smoke verification (2026-05-06)

Daemon + kitchen-sink end-to-end:

```text
✓ POST /api/v1/simulation         → 9 devices loaded on lo0
✓ GET  /api/v1/devices            → 9
✓ GET  /api/v1/topology           → nodes + links populated
✓ GET  /api/v1/stats              → packets_sent: 32, packets_received: 51
✓ GET  /api/v1/neighbors          → CDP discovery firing (cisco→brocade visible)
✓ POST /api/v1/walk/validate      → parsed cisco-c3850 walk, found 66 advisory issues
✓ DELETE /api/v1/simulation       → clean stop
```

Walk validator's 66 "issues" on `niac-cisco-c3850-48p.walk` are all
advisory (severity: info, mostly "string value not quoted" — a stylistic
recommendation). Walks load and serve correctly.
