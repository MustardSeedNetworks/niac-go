/**
 * Types for diff operations
 */
export type DiffType = 'unchanged' | 'added' | 'removed' | 'modified';

export interface DiffLine {
  lineNumber: number;
  content: string;
  type: DiffType;
  leftLineNumber?: number;
  rightLineNumber?: number;
  /** Stable React key for the line. Set when the array is built; padding
   * placeholders get a synthetic id keyed off their position in the
   * padded run so React can reconcile them across renders. */
  key?: string;
}

export interface DiffBlock {
  id: string;
  leftLines: DiffLine[];
  rightLines: DiffLine[];
  type: DiffType;
  startLine: number;
}

export interface MergeDecision {
  blockId: string;
  choice: 'left' | 'right' | 'both' | 'none';
}

export interface DiffViewerProps {
  leftContent: string;
  rightContent: string;
  leftLabel?: string;
  rightLabel?: string;
  mergeDecisions: Map<string, MergeDecision>;
  onMergeDecision: (blockId: string, choice: MergeDecision['choice']) => void;
  showMergeControls?: boolean;
}
