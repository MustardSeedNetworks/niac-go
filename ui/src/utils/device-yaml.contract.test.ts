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
  // Present on the device and deliberately NOT emitted: the daemon has no
  // device-level traffic key, so emitting it made the document unloadable.
  traffic: { enabled: true } as Device['traffic'],
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

  it('omits the traffic block the daemon cannot accept', () => {
    expect(toDaemonDevice(contractDevice)).not.toHaveProperty('traffic');
  });
});
