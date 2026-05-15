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
        ? 'bg-violet-500/20 text-violet-100'
        : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
    }`}
  >
    {icon}
  </button>
);
