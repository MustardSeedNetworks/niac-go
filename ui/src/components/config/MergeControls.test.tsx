/**
 * MergeControls.test.tsx
 *
 * Data-safety regression test: a "modified" diff block left undecided
 * (no Left/Both/Right choice) silently keeps the original (left) content
 * in generateMergedContent. Previously nothing surfaced that risk, and
 * "Preview Merged" was reachable with undecided blocks still pending.
 * This pins:
 *   1. A visible warning appears while any block is undecided.
 *   2. "Preview Merged" is disabled until every block has a decision.
 *   3. Once all blocks are decided, the warning disappears and Preview
 *      is enabled.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import type { DiffBlock, MergeDecision } from '../config/diff-viewer/types';
import { MergeControls } from './MergeControls';

const modifiedBlock: DiffBlock = {
  id: 'block-1',
  type: 'modified',
  startLine: 1,
  leftLines: [{ lineNumber: 1, content: 'left', type: 'modified' }],
  rightLines: [{ lineNumber: 1, content: 'right', type: 'modified' }],
};

const baseProps = {
  onAcceptAllLeft: vi.fn(),
  onAcceptAllRight: vi.fn(),
  onReset: vi.fn(),
  onExport: vi.fn(),
  onPreview: vi.fn(),
};

describe('MergeControls — undecided block safety', () => {
  it('warns and disables Preview when a modified block has no decision', () => {
    render(
      <MergeControls {...baseProps} diffBlocks={[modifiedBlock]} mergeDecisions={new Map()} />,
    );

    expect(screen.getByRole('alert')).toHaveTextContent(/undecided/i);
    expect(screen.getByRole('button', { name: /preview merged/i })).toBeDisabled();
  });

  it('clears the warning and enables Preview once the block is decided', () => {
    const decisions = new Map<string, MergeDecision>([
      ['block-1', { blockId: 'block-1', choice: 'left' }],
    ]);
    render(
      <MergeControls {...baseProps} diffBlocks={[modifiedBlock]} mergeDecisions={decisions} />,
    );

    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.getByRole('button', { name: /preview merged/i })).toBeEnabled();
  });
});
