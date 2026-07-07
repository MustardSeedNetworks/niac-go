import { type FC, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { SmallText } from '../../ui/Typography';
import { GlobalDebugLevelCard } from './GlobalDebugLevelCard';

/**
 * AdvancedSection wraps the page's power-user knobs (currently only
 * the global debug-level card) behind a collapsed <details>. Hidden by
 * default so the page reads as a clean Start/Stop flow for the 90% case.
 */
export const AdvancedSection: FC = () => {
  const { t } = useTranslation('pages');
  const [open, setOpen] = useState(false);
  return (
    <details
      className="rounded border border-surface-border bg-bg-base/40"
      open={open}
      onToggle={(e) => setOpen(e.currentTarget.open)}
    >
      <summary className="flex cursor-pointer items-center gap-compact px-3 py-row text-sm text-text-secondary hover:text-text-primary">
        <span className="text-text-muted">{open ? '▾' : '▸'}</span>
        <span>{t('runtime.advancedLabel')}</span>
        <SmallText className="text-text-muted">{t('runtime.advancedHint')}</SmallText>
      </summary>
      <div className="border-t border-surface-border pad-sm">
        <GlobalDebugLevelCard />
      </div>
    </details>
  );
};
