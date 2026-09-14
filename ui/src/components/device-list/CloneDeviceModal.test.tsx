/**
 * CloneDeviceModal.test.tsx — a dialog has to behave like one.
 *
 * The clone dialog was one of five overlays that hand-rolled `fixed inset-0`
 * beside the shared Modal, which has carried a focus trap and Escape handling
 * since #1668. It has been migrated since, and nothing tested it afterwards.
 *
 * The Tab-containment half of this deliberately is NOT here. `useFocusTrap`
 * decides what is focusable with `offsetParent !== null`, and jsdom reports
 * `offsetParent` as null for every element, so the trap finds zero focusable
 * elements and does nothing at all under vitest — a Tab assertion here would
 * either fail for a reason that has nothing to do with the product or, written
 * the other way round, pass while asserting nothing. The real-browser check is
 * the `FocusTrapKeepsTabInside` story in `ui/src/ui/Modal.stories.tsx`.
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
});
