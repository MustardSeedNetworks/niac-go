import { ChevronLeft, ChevronRight, ChevronsLeftRight } from 'lucide-react';
import { type FC, memo } from 'react';
import { Button } from '../../ui/Button';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { DiffLineComponent } from './DiffLineComponent';
import type { DiffBlock, MergeDecision } from './diffUtils';

interface DiffBlockComponentProps {
  block: DiffBlock;
  decision?: MergeDecision;
  onDecision: (choice: MergeDecision['choice']) => void;
  showMergeControls: boolean;
}

/**
 * Diff block with merge controls
 */
export const DiffBlockComponent: FC<DiffBlockComponentProps> = memo(
  ({ block, decision, onDecision, showMergeControls }) => {
    const isChanged = block.type !== 'unchanged';
    const maxLines = Math.max(block.leftLines.length, block.rightLines.length);

    // Pad lines to equal length for alignment
    const paddedLeftLines = [...block.leftLines];
    const paddedRightLines = [...block.rightLines];

    while (paddedLeftLines.length < maxLines) {
      paddedLeftLines.push({
        lineNumber: -1,
        content: '',
        type: 'unchanged',
      });
    }

    while (paddedRightLines.length < maxLines) {
      paddedRightLines.push({
        lineNumber: -1,
        content: '',
        type: 'unchanged',
      });
    }

    return (
      <div
        className={`${isChanged ? 'border border-white/10 rounded-lg overflow-hidden mb-2' : ''}`}
      >
        {/* Merge controls for changed blocks */}
        {isChanged && showMergeControls && (
          <div className="flex items-center justify-center gap-2 py-2 px-4 bg-gray-900/80 border-b border-white/10">
            <SmallText className="text-gray-400 mr-2">Accept:</SmallText>
            <Button
              size="sm"
              variant={decision?.choice === 'left' ? undefined : 'outline'}
              tone={decision?.choice === 'left' ? 'violet' : undefined}
              onClick={() => onDecision('left')}
              leftIcon={<ChevronLeft className="h-3 w-3" />}
            >
              Left
            </Button>
            <Button
              size="sm"
              variant={decision?.choice === 'both' ? undefined : 'outline'}
              tone={decision?.choice === 'both' ? 'violet' : undefined}
              onClick={() => onDecision('both')}
              leftIcon={<ChevronsLeftRight className="h-3 w-3" />}
            >
              Both
            </Button>
            <Button
              size="sm"
              variant={decision?.choice === 'right' ? undefined : 'outline'}
              tone={decision?.choice === 'right' ? 'violet' : undefined}
              onClick={() => onDecision('right')}
              leftIcon={<ChevronRight className="h-3 w-3" />}
            >
              Right
            </Button>
            {decision && (
              <Tag colorScheme="purple" className="ml-2 text-xs">
                {decision.choice.toUpperCase()}
              </Tag>
            )}
          </div>
        )}

        {/* Side-by-side diff display */}
        <div className="grid grid-cols-2 divide-x divide-white/10">
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
