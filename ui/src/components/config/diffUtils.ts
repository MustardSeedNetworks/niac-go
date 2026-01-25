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

/**
 * Check if current position matches LCS
 */
const isLcsMatch = (
  currentLcs: string | null,
  leftLines: string[],
  rightLines: string[],
  leftIdx: number,
  rightIdx: number,
): boolean =>
  currentLcs !== null &&
  leftIdx < leftLines.length &&
  rightIdx < rightLines.length &&
  leftLines[leftIdx] === currentLcs &&
  rightLines[rightIdx] === currentLcs;

/**
 * Append unchanged line to blocks
 */
const appendUnchangedLine = (
  blocks: DiffBlock[],
  line: DiffLine,
  blockId: number,
  leftIdx: number,
): number => {
  const lastBlock = blocks.at(-1);
  if (lastBlock && lastBlock.type === 'unchanged') {
    lastBlock.leftLines.push(line);
    lastBlock.rightLines.push({ ...line });
    return blockId;
  }

  blocks.push({
    id: `block-${blockId}`,
    leftLines: [line],
    rightLines: [{ ...line }],
    type: 'unchanged',
    startLine: leftIdx + 1,
  });

  return blockId + 1;
};

/**
 * Collect changed lines until next LCS match
 */
const collectChangedLines = (
  leftLines: string[],
  rightLines: string[],
  currentLcs: string | null,
  leftIdx: number,
  rightIdx: number,
): {
  leftChanged: DiffLine[];
  rightChanged: DiffLine[];
  nextLeftIdx: number;
  nextRightIdx: number;
} => {
  const leftChanged: DiffLine[] = [];
  const rightChanged: DiffLine[] = [];
  let nextLeftIdx = leftIdx;
  let nextRightIdx = rightIdx;

  while (
    nextLeftIdx < leftLines.length &&
    (currentLcs === null || leftLines[nextLeftIdx] !== currentLcs)
  ) {
    leftChanged.push({
      lineNumber: nextLeftIdx + 1,
      content: leftLines[nextLeftIdx],
      type: 'removed',
      leftLineNumber: nextLeftIdx + 1,
    });
    nextLeftIdx++;
  }

  while (
    nextRightIdx < rightLines.length &&
    (currentLcs === null || rightLines[nextRightIdx] !== currentLcs)
  ) {
    rightChanged.push({
      lineNumber: nextRightIdx + 1,
      content: rightLines[nextRightIdx],
      type: 'added',
      rightLineNumber: nextRightIdx + 1,
    });
    nextRightIdx++;
  }

  return { leftChanged, rightChanged, nextLeftIdx, nextRightIdx };
};

/**
 * Determine change type based on left/right changes
 */
const getChangeType = (leftChanged: DiffLine[], rightChanged: DiffLine[]): DiffType => {
  if (leftChanged.length > 0 && rightChanged.length > 0) {
    return 'modified';
  }
  if (leftChanged.length > 0) {
    return 'removed';
  }
  return 'added';
};

/**
 * Compute longest common subsequence of lines
 */
function computeLcs(left: string[], right: string[]): string[] {
  const m = left.length;
  const n = right.length;

  // Build LCS length table
  const dp: number[][] = Array.from({ length: m + 1 }, () =>
    Array.from({ length: n + 1 }, () => 0),
  );

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (left[i - 1] === right[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
      }
    }
  }

  // Backtrack to find LCS
  const lcs: string[] = [];
  let i = m;
  let j = n;

  while (i > 0 && j > 0) {
    if (left[i - 1] === right[j - 1]) {
      lcs.unshift(left[i - 1]);
      i--;
      j--;
    } else if (dp[i - 1][j] > dp[i][j - 1]) {
      i--;
    } else {
      j--;
    }
  }

  return lcs;
}

/**
 * Compute LCS-based diff between two content strings
 */
export function computeDiff(left: string, right: string): DiffBlock[] {
  const leftLines = left.split('\n');
  const rightLines = right.split('\n');

  const blocks: DiffBlock[] = [];
  let leftIdx = 0;
  let rightIdx = 0;
  let blockId = 0;

  const lcs = computeLcs(leftLines, rightLines);
  let lcsIdx = 0;

  while (leftIdx < leftLines.length || rightIdx < rightLines.length) {
    const currentLcs = lcsIdx < lcs.length ? lcs[lcsIdx] : null;

    if (isLcsMatch(currentLcs, leftLines, rightLines, leftIdx, rightIdx)) {
      const line: DiffLine = {
        lineNumber: leftIdx + 1,
        content: leftLines[leftIdx],
        type: 'unchanged',
        leftLineNumber: leftIdx + 1,
        rightLineNumber: rightIdx + 1,
      };

      blockId = appendUnchangedLine(blocks, line, blockId, leftIdx);

      leftIdx++;
      rightIdx++;
      lcsIdx++;
    } else {
      const { leftChanged, rightChanged, nextLeftIdx, nextRightIdx } = collectChangedLines(
        leftLines,
        rightLines,
        currentLcs,
        leftIdx,
        rightIdx,
      );
      leftIdx = nextLeftIdx;
      rightIdx = nextRightIdx;

      if (leftChanged.length > 0 || rightChanged.length > 0) {
        const type = getChangeType(leftChanged, rightChanged);
        blocks.push({
          id: `block-${blockId++}`,
          leftLines: leftChanged,
          rightLines: rightChanged,
          type,
          startLine: Math.min(
            leftChanged[0]?.lineNumber ?? Number.POSITIVE_INFINITY,
            rightChanged[0]?.lineNumber ?? Number.POSITIVE_INFINITY,
          ),
        });
      }
    }
  }

  return blocks;
}

/**
 * Get styling classes for diff type
 */
export function getDiffStyles(type: DiffType): {
  bg: string;
  border: string;
  text: string;
} {
  switch (type) {
    case 'added':
      return {
        bg: 'bg-green-500/10',
        border: 'border-green-500/30',
        text: 'text-green-300',
      };
    case 'removed':
      return {
        bg: 'bg-red-500/10',
        border: 'border-red-500/30',
        text: 'text-red-300',
      };
    case 'modified':
      return {
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/30',
        text: 'text-yellow-300',
      };
    default:
      return {
        bg: '',
        border: 'border-transparent',
        text: 'text-gray-300',
      };
  }
}

/**
 * Get decision lines for a block
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
