import { ChevronDown, Download } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { Button } from '../../ui/Button';

/**
 * ActionsMenu is the small "Export ▾" dropdown in the topology
 * header. Two options: PNG (image render of the canvas via
 * html-to-image) and JSON (the structured topology graph).
 *
 * Originally also carried "Reset layout" but that turned out to be
 * a recovery action operators reach for too often — Reset is now a
 * direct toolbar button. This menu stays small + scoped to "save
 * this view somewhere".
 *
 * Lives in its own file so TopologyPage.tsx stays under the
 * file-size red-flag threshold (800 lines).
 */
export const ActionsMenu: FC<{
  disabled?: boolean;
  onExportPNG: () => void;
  onExportJSON: () => void;
}> = ({ disabled, onExportPNG, onExportJSON }) => {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    const onDocClick = (e: MouseEvent) => {
      const target = e.target as Element | null;
      if (!target?.closest?.('[data-topology-actions]')) setOpen(false);
    };
    const onEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('click', onDocClick);
    document.addEventListener('keydown', onEsc);
    return () => {
      document.removeEventListener('click', onDocClick);
      document.removeEventListener('keydown', onEsc);
    };
  }, [open]);

  const handle = (fn: () => void) => () => {
    fn();
    setOpen(false);
  };

  return (
    <div className="relative" data-topology-actions>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setOpen((v) => !v)}
        disabled={disabled}
        leftIcon={<Download className="w-4 h-4" />}
        rightIcon={<ChevronDown className="w-3.5 h-3.5" />}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        Export
      </Button>
      {open && (
        <div
          role="menu"
          className="absolute right-0 mt-1 min-w-[180px] rounded-md border border-white/10 bg-gray-950/95 py-1 text-xs text-gray-200 shadow-xl backdrop-blur z-50"
        >
          <button
            type="button"
            role="menuitem"
            onClick={handle(onExportPNG)}
            className="flex w-full items-center justify-between gap-3 px-3 py-1.5 text-left transition-colors hover:bg-white/5 hover:text-white"
          >
            <span>Export as PNG</span>
            <span className="text-[10px] text-gray-500">image</span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={handle(onExportJSON)}
            className="flex w-full items-center justify-between gap-3 px-3 py-1.5 text-left transition-colors hover:bg-white/5 hover:text-white"
          >
            <span>Export topology JSON</span>
            <span className="text-[10px] text-gray-500">data</span>
          </button>
        </div>
      )}
    </div>
  );
};
