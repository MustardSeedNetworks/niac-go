/**
 * Formatting utilities.
 *
 * Two layers:
 *
 *   1. Pure functions (formatBytes, formatDurationMs, formatUptime,
 *      formatRelativeTime, getErrorMessage) — fall back to English
 *      and the browser-default locale. Safe to call from non-React
 *      code (utilities, error boundaries, console logs).
 *
 *   2. React hooks (useFormatBytes, useFormatNumber, useFormatTime,
 *      useFormatRelativeTime) — return locale-aware formatters bound
 *      to the active i18next language and pulling unit suffixes
 *      from the common.format.* locale keys.
 *
 * Prefer the hooks inside components so number/date strings flip
 * between English and Spanish (and re-render) on language change.
 * Call sites that need a one-shot format outside a component (e.g.
 * inside a Zustand store callback) can use the pure functions.
 */

import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocale } from '../hooks/useLocale';

// ---------------------------------------------------------------------------
// Pure formatters (English fallback)
// ---------------------------------------------------------------------------

export function formatNumber(value: number, locale?: string): string {
  return value.toLocaleString(locale);
}

export function formatTime(value: string, locale?: string): string {
  return new Date(value).toLocaleString(locale);
}

export function formatDuration(value: string): string {
  return value || '—';
}

/**
 * Relative-time formatter. Uses Intl.RelativeTimeFormat when a locale
 * is provided so output is fully translated; without a locale, falls
 * back to terse English (`5s ago`, `2m ago`, …).
 */
export function formatRelativeTime(timestamp: string, locale?: string): string {
  if (!timestamp) {
    return '—';
  }
  const diff = Date.now() - new Date(timestamp).getTime();
  if (diff < 0) {
    return locale
      ? new Intl.RelativeTimeFormat(locale, { numeric: 'auto' }).format(0, 'second')
      : 'just now';
  }
  const seconds = Math.floor(diff / 1000);
  if (locale && typeof Intl?.RelativeTimeFormat === 'function') {
    const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
    if (seconds < 60) return rtf.format(-seconds, 'second');
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return rtf.format(-minutes, 'minute');
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return rtf.format(-hours, 'hour');
    return rtf.format(-Math.floor(hours / 24), 'day');
  }
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function formatDurationSeconds(seconds: number): string {
  if (seconds < 0.001) return '<1ms';
  if (seconds < 1) return `${(seconds * 1000).toFixed(0)}ms`;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}m ${secs.toFixed(0)}s`;
}

export function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${ms.toFixed(0)} ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)} s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(1)} min`;
  return `${(ms / 3600000).toFixed(1)} hr`;
}

export function formatBytes(size: number): string {
  if (!Number.isFinite(size) || size <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  let idx = 0;
  let value = size;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx++;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[idx]}`;
}

/**
 * Format a bits-per-second interface speed as bps/Kbps/Mbps/Gbps.
 * Network link speeds are decimal SI (1 Mbps = 1,000,000 bps), unlike
 * formatBytes' binary (1024) steps. Returns '—' for missing/zero speed
 * (e.g. a loopback interface that never reported one) rather than "0
 * bps", since zero here means "not reported," not "measured as zero."
 */
export function formatBitsPerSecond(bps?: number): string {
  if (!bps || bps <= 0) {
    return '—';
  }
  const units = ['bps', 'Kbps', 'Mbps', 'Gbps'];
  let idx = 0;
  let value = bps;
  while (value >= 1000 && idx < units.length - 1) {
    value /= 1000;
    idx++;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[idx]}`;
}

export function formatUptime(seconds: number): string {
  if (seconds < 60) {
    return `${Math.floor(seconds)}s`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes}m ${Math.floor(seconds % 60)}s`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours}h ${minutes % 60}m`;
  }
  const days = Math.floor(hours / 24);
  return `${days}d ${hours % 24}h`;
}

/**
 * Extract a human-readable error message. Returns translated default
 * when caller provides one; otherwise English fallback.
 */
export function getErrorMessage(err: unknown, fallback?: string): string {
  if (err instanceof Error) {
    return err.message;
  }
  if (typeof err === 'string') {
    return err;
  }
  return fallback ?? 'An unexpected error occurred';
}

// ---------------------------------------------------------------------------
// Locale-aware hooks
// ---------------------------------------------------------------------------

/**
 * useFormatNumber — Intl.NumberFormat bound to the active locale.
 * Use for any count/value rendered in JSX so 1,234.56 (en) becomes
 * 1.234,56 (es) automatically.
 */
export function useFormatNumber(): (value: number, options?: Intl.NumberFormatOptions) => string {
  const locale = useLocale();
  return useCallback(
    (value, options) => new Intl.NumberFormat(locale, options).format(value),
    [locale],
  );
}

/**
 * useFormatTime — Intl.DateTimeFormat bound to the active locale.
 * Defaults to the long localized form. Pass options to override.
 */
export function useFormatTime(
  defaultOptions: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  },
): (timestamp: string | number | Date, options?: Intl.DateTimeFormatOptions) => string {
  const locale = useLocale();
  return useCallback(
    (timestamp, options) => {
      try {
        const date = timestamp instanceof Date ? timestamp : new Date(timestamp);
        return new Intl.DateTimeFormat(locale, options ?? defaultOptions).format(date);
      } catch {
        return String(timestamp);
      }
    },
    [locale, defaultOptions],
  );
}

/**
 * useFormatRelativeTime — Intl.RelativeTimeFormat for "5 minutes ago"
 * style output keyed off the active locale. ES output uses the proper
 * "hace 5 min" form.
 */
export function useFormatRelativeTime(): (timestamp: string) => string {
  const locale = useLocale();
  return useCallback((timestamp) => formatRelativeTime(timestamp, locale), [locale]);
}

/**
 * useFormatBytes — locale-aware byte-size formatter.
 * The number part flips between 1,024 (en) and 1.024 (es) decimal
 * conventions; the unit suffix (B/KB/MB/GB) is pulled from the
 * common.format.bytes* locale keys to allow translators to localize
 * the unit if needed (most ES locales keep B/KB/MB/GB verbatim).
 */
export function useFormatBytes(): (size: number) => string {
  const locale = useLocale();
  const { t } = useTranslation('common');
  return useCallback(
    (size) => {
      if (!Number.isFinite(size) || size <= 0) {
        return t('format.bytesB', { value: '0' });
      }
      const units = ['bytesB', 'bytesKB', 'bytesMB', 'bytesGB'] as const;
      let idx = 0;
      let value = size;
      while (value >= 1024 && idx < units.length - 1) {
        value /= 1024;
        idx++;
      }
      const formatted = new Intl.NumberFormat(locale, {
        maximumFractionDigits: value >= 10 ? 0 : 1,
      }).format(value);
      return t(`format.${units[idx]}`, { value: formatted });
    },
    [locale, t],
  );
}
