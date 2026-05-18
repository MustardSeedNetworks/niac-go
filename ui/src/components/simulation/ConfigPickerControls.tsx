import type { FC, ReactNode } from 'react';

/**
 * One half of the grid/list view-density toggle. Two of these live
 * inside a <fieldset> in ConfigPicker so screen readers see them as
 * an exclusive group.
 */
export const ViewToggle: FC<{
  active: boolean;
  onClick: () => void;
  icon: ReactNode;
  label: string;
}> = ({ active, onClick, icon, label }) => (
  <button
    type="button"
    onClick={onClick}
    aria-pressed={active}
    title={label}
    className={`rounded px-2 py-1 transition-colors ${
      active
        ? 'bg-brand-500/20 text-brand-100'
        : 'text-text-muted hover:bg-white/5 hover:text-text-primary'
    }`}
  >
    {icon}
  </button>
);
