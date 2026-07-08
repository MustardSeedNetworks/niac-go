/**
 * ErrorInjectionPanel.test.tsx — locks the ?errorType= deep-link preselect
 * added for the Dashboard's error-type catalog links (PR "Dashboard
 * honesty + shell sim-status"). Clicking a catalog entry on the Dashboard
 * navigates here with the type in the query string; the real injection
 * form must land with that type already selected.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { DeviceInterface, DeviceSummary, ErrorInjectionInfo } from '../api/types';
import '../i18n';
import { ErrorInjectionPanel } from './ErrorInjectionPanel';

const fetchDevices = vi.fn<() => Promise<DeviceSummary[]>>();
const fetchErrorTypes = vi.fn<() => Promise<ErrorInjectionInfo>>();
const fetchDeviceInterfaces = vi.fn<(hostname: string) => Promise<DeviceInterface[]>>();

vi.mock('../api/client', () => ({
  fetchDevices: () => fetchDevices(),
  fetchErrorTypes: () => fetchErrorTypes(),
  fetchDeviceInterfaces: (hostname: string) => fetchDeviceInterfaces(hostname),
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

const devices: DeviceSummary[] = [
  { name: 'sw-core', type: 'switch', ips: ['10.0.0.1'], protocols: [] },
  { name: 'sw-edge', type: 'switch', ips: ['10.0.0.2'], protocols: [] },
];

const swCoreInterfaces: DeviceInterface[] = [
  { name: 'Gi0/1', description: 'uplink' },
  { name: 'Gi0/2', description: '' },
];

describe('ErrorInjectionPanel', () => {
  beforeEach(() => {
    fetchDevices.mockReset().mockResolvedValue([]);
    fetchErrorTypes.mockReset().mockResolvedValue(errorInfo);
    fetchDeviceInterfaces.mockReset().mockResolvedValue([]);
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
    fetchDevices.mockResolvedValue(devices);

    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const interfaceSelect = (await screen.findByLabelText('Interface')) as HTMLSelectElement;
    expect(interfaceSelect.disabled).toBe(true);
    expect(await screen.findByText('Select a device first')).toBeInTheDocument();
    expect(fetchDeviceInterfaces).not.toHaveBeenCalled();
  });

  it("renders the selected device's interfaces and sets selectedInterface on selection", async () => {
    fetchDevices.mockResolvedValue(devices);
    fetchDeviceInterfaces.mockImplementation((hostname: string) =>
      Promise.resolve(hostname === 'sw-core' ? swCoreInterfaces : []),
    );

    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const deviceSelect = (await screen.findByLabelText('Device')) as HTMLSelectElement;
    fireEvent.change(deviceSelect, { target: { value: '10.0.0.1' } });

    await waitFor(() => expect(fetchDeviceInterfaces).toHaveBeenCalledWith('sw-core'));

    const interfaceSelect = (await screen.findByLabelText('Interface')) as HTMLSelectElement;
    await waitFor(() => expect(interfaceSelect.disabled).toBe(false));
    expect(screen.getByRole('option', { name: 'Gi0/1 (uplink)' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Gi0/2' })).toBeInTheDocument();

    fireEvent.change(interfaceSelect, { target: { value: 'Gi0/2' } });
    expect(interfaceSelect.value).toBe('Gi0/2');
  });

  it('shows a "no interfaces" hint for a known device with none configured', async () => {
    fetchDevices.mockResolvedValue(devices);
    fetchDeviceInterfaces.mockResolvedValue([]);

    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );

    const deviceSelect = (await screen.findByLabelText('Device')) as HTMLSelectElement;
    fireEvent.change(deviceSelect, { target: { value: '10.0.0.2' } });

    await waitFor(() => expect(fetchDeviceInterfaces).toHaveBeenCalledWith('sw-edge'));
    expect(await screen.findByText('This device has no configured interfaces')).toBeInTheDocument();
    const interfaceSelect = (await screen.findByLabelText('Interface')) as HTMLSelectElement;
    expect(interfaceSelect.disabled).toBe(true);
  });
});
