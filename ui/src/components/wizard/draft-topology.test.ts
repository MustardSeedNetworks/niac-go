import { describe, expect, it } from 'vitest';
import { parseDraftTopology } from './draft-topology';

describe('parseDraftTopology', () => {
  it('projects devices, reciprocal parallel links, interfaces, and saved positions', () => {
    const model = parseDraftTopology(`
devices:
  - name: core-1
    type: switch
    ips: [192.0.2.1]
    properties: { topology_x: "120.5", topology_y: "240" }
    interfaces:
      - { name: Ethernet1/1, type: ethernet, speed: 10000, duplex: full, in_utilization: 20, out_utilization: 32 }
      - { name: Ethernet1/2, type: ethernet, speed: 10000, duplex: full }
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: dist-1, remote_interface: Ethernet1/49, vlans: [200], fdb_only: true }
      - { interface: Ethernet1/2, remote_device: dist-1, remote_interface: Ethernet1/50, vlans: [210] }
  - name: dist-1
    type: switch
    ips: [192.0.2.2]
    interfaces:
      - { name: Ethernet1/49, type: ethernet }
      - { name: Ethernet1/50, type: ethernet }
      - { name: Ethernet1/51, type: ethernet }
    trunk_ports:
      - { interface: Ethernet1/49, remote_device: core-1, remote_interface: Ethernet1/1, vlans: [200], fdb_only: true }
      - { interface: Ethernet1/50, remote_device: core-1, remote_interface: Ethernet1/2, vlans: [210] }
`);
    expect(model.devices).toHaveLength(2);
    expect(model.links).toHaveLength(2);
    expect(model.links[0]).toMatchObject({
      source: 'core-1',
      target: 'dist-1',
      sourceInterface: 'Ethernet1/1',
      targetInterface: 'Ethernet1/49',
      utilizationPercent: 32,
      fdbOnly: true,
      reciprocal: true,
    });
    expect(model.positions['core-1']).toEqual({ x: 120.5, y: 240 });
    expect(model.interfaces['core-1'][0].occupied).toBe(true);
    expect(model.interfaces['dist-1'][2].occupied).toBe(false);
    expect(model.segmented).toBe(false);
  });

  it('marks one-sided link declarations as read-only', () => {
    const model = parseDraftTopology(`
devices:
  - name: core-1
    type: switch
    interfaces: [{ name: Ethernet1/1, type: ethernet }]
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: dist-1, remote_interface: Ethernet1/49, vlans: [200] }
  - name: dist-1
    type: switch
    interfaces: [{ name: Ethernet1/49, type: ethernet }]
`);
    expect(model.links).toHaveLength(1);
    expect(model.links[0].reciprocal).toBe(false);
    expect(model.interfaces['dist-1'][0].occupied).toBe(true);
  });

  it('marks reciprocal links with asymmetric properties as read-only', () => {
    const model = parseDraftTopology(`
devices:
  - name: core-1
    type: switch
    interfaces: [{ name: Ethernet1/1, type: ethernet }]
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: dist-1, remote_interface: Ethernet1/49, vlans: [200] }
  - name: dist-1
    type: switch
    interfaces: [{ name: Ethernet1/49, type: ethernet }]
    trunk_ports:
      - { interface: Ethernet1/49, remote_device: core-1, remote_interface: Ethernet1/1, vlans: [200, 210] }
`);
    expect(model.links).toHaveLength(1);
    expect(model.links[0].reciprocal).toBe(false);
  });

  it('projects devices from segmented drafts', () => {
    const model = parseDraftTopology(`
segments:
  - vlan: 200
    devices:
      - name: access-1
        type: switch
`);
    expect(model.devices.map((device) => device.name)).toEqual(['access-1']);
    expect(model.segmentByDevice).toEqual({ 'access-1': 0 });
    expect(model.segmented).toBe(true);
  });

  it('keeps config-backed segments reachable through the YAML view', () => {
    const model = parseDraftTopology(`
segments:
  - tag: 200
    config: campus.yaml
`);
    expect(model.devices).toEqual([]);
    expect(model.segmented).toBe(true);
    expect(model.configBacked).toBe(true);
  });

  it('treats a valid device-less document as an editable empty topology', () => {
    expect(parseDraftTopology('{}')).toEqual({
      devices: [],
      links: [],
      interfaces: {},
      positions: {},
      segmentByDevice: {},
      segmented: false,
      configBacked: false,
    });
    expect(parseDraftTopology('devices:')).toMatchObject({ devices: [], segmented: false });
  });

  it('uses non-empty segments instead of inactive top-level devices', () => {
    const model = parseDraftTopology(`
devices:
  - { name: inactive-device, type: host }
segments:
  - tag: 200
    devices:
      - { name: active-device, type: switch }
`);
    expect(model.devices.map((device) => device.name)).toEqual(['active-device']);
    expect(model.segmented).toBe(true);
  });

  it('rejects malformed YAML and unsupported document shapes', () => {
    expect(() => parseDraftTopology('devices: [')).toThrow();
    expect(() => parseDraftTopology('devices: nope')).toThrow('devices');
  });
});
