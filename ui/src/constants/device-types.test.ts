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
import type { DeviceType } from '../api/device-config-types';
import { DEVICE_TYPES as SCHEMA_TYPES } from '../components/device-editor/generated/sections.generated';
import {
  getDeviceColor,
  getDeviceIcon,
  getTopologyDeviceColor,
  getTopologyDeviceIcon,
  normalizeDeviceType,
} from './device-types';

describe('device type coverage', () => {
  it.each([...SCHEMA_TYPES, 'unknown'] as DeviceType[])('%s has its own topology icon', (type) => {
    if (type === 'unknown') {
      return;
    }
    expect(getTopologyDeviceIcon(type)).not.toBe(getTopologyDeviceIcon('unknown'));
  });

  it.each([...SCHEMA_TYPES, 'unknown'] as DeviceType[])(
    '%s has its own topology colour',
    (type) => {
      if (type === 'unknown') {
        return;
      }
      expect(getTopologyDeviceColor(type)).not.toBe(getTopologyDeviceColor('unknown'));
    },
  );

  it.each([...SCHEMA_TYPES, 'unknown'] as DeviceType[])('%s has a tag colour', (type) => {
    expect(getDeviceColor(type)).toBeTruthy();
  });
});

describe('surfaces agree on a device type', () => {
  it.each([...SCHEMA_TYPES, 'unknown'] as DeviceType[])(
    '%s draws the same glyph on the canvas and in a list',
    (type) => {
      expect(getTopologyDeviceIcon(type)).toBe(getDeviceIcon(type));
    },
  );
});

describe('normalizeDeviceType', () => {
  // `ap` is the exception: the schema accepts it and `access-point` for the
  // same device, and the UI folds them so an operator is not offered two
  // identical filters for one kind of device.
  it.each(([...SCHEMA_TYPES, 'unknown'] as DeviceType[]).filter((type) => type !== 'ap'))(
    'leaves the canonical value %s alone',
    (type) => {
      expect(normalizeDeviceType(type)).toBe(type);
    },
  );

  it('folds ap onto access-point', () => {
    expect(normalizeDeviceType('ap')).toBe('access-point');
  });

  // Authored YAML in the tree uses these spellings; the daemon passes a device
  // type through as written, so the UI is where they have to converge.
  it.each([
    ['access_point', 'access-point'],
    ['accessPoint', 'access-point'],
    ['wireless-ap', 'access-point'],
    ['phone', 'voip-phone'],
    ['pc', 'workstation'],
    ['Router', 'router'],
    ['SWITCH', 'switch'],
    ['LAYER3-SWITCH', 'layer3-switch'],
  ])('resolves the alias %s to %s', (raw, canonical) => {
    expect(normalizeDeviceType(raw)).toBe(canonical);
  });

  it('falls back to unknown for a type it has never seen', () => {
    expect(normalizeDeviceType('toaster')).toBe('unknown');
  });

  it('resolves an alias to the same icon as its canonical type', () => {
    expect(getTopologyDeviceIcon('access_point')).toBe(getTopologyDeviceIcon('access-point'));
  });
});
