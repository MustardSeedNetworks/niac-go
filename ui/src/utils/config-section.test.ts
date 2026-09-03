import { describe, expect, it } from 'vitest';
import { findConfigSection, spliceConfigSection } from './config-section';

const config = `# clinic branch office
networks:
  - name: clinic-lan
    subnet: 10.20.0.0/24

attachments:
  - name: tester
    connect: clinic-lan

devices:
  - name: clinic-rtr-01
    type: router
    mac: "00:1A:2B:20:00:20"
`;

describe('findConfigSection', () => {
  it('locates a section by its top-level key', () => {
    const section = findConfigSection(config, 'attachments');

    expect(section).not.toBeNull();
    expect(config.slice(section?.start, section?.end)).toBe(
      'attachments:\n  - name: tester\n    connect: clinic-lan\n',
    );
  });

  it('returns null for a key the config does not have', () => {
    expect(findConfigSection(config, 'segments')).toBeNull();
  });

  it('returns null rather than guessing at a range when the config does not parse', () => {
    expect(findConfigSection('devices: [\n  - broken', 'devices')).toBeNull();
  });
});

describe('spliceConfigSection', () => {
  it('preserves every byte outside the replaced section', () => {
    const result = spliceConfigSection(
      config,
      'networks',
      'networks:\n  - name: clinic-lan\n    subnet: 10.20.0.0/24\n    virtual_vlan: 99\n',
    );

    // The comment, the other sections and the quoted MAC all survive: a
    // whole-document re-stringify is what would reformat them.
    expect(result).toContain('# clinic branch office');
    expect(result).toContain('virtual_vlan: 99');
    expect(result).toContain('mac: "00:1A:2B:20:00:20"');
    expect(result).toContain('attachments:\n  - name: tester');
  });

  it('appends a section the config does not have yet', () => {
    const result = spliceConfigSection('devices: []\n', 'networks', 'networks:\n  - name: lab\n');

    expect(result).toBe('devices: []\n\nnetworks:\n  - name: lab\n');
  });

  it('removes the section when given empty content, rather than authoring an empty list', () => {
    const result = spliceConfigSection(config, 'attachments', '');

    // `attachments:` with nothing under it is an explicit empty list, which is
    // not the same config as having no attachments at all. Asserted exactly:
    // checking only that the key is gone cannot tell a clean removal from one
    // that leaves a stray blank line behind.
    expect(result).toBe(
      `# clinic branch office
networks:
  - name: clinic-lan
    subnet: 10.20.0.0/24

devices:
  - name: clinic-rtr-01
    type: router
    mac: "00:1A:2B:20:00:20"
`,
    );
  });

  it('leaves the blank line between sections alone', () => {
    const result = spliceConfigSection(
      config,
      'networks',
      'networks:\n  - name: clinic-lan\n    subnet: 10.20.0.0/24\n',
    );

    expect(result).toBe(config);
  });
});
