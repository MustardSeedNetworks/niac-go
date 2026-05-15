import type { FC, ReactNode } from 'react';

/**
 * Small pill button used for the All / Built-in / Saved / Local source
 * filter row above the configs list. Renders the label with a small
 * count chip on the right.
 */
export const SourceChip: FC<{
  active: boolean;
  onClick: () => void;
  label: string;
  count: number;
}> = ({ active, onClick, label, count }) => (
  <button
    type="button"
    onClick={onClick}
    aria-pressed={active}
    className={`flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
      active
        ? 'bg-violet-500/20 text-violet-100 ring-1 ring-violet-400/40'
        : 'bg-gray-800/60 text-gray-400 ring-1 ring-white/10 hover:bg-white/5 hover:text-gray-200'
    }`}
  >
    <span>{label}</span>
    <span
      className={`rounded-full px-1.5 py-0.5 text-[10px] ${
        active ? 'bg-violet-500/30 text-violet-100' : 'bg-white/10 text-gray-300'
      }`}
    >
      {count}
    </span>
  </button>
);

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
