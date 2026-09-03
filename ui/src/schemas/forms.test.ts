/**
 * forms.test.ts — pins the identity validation the device editor relies on.
 *
 * The editor's model is the authored YAML document, so these are the daemon's
 * own key names, and the rules mirror the Go side: `validate:"mac"`,
 * `validate:"ip"`, and ErrDeviceMACSourceConflict for the mac-XOR-vendor
 * choice. The schema ignores every other authored block so it can be run
 * against a whole device.
 */

import * as v from 'valibot';
import { describe, expect, it } from 'vitest';
import { AuthoredDeviceSchema, ErrorInjectionSchema } from './forms';

/** First validation message for a given top-level field, or undefined. */
function errorFor(input: unknown, field: string): string | undefined {
  const result = v.safeParse(AuthoredDeviceSchema, input);
  if (result.success) return undefined;
  const issue = result.issues.find((i) => i.path?.[0]?.key === field);
  return issue?.message;
}

describe('AuthoredDeviceSchema', () => {
  it('accepts a device identified by MAC', () => {
    const result = v.safeParse(AuthoredDeviceSchema, {
      name: 'core-switch-01',
      mac: '00:1A:2B:3C:4D:5E',
      ips: ['10.0.0.1'],
    });
    expect(result.success).toBe(true);
  });

  it('accepts a device identified by vendor', () => {
    const result = v.safeParse(AuthoredDeviceSchema, { name: 'sw1', vendor: 'cisco' });
    expect(result.success).toBe(true);
  });

  it('accepts IPv6 addresses, which the clinic scenario authors', () => {
    expect(
      v.safeParse(AuthoredDeviceSchema, {
        name: 'sw1',
        mac: '00:1A:2B:3C:4D:5E',
        ips: ['2001:db8::7'],
      }).success,
    ).toBe(true);
  });

  it('ignores the other authored blocks (runs against a whole device)', () => {
    const result = v.safeParse(AuthoredDeviceSchema, {
      name: 'sw1',
      mac: '00:1A:2B:3C:4D:5E',
      type: 'switch',
      snmp_agent: { community: 'public' },
      interfaces: [{ name: 'Gi0/1' }],
    });
    expect(result.success).toBe(true);
  });

  it('accepts hyphen-separated MAC addresses', () => {
    expect(
      v.safeParse(AuthoredDeviceSchema, { name: 'sw1', mac: '00-1A-2B-3C-4D-5E' }).success,
    ).toBe(true);
  });

  it('requires a name', () => {
    expect(errorFor({ name: '', mac: '00:1A:2B:3C:4D:5E' }, 'name')).toBe('Name is required');
  });

  it('rejects a name that does not start with a letter', () => {
    expect(errorFor({ name: '1router', mac: '00:1A:2B:3C:4D:5E' }, 'name')).toMatch(
      /must start with a letter/i,
    );
  });

  it('rejects a malformed MAC address', () => {
    expect(errorFor({ name: 'sw1', mac: 'ZZ:ZZ:ZZ:ZZ:ZZ:ZZ' }, 'mac')).toMatch(/six hex octets/i);
  });

  it('rejects a malformed address', () => {
    expect(errorFor({ name: 'sw1', mac: '00:1A:2B:3C:4D:5E', ips: ['999.1.1.1'] }, 'ips')).toMatch(
      /valid IP address/i,
    );
  });

  // The daemon rejects both and neither; the issue is forwarded to `mac` so an
  // input can display it. A bare check would carry no path and show nowhere.
  it('rejects a device carrying both a MAC and a vendor', () => {
    expect(errorFor({ name: 'sw1', mac: '00:1A:2B:3C:4D:5E', vendor: 'cisco' }, 'mac')).toMatch(
      /not both and not neither/i,
    );
  });

  it('rejects a device carrying neither', () => {
    expect(errorFor({ name: 'sw1' }, 'mac')).toMatch(/not both and not neither/i);
  });
});

/**
 * Guards #1476.
 *
 * ErrorInjectionSchema capped the interface name at 15 characters — the Linux
 * IFNAMSIZ convention, which has nothing to do with the SNMP ifName values NIAC
 * simulates. The server imposes no such limit: errorInjectionRequest
 * .validationMessage() (internal/api/interface_fault_types.go:71) only requires
 * a non-empty string.
 *
 * The result on CT304 was that the form rejected interface names its own
 * dropdown had just offered, silently, before any request was sent. Every WAN
 * router and every access point in the campus scenario was excluded from fault
 * injection: 3 of 8 sampled devices had no usable interface at all.
 */
describe('ErrorInjectionSchema interface names', () => {
  const valid = {
    selectedDevice: 'ENG-WAN-R01',
    selectedErrorType: 'FCS Errors',
    errorValue: 50,
  };

  it.each([
    ['HundredGigabitEthernet0/0/1', 27],
    ['mGigabitEthernet0', 17],
    ['TenGigabitEthernet1/1/1', 23],
  ])('accepts %s, the kind of name the dropdown offers (%i chars)', (selectedInterface) => {
    const result = v.safeParse(ErrorInjectionSchema, { ...valid, selectedInterface });
    expect(result.success, JSON.stringify(result.issues?.map((i) => i.message))).toBe(true);
  });

  it('still requires a non-empty interface, as the server does', () => {
    const result = v.safeParse(ErrorInjectionSchema, { ...valid, selectedInterface: '' });
    expect(result.success).toBe(false);
    expect(result.issues?.[0]?.message).toMatch(/required/i);
  });

  it('still rejects an absurdly long name', () => {
    const result = v.safeParse(ErrorInjectionSchema, {
      ...valid,
      selectedInterface: 'x'.repeat(300),
    });
    expect(result.success).toBe(false);
  });
});
