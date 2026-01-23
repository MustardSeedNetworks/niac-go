export type TimeDisplayMode = 'absolute' | 'relative' | 'delta';

/**
 * Format timestamp in absolute mode: HH:mm:ss.fff
 */
export function formatAbsoluteTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      fractionalSecondDigits: 3,
    });
  } catch {
    return timestamp;
  }
}

/**
 * Format timestamp as relative offset from a reference time: +N.NNNs
 */
export function formatRelativeTime(timestamp: string, referenceTimestamp: string): string {
  try {
    const current = new Date(timestamp).getTime();
    const reference = new Date(referenceTimestamp).getTime();
    const deltaMs = current - reference;
    const deltaSec = deltaMs / 1000;
    return `+${deltaSec.toFixed(3)}s`;
  } catch {
    return timestamp;
  }
}

/**
 * Format timestamp as delta from previous packet: \u0394N.NNNs
 */
export function formatDeltaTime(timestamp: string, previousTimestamp: string | null): string {
  if (!previousTimestamp) {
    return '\u03940.000s';
  }
  try {
    const current = new Date(timestamp).getTime();
    const previous = new Date(previousTimestamp).getTime();
    const deltaMs = current - previous;
    const deltaSec = deltaMs / 1000;
    return `\u0394${deltaSec.toFixed(3)}s`;
  } catch {
    return timestamp;
  }
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
  return modes[(idx + 1) % modes.length];
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
