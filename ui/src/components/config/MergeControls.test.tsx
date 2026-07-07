import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { DiffBlock, MergeDecision } from './DiffViewer';
import { MergeControls } from './MergeControls';

const diffBlocks: DiffBlock[] = [
  {
    id: 'b1',
    type: 'modified',
    leftLines: [{ lineNumber: 1, content: 'a', type: 'modified' }],
    rightLines: [{ lineNumber: 1, content: 'b', type: 'modified' }],
    startLine: 0,
  },
];

function renderControls(overrides: Partial<Parameters<typeof MergeControls>[0]> = {}) {
  const onAcceptAllLeft = vi.fn();
  const onAcceptAllRight = vi.fn();
  const onReset = vi.fn();
  const onClearAll = vi.fn();
  const onExport = vi.fn();
  const onPreview = vi.fn();

  render(
    <MergeControls
      diffBlocks={diffBlocks}
      mergeDecisions={new Map<string, MergeDecision>()}
      onAcceptAllLeft={onAcceptAllLeft}
      onAcceptAllRight={onAcceptAllRight}
      onReset={onReset}
      onClearAll={onClearAll}
      onExport={onExport}
      onPreview={onPreview}
      {...overrides}
    />,
  );

  return { onAcceptAllLeft, onAcceptAllRight, onReset, onClearAll, onExport, onPreview };
}

describe('MergeControls', () => {
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
    await user.click(confirmButtons[confirmButtons.length - 1]);

    expect(onClearAll).toHaveBeenCalledTimes(1);
    expect(onReset).not.toHaveBeenCalled();
  });
});
