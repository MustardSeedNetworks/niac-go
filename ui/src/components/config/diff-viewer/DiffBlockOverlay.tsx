import { type FC, memo } from 'react';
import { DiffBlockMergeControls } from './DiffBlock';
import { DiffLineComponent } from './DiffLine';
import type { DiffBlock, MergeDecision } from './types';

interface DiffBlockOverlayComponentProps {
  block: DiffBlock;
  decision?: MergeDecision;
  onDecision: (choice: MergeDecision['choice']) => void;
  showMergeControls: boolean;
}

/**
 * Unified diff block: removed lines followed by added lines in a single
 * column, instead of DiffBlock's side-by-side layout. Unchanged blocks
 * only need one copy of each line since left/right content is identical.
 */
export const DiffBlockOverlayComponent: FC<DiffBlockOverlayComponentProps> = memo(
  ({ block, decision, onDecision, showMergeControls }) => {
    const isChanged = block.type !== 'unchanged';

    return (
      <div
        className={`${isChanged ? 'border border-surface-border rounded-lg overflow-hidden mb-2' : ''}`}
      >
        {isChanged && showMergeControls && (
          <DiffBlockMergeControls decision={decision} onDecision={onDecision} />
        )}

        <div>
          {block.type === 'unchanged'
            ? block.leftLines.map((line, idx) => (
                <DiffLineComponent key={`overlay-${block.id}-${idx}`} line={line} side="left" />
              ))
            : [
                ...block.leftLines.map((line, idx) => (
                  <DiffLineComponent
                    key={`overlay-left-${block.id}-${idx}`}
                    line={line}
                    side="left"
                  />
                )),
                ...block.rightLines.map((line, idx) => (
                  <DiffLineComponent
                    key={`overlay-right-${block.id}-${idx}`}
                    line={line}
                    side="right"
                  />
                )),
              ]}
        </div>
      </div>
    );
  },
);

DiffBlockOverlayComponent.displayName = 'DiffBlockOverlayComponent';
