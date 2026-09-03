/**
 * The cross-language half of the device-YAML contract.
 *
 * This test asserts that the UI still produces the committed fixture; a Go
 * test in internal/converter asserts the daemon still parses it. One file,
 * two assertions, so a drift on either side fails a test rather than shipping
 * a config nobody can load.
 *
 * Regenerate deliberately with UPDATE_CONTRACT=1 when the mapping changes on
 * purpose -- then run the Go test, which is what says the daemon agrees.
 */

import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { stringify as stringifyYaml } from 'yaml';
import type { Device } from '../api/device-config-types';
import { toDaemonDevice } from './device-yaml';

const FIXTURE = join(
  import.meta.dirname,
  '../../../internal/converter/testdata/ui_device_contract.yaml',
);

/**
 * Exercises every block the mapper emits. A field missing here is a field the
 * Go side never checks, so this device is deliberately maximal rather than
 * realistic.
 */
const contractDevice: Device = {
  hostname: 'contract-sw1',
  mac: '00:11:22:33:44:55',
  type: 'access_point',
  ip: '10.0.0.1',
  ips: ['10.0.0.1', '10.0.0.2'],
  vlan: 10,
  mapToIp: '10.0.0.9',
  interfaceDetails: [
    {
      name: 'Gi0/1',
      speed: 1000,
      duplex: 'full',
      adminStatus: 'up',
      operStatus: 'up',
      description: 'uplink "quoted" \\ and a: colon',
    },
  ],
  snmpAgent: {
    community: 'public',
    sysName: 'contract-sw1',
    walkFile: 'walks/sw1.walk',
    walkFiles: ['walks/a.walk', 'walks/b.walk'],
    addMibs: [{ oid: '1.3.6.1.2.1.1.1.0', type: 'STRING', value: 'demo' }],
  } as Device['snmpAgent'],
  dhcp: {
    subnetMask: '255.255.255.0',
    router: '10.0.0.254',
    domainNameServer: '10.0.0.53',
  } as Device['dhcp'],
  dns: {
    forwardRecords: [{ name: 'sw1.example.com', ip: '10.0.0.1' }],
  } as Device['dns'],
  lldp: { enabled: true, systemDescription: 'contract lldp' } as Device['lldp'],
  cdp: { enabled: true, platform: 'contract-platform' } as Device['cdp'],
  stp: { enabled: true, bridgePriority: 32768 } as Device['stp'],
  http: { enabled: true, serverName: 'contract-httpd' } as Device['http'],
  ftp: { enabled: true, welcomeBanner: 'line one\nline two' } as Device['ftp'],
  netbios: { enabled: true, name: 'CONTRACTSW1', workgroup: 'LAB' } as Device['netbios'],
};

describe('device YAML contract', () => {
  it('matches the fixture the Go side parses', () => {
    const produced = stringifyYaml({ devices: [toDaemonDevice(contractDevice)] }, { lineWidth: 0 });

    if (process.env.UPDATE_CONTRACT === '1') {
      writeFileSync(FIXTURE, produced);
    }

    expect(produced).toBe(readFileSync(FIXTURE, 'utf8'));
  });

  it('emits no camelCase key, which is what made the old output unloadable', () => {
    const doc = toDaemonDevice(contractDevice);
    const keys: string[] = [];
    const walk = (value: unknown): void => {
      if (Array.isArray(value)) {
        for (const entry of value) walk(entry);
        return;
      }
      if (value && typeof value === 'object') {
        for (const [k, v] of Object.entries(value)) {
          keys.push(k);
          walk(v);
        }
      }
    };
    walk(doc);

    expect(keys.filter((k) => /[A-Z]/.test(k))).toEqual([]);
  });

  // Every field a device-editor section actually lets an operator set (see
  // ui/src/pages/DeviceEditorPage.tsx's visibleSections list and each
  // section's onUpdate calls). A field silently dropped here is a field the
  // operator fills in and loses on save without ever being told -- this is
  // the exact failure mode #P1-9 removed the traffic section for. Assert
  // real values, not just presence, so a mapping that emits the wrong value
  // (not just a missing one) also fails.
  it('round-trips every field the device editor can set', () => {
    const doc = toDaemonDevice(contractDevice);

    expect(doc.name).toBe('contract-sw1');
    expect(doc.type).toBe('access-point');
    expect(doc.mac).toBe('00:11:22:33:44:55');
    expect(doc.ips).toEqual(['10.0.0.1', '10.0.0.2']);

    expect(doc.interfaces).toEqual([
      {
        name: 'Gi0/1',
        speed: 1000,
        duplex: 'full',
        admin_status: 'up',
        oper_status: 'up',
        description: 'uplink "quoted" \\ and a: colon',
      },
    ]);

    expect(doc.snmp_agent).toEqual({
      community: 'public',
      sysname: 'contract-sw1',
      walk_file: 'walks/sw1.walk',
      walk_files: ['walks/a.walk', 'walks/b.walk'],
      add_mibs: [{ oid: '1.3.6.1.2.1.1.1.0', type: 'STRING', value: 'demo' }],
    });

    expect(doc.dhcp).toEqual({
      subnet_mask: '255.255.255.0',
      router: '10.0.0.254',
      domain_name_server: '10.0.0.53',
    });

    expect(doc.dns).toEqual({
      forward_records: [{ name: 'sw1.example.com', ip: '10.0.0.1' }],
    });

    expect(doc.lldp).toEqual({ enabled: true, system_description: 'contract lldp' });
    expect(doc.cdp).toEqual({ enabled: true, platform: 'contract-platform' });
    expect(doc.stp).toEqual({ enabled: true, bridge_priority: 32768 });
    expect(doc.http).toEqual({ enabled: true, server_name: 'contract-httpd' });
    expect(doc.ftp).toEqual({ enabled: true, welcome_banner: 'line one\nline two' });
    expect(doc.netbios).toEqual({ enabled: true, name: 'CONTRACTSW1', workgroup: 'LAB' });
  });

  it('has no device-level traffic concept to drop', () => {
    // Guards against the field being reintroduced on the UI side without a
    // matching daemon key -- see the block comment on toDaemonDevice.
    expect(contractDevice).not.toHaveProperty('traffic');
    expect(toDaemonDevice(contractDevice)).not.toHaveProperty('traffic');
  });
});
