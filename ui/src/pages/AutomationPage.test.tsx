/**
 * AutomationPage.test.tsx
 *
 * Covers the two things U4 names on this page: the alert-load failure is
 * reported through a translated string rather than an English literal glued
 * to `error.message`, and the two alert inputs are reachable by their visible
 * labels rather than only by placeholder.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { AutomationPage } from './AutomationPage';

const fetchAlerts = vi.fn();
const fetchStats = vi.fn();

vi.mock('../api/client', () => ({
  fetchAlerts: () => fetchAlerts(),
  fetchStats: (sessionId: string) => fetchStats(sessionId),
  updateAlerts: vi.fn(),
}));

vi.mock('../contexts/AppContext', () => ({
  useAppContext: () => ({ sessionId: null }),
}));

describe('AutomationPage', () => {
  beforeEach(async () => {
    fetchAlerts.mockReset();
    fetchStats.mockReset();
    await i18n.changeLanguage('en');
  });

  it('labels the threshold and webhook inputs', async () => {
    fetchAlerts.mockResolvedValue({ packetsThreshold: 100, webhookUrl: 'https://x.test/h' });
    render(<AutomationPage />);

    // The inputs render as soon as the fetch resolves; their values are
    // mirrored in from `data` by an effect one tick later.
    await waitFor(() => {
      expect(screen.getByLabelText('Packet threshold')).toHaveValue(100);
    });
    expect(screen.getByLabelText('Webhook URL')).toHaveValue('https://x.test/h');
  });

  it('reports a load failure through a translated string', async () => {
    fetchAlerts.mockRejectedValue(new Error('daemon unreachable'));
    await i18n.changeLanguage('es');
    render(<AutomationPage />);

    await waitFor(() => {
      expect(screen.getByText(/daemon unreachable/)).toBeInTheDocument();
    });
    // The surrounding sentence has to travel with the language; only the
    // daemon's own message stays verbatim.
    expect(screen.queryByText(/Unable to load alerts/)).not.toBeInTheDocument();
  });
});
