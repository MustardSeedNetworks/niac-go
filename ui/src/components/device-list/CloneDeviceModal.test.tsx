/**
 * CloneDeviceModal.test.tsx — a dialog has to behave like one.
 *
 * Five overlays hand-rolled `fixed inset-0` beside the shared Modal, which has
 * carried a focus trap and Escape handling since #1668. Without them Tab walks
 * out of the dialog into the page behind it and Escape does nothing, so a
 * keyboard user can be editing a form they can no longer see.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { CloneDeviceModal } from './CloneDeviceModal';

function renderModal() {
  const onCancel = vi.fn();
  const onClone = vi.fn();
  render(<CloneDeviceModal hostname="edge-1" onClone={onClone} onCancel={onCancel} />);

  return { onCancel, onClone };
}

describe('CloneDeviceModal', () => {
  it('is a modal dialog', () => {
    renderModal();

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
  });

  it('has an accessible name', () => {
    renderModal();

    expect(screen.getByRole('dialog')).toHaveAccessibleName();
  });

  it('closes on Escape', async () => {
    const user = userEvent.setup();
    const { onCancel } = renderModal();

    await user.keyboard('{Escape}');

    expect(onCancel).toHaveBeenCalled();
  });

  it('keeps Tab inside the dialog', async () => {
    const user = userEvent.setup();
    renderModal();
    const dialog = screen.getByRole('dialog');

    // Walk past every focusable element; focus must never leave the dialog.
    for (let i = 0; i < 8; i++) {
      await user.tab();
      expect(dialog).toContainElement(document.activeElement as HTMLElement);
    }
  });
});
