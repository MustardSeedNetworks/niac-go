/**
 * themeModuleColors.ts — NIAC feature-module accent tokens (canonical 2026-05-22).
 * Re-exported through theme.ts.
 *
 * IMPORTANT: Use these for icons, small badges, and legend swatches only —
 * NOT for card backgrounds. Cards should remain consistent (bg-bg-elevated)
 * across all modules; the module accent identifies the feature in icon /
 * chip form.
 *
 * The five accents are deliberately differentiated instead of using a
 * single-color ramp, which keeps feature groups visually scannable.
 *
 * All values reference CSS variables in src/index.css `@theme`:
 *   --color-module-topology    Indigo-600 #4f46e5  (= brand anchor)
 *   --color-module-protocols   Teal-600   #0d9488
 *   --color-module-analyze     Fuchsia-600 #c026d3
 *   --color-module-inject      Rose-600   #e11d48
 *   --color-module-templates   Amber-600  #d97706
 *
 * Usage:
 *   import { moduleColor } from '../styles/theme';
 *   <TopologyIcon className={moduleColor.topology.icon} />
 *   <span className={cn(moduleColor.protocols.badge, 'px-2 py-1')}>HTTP</span>
 */

export const moduleColor = {
  // Topology — network map / graph view (= brand anchor)
  topology: {
    icon: 'text-module-topology',
    badge: 'bg-module-topology/20 text-module-topology',
    border: 'border-module-topology/30',
  },
  // Protocols — protocol stack, packet types
  protocols: {
    icon: 'text-module-protocols',
    badge: 'bg-module-protocols/20 text-module-protocols',
    border: 'border-module-protocols/30',
  },
  // Analyze — capture inspection, synthesis
  analyze: {
    icon: 'text-module-analyze',
    badge: 'bg-module-analyze/20 text-module-analyze',
    border: 'border-module-analyze/30',
  },
  // Inject — traffic generation, replay
  inject: {
    icon: 'text-module-inject',
    badge: 'bg-module-inject/20 text-module-inject',
    border: 'border-module-inject/30',
  },
  // Templates — saved configs, library
  templates: {
    icon: 'text-module-templates',
    badge: 'bg-module-templates/20 text-module-templates',
    border: 'border-module-templates/30',
  },
} as const;
