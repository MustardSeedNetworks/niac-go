import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { useEffect } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ScopeProvider } from '../contexts/ScopeContext';
import { ActionButton } from './ActionButton';

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

it('keeps viewer filters and exports usable while disabling writes with a reason', async () => {
  getScope.mockResolvedValue({ scope: 'read-only' });
  const start = vi.fn();
  const filter = vi.fn();
  render(
    <ScopeProvider>
      <ActionButton action="start" onClick={start}>
        Start
      </ActionButton>
      <ActionButton onClick={filter}>Filter</ActionButton>
      <ActionButton action="export">Export</ActionButton>
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

function Page({ onMount }: { onMount: () => void }) {
  useEffect(() => onMount(), [onMount]);
  return <ActionButton action="start">Start</ActionButton>;
}

it('fails closed until scope arrives without remounting the page', async () => {
  let resolve: (value: { scope: string }) => void = () => {};
  getScope.mockReturnValue(
    new Promise((done) => {
      resolve = done;
    }),
  );
  const mounted = vi.fn();
  render(
    <ScopeProvider>
      <Page onMount={mounted} />
    </ScopeProvider>,
  );
  expect(screen.getByText('Start')).toHaveAttribute('aria-disabled', 'true');
  resolve({ scope: 'read-write' });
  await waitFor(() => expect(screen.getByText('Start')).not.toHaveAttribute('aria-disabled'));
  expect(mounted).toHaveBeenCalledOnce();
});
