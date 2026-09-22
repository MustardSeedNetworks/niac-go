import { describe, expect, it } from 'vitest';
import { parse, stringify } from 'yaml';
import { setDeviceAddress } from './device-addressing';
import { parseNetworkModel } from './network-addressing';

const config = `# clinic branch office
networks:
  - name: clinic-lan
    subnet: 10.20.0.0/24

devices:
  - name: clinic-rtr-01
    type: router
    mac: "00:1A:2B:20:00:20"
    interfaces:
      - name: GigabitEthernet0/0/1
        speed: 1000

  - name: clinic-srv-01
    type: server
    mac: "00:1A:2B:20:00:21"
`;

describe('setDeviceAddress', () => {
  it('addresses an existing interface without disturbing its other fields', () => {
    const result = setDeviceAddress(config, 'clinic-rtr-01', 'clinic-lan', '10.20.0.1/24');
    const model = parseNetworkModel(result);

    expect(model.devices[0]).toEqual({
      device: 'clinic-rtr-01',
      interfaceName: 'GigabitEthernet0/0/1',
      network: 'clinic-lan',
      address: '10.20.0.1/24',
      ports: [{ name: 'GigabitEthernet0/0/1', vlan: null, occupied: false }],
    });
    expect(result).toContain('speed: 1000');
  });

  it('creates an interface for a device that has none', () => {
    const result = setDeviceAddress(config, 'clinic-srv-01', 'clinic-lan', '10.20.0.10/24');
    const model = parseNetworkModel(result);

    expect(model.devices[1]).toEqual({
      device: 'clinic-srv-01',
      interfaceName: 'Ethernet1/1',
      network: 'clinic-lan',
      address: '10.20.0.10/24',
      ports: [{ name: 'Ethernet1/1', vlan: null, occupied: false }],
    });
  });

  it('leaves every other device and the surrounding file untouched', () => {
    const result = setDeviceAddress(config, 'clinic-rtr-01', 'clinic-lan', '10.20.0.1/24');

    // A whole-document round-trip is what would reformat these.
    expect(result).toContain('# clinic branch office');
    expect(result).toContain('mac: "00:1A:2B:20:00:21"');
    expect(result).toContain('subnet: 10.20.0.0/24');
    expect(parseNetworkModel(result).devices).toHaveLength(2);
  });

  it('returns the config unchanged for a device it cannot find', () => {
    expect(setDeviceAddress(config, 'no-such-device', 'clinic-lan', '10.20.0.9/24')).toBe(config);
  });

  it('is idempotent when applied twice with the same address', () => {
    const once = setDeviceAddress(config, 'clinic-rtr-01', 'clinic-lan', '10.20.0.1/24');
    expect(setDeviceAddress(once, 'clinic-rtr-01', 'clinic-lan', '10.20.0.1/24')).toBe(once);
  });
});

it('edits the addressed AP uplink without putting an address on its radios', () => {
  const input = `devices:
  - name: MED-AP01
    interfaces:
      - name: Dot11Radio0
        type: ieee80211
      - name: mGigabitEthernet0
        type: ethernet
        network: MED-mgmt
        address: 10.51.200.101/24
`;
  const updated = setDeviceAddress(input, 'MED-AP01', 'MED-clients', '10.51.20.5/24');
  const model = parseNetworkModel(updated);
  expect(model.devices[0]).toMatchObject({
    interfaceName: 'mGigabitEthernet0',
    network: 'MED-clients',
    address: '10.51.20.5/24',
  });
  expect(updated).toContain(
    'name: Dot11Radio0\n        type: ieee80211\n      - name: mGigabitEthernet0',
  );
  expect(updated).not.toContain('10.51.200.101');
});

it.each([
  {
    label: 'an unaddressed wired port after radios',
    interfaces: [
      { name: 'Dot11Radio0', type: 'ieee80211' },
      { name: 'eth0', type: 'ethernet' },
    ],
    target: 'eth0',
  },
  {
    label: 'an assigned network after an unused wired port',
    interfaces: [
      { name: 'eth0', type: 'ethernet' },
      { name: 'mgmt0', type: 'ethernet', network: 'old-mgmt' },
    ],
    target: 'mgmt0',
  },
  {
    label: 'a newly appended wired port when only radios exist',
    interfaces: [
      { name: 'Dot11Radio0', type: 'ieee80211' },
      { name: 'Dot11Radio1', type: 'ieee80211' },
    ],
    target: 'Ethernet1/1',
  },
])('places the address on $label', ({ interfaces, target }) => {
  const original = { devices: [{ name: 'ap-1', interfaces }] };
  const updated = setDeviceAddress(stringify(original), 'ap-1', 'mgmt', '10.20.0.2/24');
  const expected = interfaces.map((port) =>
    port.name === target ? { ...port, network: 'mgmt', address: '10.20.0.2/24' } : port,
  );
  if (!interfaces.some((port) => port.name === target)) {
    expected.push({ name: target, type: 'ethernet', network: 'mgmt', address: '10.20.0.2/24' });
  }
  const parsed: unknown = parse(updated);
  expect(parsed).toEqual({ devices: [{ name: 'ap-1', interfaces: expected }] });
  expect(parseNetworkModel(updated).devices[0]).toMatchObject({
    interfaceName: target,
    network: 'mgmt',
    address: '10.20.0.2/24',
  });
});
