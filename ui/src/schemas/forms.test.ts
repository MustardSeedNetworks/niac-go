/**
 * forms.test.ts — pins the DeviceFormSchema validation the device editor
 * relies on (#730). These cases lock the format rules that mirror the Go
 * `validate:"mac"` / `validate:"ip"` tags and guarantee the schema ignores
 * the many other device sections so it can be run against a whole Device.
 */

import * as v from 'valibot';
import { describe, expect, it } from 'vitest';
import { DeviceFormSchema } from './forms';

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
