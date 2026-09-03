# Authoring a NIAC scenario

This guide is for writing a NIAC configuration by hand. It covers the whole
vocabulary, the rules that are not obvious from the field names, and one
complete scenario you can copy.

Every field's own rule lives in `docs/schemas/niac.schema.json`, generated from
the Go structs in `internal/converter/types.go`. That schema is the single
source: this guide, the YAML editor's completion and the device editor's
per-field help all read from it. If a rule here disagrees with the schema, the
schema is right and this guide is a bug.

## Contents

- [Before you start](#before-you-start)
- [The shortest useful config](#the-shortest-useful-config)
- [The complete example](#the-complete-example)
- [Identity: name, type, mac and vendor](#identity-name-type-mac-and-vendor)
- [Addressing: networks, interfaces and attachments](#addressing-networks-interfaces-and-attachments)
- [Links and topology](#links-and-topology)
- [Services](#services)
- [Behaviour timelines](#behaviour-timelines)
- [Rules that cost people a round trip](#rules-that-cost-people-a-round-trip)
- [Validating your work](#validating-your-work)

## Before you start

Point your editor at the schema and it will complete field names and closed
vocabularies as you type. Put this on the first line of your config:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/MustardSeedNetworks/niac-go/main/docs/schemas/niac.schema.json
```

The loader is strict: an unknown key is an error, not a warning. A misspelled
field does not silently do nothing, which is deliberate — silently dropping a
key is how an authored value goes missing at replay time.

## The shortest useful config

One device that answers ping and SNMP:

```yaml
devices:
  - name: lab-sw-01
    type: switch
    mac: "00:1A:2B:00:00:01"
    ips:
      - "10.0.0.2"
    icmp:
      enabled: true
    snmp_agent:
      enabled: true
      community: public
      sysname: lab-sw-01
      sysdescr: "Lab switch"
```

That is a complete, valid scenario. Everything below adds realism to it.

## The complete example

The scenario below is a clinic branch office: one router, one access switch,
two servers, two workstations and an access point. It is kept in the repository
as [`docs/examples/clinic-branch.yaml`](examples/clinic-branch.yaml) and is
validated on every CI run, so it never drifts from the loader.

It is annotated here section by section. Read it top to bottom and you have
seen most of the vocabulary in use.

### Networks

Networks declare the routed IPv4 space. `virtual_vlan` tags a network's frames
when it is carried on a trunk; omit it for an untagged network.

```yaml
networks:
  - name: clinic-lan
    subnet: 10.20.0.0/24
  - name: clinic-mgmt
    subnet: 10.20.99.0/24
    virtual_vlan: 99
```

### Attachments

An attachment is how a network reaches the host. `name` labels the attachment —
it is what a session's binding selects at start time — and `connect` names the
network being exposed. These are easy to read backwards.

```yaml
attachments:
  - name: tester
    connect: clinic-lan
```

### The router

Note `vendor` with `mac_suffix` rather than an explicit `mac`, the `ttl` block
written as an object, and the DHCP pool sitting inside `clinic-lan`.

```yaml
devices:
  - name: clinic-rtr-01
    type: router
    vendor: cisco
    mac_suffix: 1
    ips:
      - "10.20.0.1"
      - "10.20.99.1"
    ttl:
      ttl: 255
    icmp:
      enabled: true
    lldp:
      enabled: true
      system_description: "Cisco ISR 1100 clinic edge"
      port_description: "GigabitEthernet0/0/1"
    snmp_agent:
      enabled: true
      community: msn_public
      sysname: clinic-rtr-01
      sysdescr: "Cisco IOS XE Software, ISR1100 17.9.4a"
      syslocation: "Clinic branch, comms closet"
      syscontact: "netops@clinic.example"
    dhcp:
      pool_start: "10.20.0.100"
      pool_end: "10.20.0.199"
      subnet_mask: "255.255.255.0"
      router: "10.20.0.1"
      domain_name_server: "10.20.0.10"
    dns:
      forward_records:
        - name: "rtr.clinic.example"
          ip: "10.20.0.1"
    interfaces:
      - name: GigabitEthernet0/0/1
        speed: 1000
        duplex: full
        admin_status: up
        oper_status: up
        network: clinic-lan
        address: 10.20.0.1/24
        description: "to clinic-sw-01 Gi1/0/24"
    trunk_ports:
      - interface: GigabitEthernet0/0/1
        vlans: [99]
        native_vlan: 1
        remote_device: clinic-sw-01
        remote_interface: GigabitEthernet1/0/24
```

### A host-facing switch port

The uplink above is a trunk: it carries VLAN 99 tagged and VLAN 1 untagged. A
port facing a single host is an access port — a `native_vlan` and no `vlans`:

```yaml
devices:
  - name: clinic-sw-01
    type: switch
    vendor: cisco
    mac_suffix: 2
    trunk_ports:
      - interface: GigabitEthernet1/0/1
        native_vlan: 1
        remote_device: clinic-srv-01
        remote_interface: eth0
```

### A server with SSH

The SSH password is never written in the config. `password_env` names an
environment variable, which must be set in the daemon's environment or the
device will not start.

```yaml
devices:
  - name: clinic-srv-01
    type: server
    vendor: dell
    mac_suffix: 16
    ips:
      - "10.20.0.10"
    http:
      enabled: true
      server_name: "nginx/1.24.0"
    ssh:
      enabled: true
      username: "admin"
      password_env: "NIAC_SSH_PASSWORD"
```

## Identity: name, type, mac and vendor

`name` is the device's identity everywhere else in the file — behaviour phases,
`trunk_ports.remote_device` and the topology graph all refer to it. It is
read-only once the device exists: the daemon takes a device's name from the URL,
so renaming through the editor is not supported.

`type` selects the persona: which MIBs it serves, which icon topology draws and
how a scanner classifies it. The vocabulary hyphenates:

```text
router  switch  layer3-switch  ap  access-point  firewall
server  host  workstation  iot  printer  voip-phone
```

`access_point` and `voip_phone` — with underscores — are not in it and are
rejected.

A device needs a MAC address, from exactly one of two sources:

- `mac: "00:1A:2B:20:00:20"` states it outright.
- `vendor: cisco` derives it from a vendor OUI.

They are mutually exclusive; setting both is an error. When using `vendor`, set
`mac_suffix` as well — it supplies the low 24 bits. Without it, every device of
the same vendor collides on the same address, ending `:00:00:00`.

## Addressing: networks, interfaces and attachments

There are two ways to give a device an address, and they are not interchangeable:

- `ips: ["10.20.0.10"]` — bare addresses, for a device you are not modelling
  port by port.
- `interfaces[].address: 10.20.0.1/24` — a prefix, on a modelled port.

An interface address is always written as a prefix. When the interface also
names a `network`, the address must fall inside that network's subnet **and
carry the same prefix length**: a `/32` host address on a `/24` network is
refused.

Attachments bind a network to the host. Preflight decides whether the host
interface may actually carry it, and reports `attachment_policy_denied` or
`host_interface_unavailable`. `niac validate` has no host binding and cannot
check those, which is why a file can validate and still fail preflight.

## Links and topology

There is no `links:` section. A topology edge exists because a `trunk_port`
names a `remote_device`. There are three shapes:

| Shape | How to author it | Used for |
| --- | --- | --- |
| Trunk | `vlans: [...]`, optionally `native_vlan` | Switch-to-switch, switch-to-router uplinks |
| Access | `native_vlan: N`, no `vlans` | A port facing a single host |
| Routed | neither, on a `router`, `firewall` or `layer3-switch` | Router-to-router links |

A port on a switch that declares neither tagged nor native VLANs is warned
about, because it is almost always an unfinished access port.

`port_channels` bundle member interfaces into a LAG but draw no edge on their
own. The edge comes from a `trunk_port` whose `interface` is
`port-channel<id>`.

## Services

Each service is a block on the device. They are independent — a device can
serve any combination.

| Block | Serves | Note |
| --- | --- | --- |
| `snmp_agent` | SNMP v1/v2c | `walk_file` supplies every OID not overridden here |
| `snmpv3` | SNMP v3 USM | Independent of `snmp_agent` |
| `dhcp` | DHCP | Pool must sit inside a declared routed network |
| `dns` | DNS A and PTR records | Records key the address as `ip` |
| `http`, `ftp`, `ssh` | Application listeners | Banners are what a scanner identifies |
| `lldp`, `cdp`, `edp`, `fdp` | Discovery advertisement | Overrides the fleet-wide default |
| `mdns`, `netbios` | Name advertisement | How a host gets named without SNMP |
| `icmp`, `icmpv6` | Ping, neighbour and router discovery | `icmp.enabled` is what makes a device pingable |
| `stp` | Spanning tree | Bridge priority drives the root election |
| `iperf3`, `reflector` | Throughput testing | `reflector` has no enable flag — presence enables it |
| `os_fingerprint` | TCP/IP stack shaping | For fingerprinting scanners |
| `syslog` | RFC 5424 state messages | Needs at least one receiver |

Fleet-wide discovery defaults go at the top level and a device's own block
overrides them:

```yaml
discovery_protocols:
  lldp:
    enabled: true
    interval: 30
```

## Behaviour timelines

A timeline drives traffic and faults on a schedule after the session starts.
Phase offsets are relative to the timeline, not to the previous phase, and
`repeat_count` is finite — there is no infinite repeat.

```yaml
behavior_timelines:
  - name: morning-load
    repeat_count: 3
    phases:
      - name: busy
        start_offset_ms: 0
        duration_ms: 60000
        traffic:
          - device: clinic-sw-01
            interface: GigabitEthernet1/0/24
            utilization: 80
        faults:
          - device: clinic-sw-01
            interface: GigabitEthernet1/0/6
            type: fcs_errors
            value: 5
```

Faults here are interface-scoped: they raise SNMP counters. They do not take a
service out — `fcs_errors`, `packet_discards`, `interface_errors` and
`high_utilization` are the whole vocabulary.

## Rules that cost people a round trip

These are the ones that are not guessable from the field name.

| Rule | Wrong | Right |
| --- | --- | --- |
| `ttl` is an object, not an integer | `ttl: 64` | `ttl:` then `ttl: 64` |
| DNS records key the address as `ip` | `address: 10.0.0.1` | `ip: 10.0.0.1` |
| `mac` and `vendor` are exclusive | both set | one or the other |
| `vendor` needs a per-device suffix | `vendor: cisco` alone, twice | add `mac_suffix: 1`, `2` |
| Interface addresses are prefixes | `address: 10.20.0.1` | `address: 10.20.0.1/24` |
| Prefix length must match the network | `/32` on a `/24` network | `/24` |
| Access-point type is hyphenated | `type: access_point` | `type: ap` |
| Attachment `name` is a label | `name: clinic-lan` | `name: tester`, `connect: clinic-lan` |
| SSH needs a username and an env var | `enabled: true` alone | add `username` and `password_env` |
| An empty block is a configured service | `dhcpv6: {}` | omit the block |
| NetBIOS names are short | a 20-character name | 15 characters or fewer |
| Only one capture playback | two entries | one |
| `segments` and `devices` are exclusive | both populated | one or the other |

The `dhcpv6: {}` case is worth calling out. An empty block is not "no DHCPv6" —
it is a DHCPv6 server that picks up the default lifetimes on the next load, so
the device acquires a service it was never authored with. Omit the block.

## Validating your work

```bash
niac validate my-scenario.yaml
```

Validation runs the same fabric compile the daemon runs, so a file that
validates will load. What it cannot check is anything that depends on the host
binding — attachment policy and host interface availability — because it has no
binding. Those are reported by preflight instead:

```bash
curl -sk -X POST https://localhost:8445/api/v1/simulation/preflight \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  --data-binary @request.json
```

Both surfaces report the same diagnostic codes for anything config-scoped, so a
code you see in one means the same thing in the other.
