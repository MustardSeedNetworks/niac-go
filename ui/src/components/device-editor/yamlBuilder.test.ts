/**
 * Tests for the device-editor YAML preview builder.
 *
 * Every appender is a chain of "emit this key only when the field is set", so
 * the branches that matter are present / absent / present-but-disabled. These
 * assert the emitted lines rather than merely that a string came back — a
 * dropped or misspelled key is exactly the defect a length check would miss.
 */

import { describe, expect, it } from 'vitest';
import { parse as parseYaml } from 'yaml';
import type { Device } from '../../api/types';
import { buildYamlPreview } from './yamlBuilder';

const base: Device = { hostname: 'sw1', mac: '00:11:22:33:44:55' };

/** The preview is YAML, so parse it rather than string-matching where possible. */
function previewDevice(device: Device): Record<string, unknown> {
  const parsed = parseYaml(buildYamlPreview(device)) as {
    devices: Record<string, unknown>[];
  };
  const first = parsed.devices[0];
  if (!first) throw new Error('preview produced no device');
  return first;
}

describe('escaping', () => {
  // The preview was built by string concatenation, interpolating values
  // straight into double-quoted scalars. A value carrying a quote, a
  // backslash or a newline produced a document that would not parse -- and
  // every one of these is a value a user can type into the device editor.
  it('survives a double quote in a value', () => {
    expect(previewDevice({ ...base, hostname: 'a"b' }).name).toBe('a"b');
  });

  it('survives a backslash in a value', () => {
    expect(previewDevice({ ...base, hostname: 'a\\b' }).name).toBe('a\\b');
  });

  it('survives a newline in a value', () => {
    const out = previewDevice({
      ...base,
      ftp: { enabled: true, welcomeBanner: 'line one\nline two' } as Device['ftp'],
    });

    expect((out.ftp as { welcome_banner: string }).welcome_banner).toBe('line one\nline two');
  });

  it('survives a colon-space in a value, which would otherwise start a mapping', () => {
    expect(previewDevice({ ...base, hostname: 'a: b' }).name).toBe('a: b');
  });

  it('quotes a value that would otherwise parse as a number', () => {
    expect(previewDevice({ ...base, mac: '0011' }).mac).toBe('0011');
  });
});

describe('base device lines', () => {
  it('emits only name and mac for a minimal device', () => {
    // `name`, not `hostname`: the daemon's Device struct has no hostname field.
    expect(previewDevice(base)).toEqual({ name: 'sw1', mac: '00:11:22:33:44:55' });
  });

  it('folds a scalar ip into ips, which is the only address field the daemon has', () => {
    const out = previewDevice({
      ...base,
      type: 'switch' as Device['type'],
      ip: '10.0.0.1',
      ips: ['10.0.0.1', '10.0.0.2'],
    });

    expect(out.type).toBe('switch');
    expect(out).not.toHaveProperty('ip');
    // Deduped: a device carrying the same address in both fields lists it once.
    expect(out.ips).toEqual(['10.0.0.1', '10.0.0.2']);
  });

  it('maps a UI device type onto the daemon spelling', () => {
    // The UI says access_point; the daemon's oneof says access-point.
    expect(previewDevice({ ...base, type: 'access_point' as Device['type'] }).type).toBe(
      'access-point',
    );
  });

  it('omits a type the daemon does not accept rather than failing its validation', () => {
    expect(previewDevice({ ...base, type: 'unknown' as Device['type'] })).not.toHaveProperty(
      'type',
    );
  });

  it('omits an empty ips array', () => {
    expect(previewDevice({ ...base, ips: [] })).not.toHaveProperty('ips');
  });
});

describe('interfaces', () => {
  it('is omitted when interfaceDetails is absent or empty', () => {
    expect(previewDevice(base)).not.toHaveProperty('interfaces');
    expect(previewDevice({ ...base, interfaceDetails: [] })).not.toHaveProperty('interfaces');
  });

  it('emits every populated interface attribute', () => {
    const out = previewDevice({
      ...base,
      interfaceDetails: [
        {
          name: 'Gi0/1',
          speed: 1000,
          duplex: 'full',
          adminStatus: 'up',
          operStatus: 'up',
          description: 'uplink',
        },
      ] as Device['interfaceDetails'],
    });

    expect(out.interfaces).toEqual([
      {
        name: 'Gi0/1',
        speed: 1000,
        duplex: 'full',
        admin_status: 'up',
        oper_status: 'up',
        description: 'uplink',
      },
    ]);
  });

  it('omits a zero speed but keeps the interface', () => {
    // speed is gated on `> 0`, not on presence: 0 is "unknown", not "0 Mbps".
    const out = previewDevice({
      ...base,
      interfaceDetails: [{ name: 'Gi0/1', speed: 0 }] as Device['interfaceDetails'],
    });

    expect(out.interfaces).toEqual([{ name: 'Gi0/1' }]);
  });
});

describe('snmp_agent', () => {
  it('is omitted entirely when absent', () => {
    expect(previewDevice(base)).not.toHaveProperty('snmp_agent');
  });

  it('emits the block even when every sub-field is empty', () => {
    const out = previewDevice({ ...base, snmpAgent: {} as Device['snmpAgent'] });
    expect(out).toHaveProperty('snmp_agent');
    expect(out.snmp_agent).toBeNull();
  });

  it('emits scalar fields, walk_files and add_mibs under the daemon key names', () => {
    const out = previewDevice({
      ...base,
      snmpAgent: {
        community: 'public',
        sysName: 'core',
        walkFile: 'a.walk',
        walkFiles: ['a.walk', 'b.walk'],
        addMibs: [{ oid: '1.3.6.1', type: 'STRING', value: 'x' }],
      } as Device['snmpAgent'],
    });

    expect(out.snmp_agent).toEqual({
      community: 'public',
      sysname: 'core',
      walk_file: 'a.walk',
      walk_files: ['a.walk', 'b.walk'],
      add_mibs: [{ oid: '1.3.6.1', type: 'STRING', value: 'x' }],
    });
  });

  it('omits empty walk_files and add_mibs arrays', () => {
    const out = previewDevice({
      ...base,
      snmpAgent: { community: 'public', walkFiles: [], addMibs: [] } as Device['snmpAgent'],
    });

    expect(out.snmp_agent).toEqual({ community: 'public' });
  });
});

