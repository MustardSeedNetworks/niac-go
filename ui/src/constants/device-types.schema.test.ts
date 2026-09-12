/**
 * The icon/colour maps must cover the vocabulary the daemon validates.
 *
 * The defect (#2095): `DeviceType` was a hand-written union in
 * api/device-config-types.ts that had drifted from the Go schema, so
 * `layer3-switch`, `printer` and `voip-phone` — all of which the generated
 * packs author — fell through to `unknown` and drew a question mark, while
 * `access_point`, the UI's own canonical value, was not a legal type at all.
 *
 * These read the *generated* vocabulary rather than restating it, so a schema
 * change that is not carried into the maps fails here instead of shipping a
 * question mark.
 */

import { describe, expect, it } from 'vitest';
import { DEVICE_TYPES as SCHEMA_DEVICE_TYPES } from '../components/device-editor/generated/sections.generated';
import {
  getDeviceColor,
  getTopologyDeviceColor,
  getTopologyDeviceIcon,
  normalizeDeviceType,
} from './device-types';

describe('every type the daemon accepts is drawable', () => {
  it.each(SCHEMA_DEVICE_TYPES)('%s does not fall through to unknown', (type) => {
    expect(normalizeDeviceType(type)).not.toBe('unknown');
  });

  it.each(SCHEMA_DEVICE_TYPES)('%s draws a real glyph, not the question mark', (type) => {
    expect(getTopologyDeviceIcon(type)).not.toBe(getTopologyDeviceIcon('unknown'));
  });

  it.each(SCHEMA_DEVICE_TYPES)('%s has a topology colour of its own', (type) => {
    expect(getTopologyDeviceColor(type)).not.toBe(getTopologyDeviceColor('unknown'));
  });

  it.each(SCHEMA_DEVICE_TYPES)('%s has a tag colour', (type) => {
    expect(getDeviceColor(type)).toBeTruthy();
  });
});

describe('the types the generated packs actually author', () => {
  // Counted from the seven packs: every one authors layer3-switch and
  // voip-phone, three of them author printer. These were the question marks.
  it.each(['layer3-switch', 'printer', 'voip-phone', 'access-point', 'host'])(
    '%s is drawable',
    (type) => {
      expect(getTopologyDeviceIcon(type)).not.toBe(getTopologyDeviceIcon('unknown'));
    },
  );
});

describe('unknown', () => {
  // `type` is omitempty in the schema, so absent is a real state the UI has to
  // represent — but it is not a value an author can write.
  it('is not part of the authored vocabulary', () => {
    expect(SCHEMA_DEVICE_TYPES).not.toContain('unknown');
  });

  it('is what an absent or unrecognised type becomes', () => {
    expect(normalizeDeviceType(undefined)).toBe('unknown');
    expect(normalizeDeviceType('toaster')).toBe('unknown');
  });
});
