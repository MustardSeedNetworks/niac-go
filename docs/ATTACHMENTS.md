# Where testers attach

A tester plugged into a NIAC host should see what it would see plugged into a
real switch port: its own switch as the LLDP neighbour, an address from that
port's VLAN, the right gateway, and its MAC in that switch's forwarding table
on that port. This page explains how a scenario decides where that port is,
and walks through running several testers on one pack.

## Two cables

Every session has two cables, and only one of them moves.

**The real cable** runs from the NIAC host's NIC to a real switch port, or
straight into a tester. The operator approves how it is used with
`niac daemon --attachment-policy` (`direct`, `access:VLAN` or `trunk:VLANS`),
and a session's binding picks one approved policy at start. It is fixed for the
life of the session. A scenario cannot choose a host interface or a wire VLAN.

**The virtual cable** runs, inside the scenario, from whatever arrives on the
NIC to a port on a simulated device. That is the scenario's `attachments`
entry, and it is the only thing that moves.

The two carry separate VLANs:

| | Meaning | Set by | Changes when a tester moves? |
| --- | --- | --- | --- |
| Wire VLAN | the tag the host's frames carry upstream | the operator's attachment policy | No |
| Port VLAN | which network, DHCP scope and gateway the tester lands on | the scenario | Yes |

They never have to match. The generated packs put testers on the data VLAN
(210) whatever wire VLAN the session runs on, and the CT304 presentation packs
run on wire VLANs 200 through 205.

## The attachment pool

An attachment names either a whole network (`connect`) or a pool of free ports
on one device (`at`). Use the pool. A network-scoped attachment makes every
device with an interface on that network a neighbour of the tester, so its
LLDP neighbour is whichever device owns the network's gateway, usually a core
switch several tiers from the cable.

```yaml
attachments:
  - name: cyberscope
    at:
      device: MED-ACC-SW01
      ports:
        - GigabitEthernet1/0/43
        - GigabitEthernet1/0/44
        - GigabitEthernet1/0/45
        - GigabitEthernet1/0/46
    pins:
      - mac: "00:c0:17:00:00:01"
        device: MED-ACC-SW01
        interface: GigabitEthernet1/0/44
```

How a session uses it:

- **Every client gets its own port.** Each distinct source MAC seen on the
  wire takes the first free pool port, in the order clients are first seen,
  and keeps it until the session stops. A client silent for five minutes ages
  out of the client list.
- **Pins come first.** A pinned MAC always gets its pin, and no other client
  takes a pinned port.
- **The port decides the network.** A pool port's VLAN, followed through the
  switch's uplinks to the device that routes that VLAN, picks the DHCP scope
  and gateway. Every port of one pool must land on the same network.
- **The switch reports what is plugged in.** A spare port reads `notconnect`
  (up administratively, down operationally) until a client is placed on it.
  The pool switch's forwarding table then reports that client's MAC on that
  port, and no other device reports it.
- **Only the pool switch speaks discovery at the clients.** It is the one
  device that sends LLDP, CDP, EDP, FDP or STP at the wire. All clients share
  that one wire, so they all hear the same advertisement, which names the port
  of the first client placed.
- **When the pool is full**, a further client gets no port: the client list
  shows it without one, and the daemon logs `Attachment pool full`.

Every generated pack offers a four-port pool (`GigabitEthernet1/0/43` to
`1/0/46`, data VLAN 210) on every access switch, and `TenGigabitEthernet1/0/43`
to `1/0/46` on the servers VLAN on every server switch. Its attachment, named
`cyberscope`, uses the pool on the first site's first access switch.

