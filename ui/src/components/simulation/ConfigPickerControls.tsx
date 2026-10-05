import type { ReactNode } from 'react';
import { Tooltip } from '../../ui/Tooltip';

/**
 * One half of the grid/list view-density toggle. Two of these live
 * inside a <fieldset> in ConfigPicker so screen readers see them as
 * an exclusive group.
 */
export function ViewToggle({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: ReactNode;
  label: string;
}) {
  return (
    <Tooltip text={label}>
      <button
        type="button"
        onClick={onClick}
        aria-pressed={active}
        aria-label={label}
        className={`rounded px-cell py-compact transition-colors ${
          active
            ? 'bg-brand-primary/20 text-brand-primary-strong'
            : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
        }`}
      >
        {icon}
      </button>
    </Tooltip>
  );
}
