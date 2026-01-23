// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

import { twMerge } from 'tailwind-merge';

/**
 * =============================================================================
 * NIAC DESIGN SYSTEM - Mustard Seed Networks
 * =============================================================================
 *
 * Centralized design tokens and utilities for consistent UI across the app.
 *
 * ARCHITECTURE:
 * 1. CSS Variables (index.css) - Core color tokens (dark theme)
 * 2. This file (theme.ts) - TypeScript tokens and utility functions
 * 3. Tailwind Classes - Composable styling patterns
 *
 * BRAND COLORS:
 * - Primary: Violet (#8b5cf6) - Actions, links, focus states
 * - Accent: Lighter Violet (#a78bfa) - Hover states
 *
 * STATUS COLORS (Industry Standard):
 * - Success: Green (#22c55e) - Positive states
 * - Warning: Amber (#f59e0b) - Caution states
 * - Error: Red (#ef4444) - Error/danger states
 * - Info: Blue (#3b82f6) - Informational states
 *
 * DEVICE COLORS (NIAC specific):
 * - Router: Blue - Network routing
 * - Switch: Green - Layer 2 switching
 * - Firewall: Red - Security devices
 * - Server: Orange - Compute nodes
 * - Workstation: Gray - End devices
 * - Access Point: Purple - Wireless
 * - IoT: Teal - Internet of Things
 *
 * USAGE:
 * import { spacing, button, cn, deviceColor } from '../styles/theme';
 * <button className={cn(button.base, button.variant.primary)}>Action</button>
 *
 * =============================================================================
 */

// ============================================================================
// SPACING SCALE
// ============================================================================

/**
 * Spacing scale - based on 4px grid
 * Use these semantic spacing utilities for consistency.
 */
export const spacing = {
  // Chip/pill padding
  chip: {
    sm: 'px-3 py-1',
    md: 'px-3 py-1.5',
    lg: 'px-3 py-2',
  },

  // Tab button padding
  tab: 'py-2.5 px-3',

  // Card padding
  card: {
    sm: 'p-3',
    md: 'p-4',
    lg: 'p-6',
  },

  // Modal padding
  modal: {
    sm: 'p-4',
    md: 'p-6',
    lg: 'p-8',
  },

  // Section gaps
  section: 'space-y-6',
  sectionLg: 'space-y-8',

  // Inline gaps
  inline: {
    xs: 'gap-1',
    sm: 'gap-2',
    md: 'gap-3',
    lg: 'gap-4',
  },

  // Stack gaps (vertical)
  stack: {
    xs: 'space-y-1',
    sm: 'space-y-2',
    md: 'space-y-3',
    lg: 'space-y-4',
  },

  // Badge padding
  badge: {
    xs: 'px-1.5 py-0.5',
    sm: 'px-2 py-0.5',
    md: 'px-2.5 py-1',
  },

  // Drawer padding
  drawer: 'px-4 sm:px-6 py-4',
} as const;

// ============================================================================
// TYPOGRAPHY
// ============================================================================

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

// ============================================================================
// COMPONENT VARIANTS
// ============================================================================

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

// ============================================================================
// STATUS INDICATORS
// ============================================================================

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

// ============================================================================
// ICON SIZING
// ============================================================================

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

// ============================================================================
// BORDER & RADIUS
// ============================================================================

export const radius = {
  none: 'rounded-none',
  sm: 'rounded-sm',
  default: 'rounded',
  md: 'rounded-md',
  lg: 'rounded-lg',
  xl: 'rounded-xl',
  full: 'rounded-full',
} as const;

export const border = {
  default: 'border border-white/10',
  subtle: 'border border-white/5',
  muted: 'border border-white/20',
  focus: 'border border-violet-500/50',
  error: 'border border-red-500/50',
  divider: 'border-t border-white/10',
} as const;

// ============================================================================
// LAYOUT PATTERNS
// ============================================================================

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

// ============================================================================
// DEVICE COLORS - NIAC specific
// ============================================================================

/**
 * Device type colors - accent colors for network device visualization
 */
