import { computeDiff } from './diff-algorithm';
import type { DiffBlock, DiffLine, MergeDecision } from './types';

/**
 * Get lines to include based on merge decision
 */
const getDecisionLines = (
  block: DiffBlock,
  decision?: MergeDecision,
): { leftLines: DiffLine[]; rightLines: DiffLine[] } => {
  if (!decision || decision.choice === 'none') {
    if (block.type === 'added') {
      return { leftLines: [], rightLines: block.rightLines };
    }
    return { leftLines: block.leftLines, rightLines: [] };
  }

  if (decision.choice === 'left') {
    return { leftLines: block.leftLines, rightLines: [] };
  }

  if (decision.choice === 'right') {
    return { leftLines: [], rightLines: block.rightLines };
  }

  return { leftLines: block.leftLines, rightLines: block.rightLines };
};

/**
 * Generate merged content based on decisions
 */
export function generateMergedContent(
  leftContent: string,
  rightContent: string,
  decisions: Map<string, MergeDecision>,
): string {
  const diffBlocks = computeDiff(leftContent, rightContent);
  const mergedLines: string[] = [];

  for (const block of diffBlocks) {
    if (block.type === 'unchanged') {
      // Include unchanged lines
      for (const line of block.leftLines) {
        mergedLines.push(line.content);
      }
    } else {
      const decisionLines = getDecisionLines(block, decisions.get(block.id));
      for (const line of decisionLines.leftLines) {
        mergedLines.push(line.content);
      }
      for (const line of decisionLines.rightLines) {
        mergedLines.push(line.content);
      }
    }
  }

  return mergedLines.join('\n');
}
