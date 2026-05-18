import { type FC, memo } from 'react';
import { getDiffStyles } from './diff-styles';
import type { DiffLine } from './types';

interface DiffLineComponentProps {
  line: DiffLine;
  side: 'left' | 'right';
}

/**
 * Individual diff line component
 */
export const DiffLineComponent: FC<DiffLineComponentProps> = memo(({ line, side }) => {
  const styles = getDiffStyles(line.type);
  const lineNum = side === 'left' ? line.leftLineNumber : line.rightLineNumber;

  return (
    <div className={`flex ${styles.bg} border-l-2 ${styles.border}`}>
      <span className="w-12 flex-shrink-0 px-2 py-0.5 text-right text-xs text-text-muted select-none border-r border-surface-border">
        {lineNum ?? ''}
      </span>
      <span className="px-1 py-0.5 text-xs text-text-muted select-none w-4">
        {line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '}
      </span>
      <pre
        className={`flex-1 px-2 py-0.5 text-sm font-mono ${styles.text} overflow-x-auto whitespace-pre`}
      >
        {line.content || ' '}
      </pre>
    </div>
  );
});

DiffLineComponent.displayName = 'DiffLineComponent';
