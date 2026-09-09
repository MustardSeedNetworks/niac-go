import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { ScopeProvider } from '../../contexts/ScopeContext';
import '../../i18n';
import { UnsavedChangesModal } from './UnsavedChangesModal';

vi.mock('../../api/requestCore', () => ({
  deduplicatedGet: () => Promise.resolve({ scope: 'read-only' }),
}));

it('prevents a viewer from saving through the unsaved-edit guard while allowing exit', async () => {
  const save = vi.fn();
  const discard = vi.fn();
  const cancel = vi.fn();
  render(
    <ScopeProvider>
      <UnsavedChangesModal
        open
        saving={false}
        onSave={save}
        onDiscard={discard}
        onCancel={cancel}
      />
    </ScopeProvider>,
  );

  const button = screen.getByTestId('unsaved-save');
  await waitFor(() =>
    expect(button).toHaveAttribute('title', 'Your token does not allow this action.'),
  );
  expect(button).toBeDisabled();
  fireEvent.click(button);
  expect(save).not.toHaveBeenCalled();
  expect(screen.getByTestId('unsaved-discard')).toBeEnabled();
  expect(screen.getByTestId('unsaved-cancel')).toBeEnabled();
  fireEvent.click(screen.getByTestId('unsaved-discard'));
  fireEvent.click(screen.getByTestId('unsaved-cancel'));
  expect(discard).toHaveBeenCalledOnce();
  expect(cancel).toHaveBeenCalledOnce();
});
