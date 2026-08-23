/**
 * forms.test.ts — pins the DeviceFormSchema validation the device editor
 * relies on (#730). These cases lock the format rules that mirror the Go
 * `validate:"mac"` / `validate:"ip"` tags and guarantee the schema ignores
 * the many other device sections so it can be run against a whole Device.
 */

import * as v from 'valibot';
import { describe, expect, it } from 'vitest';
import { DeviceFormSchema, ErrorInjectionSchema } from './forms';

/** First validation message for a given top-level field, or undefined. */
function errorFor(input: unknown, field: string): string | undefined {
  const result = v.safeParse(DeviceFormSchema, input);
  if (result.success) return undefined;
  const issue = result.issues.find((i) => i.path?.[0]?.key === field);
  return issue?.message;
}

describe('DeviceFormSchema', () => {
  it('accepts a valid device (hostname + MAC + IPv4)', () => {
    const result = v.safeParse(DeviceFormSchema, {
      hostname: 'core-switch-01',
      mac: '00:1A:2B:3C:4D:5E',
      ip: '10.0.0.1',
    });
    expect(result.success).toBe(true);
  });

  it('treats an empty primary IP as valid (no management IP)', () => {
    expect(
      v.safeParse(DeviceFormSchema, { hostname: 'sw1', mac: '00:1A:2B:3C:4D:5E', ip: '' }).success,
    ).toBe(true);
  });

  it('treats an omitted primary IP as valid', () => {
    expect(
      v.safeParse(DeviceFormSchema, { hostname: 'sw1', mac: '00:1A:2B:3C:4D:5E' }).success,
    ).toBe(true);
  });

  it('ignores the other device sections (runs against a whole Device)', () => {
    const result = v.safeParse(DeviceFormSchema, {
      hostname: 'sw1',
      mac: '00:1A:2B:3C:4D:5E',
      type: 'switch',
      ips: ['10.0.0.2'],
      snmpAgent: { community: 'public' },
      interfaceDetails: [{ name: 'Gi0/1' }],
    });
    expect(result.success).toBe(true);
  });

  it('accepts hyphen-separated MAC addresses', () => {
    expect(
      v.safeParse(DeviceFormSchema, { hostname: 'sw1', mac: '00-1A-2B-3C-4D-5E' }).success,
    ).toBe(true);
  });

  it('requires a hostname', () => {
    expect(errorFor({ hostname: '', mac: '00:1A:2B:3C:4D:5E' }, 'hostname')).toBe(
      'Hostname is required',
    );
  });

  it('rejects a hostname that does not start with a letter', () => {
    expect(errorFor({ hostname: '1router', mac: '00:1A:2B:3C:4D:5E' }, 'hostname')).toMatch(
      /must start with a letter/i,
    );
  });

  it('requires a MAC address', () => {
    expect(errorFor({ hostname: 'sw1', mac: '' }, 'mac')).toBe('MAC address is required');
  });

  it('rejects a malformed MAC address', () => {
    expect(errorFor({ hostname: 'sw1', mac: 'ZZ:ZZ:ZZ:ZZ:ZZ:ZZ' }, 'mac')).toMatch(
      /six hex octets/i,
    );
  });

  it('rejects a malformed primary IP', () => {
    expect(errorFor({ hostname: 'sw1', mac: '00:1A:2B:3C:4D:5E', ip: '999.1.1.1' }, 'ip')).toMatch(
      /valid IPv4/i,
    );
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
