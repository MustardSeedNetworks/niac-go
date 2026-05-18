/**
 * =============================================================================
 * TYPOGRAPHY
 * =============================================================================
 *
 * Typography tokens for consistent text styling.
 */

export const typography = {
  heading: {
    h1: 'text-2xl font-bold text-text-primary',
    h2: 'text-xl font-semibold text-text-primary',
    h3: 'text-lg font-semibold text-text-primary',
    h4: 'text-base font-medium text-text-primary',
  },

  body: {
    large: 'text-base text-text-primary',
    default: 'text-sm text-text-secondary',
    small: 'text-xs text-text-muted',
    muted: 'text-sm text-text-muted',
  },

  label: 'text-sm font-medium text-text-secondary',
  caption: 'text-xs text-text-muted',
  code: 'font-mono text-sm',

  size: {
    xs: 'text-xs',
    sm: 'text-sm',
    base: 'text-base',
    lg: 'text-lg',
    xl: 'text-xl',
  },

  weight: {
    normal: 'font-normal',
    medium: 'font-medium',
    semibold: 'font-semibold',
    bold: 'font-bold',
  },
} as const;
