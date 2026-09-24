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
 * BRAND COLORS (fleet hue set C, owner 2026-09-15; values in
 * theme/product-niac.css — this block had drifted a full generation behind it,
 * still naming the #4f46e5 indigo the CSS left in 2026-05):
 * - Primary:       Violet #6a3fa8 light / #9569d3 dark - filled buttons, glows
 * - Accent:        Violet #56328a light / #b492e3 dark - gradient second stop
 * - Text-accent:   #6a3fa8 light / #b492e3 dark - text and links
 * - Accent-gold:   Mustard #8a6208 light / #b88a1e dark - eyebrows, accent icons.
 *   Shared across the fleet in theme/msn-shared.css; it was a per-product
 *   --color-brand-gold with the same value in all four repos until UI-FLEET-1.
 *
 * STATUS COLORS (fleet-shared in theme/msn-shared.css, not NIAC's to set):
 * - Success: #2a7146 light / #3cb46e dark
 * - Warning: #845e08 light / #ca9721 dark
 * - Danger:  #b43939 light / #de8787 dark
 * - Info:    #1263a8 light / #5da5e5 dark
 *
 * PILL TEXT: text on a same-hue wash (bg-status-error/20, bg-brand-primary/10)
 * is the hue's -strong token, never the bare hue (product-niac.css,
 * pillText.test.ts). Washes under pill text stop at /20.
 *
 * MODULE ACCENTS (5 differentiated hues, icons and chart series only):
 * - Topology:  #6a3fa8 violet  - network map / graph view (= the brand anchor)
 * - Protocols: #0b6b62 teal    - protocol stack, packet types
 * - Analyze:   #9333a8 fuchsia - capture inspection
 * - Inject:    #b4143c rose    - traffic generation
 * - Templates: #96450a amber   - saved configs / library
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
