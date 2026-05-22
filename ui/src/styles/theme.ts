/**
 * =============================================================================
 * NIAC DESIGN SYSTEM - Mustard Seed Networks
 * =============================================================================
 *
 * Centralized design tokens and utilities for consistent UI across the app.
 *
 * ARCHITECTURE:
 * 1. CSS Variables (index.css) - Core color tokens for light/dark modes
 * 2. This file (theme.ts) - TypeScript tokens and utility functions
 * 3. Tailwind v4 @theme directive - CSS-first utility class generation
 *
 * BRAND COLORS (anchors constant across light + dark per 2026-05-22 audit):
 * - Primary:        Indigo #4f46e5 (Tailwind indigo-600) - filled buttons, glows
 * - Primary-strong: Indigo #3730a3 (indigo-700) - text/links on light surfaces (AA)
 * - Primary-soft:   Indigo #a5b4fc (indigo-300) - text/links on dark surfaces (AA)
 * - Brand-gold:     Mustard #d4a017 - cross-brand accent, warning, focus, premium
 *
 * STATUS COLORS (tied to brand; constant across modes):
 * - Success: #4caf50 (= seed-500)
 * - Warning: #d4a017 (= mustard-500, brand cross-accent)
 * - Danger:  #ef4444 (coral red)
 * - Info:    #1976d2 (= stem-500)
 *
 * MODULE ACCENTS (5 differentiated hues, constant across modes):
 * - Topology:  #4f46e5 indigo  - network map / graph view
 * - Protocols: #0d9488 teal    - protocol stack, packet types
 * - Analyze:   #c026d3 fuchsia - capture inspection
 * - Inject:    #e11d48 rose    - traffic generation
 * - Templates: #d97706 amber   - saved configs / library
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
export { moduleColor } from './themeModuleColors';
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