export const deviceColor = {
  router: {
    icon: 'text-blue-500',
    badge: 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
    border: 'border-blue-500/30',
    bg: 'bg-blue-500',
  },
  switch: {
    icon: 'text-emerald-500',
    badge: 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30',
    border: 'border-emerald-500/30',
    bg: 'bg-emerald-500',
  },
  firewall: {
    icon: 'text-red-500',
    badge: 'bg-red-500/20 text-red-400 border border-red-500/30',
    border: 'border-red-500/30',
    bg: 'bg-red-500',
  },
  server: {
    icon: 'text-orange-500',
    badge: 'bg-orange-500/20 text-orange-400 border border-orange-500/30',
    border: 'border-orange-500/30',
    bg: 'bg-orange-500',
  },
  workstation: {
    icon: 'text-gray-400',
    badge: 'bg-gray-500/20 text-gray-400 border border-gray-500/30',
    border: 'border-gray-500/30',
    bg: 'bg-gray-500',
  },
  ap: {
    icon: 'text-purple-500',
    badge: 'bg-purple-500/20 text-purple-400 border border-purple-500/30',
    border: 'border-purple-500/30',
    bg: 'bg-purple-500',
  },
  iot: {
    icon: 'text-teal-500',
    badge: 'bg-teal-500/20 text-teal-400 border border-teal-500/30',
    border: 'border-teal-500/30',
    bg: 'bg-teal-500',
  },
  unknown: {
    icon: 'text-gray-500',
    badge: 'bg-gray-500/20 text-gray-400 border border-gray-500/30',
    border: 'border-gray-500/30',
    bg: 'bg-gray-500',
  },
} as const;

/**
 * Protocol colors - for packet/protocol identification
 */
export const protocolColor = {
  arp: { icon: 'text-emerald-500', badge: 'bg-emerald-500/20 text-emerald-400' },
  icmp: { icon: 'text-blue-500', badge: 'bg-blue-500/20 text-blue-400' },
  dns: { icon: 'text-purple-500', badge: 'bg-purple-500/20 text-purple-400' },
  dhcp: { icon: 'text-orange-500', badge: 'bg-orange-500/20 text-orange-400' },
  snmp: { icon: 'text-teal-500', badge: 'bg-teal-500/20 text-teal-400' },
  lldp: { icon: 'text-pink-500', badge: 'bg-pink-500/20 text-pink-400' },
  cdp: { icon: 'text-yellow-500', badge: 'bg-yellow-500/20 text-yellow-400' },
  http: { icon: 'text-indigo-500', badge: 'bg-indigo-500/20 text-indigo-400' },
  tcp: { icon: 'text-cyan-500', badge: 'bg-cyan-500/20 text-cyan-400' },
  udp: { icon: 'text-violet-500', badge: 'bg-violet-500/20 text-violet-400' },
} as const;

/**
 * Link speed colors - for topology visualization
 */
export const linkSpeedColor = {
  '10m': 'text-gray-400',
  '100m': 'text-emerald-500',
  '1g': 'text-blue-500',
  '10g': 'text-purple-500',
  '25g': 'text-orange-500',
  '40g': 'text-pink-500',
  '100g': 'text-yellow-500',
  trunk: 'text-cyan-500',
} as const;

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Combine class names with Tailwind class conflict resolution.
 */
export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return twMerge(classes.filter(Boolean).join(' '));
}

/**
 * Build a button class string
 */
export function buttonClass(
  variant: keyof typeof button.variant = 'primary',
  size: keyof typeof button.size = 'md',
  className?: string,
): string {
  return cn(button.base, button.variant[variant], button.size[size], className);
}

/**
 * Build a card class string
 */
export function cardClass(
  variant: keyof typeof card.variant = 'default',
  padding: keyof typeof card.padding = 'md',
  className?: string,
): string {
  return cn(card.base, card.variant[variant], card.padding[padding], className);
}

/**
 * Build a badge class string
 */
export function badgeClass(
  variant: keyof typeof badge.variant = 'default',
  size: keyof typeof badge.size = 'sm',
  className?: string,
): string {
  return cn(badge.base, badge.variant[variant], badge.size[size], className);
}

/**
 * Build an input class string
 */
export function inputClass(
  state: keyof typeof input.state = 'default',
  size: keyof typeof input.size = 'md',
  className?: string,
): string {
  return cn(input.base, input.state[state], input.size[size], className);
}

/**
 * Build a modal class string
 */
export function modalClass(size: keyof typeof modal.size = 'md', className?: string): string {
  return cn(modal.content, modal.size[size], className);
}

/**
 * Build a drawer class string
 */
export function drawerClass(size: keyof typeof drawer.size = 'md', className?: string): string {
  return cn(drawer.content, drawer.size[size], className);
}

/**
 * Get device color config by type
 */
export function getDeviceColor(type: string): (typeof deviceColor)[keyof typeof deviceColor] {
  const normalizedType = type.toLowerCase().replace(/[^a-z]/g, '');
  if (normalizedType in deviceColor) {
    return deviceColor[normalizedType as keyof typeof deviceColor];
  }
  return deviceColor.unknown;
}
