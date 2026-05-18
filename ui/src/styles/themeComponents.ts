/**
 * themeComponents.ts — reusable component styling tokens.
 * Re-exported through theme.ts.
 *
 * All values reference semantic theme tokens declared in src/index.css via
 * Tailwind v4 @theme directive (brand-*, status-*, text-*, bg-*, border-*).
 * No raw Tailwind palette colors — change the @theme block to retheme;
 * do not hardcode here.
 */

export const button = {
  base: 'inline-flex items-center justify-center gap-2 rounded-lg font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-brand-primary/50 disabled:opacity-50 disabled:cursor-not-allowed',

  variant: {
    primary:
      'bg-gradient-to-r from-brand-primary to-brand-primary text-text-primary shadow-lg shadow-brand-primary/30 hover:from-brand-primary hover:to-brand-accent active:scale-[0.98]',
    secondary: 'bg-bg-elevated text-text-primary hover:bg-bg-overlay border border-border-default',
    ghost: 'text-text-muted hover:text-text-primary hover:bg-bg-elevated',
    outline:
      'border border-border-muted text-text-secondary hover:bg-bg-elevated hover:border-brand-primary/50',
    danger:
      'bg-status-error/20 text-status-error border border-status-error/30 hover:bg-status-error/30',
    success:
      'bg-status-success/20 text-status-success border border-status-success/30 hover:bg-status-success/30',
  },

  size: {
    xs: 'px-2 py-1 text-xs',
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base',
  },
} as const;

export const input = {
  base: 'w-full rounded-lg bg-bg-surface text-text-primary transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary/50 disabled:opacity-50 disabled:cursor-not-allowed placeholder:text-text-disabled',

  state: {
    default: 'border border-border-default',
    error: 'border border-status-error/50',
    success: 'border border-status-success/50',
  },

  size: {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-3 py-2.5 text-sm',
    lg: 'px-4 py-3 text-base',
  },
} as const;

export const card = {
  base: 'rounded-xl backdrop-blur-xl',

  variant: {
    default: 'bg-gradient-to-br from-bg-elevated to-bg-surface border border-border-default',
    elevated:
      'bg-gradient-to-br from-bg-elevated to-bg-surface border border-border-default shadow-xl shadow-black/20',
    interactive:
      'bg-gradient-to-br from-bg-elevated to-bg-surface border border-border-default hover:border-brand-primary/30 cursor-pointer transition-colors',
    glass: 'bg-bg-elevated/40 backdrop-blur-2xl border border-border-default shadow-2xl',
  },

  padding: {
    none: '',
    sm: 'p-3',
    md: 'p-4',
    lg: 'p-6',
  },
} as const;

export const badge = {
  base: 'inline-flex items-center gap-1 rounded-full text-xs font-medium',

  variant: {
    default: 'bg-bg-elevated text-text-secondary',
    success: 'bg-status-success/20 text-status-success border border-status-success/30',
    warning: 'bg-status-warning/20 text-status-warning border border-status-warning/30',
    error: 'bg-status-error/20 text-status-error border border-status-error/30',
    info: 'bg-status-info/20 text-status-info border border-status-info/30',
    primary: 'bg-brand-primary/20 text-brand-accent border border-brand-primary/30',
    new: 'bg-status-success/20 text-status-success',
    beta: 'bg-status-warning/20 text-status-warning',
  },

  size: {
    xs: 'px-1.5 py-0.5',
    sm: 'px-2 py-0.5',
    md: 'px-2.5 py-1',
  },
} as const;

export const alert = {
  base: 'px-4 py-3 rounded-lg border',

  variant: {
    error: 'bg-status-error/10 border-status-error/20 text-status-error',
    warning: 'bg-status-warning/10 border-status-warning/20 text-status-warning',
    success: 'bg-status-success/10 border-status-success/20 text-status-success',
    info: 'bg-status-info/10 border-status-info/20 text-status-info',
  },
} as const;

export const modal = {
  overlay: 'fixed inset-0 z-50 flex items-center justify-center p-4',
  backdrop: 'absolute inset-0 bg-black/60 backdrop-blur-sm',
  content:
    'relative bg-bg-surface border border-border-default rounded-xl shadow-2xl max-h-[85vh] overflow-y-auto',

  size: {
    sm: 'max-w-md w-full',
    md: 'max-w-2xl w-full',
    lg: 'max-w-4xl w-full',
    xl: 'max-w-6xl w-full',
    full: 'max-w-7xl w-full',
  },
} as const;

export const drawer = {
  overlay: 'fixed inset-0 z-50',
  backdrop: 'absolute inset-0 bg-black/50 backdrop-blur-sm',
  content:
    'absolute right-0 top-0 h-full bg-bg-surface border-l border-border-default shadow-2xl overflow-y-auto',

  size: {
    sm: 'w-80',
    md: 'w-96',
    lg: 'w-[480px]',
    xl: 'w-[600px]',
  },
} as const;

export const status = {
  dot: 'inline-block w-2 h-2 rounded-full',

  color: {
    success: 'bg-status-success',
    warning: 'bg-status-warning',
    error: 'bg-status-error',
    info: 'bg-status-info',
    inactive: 'bg-text-disabled',
    online: 'bg-status-success',
    offline: 'bg-text-disabled',
    pending: 'bg-status-info',
  },

  withLabel: 'inline-flex items-center gap-2',
  pulse: 'animate-pulse',
} as const;

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
