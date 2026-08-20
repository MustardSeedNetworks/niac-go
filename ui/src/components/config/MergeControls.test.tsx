/**
 * MergeControls.test.tsx
 *
 * Two data-safety behaviors:
 *   1. Undecided blocks — a "modified" block left undecided (no Left/Both/
 *      Right choice) silently keeps the original (left) content, so a
 *      warning must appear and "Preview Merged" must be disabled until every
 *      block has a decision.
 *   2. Clear All — the confirm copy must match actual behavior (clears both
 *      files + all decisions, no "reload the page" claim) and must call
 *      onClearAll, not onReset.
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { required } from '../../test/required';
import '../../i18n';
import type { DiffBlock, MergeDecision } from './DiffViewer';
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
  onClearAll: vi.fn(),
  onExport: vi.fn(),
  onPreview: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
});

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

describe('MergeControls — Clear All', () => {
  function renderControls(overrides: Partial<Parameters<typeof MergeControls>[0]> = {}) {
    render(
      <MergeControls
        {...baseProps}
        diffBlocks={[modifiedBlock]}
        mergeDecisions={new Map<string, MergeDecision>()}
        {...overrides}
      />,
    );
    return baseProps;
  }

  it('confirm dialog copy for Clear All matches actual behavior (no page reload claim)', async () => {
    const user = userEvent.setup();
    renderControls();

    await user.click(screen.getByRole('button', { name: 'Clear All' }));

    expect(
      screen.getByText(/Clear both loaded files and all merge decisions and start over\?/),
    ).toBeInTheDocument();
    expect(screen.queryByText(/reload the page/)).not.toBeInTheDocument();
  });

  it('calls onClearAll (not onReset) when Clear All is confirmed', async () => {
    const user = userEvent.setup();
    const { onClearAll, onReset } = renderControls();

    await user.click(screen.getByRole('button', { name: 'Clear All' }));
    const confirmButtons = screen.getAllByRole('button', { name: 'Clear All' });
    await user.click(required(confirmButtons.at(-1), 'the last confirm button'));

    expect(onClearAll).toHaveBeenCalledTimes(1);
    expect(onReset).not.toHaveBeenCalled();
  });
});
