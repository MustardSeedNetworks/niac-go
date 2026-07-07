/**
 * DebugConsolePage.test.tsx — heading naming collision (PR 2a).
 *
 * The page hardcoded "Debug Console" while the nav/help copy call this
 * surface "Logs" everywhere else (pages.debug.title / debug.label). Pins
 * that the on-page heading now goes through the same i18n key so the two
 * never drift apart again.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import '../i18n';
import { DebugConsolePage } from './DebugConsolePage';

vi.mock('../hooks/useEventSource', () => ({
  useLogStream: () => ({ data: null, connected: true, reconnect: vi.fn() }),
}));

describe('DebugConsolePage', () => {
  it('renders the "Logs" heading (matching nav/help copy), not "Debug Console"', () => {
    render(<DebugConsolePage />);

    expect(screen.getByRole('heading', { name: 'Logs' })).toBeInTheDocument();
    expect(screen.queryByText('Debug Console')).not.toBeInTheDocument();
  });
});
