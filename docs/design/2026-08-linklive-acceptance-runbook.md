# Link-Live acceptance runbook

**Status:** Active
**Covers:** program ledger M4 (`2026-08-niac-program-execution-ledger.md`)
**Script:** `scripts/lab/acceptance.sh`

Runs a presentation pack end to end: generate it the way the product does,
start it on its own trunk VLAN, discover it with a CyberScope, upload to
Link-Live, and compare the rendered result against authored truth.

## The acceptance bar

A run is complete only when a _fresh_ Link-Live analysis of a _UI/API-generated_
config shows **zero actionable findings**. None of these count on their own:

- green unit tests
- a successful upload, or `0 buffered` on the tester
- matching device and link _counts_
- a previous analysis that predates any generator, MIB, packet or comparator change

Any change to the generator, emitted MIBs, packet path or comparator
**invalidates every earlier live result** and requires a fresh final-binary run.

`scripts/lab/acceptance.sh` generates through `POST /api/v1/scenario/generate` —
the same endpoint the wizard calls — and hands that exact file to both the daemon
and the comparator. A hand-tuned YAML is a known-good oracle, never the product.

## Physical VLAN assignment

Deployment identity, deliberately not stored inside a portable pack.

| VLAN | Pack | Purpose |
| --- | --- | --- |
| 200 | hospital | presentation |
| 201 | warehouse | presentation |
| 202 | manufacturing | presentation |
| 203 | campus | presentation |
| 204 | retail | presentation |
| 205 | service-provider | presentation |
| 299 | enterprise-scale | stress only — scale and responsiveness, not map readability |

## One-time CT304 setup

Trunk the container's NIC and let the daemon accept those tags. The allowlist is
operator configuration; a scenario cannot ask for an interface or a physical VLAN.

```bash
# On pvm01. QUOTE the value so the semicolons survive the shell.
pct set 304 -net0 "name=eth0,bridge=vmbr0,trunks=200;201;202;203;204;205;299,type=veth"

# vmbr0 must carry each tag on the bridge itself and on the uplink.
for v in 200 201 202 203 204 205 299; do
  bridge vlan add dev vmbr0 vid $v self
  bridge vlan add dev nic0  vid $v
done

# A live `bridge vlan add` on the veth is not enough — reboot reapplies
# the trunk list to veth304i0 cleanly.
pct reboot 304
```

Daemon side — one capture owner per interface, allowlisted tags:

```bash
niac daemon --attachment-policy 'eth0=trunk:200,201,202,203,204,205,299'
```

Tester rig on pvm01, one sub-interface per VLAN under test:

```bash
ip link add link vmbr0 name vmbr0.201 type vlan id 201
ip addr add 10.20.201.250/24 dev vmbr0.201 && ip link set vmbr0.201 up
```

## Per-pack loop

```bash
export NIAC_URL=https://10.44.40.22:8445
export NIAC_API_TOKEN=...            # required on a non-loopback bind

scripts/lab/acceptance.sh warehouse   # generate + start, prints the metadata
# ... run Discovery on the CyberScope and upload ...
scripts/lab/acceptance.sh --compare warehouse <unit-mac>
```

Order matters on the tester: **clear Discovery results before scanning.** A stale
inventory replays the previous build's vendor/OUI cadence and produces
`name-conflict` / `type-conflict` findings that have nothing to do with the pack
under test. If the capture VLAN cannot reach Link-Live, finish and queue
Discovery first, then switch to an outbound profile to flush — switching
profiles resets the test port, so never do it mid-scan.

## Findings triage

Every finding kind the comparator can emit, and where to look first.

### Device identity

| Finding | Usual cause | First check |
| --- | --- | --- |
| `missing-device` | device never answered, or discovery ran before the session was up | session running on the right VLAN; SNMP reachable from the tester sub-interface |
| `unexpected-device` | stale tester inventory, or management-subnet nodes leaked in | clear Discovery and rerun; confirm the scan started on the pack's VLAN, not the outbound profile |
| `name-conflict` | walk `sysName` overriding the authored name, or two devices sharing a walk | authored name wins by design — confirm the walk's sysName is being skipped |
| `type-conflict` | profile `sysObjectID` does not map to the expected class | the device's profile in `profiles_catalog.go`; `layer3-switch` may legitimately render as Router or Switch |
| `address-conflict` | device answering on an address the pack did not author | duplicate IP across concurrently running sessions — VLANs isolate, but check the pack itself |
| `problem-conflict` | tester flagged a device-level problem the pack did not author | usually real; read the tester's own problem text |

### Links and topology

| Finding | Usual cause | First check |
| --- | --- | --- |
| `missing-link` | neighbour tables not synthesized, or the peer never discovered | `trunk_ports` on both ends; the comparator matches by MAC pair in either direction |
| `unexpected-link` | CDP/LLDP synthesis produced a placeholder peer identity | the affected interface's neighbour and bridge data — this is the classic false-triangle |
| `port-conflict` | one-sided rendered port, or a `left / right` pair being read as one side | already normalized; a surviving finding is usually a genuinely wrong `remote_interface` |
| `vlan-conflict` | routed link authored with an access VLAN | routed firewall/L3 links should carry LLDP/CDP but no bridge or FDB state |

### Interface telemetry

| Finding | Usual cause | First check |
| --- | --- | --- |
| `missing-interface` | authored interface absent from the walk-backed inventory | if the device has a walk or mapped agent, the walk owns the inventory and defaults are not synthesized |
| `unexpected-interface` | tester saw an interface the pack did not author | only reported for explicitly authored inventories; a hit here means the walk and the YAML disagree |
| `interface-status-conflict` | admin/oper status drift, or a behavior timeline mid-transition | whether a fault phase was active during the scan |
| `interface-speed-conflict` | authored speed disagrees with the walk's `ifSpeed` | walk-backed speed wins on the wire; fix the authored value |
| `interface-duplex-conflict` | duplex compared on an interface that has no duplex | logical and `ieee80211` interfaces are exempt; a hit means an ethernet interface really differs |
| `interface-mtu-conflict` | authored MTU differs from IF-MIB | jumbo-frame mismatch on uplinks is the common one |
| `interface-utilization-conflict` | first poll, or behavior-driven traffic | utilization is only compared once the device has produced a sample; behavior-driven interfaces are expected to move |
| `interface-error-conflict` / `interface-discard-conflict` | fault injection running during the scan, or counters not authored | whether a behavior timeline was mid-phase |
| `interface-problem-conflict` | tester flagged an interface problem not authored | usually real |

## What to record per run

Branch and commit, package version and UI hash, deployed host, session and VLAN
binding, tester identity, Link-Live analysis ID, comparator output, and any
unresolved finding. `--compare` writes `.lab/<pack>.report.json`; keep it with
the analysis ID — a report without its analysis ID cannot be re-checked later.
