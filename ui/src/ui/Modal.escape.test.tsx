import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { ConfirmModal } from './ConfirmModal';
import { Modal } from './Modal';

/**
 * Guards #1465.
 *
 * Modal wired an onKeyDown that called stopPropagation() on Escape, "to prevent
 * escape from bubbling if handled by the focus-trap hook". But useFocusTrap
 * listens on `document`, and React dispatches synthetic events from its root
 * container — below document — so that call stopped the native event before the
 * very handler it claimed to defer to could ever see it.
 *
 * The trap auto-focuses inside the dialog and keeps focus there, so the keypress
 * always originated inside the modal and Escape was always swallowed. Measured
 * on CT304: pressing Escape inside the dialog reached a document listener zero
 * times, while dispatching directly at document closed the dialog.
 *
 * These press Escape on a control *inside* the dialog, which is the only way it
 * happens in real use.
 */
describe('Modal Escape handling', () => {
  it('closes when Escape is pressed on a control inside the dialog', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(
      <Modal isOpen onClose={onClose} title="Danger">
        <button type="button">Inside</button>
      </Modal>,
    );

    await user.click(screen.getByRole('button', { name: 'Inside' }));
    await user.keyboard('{Escape}');

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('leaves Escape inert when closeOnEscape is off', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(
      <Modal isOpen onClose={onClose} closeOnEscape={false} title="Danger">
        <button type="button">Inside</button>
      </Modal>,
    );

    await user.click(screen.getByRole('button', { name: 'Inside' }));
    await user.keyboard('{Escape}');

    expect(onClose).not.toHaveBeenCalled();
  });

  it('cancels a ConfirmModal on Escape, the real CT304 case', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    const onConfirm = vi.fn();

    render(
      <ConfirmModal
        isOpen
        onConfirm={onConfirm}
        onCancel={onCancel}
        title="Stop simulation?"
        message="Are you sure?"
        confirmLabel="Stop"
      />,
    );

    // Focus without clicking: clicking Cancel would call onCancel by itself and
    // the test would pass whether or not Escape works.
    screen.getByRole('button', { name: 'Stop' }).focus();
    await user.keyboard('{Escape}');

    expect(onCancel).toHaveBeenCalled();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});
