# SNMP walk workflow

NIAC can serve captured SNMP OID/value pairs from walk files. Use captured data
only when the built-in simulated MIBs do not cover the required device profile.

## Create a walk

Capture the narrowest useful subtree. Never commit credentials or a raw
customer walk.

```bash
snmpwalk -v2c -c "$SNMP_COMMUNITY" device.example.com \
  .1.3.6.1.2.1 > device.walk
```

For basic discovery, retain the system, interface, IP, bridge, and entity MIB
trees that the target workflow actually queries. Preserve numeric OIDs when
possible so loading does not depend on locally installed MIB text files.

## Sanitize and validate

Before a walk enters the catalog:

1. Replace hostnames, addresses, contact details, locations, asset tags, and
   serial numbers with obviously synthetic values.
2. Remove credentials and private enterprise data unrelated to the profile.
3. Confirm every retained line has an OID, type, and value.
4. Run the profile in NIAC and walk it through the same SNMP version used by
   the acceptance client.
5. Compare returned indexes across IF-MIB, BRIDGE-MIB, LLDP-MIB, and any
   vendor-specific tables used by the scenario.

Use `niac snmp-walk sanitize` and `niac snmp-walk validate` for the supported
sanitization and validation workflow. See `niac snmp-walk --help` for current
flags.

## Configure a device

Reference the committed catalog path from the device SNMP configuration. Paths
are resolved by the configuration loader and must remain inside the managed
configuration roots.

```yaml
devices:
  - name: access-switch-1
    type: switch
    ip_addresses:
      - 192.0.2.10
    snmp_config:
      community: public
      walk_file: examples/walks/cisco/example.walk
```

## Contribution acceptance

A contributed walk must:

- identify its vendor, model family, and software family;
- contain no customer or credential data;
- include only OIDs needed for the documented profile;
- pass the walk validator and repository tests;
- produce consistent interface and bridge indexes; and
- include an acceptance note naming the client and queried MIBs.

Walks are simulation fixtures, not proof that every device feature is modeled.
Document omissions explicitly.
