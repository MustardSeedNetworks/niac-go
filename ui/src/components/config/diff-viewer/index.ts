// Types

// Components
export { DiffBlockComponent, DiffBlockMergeControls } from './DiffBlock';
export { DiffBlockOverlayComponent } from './DiffBlockOverlay';
export { DiffLineComponent } from './DiffLine';
// Utilities
export { computeDiff, computeLcs } from './diff-algorithm';
export type { DiffStyleSet } from './diff-styles';
export { getDiffStyles } from './diff-styles';
export { generateMergedContent } from './merge-utils';
export type { DiffBlock, DiffLine, DiffType, DiffViewerProps, MergeDecision } from './types';
