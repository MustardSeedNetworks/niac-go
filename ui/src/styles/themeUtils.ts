import { twMerge } from 'tailwind-merge';

import { badge, button, card, drawer, input, modal } from './themeComponents';
import { deviceColor } from './themeDeviceColors';

/**
 * =============================================================================
 * UTILITY FUNCTIONS
 * =============================================================================
 *
 * Helper functions for building class strings and accessing design tokens.
 */

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
