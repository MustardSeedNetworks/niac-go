/**
 * =============================================================================
 * NIAC DESIGN SYSTEM - Mustard Seed Networks
 * =============================================================================
 *
 * Centralized design tokens and utilities for consistent UI across the app.
 *
 * ARCHITECTURE:
 * 1. theme/*.css - every colour value, light and dark, and the Tailwind @theme
 * 2. This file (theme.ts) - TypeScript class tokens and utility functions
 *
 * BRAND (fleet hue set C): brand-primary, brand-accent, text-accent, set in
 * theme/product-niac.css; accent-gold is fleet-shared in theme/msn-shared.css.
 *
 * STATUS (fleet-shared in theme/msn-shared.css, not NIAC's to set):
 * status-success, status-warning, status-error, status-info.
 *
 * PILL TEXT: text on a same-hue wash (bg-status-error/20, bg-brand-primary/10)
 * is the hue's -strong token, never the bare hue (product-niac.css,
 * pillText.test.ts). Washes under pill text stop at /20.
 *
 * MODULE ACCENTS (icons and chart series only): module-topology (= the brand
 * anchor), module-protocols, module-analyze, module-inject, module-templates.
 *
 * DEVICE COLORS (NIAC-specific, orthogonal to brand — see themeDeviceColors.ts):
 * Router, Switch, Firewall, Server, Workstation, AP, IoT, Unknown.
 *
 * USAGE:
 * import { spacing, button, cn, deviceColor } from '../styles/theme';
 * <button className={cn(button.base, button.variant.primary)}>Action</button>
 *
 * See THEMING.md for the full canonical token map and migration phase notes.
 * =============================================================================
 */

// biome-ignore lint/performance/noBarrelFile: Design system barrel file is intentional for API stability
export { alert, badge, button, card, drawer, icon, input, modal, status } from './themeComponents';
export { deviceColor, linkSpeedColor, protocolColor } from './themeDeviceColors';
export { border, layout, radius } from './themeLayout';
export { spacing } from './themeSpacing';
export { typography } from './themeTypography';
export {
  badgeClass,
  buttonClass,
  cardClass,
  cn,
  drawerClass,
  getDeviceColor,
  inputClass,
  modalClass,
} from './themeUtils';
