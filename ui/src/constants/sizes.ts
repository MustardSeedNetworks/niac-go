/**
 * Icon size tokens. Use these instead of raw `h-N w-N` Tailwind classes
 * on lucide icons so sizes stay consistent across the UI.
 *
 * Convention:
 *   xs  → inline badges, dense lists                (12px)
 *   sm  → secondary chrome (chevrons, copy-icons)   (14px)
 *   md  → buttons, table rows, default              (16px)
 *   lg  → primary nav, card headers                 (20px)
 *   xl  → hero icons in cards                       (24px)
 *   2xl → large empty-state placeholders            (32px)
 *   3xl → page-level empty-state placeholders       (48px)
 */
export const iconSizes = {
  xs: 'h-3 w-3',
  sm: 'h-3.5 w-3.5',
  md: 'h-4 w-4',
  lg: 'h-5 w-5',
  xl: 'h-6 w-6',
  '2xl': 'h-8 w-8',
  '3xl': 'h-12 w-12',
} as const;

export type IconSize = keyof typeof iconSizes;

/**
 * Text size tokens. Tailwind already ships `text-xs … text-3xl`, but
 * exposing them as named constants lets us swap the scale in one place
 * if the design system changes.
 */
export const textSizes = {
  xs: 'text-xs',
  sm: 'text-sm',
  base: 'text-base',
  lg: 'text-lg',
  xl: 'text-xl',
  '2xl': 'text-2xl',
  '3xl': 'text-3xl',
} as const;

export type TextSize = keyof typeof textSizes;
