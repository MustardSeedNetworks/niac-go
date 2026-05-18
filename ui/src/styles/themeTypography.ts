// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * =============================================================================
 * TYPOGRAPHY
 * =============================================================================
 *
 * Typography tokens for consistent text styling.
 */

export const typography = {
  heading: {
    h1: 'text-2xl font-bold text-white',
    h2: 'text-xl font-semibold text-white',
    h3: 'text-lg font-semibold text-white',
    h4: 'text-base font-medium text-white',
  },

  body: {
    large: 'text-base text-gray-200',
    default: 'text-sm text-gray-300',
    small: 'text-xs text-gray-400',
    muted: 'text-sm text-gray-500',
  },

  label: 'text-sm font-medium text-gray-300',
  caption: 'text-xs text-gray-500',
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
