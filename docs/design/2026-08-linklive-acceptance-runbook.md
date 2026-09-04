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

**Make the bridge membership persistent, or it is gone on the next reboot.**
`bridge-vids` in `/etc/network/interfaces` programs the bridge _ports_, not the
bridge itself, and the lab terminates its VLAN sub-interfaces on the bridge. A
reboot on 2026-08-09 wiped every `self` entry and the packs went silent in a way
that reads exactly like a NIAC fault — SNMP timing out on every VLAN while the
daemon was healthy and the sessions were running. Add to the `vmbr0` stanza:

```text
post-up for v in 200 201 202 203 204 205 299; do bridge vlan add dev vmbr0 vid $v self; done
```

Check it with `bridge vlan show dev vmbr0`: if only VLAN 1 is listed, that is why
nothing answers.

Daemon side — one capture owner per interface, allowlisted tags:

```bash
niac daemon --attachment-policy 'eth0=trunk:200,201,202,203,204,205,299'
```

Tester rig on pvm01, one sub-interface per VLAN under test:

```bash
ip link add link vmbr0 name vmbr0.201 type vlan id 201
ip addr add 10.20.201.250/24 dev vmbr0.201 && ip link set vmbr0.201 up
```

## Driving the CyberScope

The unit is `10.44.10.184`. **VNC on 5900 is the only control path** — nothing
else is open, no SSH and no web UI. There is no password.

```bash
# One-time
python3 -m venv ~/vncenv && ~/vncenv/bin/pip install vncdotool
```

Chain every action and the screenshot into **one** `vncdo` invocation. A
separate `capture` call opens a fresh session, receives no framebuffer update,
and writes a black frame that looks exactly like a sleeping screen — this costs
an hour if you do not know it.

```bash
# Wrong: two calls, second one writes black
vncdo -s 10.44.10.184::5900 move 360 640 click 1
vncdo -s 10.44.10.184::5900 capture shot.png

# Right: one session
vncdo -s 10.44.10.184::5900 move 360 640 click 1 pause 2 capture shot.png
```

Screen is 720x1280. Android navigation bar: back `180,1232`, home `360,1232`.
`key ctrl-a` types a literal `a` rather than selecting all — clear a text field
with repeated `key bsp`.

### Capture walkthrough

1. **Home** (`360,1232`) → **AutoTest** (`111,148`) → hamburger (`55,104`) →
   expand the profile list (`451,96`) → **Wired Profile VLAN 200** (`300,481`).
   Wait for DHCP to show `10.254.200.100`; that is the pack's own DHCP server
   answering, and it is the proof the session is live on the right VLAN.
2. **Home** → **Discovery** (`443,367`) → ⋮ (`676,104`) → **Refresh Discovery**
   (`481,104`) → **CLEAR AND RERUN DISCOVERY** (`320,678`).
3. Wait for the count to stop climbing. A good run reaches the pack's device
   count in about five minutes. **If it stalls at a few dozen and never walks
   past the first hop, that is the unit, not NIAC** — see below.
4. ⋮ → **Upload To Link-Live** (`490,392`), set the Analysis Name, dismiss the
   keyboard with `key esc`, then **UPLOAD TO ANALYSIS** (`293,1127`).
5. The capture VLAN has no internet, so the upload queues. Switch AutoTest to
   **EZ Wired Profile** to flush it. Confirm in the **Link-Live** app
   (`111,805`) that the header reads `Link-Live (0 buffered)` and
   `Link-Live is reachable`; anything else means the queue has not drained.

Only one AutoTest profile can be linked at a time. VLAN 200 runs the pack;
EZ Wired (`10.44.10.x`) is the only path out. Never switch profiles mid-scan —
it resets the test port.

### When Discovery stalls

Symptom: the count sticks at a few dozen, the list is all `10.44.x` management
hosts, and `LAB-EDGE-R1` appears with no MAC or vendor — meaning the unit found
it by ARP but never SNMP-walked it, so it never followed the route into the
pack's subnets.

