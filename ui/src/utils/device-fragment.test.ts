import { describe, expect, it } from 'vitest';
import { findDeviceFragment, spliceDeviceFragment } from './device-fragment';

// A config that carries the things a naive splice destroys: comments above and
// inside a device, a trailing device after the target, blank lines, and a
// top-level key after the list.
const CONFIG = `# NIAC demo topology
devices:
  # the edge router
  - name: api-router
    type: router
    ips:
      - "10.10.0.1"
    snmp_agent:
      community: "public" # not the default on purpose

  - name: core-switch
    type: switch
    ips:
      - "10.10.0.2"

segments:
  - vlan: 10
`;

describe('findDeviceFragment', () => {
  it('returns the selected device dedented, so the editor shows a standalone document', () => {
    const fragment = findDeviceFragment(CONFIG, 'api-router');
    expect(fragment).not.toBeNull();
    expect(fragment?.text).toBe(
      `name: api-router
type: router
ips:
  - "10.10.0.1"
snmp_agent:
  community: "public" # not the default on purpose
`,
    );
  });

  it('keeps a comment that sits inside the device', () => {
    expect(findDeviceFragment(CONFIG, 'api-router')?.text).toContain(
      '# not the default on purpose',
    );
  });

  it('stops at the next device rather than swallowing it', () => {
    expect(findDeviceFragment(CONFIG, 'api-router')?.text).not.toContain('core-switch');
  });

  it('finds the last device without running into the next top-level key', () => {
    const fragment = findDeviceFragment(CONFIG, 'core-switch');
    expect(fragment?.text).toBe(
      `name: core-switch
type: switch
ips:
  - "10.10.0.2"
`,
    );
  });

  it('returns null for a device the config does not have', () => {
    expect(findDeviceFragment(CONFIG, 'nope')).toBeNull();
  });

  it('returns null when the config has no devices list at all', () => {
    expect(findDeviceFragment('segments:\n  - vlan: 10\n', 'api-router')).toBeNull();
  });

  it('returns null for YAML that does not parse, rather than guessing', () => {
    expect(findDeviceFragment('devices:\n  - name: [unclosed\n', 'api-router')).toBeNull();
  });
});

describe('spliceDeviceFragment', () => {
  it('replaces only the selected device, leaving every other byte identical', () => {
    const fragment = findDeviceFragment(CONFIG, 'api-router');
    if (!fragment) {
      throw new Error('fixture regressed: api-router not found');
    }
    const next = spliceDeviceFragment(CONFIG, fragment, 'name: api-router\ntype: firewall\n');

    expect(next).toContain('# NIAC demo topology');
    expect(next).toContain('  # the edge router');
    expect(next).toContain('type: firewall');
    expect(next).not.toContain('10.10.0.1');
    // Everything after the edited device survives byte for byte.
    expect(next.slice(next.indexOf('  - name: core-switch'))).toBe(
      CONFIG.slice(CONFIG.indexOf('  - name: core-switch')),
    );
  });

  it('re-indents the fragment to the depth the list uses', () => {
    const fragment = findDeviceFragment(CONFIG, 'core-switch');
    if (!fragment) {
      throw new Error('fixture regressed: core-switch not found');
    }
    const next = spliceDeviceFragment(
      CONFIG,
      fragment,
      'name: core-switch\nips:\n  - "10.0.0.9"\n',
    );

    expect(next).toContain('  - name: core-switch\n    ips:\n      - "10.0.0.9"\n');
  });

  it('round-trips unchanged when the fragment is put back as-is', () => {
    for (const name of ['api-router', 'core-switch']) {
      const fragment = findDeviceFragment(CONFIG, name);
      if (!fragment) {
        throw new Error(`fixture regressed: ${name} not found`);
      }
      expect(spliceDeviceFragment(CONFIG, fragment, fragment.text)).toBe(CONFIG);
    }
  });
});
