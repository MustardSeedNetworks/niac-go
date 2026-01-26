// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * =============================================================================
 * COMPONENT VARIANTS
 * =============================================================================
 *
 * Reusable component styling tokens for buttons, inputs, cards, modals, etc.
 */

/**
 * Button variants - consistent button styling
 */
export const button = {
  base: 'inline-flex items-center justify-center gap-2 rounded-lg font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-violet-500/50 disabled:opacity-50 disabled:cursor-not-allowed',

  variant: {
    primary:
      'bg-gradient-to-r from-violet-600 to-violet-500 text-white shadow-lg shadow-violet-500/30 hover:from-violet-500 hover:to-violet-400 active:scale-[0.98]',
    secondary: 'bg-white/10 text-white hover:bg-white/15 border border-white/10',
    ghost: 'text-gray-400 hover:text-white hover:bg-white/10',
    outline: 'border border-white/20 text-gray-300 hover:bg-white/5 hover:border-violet-500/50',
    danger: 'bg-red-500/20 text-red-400 border border-red-500/30 hover:bg-red-500/30',
    success:
      'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 hover:bg-emerald-500/30',
  },

  size: {
    xs: 'px-2 py-1 text-xs',
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base',
  },
} as const;

/**
 * Input variants - consistent form input styling
 */
export const input = {
  base: 'w-full rounded-lg bg-gray-900/50 text-white transition-colors focus:outline-none focus:ring-2 focus:ring-violet-500/50 disabled:opacity-50 disabled:cursor-not-allowed placeholder:text-gray-500',

  state: {
    default: 'border border-white/10',
    error: 'border border-red-500/50',
    success: 'border border-emerald-500/50',
  },

  size: {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-3 py-2.5 text-sm',
    lg: 'px-4 py-3 text-base',
  },
} as const;

/**
 * Card variants - glass morphism cards
 */
export const card = {
  base: 'rounded-xl backdrop-blur-xl',

  variant: {
    default: 'bg-gradient-to-br from-gray-800/50 to-gray-900/50 border border-white/10',
    elevated:
      'bg-gradient-to-br from-gray-800/70 to-gray-900/70 border border-white/10 shadow-xl shadow-black/20',
    interactive:
      'bg-gradient-to-br from-gray-800/50 to-gray-900/50 border border-white/10 hover:border-violet-500/30 cursor-pointer transition-colors',
    glass: 'bg-white/5 backdrop-blur-2xl border border-white/10 shadow-2xl',
  },

  padding: {
    none: '',
    sm: 'p-3',
    md: 'p-4',
    lg: 'p-6',
  },
} as const;

/**
 * Badge variants - status badges and tags
 */
export const badge = {
  base: 'inline-flex items-center gap-1 rounded-full text-xs font-medium',

  variant: {
    default: 'bg-white/10 text-gray-300',
    success: 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30',
    warning: 'bg-amber-500/20 text-amber-400 border border-amber-500/30',
    error: 'bg-red-500/20 text-red-400 border border-red-500/30',
    info: 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
    primary: 'bg-violet-500/20 text-violet-400 border border-violet-500/30',
    new: 'bg-emerald-500/20 text-emerald-300',
    beta: 'bg-amber-500/20 text-amber-300',
  },

  size: {
    xs: 'px-1.5 py-0.5',
    sm: 'px-2 py-0.5',
    md: 'px-2.5 py-1',
  },
} as const;

/**
 * Alert/Banner variants
 */
export const alert = {
  base: 'px-4 py-3 rounded-lg border',

  variant: {
    error: 'bg-red-500/10 border-red-500/20 text-red-400',
    warning: 'bg-amber-500/10 border-amber-500/20 text-amber-400',
    success: 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400',
    info: 'bg-blue-500/10 border-blue-500/20 text-blue-400',
  },
} as const;

/**
 * Modal/Dialog variants
 */
export const modal = {
  overlay: 'fixed inset-0 z-50 flex items-center justify-center p-4',
  backdrop: 'absolute inset-0 bg-black/60 backdrop-blur-sm',
  content:
    'relative bg-gray-900 border border-white/10 rounded-xl shadow-2xl max-h-[85vh] overflow-y-auto',

  size: {
    sm: 'max-w-md w-full',
    md: 'max-w-2xl w-full',
    lg: 'max-w-4xl w-full',
    xl: 'max-w-6xl w-full',
    full: 'max-w-7xl w-full',
  },
} as const;

/**
 * Drawer variants - side panels
 */
export const drawer = {
  overlay: 'fixed inset-0 z-50',
  backdrop: 'absolute inset-0 bg-black/50 backdrop-blur-sm',
  content:
    'absolute right-0 top-0 h-full bg-gray-900 border-l border-white/10 shadow-2xl overflow-y-auto',

  size: {
    sm: 'w-80',
    md: 'w-96',
    lg: 'w-[480px]',
    xl: 'w-[600px]',
  },
} as const;

/**
 * Status indicator variants - for connection status, health, etc.
 */
export const status = {
  dot: 'inline-block w-2 h-2 rounded-full',

  color: {
    success: 'bg-emerald-500',
    warning: 'bg-amber-500',
    error: 'bg-red-500',
    info: 'bg-blue-500',
    inactive: 'bg-gray-500',
    online: 'bg-emerald-500',
    offline: 'bg-gray-500',
    pending: 'bg-blue-500',
  },

  withLabel: 'inline-flex items-center gap-2',
  pulse: 'animate-pulse',
} as const;

/**
 * Icon sizing utilities
 */
export const icon = {
  size: {
    xs: 'w-3 h-3',
    sm: 'w-4 h-4',
    md: 'w-5 h-5',
    lg: 'w-6 h-6',
    xl: 'w-8 h-8',
  },

  inline: 'inline-flex items-center gap-1.5',
  button: 'inline-flex items-center gap-2',
  leading: 'flex items-center gap-2',
} as const;
