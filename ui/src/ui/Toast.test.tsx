import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/errors';
import { useToast } from '../hooks/useToast';
import '../i18n';
import { ToastProvider } from './Toast';

function Probe() {
  const { addToast, showError } = useToast();
  return (
    <>
      <button type="button" onClick={() => addToast({ type: 'success', title: 'Saved' })}>
        ok
      </button>
      <button
        type="button"
        onClick={() => addToast({ title: 'Pinned', message: 'stays', duration: 0 })}
      >
        pin
      </button>
      <button type="button" onClick={() => showError(new Error('daemon unreachable'))}>
        fail
      </button>
      <button
        type="button"
        onClick={() =>
          showError(
            new ApiError('invalid config', 400, 'config_invalid', [
              { line: 65, issue: 'field enabled not found' },
              { issue: 'no devices defined' },
            ]),
            'Import failed',
          )
        }
      >
        fail-api
      </button>
    </>
  );
}

function renderProbe() {
  return render(
    <ToastProvider>
      <Probe />
    </ToastProvider>,
  );
}

afterEach(() => {
  vi.useRealTimers();
});

describe('toast system', () => {
  it('shows a toast and dismisses it from its close button', async () => {
    const user = userEvent.setup();
    renderProbe();

    await user.click(screen.getByRole('button', { name: 'ok' }));
    expect(screen.getByRole('alert')).toHaveTextContent('Saved');

    await user.click(screen.getByRole('button', { name: 'Dismiss notification' }));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('auto-dismisses after the default duration but keeps a pinned toast', () => {
    vi.useFakeTimers();
    renderProbe();

    act(() => {
      screen.getByRole('button', { name: 'ok' }).click();
      screen.getByRole('button', { name: 'pin' }).click();
    });
    expect(screen.getAllByRole('alert')).toHaveLength(2);

    act(() => {
      vi.advanceTimersByTime(5_000);
    });
    const remaining = screen.getAllByRole('alert');
    expect(remaining).toHaveLength(1);
    expect(remaining[0]).toHaveTextContent('Pinned');
  });

  it('stacks toasts raised together', async () => {
    const user = userEvent.setup();
    renderProbe();

    await user.click(screen.getByRole('button', { name: 'ok' }));
    await user.click(screen.getByRole('button', { name: 'fail' }));

    expect(screen.getAllByRole('alert')).toHaveLength(2);
  });

  it('showError reports the failure under the default title', async () => {
    const user = userEvent.setup();
    renderProbe();

    await user.click(screen.getByRole('button', { name: 'fail' }));

    const toast = screen.getByRole('alert');
    expect(toast).toHaveTextContent('Request failed');
    expect(toast).toHaveTextContent('daemon unreachable');
  });

  it('showError lists the server details and honours a custom title', async () => {
    const user = userEvent.setup();
    renderProbe();

    await user.click(screen.getByRole('button', { name: 'fail-api' }));

    const toast = screen.getByRole('alert');
    expect(toast).toHaveTextContent('Import failed');
    expect(toast).toHaveTextContent('line 65: field enabled not found');
    expect(toast).toHaveTextContent('no devices defined');
  });

  it('useToast refuses to run outside a ToastProvider', () => {
    const quiet = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    expect(() => render(<Probe />)).toThrow('useToast must be used within a ToastProvider');
    quiet.mockRestore();
  });
});
