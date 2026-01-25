import type { DiffBlock, DiffLine, DiffType } from './types';

/**
 * Check if current position matches LCS element
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
 * Append unchanged line to blocks, creating new block if needed
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
 * Determine change type based on changed lines
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
export function computeLcs(left: string[], right: string[]): string[] {
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

  // Simple line-by-line diff using longest common subsequence approach
  const lcs = computeLcs(leftLines, rightLines);
  let lcsIdx = 0;

  while (leftIdx < leftLines.length || rightIdx < rightLines.length) {
    const currentLcs = lcsIdx < lcs.length ? lcs[lcsIdx] : null;

    if (isLcsMatch(currentLcs, leftLines, rightLines, leftIdx, rightIdx)) {
      // Unchanged line
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
