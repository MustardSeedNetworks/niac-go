/**
 * Polling intervals for API resource fetching
 */
export const POLL_INTERVALS = {
  fast: 2000, // 2s - Real-time simulation status
  medium: 5000, // 5s - Live stats
  slow: 15000, // 15s - Historical data
  verySlow: 60000, // 1m - Static data like version
} as const;

export type PollInterval = (typeof POLL_INTERVALS)[keyof typeof POLL_INTERVALS];
