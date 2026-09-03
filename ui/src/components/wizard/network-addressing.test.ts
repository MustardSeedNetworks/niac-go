import { describe, expect, it } from 'vitest';
import {
  nextFreeAddress,
  parseNetworkModel,
  serializeAttachments,
  serializeNetworks,
  takenAddresses,
} from './network-addressing';

const clinic = `networks:
  - name: clinic-lan
    subnet: 10.20.0.0/24
  - name: clinic-mgmt
    subnet: 10.20.99.0/24
    virtual_vlan: 99

attachments:
  - name: tester
    connect: clinic-lan

devices:
  - name: clinic-rtr-01
    type: router
    ips:
      - "10.20.0.1"
    interfaces:
      - name: GigabitEthernet0/0/1
        network: clinic-lan
        address: 10.20.0.1/24
  - name: clinic-srv-01
    type: server
    ips:
      - "10.20.0.10"
`;

describe('parseNetworkModel', () => {
  it('reads networks, attachments and per-device addressing out of a config', () => {
    const model = parseNetworkModel(clinic);

    expect(model.networks).toEqual([
      { name: 'clinic-lan', subnet: '10.20.0.0/24' },
      { name: 'clinic-mgmt', subnet: '10.20.99.0/24', virtualVlan: 99 },
    ]);
    expect(model.attachments).toEqual([{ name: 'tester', connect: 'clinic-lan' }]);
    expect(model.devices).toEqual([
      {
        device: 'clinic-rtr-01',
        interfaceName: 'GigabitEthernet0/0/1',
        network: 'clinic-lan',
        address: '10.20.0.1/24',
      },
      { device: 'clinic-srv-01', interfaceName: null, network: null, address: null },
    ]);
  });

  it('returns empty lists for a config that does not parse', () => {
    expect(parseNetworkModel('devices: [\n  - broken')).toEqual({
      networks: [],
      attachments: [],
      devices: [],
    });
  });
});

describe('serializeNetworks', () => {
  it('round-trips through the parser', () => {
    const model = parseNetworkModel(clinic);
    expect(parseNetworkModel(serializeNetworks(model.networks)).networks).toEqual(model.networks);
  });

  it('omits an absent VLAN rather than writing zero', () => {
    // `virtual_vlan: 0` is out of the schema's 1..4094 range, so writing it
    // for an untagged network would author a config that fails validation.
    expect(serializeNetworks([{ name: 'lab', subnet: '10.0.0.0/24' }])).toBe(
      'networks:\n  - name: lab\n    subnet: 10.0.0.0/24\n',
    );
  });

  it('returns empty string for no networks, which the splice reads as removal', () => {
    expect(serializeNetworks([])).toBe('');
    expect(serializeAttachments([])).toBe('');
  });
});

describe('nextFreeAddress', () => {
  it('skips the network address and any address already in use', () => {
    expect(nextFreeAddress('10.20.0.0/24', [])).toBe('10.20.0.1/24');
    expect(nextFreeAddress('10.20.0.0/24', ['10.20.0.1/24', '10.20.0.2'])).toBe('10.20.0.3/24');
  });

  it('carries the network prefix length, not a host prefix', () => {
    // The fabric compiler requires an interface address to match its
    // network's prefix length exactly; a /32 is refused at start.
    expect(nextFreeAddress('10.20.0.0/24', [])?.endsWith('/24')).toBe(true);
  });

  it('stops before the broadcast address', () => {
    // /30 holds exactly two usable hosts.
    expect(nextFreeAddress('192.0.2.0/30', ['192.0.2.1', '192.0.2.2'])).toBeNull();
  });

  it('returns null for a malformed subnet rather than a wrong address', () => {
    expect(nextFreeAddress('not-a-subnet', [])).toBeNull();
    expect(nextFreeAddress('10.20.0.0', [])).toBeNull();
    expect(nextFreeAddress('999.0.0.0/24', [])).toBeNull();
  });

  it('handles a base address that is not on its own subnet boundary', () => {
    expect(nextFreeAddress('10.20.0.37/24', [])).toBe('10.20.0.1/24');
  });
});

describe('takenAddresses', () => {
  it('counts bare ips as well as interface prefixes', () => {
    // A device addressed through `ips` occupies the address just as much as
    // one addressed through an interface, so auto-assign must not reissue it.
    expect(takenAddresses(clinic).sort((a, b) => a.localeCompare(b))).toEqual([
      '10.20.0.1',
      '10.20.0.1/24',
      '10.20.0.10',
    ]);
  });
});
