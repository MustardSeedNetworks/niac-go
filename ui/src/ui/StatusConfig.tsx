/**
 * Status Configuration
 *
 * Centralized configuration for status indicators (success, warning, error, etc.).
 * Used by StatusBadge, Card, and other components that need consistent status styling.
 *
 * Adapted from Seed UI for NIAC's dark glassmorphism theme.
 */

import type { ReactNode } from 'react';

export type Status = 'success' | 'warning' | 'error' | 'unknown' | 'loading';

interface StatusConfigItem {
  readonly icon: ReactNode;
  readonly color: string;
  readonly bgColor: string;
  readonly label: string;
}

// Centralized status configuration - icons, colors, and labels
// Using NIAC's dark theme color palette
export const statusConfig: Record<Status, StatusConfigItem> = {
  success: {
    icon: (
      <svg className="w-full h-full" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-10.707a1 1 0 00-1.414-1.414L9 9.172 7.707 7.879a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: 'text-status-success',
    bgColor: 'bg-status-success/15',
    label: 'Status: success',
  },
  warning: {
    icon: (
      <svg className="w-full h-full" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path
          fillRule="evenodd"
          d="M8.257 3.099c.765-1.36 2.72-1.36 3.485 0l6.518 11.6c.75 1.334-.214 3.001-1.742 3.001H3.48c-1.528 0-2.492-1.667-1.742-3.001l6.52-11.6zM11 14a1 1 0 11-2 0 1 1 0 012 0zm-1-2a1 1 0 01-1-1V8a1 1 0 112 0v3a1 1 0 01-1 1z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: 'text-status-warning',
    bgColor: 'bg-status-warning/15',
    label: 'Status: warning',
  },
  error: {
    icon: (
      <svg className="w-full h-full" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: 'text-status-error',
    bgColor: 'bg-status-error/15',
    label: 'Status: error',
  },
  unknown: {
    icon: (
      <svg className="w-full h-full" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path
          fillRule="evenodd"
          d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: 'text-text-muted',
    bgColor: 'bg-bg-muted/15',
    label: 'Status: unknown',
  },
  loading: {
    icon: (
      <svg
        className="w-full h-full animate-spin"
        viewBox="0 0 20 20"
        fill="none"
        aria-hidden="true"
      >
        <circle
          className="opacity-25"
          cx="10"
          cy="10"
          r="8"
          stroke="currentColor"
          strokeWidth="3"
        />
        <path className="opacity-75" fill="currentColor" d="M18 10a8 8 0 00-8-8v4a4 4 0 014 4h4z" />
      </svg>
    ),
    color: 'text-status-info',
    bgColor: 'bg-status-info/15',
    label: 'Status: loading',
  },
};

// Size configurations
export const sizeConfig = {
  sm: {
    icon: 'w-4 h-4',
    dot: 'w-2 h-2',
    padding: 'p-0.5',
  },
  md: {
    icon: 'w-5 h-5',
    dot: 'w-2.5 h-2.5',
    padding: 'p-1',
  },
  lg: {
    icon: 'w-6 h-6',
    dot: 'w-3 h-3',
    padding: 'p-1.5',
  },
} as const;

export type SizeKey = keyof typeof sizeConfig;

/**
 * Type-safe getter for status configuration.
 * Uses exhaustive switch for type safety.
 */
export function getStatusConfig(status: Status): StatusConfigItem {
  switch (status) {
    case 'success':
      return statusConfig.success;
    case 'warning':
      return statusConfig.warning;
    case 'error':
      return statusConfig.error;
    case 'unknown':
      return statusConfig.unknown;
    case 'loading':
      return statusConfig.loading;
  }
}

/**
 * Type-safe getter for size configuration.
 */
export function getSizeConfig(size: SizeKey): (typeof sizeConfig)[SizeKey] {
  switch (size) {
    case 'sm':
      return sizeConfig.sm;
    case 'md':
      return sizeConfig.md;
    case 'lg':
      return sizeConfig.lg;
  }
}
