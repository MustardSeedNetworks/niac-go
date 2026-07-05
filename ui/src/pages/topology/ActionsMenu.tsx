import { ChevronDown, Download } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../ui/Button';

/**
 * ActionsMenu is the small "Export ▾" dropdown in the topology
 * header. PNG (image render of the canvas via html-to-image) and
 * JSON (the client-side graph snapshot) export the current view;
 * DOT and GraphML fetch the daemon's rendered topology from the
 * server for Graphviz / yEd / gephi interop.
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
  onExportDOT: () => void;
  onExportGraphML: () => void;
}> = ({ disabled, onExportPNG, onExportJSON, onExportDOT, onExportGraphML }) => {
  const { t } = useTranslation('pages');
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
          className="absolute right-0 mt-tight min-w-[180px] rounded-md border border-surface-border bg-bg-base/95 py-compact text-xs text-text-primary shadow-xl backdrop-blur z-50"
        >
          <button
            type="button"
            role="menuitem"
            onClick={handle(onExportPNG)}
            className="flex w-full items-center justify-between gap-default px-3 py-compact-md text-left transition-colors hover:bg-surface-hover hover:text-text-primary"
          >
            <span>{t('topology.actionsMenu.exportPng')}</span>
            <span className="text-[10px] text-text-muted">image</span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={handle(onExportJSON)}
            className="flex w-full items-center justify-between gap-default px-3 py-compact-md text-left transition-colors hover:bg-surface-hover hover:text-text-primary"
          >
            <span>{t('topology.actionsMenu.exportJson')}</span>
            <span className="text-[10px] text-text-muted">data</span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={handle(onExportDOT)}
            className="flex w-full items-center justify-between gap-default px-3 py-compact-md text-left transition-colors hover:bg-surface-hover hover:text-text-primary"
          >
            <span>{t('topology.actionsMenu.exportDot')}</span>
            <span className="text-[10px] text-text-muted">graphviz</span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={handle(onExportGraphML)}
            className="flex w-full items-center justify-between gap-default px-3 py-compact-md text-left transition-colors hover:bg-surface-hover hover:text-text-primary"
          >
            <span>{t('topology.actionsMenu.exportGraphml')}</span>
            <span className="text-[10px] text-text-muted">graphml</span>
          </button>
        </div>
      )}
    </div>
  );
};
