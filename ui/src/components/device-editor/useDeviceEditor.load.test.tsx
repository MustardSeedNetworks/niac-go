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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ResourceProvider } from '../../contexts/ResourceProvider';
import { useApiResource } from '../../hooks/useApiResource';
import { MemoryDataRouter } from '../../test/MemoryDataRouter';

const wrapper = ({ children }: { children: ReactNode }) => (
  <ResourceProvider>
    <MemoryDataRouter>{children}</MemoryDataRouter>
  </ResourceProvider>
);

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
  it('does not overwrite an edited cached document when its refresh completes', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { fetchConfigDevice } = await import('../../api/client');
    type Response = Awaited<ReturnType<typeof fetchConfigDevice>>;
    let resolveFresh: (response: Response) => void = () => {};
    const fresh = new Promise<Response>((resolve) => {
      resolveFresh = resolve;
    });
    vi.mocked(fetchConfigDevice).mockReturnValueOnce(fresh);
    const client = new QueryClient();
    const key = ['config-device', 'LAB-EDGE-R1', false] as const;
    client.setQueryData(key, { hostname: 'LAB-EDGE-R1', rawYaml: RAW_YAML });
    const { result, unmount } = renderHook(
      () => ({
        editor: useDeviceEditor(),
        server: useApiResource(() => fetchConfigDevice('LAB-EDGE-R1'), key),
      }),
      {
        wrapper: ({ children }) => (
          <QueryClientProvider client={client}>
            <MemoryDataRouter>{children}</MemoryDataRouter>
          </QueryClientProvider>
        ),
      },
    );
    try {
      await waitFor(() => expect(result.current.editor.device.name).toBe('LAB-EDGE-R1'));
      act(() => result.current.editor.updateField('vendor', 'operator edit'));
      const response = {
        hostname: 'LAB-EDGE-R1',
        mac: '00:00:0c:00:01:01',
        type: 'router' as const,
        rawYaml: `${RAW_YAML}vendor: refreshed\n`,
      };
      act(() => resolveFresh(response));
      await waitFor(() => expect(result.current.server.data?.rawYaml).toBe(response.rawYaml));
      expect(result.current.editor.device.vendor).toBe('operator edit');
      expect(result.current.editor.isDirty).toBe(true);
    } finally {
      resolveFresh({
        hostname: 'LAB-EDGE-R1',
        mac: '00:00:0c:00:01:01',
        type: 'router',
        rawYaml: RAW_YAML,
      });
      unmount();
      client.clear();
    }
  });

  beforeEach(async () => {
    vi.clearAllMocks();
    const { fetchConfigDevice } = await import('../../api/client');
    vi.mocked(fetchConfigDevice).mockReset().mockResolvedValue({
      hostname: 'LAB-EDGE-R1',
      mac: '00:00:0c:00:01:01',
      type: 'router',
      rawYaml: RAW_YAML,
    });
  });

  it('retains a saved document when the following refresh fails', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { fetchConfigDevice } = await import('../../api/client');
    mockUpdateDevice.mockResolvedValue({});
    const { result } = renderHook(() => useDeviceEditor(), { wrapper });
    await waitFor(() => expect(result.current.device.name).toBe('LAB-EDGE-R1'));
    act(() => result.current.updateField('ips', ['192.0.2.10']));
    vi.mocked(fetchConfigDevice).mockRejectedValueOnce(new Error('refresh unavailable'));

    act(() => {
      void result.current.handleSave();
    });

    await waitFor(() => expect(result.current.error?.message).toBe('refresh unavailable'));
    expect(mockUpdateDevice).toHaveBeenCalledWith(
      'LAB-EDGE-R1',
      expect.stringContaining('192.0.2.10'),
    );
    expect(result.current.device.ips).toEqual(['192.0.2.10']);
    expect(result.current.originalDevice?.ips).toEqual(['192.0.2.10']);
    expect(result.current.isDirty).toBe(false);
  });

  it('populates the form from the authored document', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor(), { wrapper });

    await waitFor(() => expect(result.current.device.name).toBe('LAB-EDGE-R1'));

    expect(result.current.device.mac).toBe('00:00:0c:00:01:01');
    expect(result.current.device.type).toBe('router');
    // `mdns` has no camelCase Device property, which is why the projection lost
    // it and why the editor reads the document instead.
    expect(result.current.device.mdns?.services?.[0]?.type).toBe('_workstation._tcp');
  });

  it('captures originalDevice so Discard has something to restore', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor(), { wrapper });

    await waitFor(() => expect(result.current.originalDevice).not.toBeNull());

    expect(result.current.originalDevice?.name).toBe('LAB-EDGE-R1');
  });

  it('is not dirty until something is edited, so the guard stays quiet', async () => {
    const { useDeviceEditor } = await import('./useDeviceEditor');
    const { result } = renderHook(() => useDeviceEditor(), { wrapper });

    await waitFor(() => expect(result.current.originalDevice).not.toBeNull());

    expect(result.current.isDirty).toBe(false);
  });
});
