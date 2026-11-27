import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationRecovery } from '../../api/types';
import '../../i18n';
import { renderWithToasts } from '../../test/renderWithToasts';
import { SavedSessionRecoveryCard } from './SavedSessionRecoveryCard';

const forgetSavedSession = vi.fn<(sessionId: string) => Promise<{ status: string }>>();
vi.mock('../../api/client', () => ({
  forgetSavedSession: (sessionId: string) => forgetSavedSession(sessionId),
}));

let permission: { disabled: boolean; title?: string } = { disabled: false };
vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => permission,
}));

const failed: SimulationRecovery = {
  state: 'failed',
  message: 'Recovery failed: session "modern": interface does not exist: en16.',
  attemptedAt: '2026-11-27T13:36:07Z',
  failed: [
    { sessionId: 'modern', interface: 'en16', reason: 'interface does not exist: en16' },
    { sessionId: 'clinic', interface: 'en17', reason: 'interface does not exist: en17' },
  ],
};

describe('SavedSessionRecoveryCard', () => {
  beforeEach(() => {
    forgetSavedSession.mockReset();
    permission = { disabled: false };
  });

  it.each([
    ['recovered', { ...failed, state: 'recovered' as const }],
    ['failed without per-session detail', { ...failed, failed: [] }],
    ['failed with the field absent', { state: 'failed' as const, message: 'x', attemptedAt: 'x' }],
  ])('renders nothing for %s', (_label, recovery) => {
    renderWithToasts(<SavedSessionRecoveryCard recovery={recovery} onForgotten={vi.fn()} />);
    expect(screen.queryByTestId('saved-session-recovery-card')).not.toBeInTheDocument();
  });

  it('names every failed session with its reason', () => {
    renderWithToasts(<SavedSessionRecoveryCard recovery={failed} onForgotten={vi.fn()} />);
    expect(screen.getByTestId('saved-session-modern')).toHaveTextContent('modern on en16');
    expect(screen.getByTestId('saved-session-modern')).toHaveTextContent(
      'interface does not exist: en16',
    );
    expect(screen.getByTestId('saved-session-clinic')).toHaveTextContent('clinic on en17');
  });

  it('forgets only after the confirmation, then reloads the status', async () => {
    const user = userEvent.setup();
    const onForgotten = vi.fn();
    forgetSavedSession.mockResolvedValue({ status: 'forgotten' });
    renderWithToasts(<SavedSessionRecoveryCard recovery={failed} onForgotten={onForgotten} />);

    await user.click(screen.getByTestId('forget-saved-session-modern'));
    expect(forgetSavedSession).not.toHaveBeenCalled();
    await user.click(await screen.findByRole('button', { name: 'Forget' }));

    await waitFor(() => expect(forgetSavedSession).toHaveBeenCalledExactlyOnceWith('modern'));
    await waitFor(() => expect(onForgotten).toHaveBeenCalledOnce());
  });

  it('cancel leaves the session alone', async () => {
    const user = userEvent.setup();
    renderWithToasts(<SavedSessionRecoveryCard recovery={failed} onForgotten={vi.fn()} />);

    await user.click(screen.getByTestId('forget-saved-session-modern'));
    await user.click(await screen.findByRole('button', { name: /^cancel$/i }));

    expect(forgetSavedSession).not.toHaveBeenCalled();
  });

  it('shows the failure as a toast and does not reload', async () => {
    const user = userEvent.setup();
    const onForgotten = vi.fn();
    forgetSavedSession.mockRejectedValue(new Error('admin scope required'));
    renderWithToasts(<SavedSessionRecoveryCard recovery={failed} onForgotten={onForgotten} />);

    await user.click(screen.getByTestId('forget-saved-session-modern'));
    await user.click(await screen.findByRole('button', { name: 'Forget' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('admin scope required');
    expect(onForgotten).not.toHaveBeenCalled();
  });

  it('does not offer the action below admin scope', async () => {
    const user = userEvent.setup();
    permission = { disabled: true, title: 'Requires an admin token' };
    renderWithToasts(<SavedSessionRecoveryCard recovery={failed} onForgotten={vi.fn()} />);

    await user.click(screen.getByTestId('forget-saved-session-modern'));

    expect(forgetSavedSession).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
