import { describe, expect, it } from 'vitest';
import { parseNetworkModel, serializeAttachments } from './network-addressing';

const poolYaml = `networks:
  - name: med-data
    subnet: 10.51.210.0/24
    virtual_vlan: 210
attachments:
  - name: cyberscope
    at:
      device: MED-ACC-SW01
      ports:
        - GigabitEthernet1/0/20
        - GigabitEthernet1/0/21
    pins:
      - mac: 00:c0:17:aa:bb:cc
        device: MED-ACC-SW01
        interface: GigabitEthernet1/0/21
`;

/**
 * AP-1. An attachment now says where in the scenario a tester appears — a pool
 * of free ports on one switch — instead of naming a whole network. The wizard
 * has to read and write both forms, or authoring one in YAML loses it on the
 * next edit.
 */
describe('attachment pools in the network model', () => {
  it('reads the pool and its pins', () => {
    const model = parseNetworkModel(poolYaml);

    expect(model.attachments).toHaveLength(1);
    const attachment = model.attachments[0];
    expect(attachment?.connect).toBe('');
    expect(attachment?.at).toEqual({
      device: 'MED-ACC-SW01',
      ports: ['GigabitEthernet1/0/20', 'GigabitEthernet1/0/21'],
    });
    expect(attachment?.pins).toEqual([
      {
        mac: '00:c0:17:aa:bb:cc',
        device: 'MED-ACC-SW01',
        interface: 'GigabitEthernet1/0/21',
      },
    ]);
  });

  it('writes a pool back as YAML the loader reads', () => {
    const written = serializeAttachments(parseNetworkModel(poolYaml).attachments);

    expect(written).toContain('at:');
    expect(written).toContain('device: MED-ACC-SW01');
    expect(written).toContain('- GigabitEthernet1/0/20');
    expect(written).toContain('mac: "00:c0:17:aa:bb:cc"');
    expect(written).not.toContain('connect:');
    expect(parseNetworkModel(`${written}`).attachments[0]?.at?.ports).toHaveLength(2);
  });

  it('keeps writing the network form for a network attachment', () => {
    const written = serializeAttachments([{ name: 'cyberscope', connect: 'lab-transit' }]);

    expect(written).toContain('connect: lab-transit');
    expect(written).not.toContain('at:');
    expect(written).not.toContain('pins:');
  });

  // An attachment with no ports is not a pool; writing an empty `ports:` list
  // would make the daemon refuse the file the wizard just produced.
  it('omits an empty pool rather than writing an empty port list', () => {
    const written = serializeAttachments([
      { name: 'cyberscope', connect: '', at: { device: 'SW1', ports: [] } },
    ]);

    expect(written).not.toContain('ports:');
  });
});
