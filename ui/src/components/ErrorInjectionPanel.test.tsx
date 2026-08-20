/**
 * ErrorInjectionPanel.test.tsx — locks the ?errorType= deep-link preselect
 * added for the Dashboard's error-type catalog links (PR "Dashboard
 * honesty + shell sim-status"). Clicking a catalog entry on the Dashboard
 * navigates here with the type in the query string; the real injection
 * form must land with that type already selected.
 */

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ErrorInjectionInfo } from '../api/types';
import { required } from '../test/required';
import '../i18n';
import { ErrorInjectionPanel } from './ErrorInjectionPanel';

const fetchErrorTypes = vi.fn<() => Promise<ErrorInjectionInfo>>();
const clearError = vi.fn<(device: string, iface: string, errorType: string) => Promise<unknown>>();

vi.mock('../api/client', () => ({
  fetchErrorTypes: () => fetchErrorTypes(),
  injectError: vi.fn(),
  clearError: (device: string, iface: string, errorType: string) =>
    clearError(device, iface, errorType),
  clearAllErrors: vi.fn(),
}));

const errorInfo: ErrorInjectionInfo = {
  availableTypes: [
    { type: 'drop', description: 'Drop packets' },
    { type: 'delay', description: 'Delay packets' },
  ],
  info: 'Inject errors on a device interface.',
  targets: [
    { device: 'sw-core', address: '10.0.0.1', interfaces: ['Gi0/1', 'Gi0/2'] },
    { device: 'sw-edge', address: '10.0.0.2', interfaces: [] },
  ],
};

describe('ErrorInjectionPanel', () => {
  beforeEach(() => {
    fetchErrorTypes.mockReset().mockResolvedValue(errorInfo);
    clearError.mockReset().mockResolvedValue({});
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

  it('disables the interface dropdown with a hint until a device is selected', async () => {
    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const interfaceSelect = (await screen.findByLabelText('Interface')) as HTMLSelectElement;
    expect(interfaceSelect.disabled).toBe(true);
    expect(await screen.findByText('Select a device first')).toBeInTheDocument();
  });

  it("renders the selected device's interfaces and sets selectedInterface on selection", async () => {
    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const deviceSelect = (await screen.findByLabelText('Device')) as HTMLSelectElement;
    fireEvent.change(deviceSelect, { target: { value: 'sw-core' } });

    const interfaceSelect = (await screen.findByLabelText('Interface')) as HTMLSelectElement;
    await waitFor(() => expect(interfaceSelect.disabled).toBe(false));
    expect(screen.getByRole('option', { name: 'Gi0/1' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Gi0/2' })).toBeInTheDocument();

    fireEvent.change(interfaceSelect, { target: { value: 'Gi0/2' } });
    expect(interfaceSelect.value).toBe('Gi0/2');
  });

  it('shows a "no interfaces" hint for a known device with none configured', async () => {
    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const deviceSelect = (await screen.findByLabelText('Device')) as HTMLSelectElement;
    fireEvent.change(deviceSelect, { target: { value: 'sw-edge' } });

    expect(await screen.findByText('This device has no configured interfaces')).toBeInTheDocument();
    const interfaceSelect = (await screen.findByLabelText('Interface')) as HTMLSelectElement;
    expect(interfaceSelect.disabled).toBe(true);
  });

  it('clears only the selected active fault', async () => {
    fetchErrorTypes.mockResolvedValue({
      ...errorInfo,
      activeErrors: {
        'sw-core': {
          'Gi0/1': { 'FCS Errors': 25, 'Packet Discards': 40 },
        },
      },
    });

    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const clearButtons = await screen.findAllByRole('button', { name: /Clear error on/ });
    fireEvent.click(required(clearButtons[0], 'a clear button'));
    await waitFor(() => expect(clearError).toHaveBeenCalledWith('sw-core', 'Gi0/1', 'FCS Errors'));
  });
});
