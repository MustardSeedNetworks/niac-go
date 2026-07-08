import { AlignJustify, Columns2 } from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { DiffBlockComponent } from './diff-viewer/DiffBlock';
import { DiffBlockOverlayComponent } from './diff-viewer/DiffBlockOverlay';
import { computeDiff } from './diff-viewer/diff-algorithm';
import type { DiffViewerProps, MergeDecision } from './diff-viewer/types';

type ViewMode = 'block' | 'overlay';

export { computeDiff } from './diff-viewer/diff-algorithm';
export { generateMergedContent } from './diff-viewer/merge-utils';
// Re-export types and utilities for backwards compatibility
export type { DiffBlock, DiffLine, DiffType, MergeDecision } from './diff-viewer/types';

/**
 * Statistics bar showing additions, deletions, and modifications
 */
const DiffStatsBar: FC<{
  additions: number;
  deletions: number;
  modifications: number;
  decisionsCount: number;
  changedBlocksCount: number;
  showMergeControls: boolean;
}> = ({
  additions,
  deletions,
  modifications,
  decisionsCount,
  changedBlocksCount,
  showMergeControls,
}) => {
  const { t } = useTranslation('pages');
  return (
    <div className="flex items-center gap-comfortable flex-wrap">
      <SmallText className="text-text-muted">{t('configDiff.changesLabel')}</SmallText>
      <Tag colorScheme="green" className="text-xs">
        {t('configDiff.additionsCount', { count: additions })}
      </Tag>
      <Tag colorScheme="red" className="text-xs">
        {t('configDiff.deletionsCount', { count: deletions })}
      </Tag>
      <Tag colorScheme="yellow" className="text-xs">
        {t('configDiff.modifiedBlocksCount', { count: modifications })}
      </Tag>
      {showMergeControls && (
        <SmallText className="text-text-muted ml-auto">
          {t('configDiff.decisionsMadeCount', {
            decisionsCount,
            changedBlocksCount,
          })}
        </SmallText>
      )}
    </div>
  );
};

/**
 * Column headers for left and right panels
 */
const ColumnHeaders: FC<{
  leftLabel: string;
  rightLabel: string;
}> = ({ leftLabel, rightLabel }) => (
  <div className="grid grid-cols-2 gap-px bg-bg-elevated rounded-t-lg overflow-hidden">
    <div className="bg-bg-surface/90 px-4 py-row">
      <SmallText className="text-text-secondary font-semibold">{leftLabel}</SmallText>
    </div>
    <div className="bg-bg-surface/90 px-4 py-row">
      <SmallText className="text-text-secondary font-semibold">{rightLabel}</SmallText>
    </div>
  </div>
);

/**
 * Block/Overlay merge-view toggle. "Block" keeps the two files in
 * separate side-by-side columns; "Overlay" shows a single unified
 * column with removed/added lines inlined.
 */
const ViewModeToggle: FC<{
  viewMode: ViewMode;
  onChange: (mode: ViewMode) => void;
}> = ({ viewMode, onChange }) => {
  const { t } = useTranslation('pages');
  return (
    <fieldset
      className="flex rounded border border-surface-border bg-bg-surface/60 p-0.5"
      aria-label={t('configDiff.viewModeLabel')}
    >
      <button
        type="button"
        data-testid="diff-view-mode-block"
        aria-pressed={viewMode === 'block'}
        title={t('configDiff.blockViewOption')}
        onClick={() => onChange('block')}
        className={`flex items-center gap-tight rounded px-cell py-compact text-xs transition-colors ${
          viewMode === 'block'
            ? 'bg-brand-primary/20 text-brand-accent'
            : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
        }`}
      >
        <Columns2 className={iconSizes.xs} />
        {t('configDiff.blockViewOption')}
      </button>
      <button
        type="button"
        data-testid="diff-view-mode-overlay"
        aria-pressed={viewMode === 'overlay'}
        title={t('configDiff.overlayViewOption')}
        onClick={() => onChange('overlay')}
        className={`flex items-center gap-tight rounded px-cell py-compact text-xs transition-colors ${
          viewMode === 'overlay'
            ? 'bg-brand-primary/20 text-brand-accent'
            : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
        }`}
      >
        <AlignJustify className={iconSizes.xs} />
        {t('configDiff.overlayViewOption')}
      </button>
    </fieldset>
  );
};

/**
 * Empty state when no content is provided
 */
const EmptyState: FC = () => {
  const { t } = useTranslation('pages');
  return (
    <div className="flex-center h-64 text-text-muted">
      <SmallText>{t('configDiff.uploadFilesToCompare')}</SmallText>
    </div>
  );
};

/**
 * Side-by-side diff viewer component
 */
export const DiffViewer: FC<DiffViewerProps> = ({
  leftContent,
  rightContent,
  leftLabel = 'Original',
  rightLabel = 'Modified',
  mergeDecisions,
  onMergeDecision,
  showMergeControls = true,
}) => {
  const [viewMode, setViewMode] = useState<ViewMode>('block');

  // Compute diff blocks
  const diffBlocks = useMemo(
    () => computeDiff(leftContent, rightContent),
    [leftContent, rightContent],
  );

  // Count changes
  const stats = useMemo(() => {
    let additions = 0;
    let deletions = 0;
    let modifications = 0;

    for (const block of diffBlocks) {
      if (block.type === 'added') {
        additions += block.rightLines.length;
      } else if (block.type === 'removed') {
        deletions += block.leftLines.length;
      } else if (block.type === 'modified') {
        modifications++;
      }
    }

    return { additions, deletions, modifications };
  }, [diffBlocks]);

  const changedBlocksCount = useMemo(
    () => diffBlocks.filter((b) => b.type !== 'unchanged').length,
    [diffBlocks],
  );

  const handleDecision = useCallback(
    (blockId: string) => (choice: MergeDecision['choice']) => {
      onMergeDecision(blockId, choice);
    },
    [onMergeDecision],
  );

  if (!(leftContent || rightContent)) {
    return <EmptyState />;
  }

  return (
    <div className="stack-lg">
      {/* Stats bar + view mode toggle */}
      <div className="flex items-center justify-between gap-comfortable flex-wrap">
        <DiffStatsBar
          additions={stats.additions}
          deletions={stats.deletions}
          modifications={stats.modifications}
          decisionsCount={mergeDecisions.size}
          changedBlocksCount={changedBlocksCount}
          showMergeControls={showMergeControls}
        />
        <ViewModeToggle viewMode={viewMode} onChange={setViewMode} />
      </div>

      {/* Column headers (block mode only — overlay is a single column) */}
      {viewMode === 'block' && <ColumnHeaders leftLabel={leftLabel} rightLabel={rightLabel} />}

      {/* Diff content */}
      <div className="rounded-lg border border-surface-border bg-bg-base/50 overflow-hidden max-h-[500px] overflow-y-auto">
        {diffBlocks.map((block) =>
          viewMode === 'block' ? (
            <DiffBlockComponent
              key={block.id}
              block={block}
              decision={mergeDecisions.get(block.id)}
              onDecision={handleDecision(block.id)}
              showMergeControls={showMergeControls}
            />
          ) : (
            <DiffBlockOverlayComponent
              key={block.id}
              block={block}
              decision={mergeDecisions.get(block.id)}
              onDecision={handleDecision(block.id)}
              showMergeControls={showMergeControls}
            />
          ),
        )}
      </div>
    </div>
  );
};
