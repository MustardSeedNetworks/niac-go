/**
 * =============================================================================
 * SPACING SCALE
 * =============================================================================
 *
 * Spacing tokens based on 4px grid.
 * Use these semantic spacing utilities for consistency.
 */

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
