/**
 * Device Library filtering — regression guard for D9.
 *
 * `getDeviceProtocols` re-derived the protocol list by probing per-protocol
 * sub-objects (`snmpAgent`, `lldp`, `cdp`, …). `GET /api/v1/config/devices`
 * returns a *summary* shape that carries none of them — the union of keys
 * across all 75 devices in a real response is:
 *
 *   hostname · interfaceDetails · interfaces · ips · mac · protocols · type · vlan
 *
 * so every predicate was `Boolean(undefined)`, the column read "No protocols"
 * for every device, the CSV export shipped an empty column, and the protocol
 * dropdown — built from the protocols found across devices — rendered with a
 * single option and could never match.
 *
 * The server had already done the work: it sends `protocols: [...]`.
 *
 * These fixtures deliberately use the **summary** shape. A fixture with
 * `snmpAgent` populated would pass against the broken code, which is exactly
 * why this shipped with no test at all.
 */

import { describe, expect, it } from 'vitest';
import type { Device } from '../../api/types';
import { getDeviceProtocols, matchesProtocolFilter } from './deviceFilters';

/** What the list endpoint actually returns — no protocol sub-objects. */
const summaryDevice = {
  hostname: 'LAB-EDGE-R1',
  type: 'router',
  mac: '00:00:0c:00:01:01',
  ips: ['10.254.200.1', '203.0.113.1'],
  interfaces: ['TenGigabitEthernet0/0/0'],
  protocols: ['SNMP', 'DHCP', 'DNS', 'LLDP', 'CDP'],
  vlan: 200,
} as unknown as Device;

const noProtocols = {
  hostname: 'BARE-HOST-01',
  type: 'host',
  mac: '00:00:0c:00:02:01',
  ips: ['10.254.200.50'],
  protocols: [],
} as unknown as Device;

describe('getDeviceProtocols', () => {
  it('reads the protocols the list endpoint already provides', () => {
    expect(getDeviceProtocols(summaryDevice)).toEqual(['SNMP', 'DHCP', 'DNS', 'LLDP', 'CDP']);
  });

  it('returns an empty list for a device that speaks nothing', () => {
    expect(getDeviceProtocols(noProtocols)).toEqual([]);
  });

  it('does not fall over when the field is missing entirely', () => {
    const legacy = { hostname: 'X', type: 'host', mac: '00:00:00:00:00:01' } as unknown as Device;
    expect(getDeviceProtocols(legacy)).toEqual([]);
  });
});

describe('matchesProtocolFilter', () => {
  it('matches a device that speaks the selected protocol', () => {
    expect(matchesProtocolFilter(summaryDevice, 'LLDP')).toBe(true);
  });

  it('rejects a device that does not', () => {
    expect(matchesProtocolFilter(summaryDevice, 'FTP')).toBe(false);
    expect(matchesProtocolFilter(noProtocols, 'SNMP')).toBe(false);
  });

  it('passes everything through when no protocol is selected', () => {
    expect(matchesProtocolFilter(noProtocols, 'all')).toBe(true);
  });
});
