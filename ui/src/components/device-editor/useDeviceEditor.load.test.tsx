/**
 * Device editor load path — regression guard for D10.
 *
 * `GET /api/v1/config/devices/{hostname}` returns the device **flat**, but the
 * hook waited for a `{ device: … }` wrapper that the server never sends. The
 * condition was therefore never true: `reset()` never ran, the form kept
 * react-hook-form's empty defaults, and `originalDevice` stayed null so Discard
 * had nothing to restore. Save remained enabled on a blank form bound to a real
 * hostname, so opening a device and saving would overwrite it with an empty one.
 *
 * The existing useDeviceEditor.test.tsx pins `useParams` to `{hostname:'new'}`
 * for the whole file, and the hook short-circuits that case — so the edit-load
 * path had no coverage at all. This file supplies it, and deliberately mocks
 * `fetchConfigDevice` with the **flat** shape the server actually returns.
 */

import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const EXISTING_DEVICE = {
  hostname: 'LAB-EDGE-R1',
  type: 'router',
  mac: '00:00:0c:00:01:01',
  ip: '10.254.200.1',
  ips: ['10.254.200.1', '203.0.113.1'],
  interfaceDetails: [{ name: 'TenGigabitEthernet0/0/0', speed: 10000, duplex: 'full' }],
  snmpAgent: { enabled: true, community: 'NetAllyDemo', sysname: 'LAB-EDGE-R1' },
};

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router');
  return {
    ...actual,
    useParams: () => ({ hostname: 'LAB-EDGE-R1' }),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: '/device-config/LAB-EDGE-R1', hash: '', search: '' }),
  };
});

vi.mock('../../api/client', () => ({
  // Flat — exactly what handleDeviceGet writes. Not { device: … }.
  fetchConfigDevice: vi.fn().mockResolvedValue(EXISTING_DEVICE),
  fetchDeviceEditorSchema: vi.fn().mockResolvedValue({ visibleSections: [] }),
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
  deleteDevice: vi.fn(),
}));

vi.mock('../../api/library-client', () => ({
  fetchLibraryWalks: vi.fn().mockResolvedValue([]),
}));

describe('useDeviceEditor — loading an existing device', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('populates the form from the flat device response', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor());

    await waitFor(() => {
      expect(result.current.device.hostname).toBe('LAB-EDGE-R1');
    });

    expect(result.current.device.mac).toBe('00:00:0c:00:01:01');
    expect(result.current.device.type).toBe('router');
  });

  it('captures originalDevice so Discard has something to restore', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor());

    await waitFor(() => {
      expect(result.current.originalDevice).not.toBeNull();
    });

    expect(result.current.originalDevice?.hostname).toBe('LAB-EDGE-R1');
  });
});
