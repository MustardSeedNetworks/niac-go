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
        bg: 'bg-green-500/10',
        border: 'border-green-500/30',
        text: 'text-green-300',
      };
    case 'removed':
      return {
        bg: 'bg-red-500/10',
        border: 'border-red-500/30',
        text: 'text-red-300',
      };
    case 'modified':
      return {
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/30',
        text: 'text-yellow-300',
      };
    default:
      return {
        bg: '',
        border: 'border-transparent',
        text: 'text-gray-300',
      };
  }
}
