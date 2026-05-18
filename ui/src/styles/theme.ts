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
