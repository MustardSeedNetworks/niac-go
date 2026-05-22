/**
 * themeTypography.ts — semantic typography tokens. CSS utility classes are
 * defined in index.css `@layer components`; use these TS constants for
 * programmatic styling. Re-exported through theme.ts.
 *
 * Canonical responsive type scale (2026-05-22) — harmonized with Seed and Stem.
 * Replaces the pre-audit flat scale that lacked responsive sizing and tracking.
 */

export const typography = {
  // Semantic heading classes (match CSS utilities in index.css).
  // Preferred way to style headings.
  heading: {
    h1: 'heading-1', // Page titles: 24px -> sm:30px bold, leading-tight, tracking-tight
    h2: 'heading-2', // Section/modal titles: 20px -> sm:24px semibold, leading-snug
    h3: 'heading-3', // Card titles: 18px -> sm:20px semibold, leading-snug
    h4: 'heading-4', // Subsections: 16px -> sm:18px medium, leading-snug
    section: 'section-title', // Category labels: 12px uppercase tracking-wider, fg-muted
  },

  // Body text variants
  body: {
    large: 'body-large', // 18px primary, leading-relaxed
    default: 'body', // 16px primary, leading-relaxed (most common)
    small: 'body-small', // 14px secondary, leading-relaxed
    caption: 'caption', // 12px muted, leading-normal (metadata)
    muted: 'body-small', // Legacy alias retained for backwards compatibility
  },

  // Utility classes
  label: 'label', // Form labels: 14px medium
  caption: 'caption', // Legacy top-level alias (prefer body.caption)
  code: 'code', // Monospace with background chip

  // Raw size classes (use sparingly — prefer semantic variants above)
  size: {
    xs: 'text-xs', // 12px
    sm: 'text-sm', // 14px
    base: 'text-base', // 16px
    lg: 'text-lg', // 18px
    xl: 'text-xl', // 20px
    '2xl': 'text-2xl', // 24px
    '3xl': 'text-3xl', // 30px
  },

  // Font weights
  weight: {
    normal: 'font-normal', // 400
    medium: 'font-medium', // 500
    semibold: 'font-semibold', // 600
    bold: 'font-bold', // 700
  },

  // Font families
  family: {
    body: 'font-body', // Inter Variable
    display: 'font-display', // Inter Variable (display)
    mono: 'font-mono', // JetBrains Mono
  },

  // Line heights
  leading: {
    tight: 'leading-tight', // 1.25 — headings
    snug: 'leading-snug', // 1.375 — subheadings
    normal: 'leading-normal', // 1.5 — default
    relaxed: 'leading-relaxed', // 1.625 — body text
  },
} as const;
