/**
 * Polling intervals for API resource fetching
 */
export const POLL_INTERVALS = {
  FAST: 2000,      // 2s - Real-time simulation status
  MEDIUM: 5000,    // 5s - Live stats
  SLOW: 15000,     // 15s - Historical data
  VERY_SLOW: 60000, // 1m - Static data like version
} as const;

export type PollInterval = typeof POLL_INTERVALS[keyof typeof POLL_INTERVALS];