describe('protocols gated on an enabled flag', () => {
  const cases: {
    key: string;
    enabled: Partial<Device>;
    disabled: Partial<Device>;
    expected: Record<string, unknown>;
  }[] = [
    {
      key: 'lldp',
      enabled: { lldp: { enabled: true, systemDescription: 'desc' } as Device['lldp'] },
      disabled: { lldp: { enabled: false, systemDescription: 'desc' } as Device['lldp'] },
      expected: { enabled: true, system_description: 'desc' },
    },
    {
      key: 'cdp',
      enabled: { cdp: { enabled: true, platform: 'cat9k' } as Device['cdp'] },
      disabled: { cdp: { enabled: false } as Device['cdp'] },
      expected: { enabled: true, platform: 'cat9k' },
    },
    {
      key: 'stp',
      enabled: { stp: { enabled: true, bridgePriority: 4096 } as Device['stp'] },
      disabled: { stp: { enabled: false } as Device['stp'] },
      expected: { enabled: true, bridge_priority: 4096 },
    },
    {
      key: 'http',
      enabled: { http: { enabled: true, serverName: 'nginx' } as Device['http'] },
      disabled: { http: { enabled: false } as Device['http'] },
      expected: { enabled: true, server_name: 'nginx' },
    },
    {
      key: 'ftp',
      enabled: { ftp: { enabled: true, welcomeBanner: 'hi' } as Device['ftp'] },
      disabled: { ftp: { enabled: false } as Device['ftp'] },
      expected: { enabled: true, welcome_banner: 'hi' },
    },
    {
      key: 'netbios',
      enabled: {
        netbios: { enabled: true, name: 'PC1', workgroup: 'WG' } as Device['netbios'],
      },
      disabled: { netbios: { enabled: false } as Device['netbios'] },
      expected: { enabled: true, name: 'PC1', workgroup: 'WG' },
    },
  ];

  for (const { key, enabled, disabled, expected } of cases) {
    it(`${key}: emitted when enabled`, () => {
      expect(previewDevice({ ...base, ...enabled })[key]).toEqual(expected);
    });

    it(`${key}: omitted when disabled`, () => {
      expect(previewDevice({ ...base, ...disabled })).not.toHaveProperty(key);
    });

    it(`${key}: omitted when absent`, () => {
      expect(previewDevice(base)).not.toHaveProperty(key);
    });
  }

  it('emits only the enabled flag when optional sub-fields are unset', () => {
    const out = previewDevice({ ...base, lldp: { enabled: true } as Device['lldp'] });
    expect(out.lldp).toEqual({ enabled: true });
  });

  it('stp emits a zero bridge_priority, which is a real value', () => {
    // Gated on `!== undefined`, so priority 0 must survive.
    const out = previewDevice({
      ...base,
      stp: { enabled: true, bridgePriority: 0 } as Device['stp'],
    });
    expect(out.stp).toEqual({ enabled: true, bridge_priority: 0 });
  });
});

describe('protocols gated on presence only', () => {
  it('dhcp emits each populated option', () => {
    const out = previewDevice({
      ...base,
      dhcp: {
        subnetMask: '255.255.255.0',
        router: '10.0.0.1',
        domainNameServer: '10.0.0.53',
      } as Device['dhcp'],
    });

    expect(out.dhcp).toEqual({
      subnet_mask: '255.255.255.0',
      router: '10.0.0.1',
      domain_name_server: '10.0.0.53',
    });
  });

  it('dhcp is present but empty when it has no options', () => {
    expect(previewDevice({ ...base, dhcp: {} as Device['dhcp'] }).dhcp).toBeNull();
  });

  it('dns emits forward records as name/ip pairs', () => {
    const out = previewDevice({
      ...base,
      dns: {
        forwardRecords: [
          { name: 'a.example', ip: '10.0.0.5' },
          { name: 'b.example', ip: '10.0.0.6' },
        ],
      } as Device['dns'],
    });

    expect(out.dns).toEqual({
      forward_records: [
        { name: 'a.example', ip: '10.0.0.5' },
        { name: 'b.example', ip: '10.0.0.6' },
      ],
    });
  });

  it('dns omits an empty forward_records array', () => {
    expect(previewDevice({ ...base, dns: { forwardRecords: [] } as Device['dns'] }).dns).toBeNull();
  });
});

describe('ordering and failure', () => {
  it('emits snmp_agent before interfaces, following the daemon struct order', () => {
    // The daemon does not care, but a reordering here would be an unreviewed
    // change to a user-visible preview. The order now follows the field order
    // of converter.Device, so the preview reads like the schema it targets.
    const yaml = buildYamlPreview({
      ...base,
      interfaceDetails: [{ name: 'Gi0/1' }] as Device['interfaceDetails'],
      snmpAgent: { community: 'public' } as Device['snmpAgent'],
    });

    expect(yaml.indexOf('snmp_agent:')).toBeLessThan(yaml.indexOf('interfaces:'));
  });

  it('returns the error placeholder instead of throwing', () => {
    // hostname is read unguarded; a null device exercises the catch.
    expect(buildYamlPreview(null as unknown as Device)).toBe('# Error generating YAML preview');
  });
});