Author the pool in the wizard's **Networks** step (**Spare ports on one
device**) or in YAML; see the [authoring guide](AUTHORING_GUIDE.md#addressing-networks-interfaces-and-attachments).

## Moving a tester

Moving a client to another port of its pool is a pin, and it happens on the
running session:

- **Web UI:** **Attached clients** on the Simulation page, or **Tester ports**
  in a pool switch's details on the Topology page. Pick the client and a free
  port, then **Move client**.
- **API:** `POST /api/v1/sessions/{id}/pins` with
  `{"mac": "...", "device": "...", "interface": "..."}`. See the
  [REST API](REST_API.md) for the errors it returns.

The pin is written into the scenario file the session runs, so the client stays
on its new port across restarts. The session is not restarted: every other
client keeps its port and lease, the session keeps its binding and wire tag,
and other sessions on the same NIC are not touched. The port the client left
returns to `notconnect`.

A pool covers one switch, so a pin cannot cross to another switch. To put
testers on a different switch, point the attachment's `at` at that switch's
spare ports and restart the session; every tester moves together. A live move
between switches is tracked in
[#2505](https://github.com/MustardSeedNetworks/niac-go/issues/2505).

## Deployment shapes

The pool works the same in all three, because the scenario never sees the wire
tag:

1. **Tester straight into the NIC** (`direct`): untagged both ways.
2. **NIC in an upstream access port** (`access:VLAN`): the switch strips the
   tag, so NIAC sees untagged frames, as in shape 1.
3. **NIC on a trunk, several scenarios** (`trunk:VLANS`): each session owns
   one wire tag. Moving a tester inside one scenario changes that scenario's
   internal network, not its tag, so other scenarios never notice.

Two limits follow from the wire tag being how a frame finds its session:
two testers on the same tag always see the same scenario, and one tester
cannot see two scenarios at once.

## Runbook: three testers on one pack, move one

Runs on the lab: the hospital pack on CT304's wire VLAN 200, testers on pvm01.
It uses only the API and web UI and changes nothing on CT304 itself.

**Status: written 2026-10-04, not yet executed on CT304.** Correct this section
from the first run's evidence.

### 1. Start the pack

From a checkout, with the daemon token for CT304:

```bash
export NIAC_URL=https://10.44.40.22:8445
export NIAC_API_TOKEN=...
scripts/lab/acceptance.sh hospital
```

The session id is `hospital`. Its pool is `MED-ACC-SW01`
`GigabitEthernet1/0/43` to `1/0/46`, on `med-data` (`10.51.210.0/24`).

### 2. Attach three testers

Tester one is the CyberScope on **Wired Profile VLAN 200**; run an AutoTest so
it DHCPs. Testers two and three are network namespaces on pvm01, each with its
own MAC on wire VLAN 200. `vmbr0.200` is a VLAN 200 sub-interface of `vmbr0`;
create it as in the Link-Live runbook's tester rig if it does not exist.

```bash
for n in 2 3; do
  ip netns add tester$n
  ip link add link vmbr0.200 name t$n type macvlan mode bridge
  ip link set t$n address 02:00:00:5a:00:0$n netns tester$n
  ip -n tester$n link set t$n up
  ip netns exec tester$n dhclient -v t$n
done
```

### 3. Check where each landed

```bash
curl -sk -H "Authorization: Bearer $NIAC_API_TOKEN" \
  "$NIAC_URL/api/v1/sessions/hospital/clients" | jq '.[] | {mac, ip, device, interface}'
```

Expect three entries on `MED-ACC-SW01`, on three different ports of
`1/0/43`..`1/0/45`, each with a `10.51.210.x` address. Confirm the switch agrees,
from a tester:

```bash
ip netns exec tester2 snmpwalk -v2c -c NetAllyDemo 10.51.200.21 \
  1.3.6.1.2.1.17.7.1.2.2.1.2
```

Each tester MAC appears exactly once, on its own port's bridge port number. On
the CyberScope, the LLDP neighbour is `MED-ACC-SW01`.

### 4. Move one

Move tester three to `GigabitEthernet1/0/46`, from **Attached clients** or:

```bash
csrf=$(curl -sk -H "Authorization: Bearer $NIAC_API_TOKEN" \
  "$NIAC_URL/api/v1/csrf-token" | jq -r .token)
curl -sk -X POST -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-CSRF-Token: $csrf" -H 'Content-Type: application/json' \
  -d '{"mac":"02:00:00:5a:00:03","device":"MED-ACC-SW01","interface":"GigabitEthernet1/0/46"}' \
  "$NIAC_URL/api/v1/sessions/hospital/pins"
```

Then repeat step 3. Pass criteria:

- tester three is on `GigabitEthernet1/0/46`, and the forwarding table reports
  its MAC there and nowhere else;
- the port it left reads `notconnect` (`ifOperStatus` down);
- testers one and two kept their ports and addresses;
- the session was not restarted: `packets_sent` in
  `GET /api/v1/sessions/hospital/runtime` kept counting up rather than starting
  again from zero.

### 5. Clean up

```bash
for n in 2 3; do ip netns del tester$n; done
scripts/lab/acceptance.sh hospital
```

Re-running the acceptance script regenerates the pack, which drops the pin
written in step 4.
