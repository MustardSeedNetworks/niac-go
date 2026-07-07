/**
 * DebugConsolePage.test.tsx — heading naming collision (PR 2a).
 *
 * The page hardcoded "Debug Console" while the nav/help copy call this
 * surface "Logs" everywhere else (pages.debug.title / debug.label). Pins
 * that the on-page heading now goes through the same i18n key so the two
 * never drift apart again.
 */
import { act, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { useUIStore } from '../stores/ui-store';
import { ToastContainer } from '../ui/ToastContainer';
import { DebugConsolePage } from './DebugConsolePage';

let onConnect: (() => void) | undefined;
let onDisconnect: (() => void) | undefined;

vi.mock('../hooks/useEventSource', () => ({
  useLogStream: (options: { onConnect?: () => void; onDisconnect?: () => void }) => {
    onConnect = options.onConnect;
    onDisconnect = options.onDisconnect;
    return { data: null, connected: true, reconnect: vi.fn() };
  },
}));

describe('DebugConsolePage', () => {
  beforeEach(() => {
    onConnect = undefined;
    onDisconnect = undefined;
    useUIStore.getState().clearNotifications();
  });

  it('renders the "Logs" heading (matching nav/help copy), not "Debug Console"', () => {
    render(<DebugConsolePage />);

    expect(screen.getByRole('heading', { name: 'Logs' })).toBeInTheDocument();
    expect(screen.queryByText('Debug Console')).not.toBeInTheDocument();
  });

  it('does not toast a drop before the stream has ever connected', () => {
    render(<DebugConsolePage />);

    act(() => onDisconnect?.());

    expect(useUIStore.getState().notifications).toHaveLength(0);
  });

  it('toasts once the log stream drops after having connected', () => {
    render(
      <>
        <DebugConsolePage />
        <ToastContainer />
      </>,
    );

    act(() => onConnect?.());
    act(() => onDisconnect?.());

    expect(screen.getByRole('alert')).toHaveTextContent('Log stream disconnected');
  });
});