Before suspecting the simulation, prove NIAC from `pvm01`:

```bash
ip link add link vmbr0 name vmbr0.200 type vlan id 200
ip addr add 10.254.200.251/24 dev vmbr0.200 && ip link set vmbr0.200 up
ip route add 10.51.0.0/16 via 10.254.200.1 dev vmbr0.200

snmpget -v2c -c NetAllyDemo 10.254.200.1  1.3.6.1.2.1.1.5.0   # LAB-EDGE-R1
snmpget -v2c -c NetAllyDemo 10.51.200.21  1.3.6.1.2.1.1.5.0   # an access switch
```

If those answer, the simulation is fine. Discovery Settings → SNMP will still
show the communities configured; the stall is unit-side.

**The cure: re-select the AutoTest profile, then clear and rerun.** Drawer →
**AutoTest** (the row, not the chevron) opens the profile list; tap **EZ Wired
Profile** and **START**, then tap **Wired Profile VLAN 200** and **START**
again. Discovery walks the pack on the next clear-and-rerun. This worked twice
on 2026-08-08, each time after the same stall, and nothing else did — SNMP
settings, credentials and Feature Access were all correct throughout.

What a stalled unit is actually doing is worth knowing, because it is the
fastest way to place the blame: it emits **no SNMP at all**. `tcpdump -i vmbr0
'vlan and udp port 161'` on pvm01 captured zero packets across a full
clear-and-rerun; the same filter after the profile bounce showed GetRequests to
`10.254.200.1` within seconds. The stalled unit also loses its VLAN-200 address
— it appears in its own Discovery list under the management IP rather than a
`10.254.200.x` one.

Remove any `vmbr0.<vlan>` sub-interface you added to prove the simulation
before capturing, or it is discovered too and files as an unexpected device.

### Two things that decide whether a capture is usable

**Link-Live computes utilization from two SNMP polls.** Upload straight after a
clear-and-rerun and every interface comes back `util 0`, which the comparator
files as one `interface-utilization-conflict` per interface — 110 of them on
manufacturing, 120 on warehouse. Run **Refresh Discovery** (it keeps the
inventory and takes the second sample), wait for the count to settle, and upload
that. Both went to zero findings unchanged otherwise.

**The tester only sweeps subnets listed under Discovery Settings → Extended
Ranges.** Missing devices are usually a missing range rather than a NIAC fault:
manufacturing's endpoints and servers stayed absent until `10.91.210.0/24` and
`10.91.240.0/24` were added, taking it 40 → 14 → 0. Add each pack's `.210`
(clients) and `.240` (servers) subnets before its first capture — Discovery →
drawer → **Discovery Settings** → **Extended Ranges** → **+**. The ranges for
every pack are in place as of 2026-08-08.

## Link-Live API

Credentials live in `~/.linklive/token.env` as `LINKLIVE_ACCESS_TOKEN`, kept
alive by `~/.linklive/refresh.sh` under launchd so the account's MFA is never
needed again.

Two things that will waste time:

- **Paths are `/v1/...`, not `/api/v1/...`.** The wrong path returns
  `InvalidAuthHeader`, which reads like a credential problem and is not.
- **The API rejects a token with `426 JWTNeedsToBeRefreshed` well before its
  `exp` claim.** The unforced refresh script reports "still fresh" and does
  nothing. Force it.

```bash
bash ~/.linklive/refresh.sh --force
set -a; . ~/.linklive/token.env; set +a

curl -s -H "Authorization: Access $LINKLIVE_ACCESS_TOKEN" \
  https://link-live.com/v1/admin/analysis |
  python3 -c 'import sys,json; rows=json.load(sys.stdin);
rows.sort(key=lambda r: r["created_at"], reverse=True)
[print(r["created_at"], r["_id"], r.get("fileName")) for r in rows[:5]]'
```

The header scheme is `Access`, not `Bearer`.

