export type TimeDisplayMode = 'absolute' | 'relative' | 'delta';

/**
 * Milliseconds since the epoch, or null when the timestamp does not parse.
 *
 * A bad timestamp has to be tested for rather than caught: `new Date(bad)`
 * does not throw, and neither does arithmetic on the NaN it yields.
 */
function epochMs(timestamp: string): number | null {
  const ms = new Date(timestamp).getTime();
  return Number.isNaN(ms) ? null : ms;
}

/**
 * Format timestamp in absolute mode: HH:mm:ss.fff
 */
export function formatAbsoluteTime(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
  });
}

/**
 * Format timestamp as relative offset from a reference time: +N.NNNs
 */
export function formatRelativeTime(timestamp: string, referenceTimestamp: string): string {
  const current = epochMs(timestamp);
  const reference = epochMs(referenceTimestamp);
  if (current === null || reference === null) {
    return timestamp;
  }
  return `+${((current - reference) / 1000).toFixed(3)}s`;
}

/**
 * Format timestamp as delta from previous packet: \u0394N.NNNs
 */
export function formatDeltaTime(timestamp: string, previousTimestamp: string | null): string {
  if (!previousTimestamp) {
    return '\u03940.000s';
  }
  const current = epochMs(timestamp);
  const previous = epochMs(previousTimestamp);
  if (current === null || previous === null) {
    return timestamp;
  }
  return `\u0394${((current - previous) / 1000).toFixed(3)}s`;
}

/**
 * Format a packet timestamp based on the current display mode.
 */
export function formatTimeByMode(
  timestamp: string,
  mode: TimeDisplayMode,
  firstTimestamp: string | null,
  previousTimestamp: string | null,
): string {
  switch (mode) {
    case 'absolute':
      return formatAbsoluteTime(timestamp);
    case 'relative':
      return formatRelativeTime(timestamp, firstTimestamp ?? timestamp);
    case 'delta':
      return formatDeltaTime(timestamp, previousTimestamp);
  }
}

/**
 * Cycle to the next time display mode.
 */
export function nextTimeDisplayMode(current: TimeDisplayMode): TimeDisplayMode {
  const modes: TimeDisplayMode[] = ['absolute', 'relative', 'delta'];
  const idx = modes.indexOf(current);
  // The modulo keeps this in range; the fallback is what says so to the type
  // system, and returning the first mode is what wrapping already means.
  return modes[(idx + 1) % modes.length] ?? 'absolute';
}

/**
 * Get a human-readable label for the time display mode.
 */
export function getTimeDisplayLabel(mode: TimeDisplayMode): string {
  switch (mode) {
    case 'absolute':
      return 'Time';
    case 'relative':
      return 'Relative';
    case 'delta':
      return 'Delta';
  }
}
