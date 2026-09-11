/**
 * Device type icon/colour mapping.
 *
 * The regression these cover (#2052): the topology canvas and the topology
 * legend read two different maps for the same device, so an access point drew
 * a Wifi glyph in the legend and a generic `unknown` glyph on the canvas. The
 * assertions below are about agreement between surfaces, not about which
 * glyph any one type gets.
 */

import { describe, expect, it } from 'vitest';
import { DEVICE_TYPES } from '../api/device-config-types';
import {
  getDeviceColor,
  getDeviceIcon,
  getTopologyDeviceColor,
  getTopologyDeviceIcon,
  normalizeDeviceType,
} from './device-types';

describe('device type coverage', () => {
  it.each(DEVICE_TYPES)('%s has its own topology icon', (type) => {
    if (type === 'unknown') {
      return;
    }
    expect(getTopologyDeviceIcon(type)).not.toBe(getTopologyDeviceIcon('unknown'));
  });

  it.each(DEVICE_TYPES)('%s has its own topology colour', (type) => {
    if (type === 'unknown') {
      return;
    }
    expect(getTopologyDeviceColor(type)).not.toBe(getTopologyDeviceColor('unknown'));
  });

  it.each(DEVICE_TYPES)('%s has a tag colour', (type) => {
    expect(getDeviceColor(type)).toBeTruthy();
  });
});

describe('surfaces agree on a device type', () => {
  it.each(DEVICE_TYPES)('%s draws the same glyph on the canvas and in a list', (type) => {
    expect(getTopologyDeviceIcon(type)).toBe(getDeviceIcon(type));
  });
});

describe('normalizeDeviceType', () => {
  it.each(DEVICE_TYPES)('leaves the canonical value %s alone', (type) => {
    expect(normalizeDeviceType(type)).toBe(type);
  });

  // Authored YAML in the tree uses these spellings; the daemon passes a device
  // type through as written, so the UI is where they have to converge.
  it.each([
    ['ap', 'access_point'],
    ['access-point', 'access_point'],
    ['accessPoint', 'access_point'],
    ['host', 'workstation'],
    ['Router', 'router'],
    ['SWITCH', 'switch'],
  ])('resolves the alias %s to %s', (raw, canonical) => {
    expect(normalizeDeviceType(raw)).toBe(canonical);
  });

  it('falls back to unknown for a type it has never seen', () => {
    expect(normalizeDeviceType('toaster')).toBe('unknown');
  });

  it('resolves an alias to the same icon as its canonical type', () => {
    expect(getTopologyDeviceIcon('ap')).toBe(getTopologyDeviceIcon('access_point'));
  });
});
