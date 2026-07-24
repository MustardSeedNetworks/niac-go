/**
 * DeviceListPage.test.tsx
 *
 * Regression test for PR "3b — confirm-modal consolidation": the
 * per-device delete confirmation used to be a bespoke
 * `ConfirmDeleteModal` clone. This pins the migrated behavior onto the
 * shared `ConfirmModal`: clicking the row delete action opens the modal
 * without deleting anything, Cancel dismisses it without calling
 * `deleteDevice`, and confirming calls it exactly once with the right
 * hostname.
 */
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Device, DeviceBatchDeleteResponse, DeviceListResponse } from '../api/types';
import '../i18n';
import { DeviceListPage } from './DeviceListPage';

const deleteDevice = vi.fn<(hostname: string) => Promise<void>>();
const deleteDevices = vi.fn<(hostnames: string[]) => Promise<DeviceBatchDeleteResponse>>();
const fetchConfigDevices = vi.fn<() => Promise<DeviceListResponse>>();

vi.mock('../api/client', () => ({
  fetchConfigDevices: () => fetchConfigDevices(),
  deleteDevice: (hostname: string) => deleteDevice(hostname),
  deleteDevices: (hostnames: string[]) => deleteDevices(hostnames),
  cloneDevice: vi.fn(),
}));

const devices: Device[] = [
  { hostname: 'edge-01', mac: '00:11:22:33:44:55', type: 'router' },
  { hostname: 'edge-02', mac: '00:11:22:33:44:66', type: 'router' },
];

afterEach(() => {
  vi.clearAllMocks();
});

function renderPage() {
  return render(createElement(MemoryRouter, null, createElement(DeviceListPage)));
}

describe('DeviceListPage — delete confirmation', () => {
  it('does not delete until the confirm modal is accepted', async () => {
    fetchConfigDevices.mockResolvedValue({ devices, total: devices.length });
    renderPage();

    await waitFor(() => expect(screen.getByText('edge-01')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /delete device edge-01/i }));

    expect(await screen.findByText(/delete device\?/i)).toBeInTheDocument();
    expect(deleteDevice).not.toHaveBeenCalled();
  });

  it('cancel dismisses the modal without deleting', async () => {
    fetchConfigDevices.mockResolvedValue({ devices, total: devices.length });
    renderPage();

    await waitFor(() => expect(screen.getByText('edge-01')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /delete device edge-01/i }));
    await screen.findByText(/delete device\?/i);

    fireEvent.click(screen.getByRole('button', { name: /^cancel$/i }));

    await waitFor(() => expect(screen.queryByText(/delete device\?/i)).not.toBeInTheDocument());
    expect(deleteDevice).not.toHaveBeenCalled();
  });

  it('confirming the modal deletes exactly the targeted device', async () => {
    fetchConfigDevices.mockResolvedValue({ devices, total: devices.length });
    deleteDevice.mockResolvedValue(undefined);
    renderPage();

    await waitFor(() => expect(screen.getByText('edge-01')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /delete device edge-01/i }));
    await screen.findByText(/delete device\?/i);

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /^delete$/i }));
    });

    await waitFor(() => expect(deleteDevice).toHaveBeenCalledTimes(1));
    expect(deleteDevice).toHaveBeenCalledWith('edge-01');
  });
});

describe('DeviceListPage — bulk delete', () => {
  it('sends every selected hostname in a single deleteDevices call', async () => {
    fetchConfigDevices.mockResolvedValue({ devices, total: devices.length });
    deleteDevices.mockResolvedValue({
      results: [
        { hostname: 'edge-01', success: true },
        { hostname: 'edge-02', success: true },
      ],
      deleted: 2,
      failed: 0,
    });
    renderPage();

    await waitFor(() => expect(screen.getByText('edge-01')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('checkbox', { name: /select edge-01/i }));
    fireEvent.click(screen.getByRole('checkbox', { name: /select edge-02/i }));

    fireEvent.click(screen.getByRole('button', { name: /delete selected/i }));
    const confirmButton = await screen.findByRole('button', { name: /delete all/i });

    await act(async () => {
      fireEvent.click(confirmButton);
    });

    await waitFor(() => expect(deleteDevices).toHaveBeenCalledTimes(1));
    expect(deleteDevices).toHaveBeenCalledWith(expect.arrayContaining(['edge-01', 'edge-02']));
    expect(deleteDevice).not.toHaveBeenCalled();
  });

  it('names the hostnames that failed on a partial-failure response', async () => {
    fetchConfigDevices.mockResolvedValue({ devices, total: devices.length });
    deleteDevices.mockResolvedValue({
      results: [
        { hostname: 'edge-01', success: true },
        { hostname: 'edge-02', success: false, error: "Device 'edge-02' not found" },
      ],
      deleted: 1,
      failed: 1,
    });
    renderPage();

    await waitFor(() => expect(screen.getByText('edge-01')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('checkbox', { name: /select edge-01/i }));
    fireEvent.click(screen.getByRole('checkbox', { name: /select edge-02/i }));

    fireEvent.click(screen.getByRole('button', { name: /delete selected/i }));
    const confirmButton = await screen.findByRole('button', { name: /delete all/i });

    await act(async () => {
      fireEvent.click(confirmButton);
    });

    await waitFor(() => expect(deleteDevices).toHaveBeenCalledTimes(1));
    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(/edge-02/i);
  });
});
