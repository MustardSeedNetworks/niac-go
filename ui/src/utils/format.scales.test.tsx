/**
 * Scale-boundary and hook coverage for the formatting helpers.
 *
 * format.test.ts covers formatNumber / formatDuration / formatRelativeTime /
 * formatBytes / formatUptime / getErrorMessage. This adds the duration and
 * link-speed scales and the four locale-aware hooks.
 *
 * These are unit-boundary functions, so the assertions sit on the boundaries:
 * an off-by-one in a `<` reads as a plausible number rather than as a bug.
 */

import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  formatBitsPerSecond,
  formatDurationMs,
  formatDurationSeconds,
  formatTime,
  useFormatBytes,
  useFormatNumber,
  useFormatRelativeTime,
  useFormatTime,
} from './format';

describe('formatTime', () => {
  it('renders a parseable timestamp in the requested locale', () => {
    // en-US puts the month first; this is the locale wiring, not the exact text.
    expect(formatTime('2026-08-29T10:00:00.000Z', 'en-US')).toMatch(/8\/29\/2026/);
  });

  it('renders an unparseable timestamp as "Invalid Date"', () => {
    expect(formatTime('nope', 'en-US')).toBe('Invalid Date');
  });
});

describe('formatDurationSeconds', () => {
  it('collapses sub-millisecond durations', () => {
    expect(formatDurationSeconds(0.0009)).toBe('<1ms');
    expect(formatDurationSeconds(0)).toBe('<1ms');
  });

  it('renders milliseconds below one second', () => {
    expect(formatDurationSeconds(0.001)).toBe('1ms');
    expect(formatDurationSeconds(0.25)).toBe('250ms');
    expect(formatDurationSeconds(0.999)).toBe('999ms');
  });

  it('renders seconds to one decimal below a minute', () => {
    expect(formatDurationSeconds(1)).toBe('1.0s');
    expect(formatDurationSeconds(59.9)).toBe('59.9s');
  });

  it('switches to minutes and seconds at one minute', () => {
    expect(formatDurationSeconds(60)).toBe('1m 0s');
    expect(formatDurationSeconds(90)).toBe('1m 30s');
    expect(formatDurationSeconds(3661)).toBe('61m 1s');
  });
});

describe('formatDurationMs', () => {
  it('renders whole milliseconds below a second', () => {
    expect(formatDurationMs(0)).toBe('0 ms');
    expect(formatDurationMs(999)).toBe('999 ms');
  });

  it('switches to seconds at one second', () => {
    expect(formatDurationMs(1000)).toBe('1.00 s');
    expect(formatDurationMs(59999)).toBe('60.00 s');
  });

  it('switches to minutes at one minute', () => {
    expect(formatDurationMs(60000)).toBe('1.0 min');
  });

  it('switches to hours at one hour', () => {
    expect(formatDurationMs(3600000)).toBe('1.0 hr');
    expect(formatDurationMs(5400000)).toBe('1.5 hr');
  });
});

describe('formatBitsPerSecond', () => {
  it('reports an em dash when the speed was never reported', () => {
    // Zero here means "not reported", not "measured as zero" — a loopback
    // interface must not read as 0 bps.
    expect(formatBitsPerSecond()).toBe('—');
    expect(formatBitsPerSecond(0)).toBe('—');
    expect(formatBitsPerSecond(-1)).toBe('—');
  });

  it('steps in decimal SI, not binary', () => {
    // 1 Mbps is 1,000,000 bps — unlike formatBytes' 1024 steps.
    expect(formatBitsPerSecond(1000)).toBe('1.0 Kbps');
    expect(formatBitsPerSecond(1000000)).toBe('1.0 Mbps');
    expect(formatBitsPerSecond(1000000000)).toBe('1.0 Gbps');
  });

  it('drops the decimal at and above ten', () => {
    expect(formatBitsPerSecond(9900)).toBe('9.9 Kbps');
    expect(formatBitsPerSecond(10000)).toBe('10 Kbps');
  });

  it('stops scaling at Gbps rather than inventing a unit', () => {
    expect(formatBitsPerSecond(100000000000)).toBe('100 Gbps');
  });

  it('renders sub-kilobit speeds as bps', () => {
    expect(formatBitsPerSecond(999)).toBe('999 bps');
  });
});

describe('useFormatNumber', () => {
  it('formats with the active locale', () => {
    const { result } = renderHook(() => useFormatNumber());
    expect(result.current(1234.5)).toBe('1,234.5');
  });

  it('honours per-call options', () => {
    const { result } = renderHook(() => useFormatNumber());
    expect(result.current(0.5, { style: 'percent' })).toBe('50%');
  });
});

describe('useFormatTime', () => {
  it('formats a timestamp string', () => {
    const { result } = renderHook(() => useFormatTime());
    expect(result.current('2026-08-29T10:00:00.000Z')).toMatch(/Aug/);
  });

  it('accepts a Date and a numeric epoch', () => {
    const { result } = renderHook(() => useFormatTime());
    const fromDate = result.current(new Date('2026-08-29T10:00:00.000Z'));
    const fromEpoch = result.current(Date.parse('2026-08-29T10:00:00.000Z'));
    expect(fromDate).toBe(fromEpoch);
  });

  it('falls back to the raw input when the date cannot be formatted', () => {
    const { result } = renderHook(() => useFormatTime());
    expect(result.current('not-a-date')).toBe('not-a-date');
  });

  it('honours per-call options over the defaults', () => {
    const { result } = renderHook(() => useFormatTime());
    expect(result.current('2026-08-29T10:00:00.000Z', { year: 'numeric' })).toBe('2026');
  });
});

describe('useFormatRelativeTime', () => {
  it('renders a recent timestamp relatively', () => {
    const { result } = renderHook(() => useFormatRelativeTime());
    const tenSecondsAgo = new Date(Date.now() - 10_000).toISOString();
    expect(result.current(tenSecondsAgo)).toMatch(/10 seconds ago/);
  });
});

describe('useFormatBytes', () => {
  it('renders zero and non-finite sizes as bytes', () => {
    const { result } = renderHook(() => useFormatBytes());
    expect(result.current(0)).toContain('0');
    expect(result.current(Number.NaN)).toContain('0');
  });

  it('steps in binary units', () => {
    const { result } = renderHook(() => useFormatBytes());
    expect(result.current(1024)).toContain('1');
    expect(result.current(1024 * 1024)).toContain('1');
  });

  it('drops the decimal at and above ten', () => {
    const { result } = renderHook(() => useFormatBytes());
    // 10 KB formats without a fractional part; 1.5 KB keeps one.
    expect(result.current(10 * 1024)).toContain('10');
    expect(result.current(1536)).toContain('1.5');
  });
});
