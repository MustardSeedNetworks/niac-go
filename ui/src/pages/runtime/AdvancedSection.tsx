import { type FC, useState } from 'react';
import { SmallText } from '../../ui/Typography';
import { GlobalDebugLevelCard } from './GlobalDebugLevelCard';

/**
 * AdvancedSection wraps the page's power-user knobs (currently only
 * the global debug-level card) behind a collapsed <details>. Hidden by
 * default so the page reads as a clean Start/Stop flow for the 90% case.
 */
export const AdvancedSection: FC = () => {
  const [open, setOpen] = useState(false);
  return (
    <details
      className="rounded border border-white/10 bg-gray-950/40"
      open={open}
      onToggle={(e) => setOpen(e.currentTarget.open)}
    >
      <summary className="flex cursor-pointer items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:text-white">
        <span className="text-gray-500">{open ? '▾' : '▸'}</span>
        <span>Advanced</span>
        <SmallText className="text-gray-500">(global protocol debug level)</SmallText>
      </summary>
      <div className="border-t border-white/10 p-3">
        <GlobalDebugLevelCard />
      </div>
    </details>
  );
};
