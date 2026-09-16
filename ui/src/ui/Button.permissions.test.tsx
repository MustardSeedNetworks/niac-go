import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { useEffect } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ScopeProvider } from '../contexts/ScopeContext';
import { ActionButton } from './ActionButton';
import { Button } from './Button';
import { ConfirmModal } from './ConfirmModal';

const getScope = vi.fn();
vi.mock('../api/requestCore', () => ({ deduplicatedGet: () => getScope() }));
beforeEach(() => {
  getScope.mockReset();
});

it('fails closed for native actions when scope lookup fails', async () => {
  getScope.mockRejectedValue(new Error('offline'));
  const remove = vi.fn();
  render(
    <ScopeProvider>
      <ActionButton action="delete" onClick={remove}>
        Delete
      </ActionButton>
    </ScopeProvider>,
  );
  await waitFor(() =>
    expect(screen.getByText('Delete')).toHaveAccessibleDescription(
      'Permissions could not be verified. Reconnect before making changes.',
    ),
  );
  fireEvent.click(screen.getByText('Delete'));
  expect(remove).not.toHaveBeenCalled();
});

it('gates confirmation actions without trapping the viewer in the dialog', async () => {
  getScope.mockResolvedValue({ scope: 'read-only' });
  render(
    <ScopeProvider>
      <ConfirmModal
        isOpen
        action="delete"
        title="Delete device"
        message="Remove?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    </ScopeProvider>,
  );
  await waitFor(() =>
    expect(screen.getByText('Confirm')).toHaveAccessibleDescription(
      'Your token does not allow this action.',
    ),
  );
  expect(screen.getByText('Confirm')).toHaveAttribute('aria-disabled', 'true');
  expect(screen.getByText('Cancel')).toBeEnabled();
});

it('keeps viewer filters and exports usable while disabling writes with a reason', async () => {
  getScope.mockResolvedValue({ scope: 'read-only' });
  const start = vi.fn();
  const filter = vi.fn();
  render(
    <ScopeProvider>
      <Button action="start" onClick={start}>
        Start
      </Button>
      <Button onClick={filter}>Filter</Button>
      <Button action="export">Export</Button>
    </ScopeProvider>,
  );
  await waitFor(() => expect(screen.getByText('Export')).not.toHaveAttribute('aria-disabled'));
  expect(screen.getByText('Start')).toHaveAttribute('aria-disabled', 'true');
  expect(screen.getByText('Start')).toHaveAccessibleDescription(
    'Your token does not allow this action.',
  );
  fireEvent.click(screen.getByText('Start'));
  fireEvent.click(screen.getByText('Filter'));
  expect(start).not.toHaveBeenCalled();
  expect(filter).toHaveBeenCalledOnce();
});

it('fails closed until scope arrives without remounting the page', async () => {
  let resolve: (value: { scope: string }) => void = () => {};
  getScope.mockReturnValue(
    new Promise((done) => {
      resolve = done;
    }),
  );
  const mounted = vi.fn();
  function Page() {
    useEffect(mounted, []);
    return <Button action="start">Start</Button>;
  }
  render(
    <ScopeProvider>
      <Page />
    </ScopeProvider>,
  );
  expect(screen.getByText('Start')).toHaveAttribute('aria-disabled', 'true');
  resolve({ scope: 'read-write' });
  await waitFor(() => expect(screen.getByText('Start')).not.toHaveAttribute('aria-disabled'));
  expect(mounted).toHaveBeenCalledOnce();
});
