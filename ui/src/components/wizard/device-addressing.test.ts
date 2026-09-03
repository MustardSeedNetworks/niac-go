import { describe, expect, it } from 'vitest';
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
