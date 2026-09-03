/**
 * Device editor load path.
 *
 * Two defects live here. D10: the hook waited for a `{ device: … }` wrapper the
 * server never sends, so the form kept its empty defaults while bound to a real
 * hostname and a save would overwrite the device with nothing. P1b-2: the form
 * then held the camelCase projection, which covers 56 of the 223 authored
 * fields, so every other field was dropped on save.
 *
 * The hook now loads `rawYaml` — the document the daemon itself serialized —
 * so this mocks the response shape the server actually returns, and asserts a
 * field the projection does not carry survives the load.
 */

import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const RAW_YAML = `name: LAB-EDGE-R1
type: router
mac: 00:00:0c:00:01:01
ips:
  - 10.254.200.1
  - 203.0.113.1
snmp_agent:
  enabled: true
  community: NetAllyDemo
  sysname: LAB-EDGE-R1
mdns:
  enabled: true
  services:
    - type: _workstation._tcp
      port: 9
`;

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router');
  return {
    ...actual,
    useParams: () => ({ hostname: 'LAB-EDGE-R1' }),
    useNavigate: () => vi.fn(),
    useLocation: () => ({ pathname: '/device-config/LAB-EDGE-R1', hash: '', search: '' }),
  };
});

const mockUpdateDevice = vi.fn();
vi.mock('../../api/client', () => ({
  // Flat, with the authored document beside the projection — exactly what
  // handleDeviceGet writes. Not { device: … }.
  fetchConfigDevice: vi
    .fn()
    .mockResolvedValue({ hostname: 'LAB-EDGE-R1', type: 'router', rawYaml: RAW_YAML }),
  fetchDeviceEditorSchema: vi.fn().mockResolvedValue({ visibleSections: [] }),
  createDevice: vi.fn(),
  updateDevice: (...args: unknown[]) => mockUpdateDevice(...args),
  deleteDevice: vi.fn(),
}));

vi.mock('../../api/library-client', () => ({
  fetchLibraryWalks: vi.fn().mockResolvedValue([]),
}));

describe('useDeviceEditor — loading an existing device', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('populates the form from the authored document', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor());

    await waitFor(() => expect(result.current.device.name).toBe('LAB-EDGE-R1'));

    expect(result.current.device.mac).toBe('00:00:0c:00:01:01');
    expect(result.current.device.type).toBe('router');
    // `mdns` has no camelCase Device property, which is why the projection lost
    // it and why the editor reads the document instead.
    expect(result.current.device.mdns?.services?.[0]?.type).toBe('_workstation._tcp');
  });

  it('captures originalDevice so Discard has something to restore', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor());

    await waitFor(() => expect(result.current.originalDevice).not.toBeNull());

    expect(result.current.originalDevice?.name).toBe('LAB-EDGE-R1');
  });

  it('is not dirty until something is edited, so the guard stays quiet', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor());

    await waitFor(() => expect(result.current.originalDevice).not.toBeNull());

    expect(result.current.isDirty).toBe(false);
  });
});
