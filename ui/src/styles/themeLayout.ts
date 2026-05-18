/**
 * =============================================================================
 * LAYOUT PATTERNS
 * =============================================================================
 *
 * Layout utilities for flexbox, grid, borders, and border radius.
 */

export const layout = {
  flex: {
    center: 'flex items-center justify-center',
    between: 'flex items-center justify-between',
    start: 'flex items-center justify-start',
    end: 'flex items-center justify-end',
    col: 'flex flex-col',
    colCenter: 'flex flex-col items-center justify-center',
    wrap: 'flex flex-wrap',
  },

  grid: {
    cards: 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6',
    form2col: 'grid grid-cols-2 gap-2',
    data2col: 'grid grid-cols-2 gap-x-4 gap-y-2',
  },

  inline: {
    tight: 'flex items-center gap-1',
    default: 'flex items-center gap-2',
    comfortable: 'flex items-center gap-3',
    spacious: 'flex items-center gap-4',
    wrap: 'flex flex-wrap items-center gap-2',
  },

  stack: {
    tight: 'flex flex-col gap-1',
    default: 'flex flex-col gap-2',
    comfortable: 'flex flex-col gap-3',
    spacious: 'flex flex-col gap-4',
  },
} as const;

/**
 * Border radius utilities
 */
export const radius = {
  none: 'rounded-none',
  sm: 'rounded-sm',
  default: 'rounded',
  md: 'rounded-md',
  lg: 'rounded-lg',
  xl: 'rounded-xl',
  full: 'rounded-full',
} as const;

/**
 * Border utilities
 */
export const border = {
  default: 'border border-border-default',
  subtle: 'border border-border-subtle',
  muted: 'border border-border-muted',
  focus: 'border border-brand-primary/50',
  error: 'border border-status-error/50',
  divider: 'border-t border-border-default',
} as const;