Then compare against authored truth:

```bash
go run ./tools/linklive-acceptance -config <pack>.yaml -analysis <_id>
```

The runner prints a JSON report and then a plain-text failure line, so the
output is **not** valid JSON as a whole — parse only up to the closing brace.

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

## Start-up failures

Before a discovery is worth running, the session has to actually start. Two
failures seen in the lab:

| Symptom | Cause | Fix |
| --- | --- | --- |
| `preflight` returns `safe: false` with `unknown_attachment` | The generated config declares a logical attachment (`cyberscope` connected to `lab-transit`) and the start request did not name one. A config that declares attachments requires the binding to pick one. | Pass `attachment` in the start request; `acceptance.sh` reads it from the pack's `attachmentName`. |
| `POST /api/v1/simulation` returns a bare `500 simulation_start_failed` | Deliberate: the error may contain config-derived secrets, so neither the response nor the log carries detail. | Run `preflight` with the same payload — it reports the real diagnostic. |

`niac-demo-lab.service` requests attachment mode `access` on the same interface
the presentation packs hold as `trunk`. Mixing modes on one interface is
rejected by design, so that unit fails whenever the trunk sessions are up. It
predates the concurrent-VLAN model and should be disabled rather than debugged.

## Findings triage

Every finding kind the comparator can emit, and where to look first.

### Device identity

| Finding | Usual cause | First check |
| --- | --- | --- |
| `missing-device` | device never answered, or discovery ran before the session was up | session running on the right VLAN; SNMP reachable from the tester sub-interface |
| `unexpected-device` | stale tester inventory, or management-subnet nodes leaked in | clear Discovery and rerun; confirm the scan started on the pack's VLAN, not the outbound profile |
| `name-conflict` | walk `sysName` overriding the authored name, or two devices sharing a walk | authored name wins by design — confirm the walk's sysName is being skipped. A device shown as a **bare IP** is usually discovery timing rather than a defect: probe it directly before believing it (`snmpget` for an appliance, an NBSTAT node-status query **bound to source port 137** for a Windows endpoint — an unbound probe sees nothing even when the responder is working). |
| `interface-problem-conflict` / `problem-conflict` | Link-Live flagged something the pack did not author | expected where the pack authors congestion — an interface at or above 80% is _meant_ to warn and no longer files as a finding (hospital's imaging uplinks). Anywhere else it is real. |
| `type-conflict` | profile `sysObjectID` does not map to the expected class | the device's profile in `profiles_catalog.go`; `layer3-switch` may legitimately render as Router or Switch. An **endpoint** filed as `SNMP Agent` is not a finding and is no longer reported — a clinical or industrial appliance that answers SNMP is managed gear, and Link-Live is right to say so. A **switch** filed as `SNMP Agent` still is a finding. |
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
| `interface-utilization-conflict` | first poll, or behavior-driven traffic | utilization is only compared once the device has produced a sample; behavior-driven interfaces are expected to move. Only **switch and router** ports are compared: Link-Live takes no utilization sample from a leaf node — endpoint, server, wireless controller, **access point** — even though ours all serve `ifHCInOctets` (measured: `MED-DNS01`, `MED-WLC01`, `MED-PUMP-B01-F01-02` all return Counter64 and Link-Live still reports none; on the 2026-08-08 hospital capture all five interfaces of all 30 APs came back `util 0`, wired uplink included, while the switch port facing that uplink read 71.57%). |
| `interface-error-conflict` / `interface-discard-conflict` | fault injection running during the scan, or counters not authored | whether a behavior timeline was mid-phase |
| `interface-problem-conflict` | tester flagged an interface problem not authored | usually real |

## What to record per run

Branch and commit, package version and UI hash, deployed host, session and VLAN
binding, tester identity, Link-Live analysis ID, comparator output, and any
unresolved finding. `--compare` writes `.lab/<pack>.report.json`; keep it with
the analysis ID — a report without its analysis ID cannot be re-checked later.
