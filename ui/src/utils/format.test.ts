import { describe, expect, it } from 'vitest';
import {
  formatBytes,
  formatDuration,
  formatNumber,
  formatRelativeTime,
  formatUptime,
  getErrorMessage,
} from './format';

describe('formatNumber', () => {
  it('formats integers with locale separators', () => {
    expect(formatNumber(1000)).toBe('1,000');
    expect(formatNumber(1000000)).toBe('1,000,000');
  });

  it('handles zero', () => {
    expect(formatNumber(0)).toBe('0');
  });

  it('handles negative numbers', () => {
    expect(formatNumber(-1500)).toBe('-1,500');
  });
});

describe('formatDuration', () => {
  it('returns value when non-empty', () => {
    expect(formatDuration('5m 30s')).toBe('5m 30s');
  });

  it('returns dash for empty string', () => {
    expect(formatDuration('')).toBe('\u2014');
  });
});

describe('formatRelativeTime', () => {
  it('returns dash for empty input', () => {
    expect(formatRelativeTime('')).toBe('\u2014');
  });

  it('returns just now for future timestamps', () => {
    const future = new Date(Date.now() + 60000).toISOString();
    expect(formatRelativeTime(future)).toBe('just now');
  });

  it('returns seconds ago for recent timestamps', () => {
    const recent = new Date(Date.now() - 30000).toISOString();
    expect(formatRelativeTime(recent)).toBe('30s ago');
  });

  it('returns minutes ago', () => {
    const fiveMinAgo = new Date(Date.now() - 300000).toISOString();
    expect(formatRelativeTime(fiveMinAgo)).toBe('5m ago');
  });

  it('returns hours ago', () => {
    const twoHoursAgo = new Date(Date.now() - 7200000).toISOString();
    expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');
  });

  it('returns days ago', () => {
    const threeDaysAgo = new Date(Date.now() - 259200000).toISOString();
    expect(formatRelativeTime(threeDaysAgo)).toBe('3d ago');
  });

  // Phase 5 added an optional locale parameter that swaps the English
  // "Ns ago" output for the Intl.RelativeTimeFormat localized form.
  describe('with locale', () => {
    it('renders Spanish form when locale is es-ES', () => {
      const fiveMinAgo = new Date(Date.now() - 300000).toISOString();
      const out = formatRelativeTime(fiveMinAgo, 'es-ES');
      // Intl uses the locale's standard relative-time phrasing. Don't
      // assert exact ICU output (varies by Node/Intl version); assert
      // it's Spanish-shaped (contains 'hace' or 'minuto') and not the
      // English fallback ('m ago').
      expect(out).not.toContain('ago');
      expect(out.toLowerCase()).toMatch(/(hace|minuto)/);
    });

    it('renders English form when locale is en-US', () => {
      const twoHoursAgo = new Date(Date.now() - 7200000).toISOString();
      const out = formatRelativeTime(twoHoursAgo, 'en-US');
      // Intl en-US uses "2 hours ago" not the terse "2h ago" fallback.
      expect(out).toMatch(/2\s+(hours?|hr)/i);
      expect(out).toContain('ago');
    });

    it('falls back to terse English when locale is omitted', () => {
      const twoHoursAgo = new Date(Date.now() - 7200000).toISOString();
      expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');
    });
  });
});

describe('formatBytes', () => {
  it('returns 0 B for zero', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('returns 0 B for negative', () => {
    expect(formatBytes(-100)).toBe('0 B');
  });

  it('returns 0 B for NaN', () => {
    expect(formatBytes(Number.NaN)).toBe('0 B');
  });

  it('formats bytes', () => {
    expect(formatBytes(512)).toBe('512 B');
  });

  it('formats kilobytes', () => {
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
  });

  it('formats megabytes', () => {
    expect(formatBytes(1048576)).toBe('1.0 MB');
    expect(formatBytes(15728640)).toBe('15 MB');
  });

  it('formats gigabytes', () => {
    expect(formatBytes(1073741824)).toBe('1.0 GB');
  });
});

describe('formatUptime', () => {
  it('formats seconds only', () => {
    expect(formatUptime(45)).toBe('45s');
  });

  it('formats minutes and seconds', () => {
    expect(formatUptime(90)).toBe('1m 30s');
  });

  it('formats hours and minutes', () => {
    expect(formatUptime(3700)).toBe('1h 1m');
  });

  it('formats days and hours', () => {
    expect(formatUptime(90000)).toBe('1d 1h');
  });
});

describe('getErrorMessage', () => {
  it('extracts message from Error instances', () => {
    expect(getErrorMessage(new Error('test error'))).toBe('test error');
  });

  it('returns string errors directly', () => {
    expect(getErrorMessage('string error')).toBe('string error');
  });

  it('returns default message for other types', () => {
    expect(getErrorMessage(42)).toBe('An unexpected error occurred');
    expect(getErrorMessage(null)).toBe('An unexpected error occurred');
    expect(getErrorMessage(undefined)).toBe('An unexpected error occurred');
  });
});
