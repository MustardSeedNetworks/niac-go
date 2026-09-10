import { describe, expect, it } from 'vitest';
import {
  behaviorFaultMaximum,
  deviceBehaviorFaultTypes,
  isDeviceBehaviorFaultType,
  isInterfaceBehaviorFaultType,
  isResourceBehaviorFaultType,
  resourceBehaviorFaultTypes,
  validBehaviorFault,
  validFaultAddress,
} from './behavior-fault-types';

const resources = ['cpu_percent', 'memory_percent', 'disk_percent'] as const;

it('accepts only usable IPv4 address payloads for duplicate offers', () => {
  for (const address of ['192.0.2.20', '240.0.0.1']) {
    expect(validFaultAddress(address)).toBe(true);
    expect(validBehaviorFault({ device: 'server', type: 'duplicate_dhcp_offer', address })).toBe(
      true,
    );
  }
  for (const address of [
    '',
    'invalid',
    '192.0.2.999',
    '192.0.2.01',
    '::1',
    '224.0.0.1',
    '0.0.0.0',
    '255.255.255.255',
  ]) {
    expect(validFaultAddress(address)).toBe(false);
    expect(validBehaviorFault({ device: 'server', type: 'duplicate_dhcp_offer', address })).toBe(
      false,
    );
  }
});

it('validates captive portal as a binary device outcome', () => {
  const type = 'captive_portal';
  expect(isDeviceBehaviorFaultType(type)).toBe(true);
  expect(isInterfaceBehaviorFaultType(type)).toBe(false);
  if (!isDeviceBehaviorFaultType(type)) throw new Error('Expected device-scoped captive portal');
  expect(behaviorFaultMaximum(type)).toBe(1);
  expect(validBehaviorFault({ device: 'gateway-1', type, value: 1 })).toBe(true);
  for (const value of [0, -1, 2, 100, 1.5]) {
    expect(validBehaviorFault({ device: 'gateway-1', type, value })).toBe(false);
  }
});

describe('resource behavior faults', () => {
  it.each(resources)('authors %s on the device without an interface', (type) => {
    expect(deviceBehaviorFaultTypes).toContain(type);
    expect(resourceBehaviorFaultTypes).toContain(type);
    expect(isResourceBehaviorFaultType(type)).toBe(true);
    expect(isDeviceBehaviorFaultType(type)).toBe(true);
    expect(isInterfaceBehaviorFaultType(type)).toBe(false);
    if (!isDeviceBehaviorFaultType(type)) throw new Error('Expected device-scoped resource');
    expect(behaviorFaultMaximum(type)).toBe(100);
    for (const value of [1, 50, 100]) {
      expect(validBehaviorFault({ device: 'hospital-server-1', type, value })).toBe(true);
    }
    for (const value of [0, -1, 101, 1.5, Number.NaN, Number.POSITIVE_INFINITY]) {
      expect(validBehaviorFault({ device: 'hospital-server-1', type, value })).toBe(false);
    }
    expect(validBehaviorFault({ device: '', type, value: 50 })).toBe(false);
  });

  it.each(['latency', 'dns_timeout', 'high_utilization', 'other_percent'])(
    'does not classify %s as a resource target',
    (type) => expect(isResourceBehaviorFaultType(type)).toBe(false),
  );
});
