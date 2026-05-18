import type { DiffType } from './types';

export interface DiffStyleSet {
  bg: string;
  border: string;
  text: string;
}

/**
 * Get styling classes for diff type
 */
export function getDiffStyles(type: DiffType): DiffStyleSet {
  switch (type) {
    case 'added':
      return {
        bg: 'bg-status-success/10',
        border: 'border-status-success/30',
        text: 'text-status-success',
      };
    case 'removed':
      return {
        bg: 'bg-status-error/10',
        border: 'border-status-error/30',
        text: 'text-status-error',
      };
    case 'modified':
      return {
        bg: 'bg-status-warning/10',
        border: 'border-status-warning/30',
        text: 'text-status-warning',
      };
    default:
      return {
        bg: '',
        border: 'border-transparent',
        text: 'text-text-secondary',
      };
  }
}
