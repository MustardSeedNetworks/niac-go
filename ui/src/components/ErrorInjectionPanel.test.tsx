/**
 * ErrorInjectionPanel.test.tsx — locks the ?errorType= deep-link preselect
 * added for the Dashboard's error-type catalog links (PR "Dashboard
 * honesty + shell sim-status"). Clicking a catalog entry on the Dashboard
 * navigates here with the type in the query string; the real injection
 * form must land with that type already selected.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ErrorInjectionInfo } from '../api/types';
import '../i18n';
import { ErrorInjectionPanel } from './ErrorInjectionPanel';

const fetchDevices = vi.fn();
const fetchErrorTypes = vi.fn<() => Promise<ErrorInjectionInfo>>();

vi.mock('../api/client', () => ({
  fetchDevices: () => fetchDevices(),
  fetchErrorTypes: () => fetchErrorTypes(),
  injectError: vi.fn(),
  clearError: vi.fn(),
  clearAllErrors: vi.fn(),
}));

const errorInfo: ErrorInjectionInfo = {
  availableTypes: [
    { type: 'drop', description: 'Drop packets' },
    { type: 'delay', description: 'Delay packets' },
  ],
  info: 'Inject errors on a device interface.',
};

describe('ErrorInjectionPanel', () => {
  beforeEach(() => {
    fetchDevices.mockReset().mockResolvedValue([]);
    fetchErrorTypes.mockReset().mockResolvedValue(errorInfo);
  });

  it('preselects the error type from the ?errorType= query param', async () => {
    render(
      <MemoryRouter initialEntries={['/traffic?errorType=delay']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const select = (await screen.findByLabelText('Error Type')) as HTMLSelectElement;
    await waitFor(() => expect(select.value).toBe('delay'));
  });

  it('defaults to no selection when no ?errorType= param is present', async () => {
    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const select = (await screen.findByLabelText('Error Type')) as HTMLSelectElement;
    expect(select.value).toBe('');
  });
});
