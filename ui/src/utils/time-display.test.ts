/**
 * Tests for packet timestamp formatting.
 *
 * The packet inspector's time column is one of the few places a wrong value is
 * indistinguishable from a right one at a glance, so these assert the formatted
 * strings, including the sign and the fixed 3-decimal precision.
 */

import { describe, expect, it } from 'vitest';
import {
  formatAbsoluteTime,
  formatDeltaTime,
  formatRelativeTime,
  formatTimeByMode,
  getTimeDisplayLabel,
  nextTimeDisplayMode,
} from './time-display';

const T0 = '2026-08-29T10:00:00.000Z';
const T1 = '2026-08-29T10:00:01.250Z';

describe('formatAbsoluteTime', () => {
  it('formats as 24-hour HH:mm:ss.fff', () => {
    expect(formatAbsoluteTime(T0)).toMatch(/^\d{2}:\d{2}:\d{2}\.\d{3}$/);
  });

  it('falls back to the raw timestamp when it does not parse', () => {
    // toLocaleTimeString returns the string "Invalid Date" rather than
    // throwing, so a try/catch cannot see this -- the date has to be tested.
    expect(formatAbsoluteTime('not-a-date')).toBe('not-a-date');
  });
});

describe('formatRelativeTime', () => {
  it('renders a positive offset to three decimals', () => {
    expect(formatRelativeTime(T1, T0)).toBe('+1.250s');
  });

  it('renders a zero offset against its own reference', () => {
    expect(formatRelativeTime(T0, T0)).toBe('+0.000s');
  });

  it('keeps the leading + even when the offset is negative', () => {
    // A packet before the reference reads "+-1.250s"; pinned so a change to the
    // sign handling is a deliberate one.
    expect(formatRelativeTime(T0, T1)).toBe('+-1.250s');
  });

  it('falls back to the raw timestamp when it does not parse', () => {
    // NaN arithmetic does not throw, so the NaN used to propagate into the
    // column as "+NaNs".
    expect(formatRelativeTime('nope', T0)).toBe('nope');
  });

  it('falls back to the raw timestamp when the reference does not parse', () => {
    expect(formatRelativeTime(T1, 'nope')).toBe(T1);
  });
});

describe('formatDeltaTime', () => {
  it('returns a zero delta when there is no previous packet', () => {
    expect(formatDeltaTime(T1, null)).toBe('Δ0.000s');
  });

  it('renders the gap from the previous packet', () => {
    expect(formatDeltaTime(T1, T0)).toBe('Δ1.250s');
  });

  it('renders a negative delta for an out-of-order packet', () => {
    expect(formatDeltaTime(T0, T1)).toBe('Δ-1.250s');
  });

  it('falls back to the raw timestamp when either timestamp does not parse', () => {
    expect(formatDeltaTime('nope', T0)).toBe('nope');
    expect(formatDeltaTime(T1, 'nope')).toBe(T1);
  });
});

describe('formatTimeByMode', () => {
  it('dispatches to the absolute formatter', () => {
    expect(formatTimeByMode(T0, 'absolute', T0, null)).toBe(formatAbsoluteTime(T0));
  });

  it('dispatches to the relative formatter', () => {
    expect(formatTimeByMode(T1, 'relative', T0, null)).toBe('+1.250s');
  });

  it('falls back to its own timestamp when there is no first packet', () => {
    expect(formatTimeByMode(T1, 'relative', null, null)).toBe('+0.000s');
  });

  it('dispatches to the delta formatter', () => {
    expect(formatTimeByMode(T1, 'delta', T0, T0)).toBe('Δ1.250s');
  });
});

describe('nextTimeDisplayMode', () => {
  it('cycles absolute → relative → delta → absolute', () => {
    expect(nextTimeDisplayMode('absolute')).toBe('relative');
    expect(nextTimeDisplayMode('relative')).toBe('delta');
    expect(nextTimeDisplayMode('delta')).toBe('absolute');
  });
});

describe('getTimeDisplayLabel', () => {
  it('labels every mode', () => {
    expect(getTimeDisplayLabel('absolute')).toBe('Time');
    expect(getTimeDisplayLabel('relative')).toBe('Relative');
    expect(getTimeDisplayLabel('delta')).toBe('Delta');
  });
});
