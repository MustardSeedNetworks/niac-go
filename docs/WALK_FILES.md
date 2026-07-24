# Walk file catalog

Walk files are sanitized SNMP response fixtures used to model a specific device
profile. The catalog supplements NIAC's simulated MIBs; it does not replace the
authoritative runtime state projected by those MIBs.

## Catalog layout

Store fixtures below `examples/walks/<vendor>/`. Use lowercase names that
identify the vendor, model family, and software family:

```text
examples/walks/
  cisco/
    catalyst-9300-ios-xe.walk
  juniper/
    ex-series-junos.walk
```

Do not encode a customer name, site, address, or serial number in a path.

## Source classes

- **Captured:** collected from hardware or a vendor virtual appliance, then
  sanitized and reviewed.
- **Synthetic:** generated from a checked-in profile and reproducible from the
  repository tooling.

The catalog manifest must identify the source class. A synthetic fixture must
not be described as a hardware capture.

## Required content

Include only the trees needed by the profile. Discovery-oriented switch
profiles commonly need:

- system and entity identity;
- IF-MIB and IF-MIB high-capacity counters;
- IP address and route projections;
- BRIDGE-MIB forwarding and bridge-port mappings;
- LLDP-MIB or CDP neighbor data; and
- vendor-specific identity fields required by the target client.

Indexes are a single contract. An interface index referenced by bridge,
neighbor, or vendor tables must resolve to the same interface in IF-MIB.

## Generate and validate

Enumerate supported generators and build a fixture with the CLI:

```bash
niac snmp-walk generate --list-devices
niac snmp-walk generate \
  --device cisco-catalyst-9300 \
  --hostname access-switch-1 \
  --output examples/walks/cisco/catalyst-9300-ios-xe.walk
niac snmp-walk validate \
  examples/walks/cisco/catalyst-9300-ios-xe.walk
```

Use `niac snmp-walk --help` as the source for current flags. Do not copy old
command examples into scripts without verifying them against the installed
binary.

## Sanitization

Run the supported sanitizer before review:

```bash
niac snmp-walk sanitize \
  --input captured.walk \
  --output sanitized.walk
```

Then inspect the result for hostnames, addresses, contacts, locations, asset
tags, serial numbers, community strings, and private enterprise values. A
sanitizer result is evidence, not a substitute for review.

## Configuration

Reference a catalog fixture from the device SNMP configuration:

```yaml
devices:
  - name: access-switch-1
    type: switch
    ip_addresses:
      - 192.0.2.10
    snmp_config:
      community: public
      walk_file: examples/walks/cisco/catalyst-9300-ios-xe.walk
```

The loader rejects unresolved paths and paths outside managed configuration
roots.

## Acceptance

Before merging a fixture:

1. validate its format;
2. verify sanitization;
3. start the profile in NIAC;
4. query it with the target SNMP client;
5. compare identity, interface, bridge, neighbor, and route tables; and
6. record any intentionally omitted trees.

For the routed CyberScope scenario, also confirm that Link-Live reports the
expected nearest switch, port, and VLAN from the same indexed data.

See [SNMP walk workflow](SNMP_WALKS.md) for capture and contribution guidance.
