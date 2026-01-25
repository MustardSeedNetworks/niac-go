import { type FC, useCallback, useMemo } from 'react';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { DiffBlockComponent } from './diff-viewer/DiffBlock';
import { computeDiff } from './diff-viewer/diff-algorithm';
import type { DiffViewerProps, MergeDecision } from './diff-viewer/types';

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
}) => (
  <div className="flex items-center gap-4 flex-wrap">
    <SmallText className="text-gray-400">Changes:</SmallText>
    <Tag colorScheme="green" className="text-xs">
      +{additions} additions
    </Tag>
    <Tag colorScheme="red" className="text-xs">
      -{deletions} deletions
    </Tag>
    <Tag colorScheme="yellow" className="text-xs">
      {modifications} modified blocks
    </Tag>
    {showMergeControls && (
      <SmallText className="text-gray-400 ml-auto">
        {decisionsCount} / {changedBlocksCount} decisions made
      </SmallText>
    )}
  </div>
);

/**
 * Column headers for left and right panels
 */
const ColumnHeaders: FC<{
  leftLabel: string;
  rightLabel: string;
}> = ({ leftLabel, rightLabel }) => (
  <div className="grid grid-cols-2 gap-px bg-gray-800 rounded-t-lg overflow-hidden">
    <div className="bg-gray-900/90 px-4 py-2">
      <SmallText className="text-gray-300 font-semibold">{leftLabel}</SmallText>
    </div>
    <div className="bg-gray-900/90 px-4 py-2">
      <SmallText className="text-gray-300 font-semibold">{rightLabel}</SmallText>
    </div>
  </div>
);

/**
 * Empty state when no content is provided
 */
const EmptyState: FC = () => (
  <div className="flex items-center justify-center h-64 text-gray-400">
    <SmallText>Upload files to compare</SmallText>
  </div>
);

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
    <div className="space-y-4">
      {/* Stats bar */}
      <DiffStatsBar
        additions={stats.additions}
        deletions={stats.deletions}
        modifications={stats.modifications}
        decisionsCount={mergeDecisions.size}
        changedBlocksCount={changedBlocksCount}
        showMergeControls={showMergeControls}
      />

      {/* Column headers */}
      <ColumnHeaders leftLabel={leftLabel} rightLabel={rightLabel} />

      {/* Diff content */}
      <div className="rounded-lg border border-white/10 bg-gray-950/50 overflow-hidden max-h-[500px] overflow-y-auto">
        {diffBlocks.map((block) => (
          <DiffBlockComponent
            key={block.id}
            block={block}
            decision={mergeDecisions.get(block.id)}
            onDecision={handleDecision(block.id)}
            showMergeControls={showMergeControls}
          />
        ))}
      </div>
    </div>
  );
};
