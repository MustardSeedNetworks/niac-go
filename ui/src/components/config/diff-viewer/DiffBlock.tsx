import { ChevronLeft, ChevronRight, ChevronsLeftRight } from 'lucide-react';
import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../../constants/sizes';
import { Button } from '../../../ui/Button';
import { Tag } from '../../../ui/Tag';
import { SmallText } from '../../../ui/Typography';
import { DiffLineComponent } from './DiffLine';
import type { DiffBlock, DiffLine, MergeDecision } from './types';

interface DiffBlockComponentProps {
  block: DiffBlock;
  decision?: MergeDecision;
  onDecision: (choice: MergeDecision['choice']) => void;
  showMergeControls: boolean;
}

/**
 * Create padded lines array for alignment
 */
function padLines(lines: DiffLine[], targetLength: number): DiffLine[] {
  const paddedLines = [...lines];
  while (paddedLines.length < targetLength) {
    paddedLines.push({
      lineNumber: -1,
      content: '',
      type: 'unchanged',
    });
  }
  return paddedLines;
}

/**
 * Merge control buttons for a diff block. Exported so both the
 * side-by-side (DiffBlock) and unified (DiffBlockOverlay) renderers
 * share one implementation.
 */
export const DiffBlockMergeControls: FC<{
  decision?: MergeDecision;
  onDecision: (choice: MergeDecision['choice']) => void;
}> = ({ decision, onDecision }) => {
  const { t } = useTranslation('pages');
  const choiceLabels: Record<MergeDecision['choice'], string> = {
    left: t('configDiff.leftChoiceLabel'),
    both: t('configDiff.bothChoiceLabel'),
    right: t('configDiff.rightChoiceLabel'),
    none: t('configDiff.noneChoiceLabel'),
  };

  return (
    <div className="flex-center gap-compact py-row px-4 bg-bg-surface/80 border-b border-surface-border">
      <SmallText className="text-text-muted mr-2">{t('configDiff.acceptLabel')}</SmallText>
      <Button
        size="sm"
        variant={decision?.choice === 'left' ? undefined : 'outline'}
        tone={decision?.choice === 'left' ? 'violet' : undefined}
        onClick={() => onDecision('left')}
        leftIcon={<ChevronLeft className={iconSizes.xs} />}
      >
        {choiceLabels.left}
      </Button>
      <Button
        size="sm"
        variant={decision?.choice === 'both' ? undefined : 'outline'}
        tone={decision?.choice === 'both' ? 'violet' : undefined}
        onClick={() => onDecision('both')}
        leftIcon={<ChevronsLeftRight className={iconSizes.xs} />}
      >
        {choiceLabels.both}
      </Button>
      <Button
        size="sm"
        variant={decision?.choice === 'right' ? undefined : 'outline'}
        tone={decision?.choice === 'right' ? 'violet' : undefined}
        onClick={() => onDecision('right')}
        leftIcon={<ChevronRight className={iconSizes.xs} />}
      >
        {choiceLabels.right}
      </Button>
      {decision && (
        <Tag colorScheme="purple" className="ml-inline text-xs">
          {choiceLabels[decision.choice].toUpperCase()}
        </Tag>
      )}
    </div>
  );
};

/**
 * Diff block with merge controls
 */
export const DiffBlockComponent: FC<DiffBlockComponentProps> = memo(
  ({ block, decision, onDecision, showMergeControls }) => {
    const isChanged = block.type !== 'unchanged';
    const maxLines = Math.max(block.leftLines.length, block.rightLines.length);

    const paddedLeftLines = padLines(block.leftLines, maxLines);
    const paddedRightLines = padLines(block.rightLines, maxLines);

    return (
      <div
        className={`${isChanged ? 'border border-surface-border rounded-lg overflow-hidden mb-2' : ''}`}
      >
        {/* Merge controls for changed blocks */}
        {isChanged && showMergeControls && (
          <DiffBlockMergeControls decision={decision} onDecision={onDecision} />
        )}

        {/* Side-by-side diff display */}
        <div className="grid grid-cols-2 divide-x divide-knob/10">
          {/* Left panel */}
          <div className="overflow-x-auto">
            {paddedLeftLines.map((line, idx) => (
              <DiffLineComponent key={`left-${block.id}-${idx}`} line={line} side="left" />
            ))}
          </div>

          {/* Right panel */}
          <div className="overflow-x-auto">
            {paddedRightLines.map((line, idx) => (
              <DiffLineComponent key={`right-${block.id}-${idx}`} line={line} side="right" />
            ))}
          </div>
        </div>
      </div>
    );
  },
);

DiffBlockComponent.displayName = 'DiffBlockComponent';
