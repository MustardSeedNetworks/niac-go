/**
 * DiffViewer.test.tsx
 *
 * Block/Overlay merge-view toggle: default is "Block" (existing
 * side-by-side columns); switching to "Overlay" collapses the two
 * columns into one unified column without losing any diff lines.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { DiffViewer } from './DiffViewer';

const leftContent = 'a\nb\nc';
const rightContent = 'a\nchanged\nc';

function renderDiffViewer() {
  return render(
    <DiffViewer
      leftContent={leftContent}
      rightContent={rightContent}
      leftLabel="left.yaml"
      rightLabel="right.yaml"
      mergeDecisions={new Map()}
      onMergeDecision={vi.fn()}
    />,
  );
}

describe('DiffViewer — Block/Overlay view mode toggle', () => {
  it('defaults to Block mode with left/right column headers', () => {
    renderDiffViewer();

    const blockToggle = screen.getByTestId('diff-view-mode-block');
    const overlayToggle = screen.getByTestId('diff-view-mode-overlay');

    expect(blockToggle).toHaveAttribute('aria-pressed', 'true');
    expect(overlayToggle).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByText('left.yaml')).toBeInTheDocument();
    expect(screen.getByText('right.yaml')).toBeInTheDocument();
  });

  it('switches to Overlay mode and drops the column headers', async () => {
    const user = userEvent.setup();
    renderDiffViewer();

    await user.click(screen.getByTestId('diff-view-mode-overlay'));

    expect(screen.getByTestId('diff-view-mode-overlay')).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByTestId('diff-view-mode-block')).toHaveAttribute('aria-pressed', 'false');
    expect(screen.queryByText('left.yaml')).not.toBeInTheDocument();
    expect(screen.queryByText('right.yaml')).not.toBeInTheDocument();

    // Both the removed ("b") and added ("changed") lines still render
    // in the unified column.
    expect(screen.getByText('b')).toBeInTheDocument();
    expect(screen.getByText('changed')).toBeInTheDocument();
  });
});
