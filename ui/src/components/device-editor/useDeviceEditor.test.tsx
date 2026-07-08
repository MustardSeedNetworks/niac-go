/**
 * useDeviceEditor.test.tsx — locks the react-hook-form migration (#730).
 *
 * The hook runs rhf headless: section components still call `updateField`
 * and read `device`, while save-time validation moved from imperative
 * presence checks to a valibot `safeParse`. These tests pin that contract:
 * updateField flows into `device`, an invalid save is blocked with an inline
 * field error, and a valid save reaches the API.
 */
import { act, renderHook, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useDeviceEditor } from './useDeviceEditor';

const mockNavigate = vi.fn();
let mockLocation: { pathname: string; hash: string } = {
  pathname: '/device-config/new',
  hash: '',
};
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ hostname: 'new' }),
    useLocation: () => mockLocation,
  };
});

const mockCreateDevice = vi.fn();
vi.mock('../../api/client', () => ({
  createDevice: (...args: unknown[]) => mockCreateDevice(...args),
  updateDevice: vi.fn(),
  deleteDevice: vi.fn(),
  fetchConfigDevice: () => Promise.resolve({ device: null }),
  fetchDeviceEditorSchema: () => Promise.resolve({ visibleSections: [] }),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryWalks: () => Promise.resolve([]),
}));

const wrapper = ({ children }: { children: React.ReactNode }): React.ReactElement =>
  createElement(MemoryRouter, null, children);

afterEach(() => {
  vi.clearAllMocks();
  mockLocation = { pathname: '/device-config/new', hash: '' };
});

function render() {
  return renderHook(() => useDeviceEditor(), { wrapper });
}

describe('useDeviceEditor', () => {
  it('flows updateField changes into device', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('hostname', 'edge-01'));
    expect(result.current.device.hostname).toBe('edge-01');
  });

  it('blocks save on an invalid MAC and surfaces an inline field error', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('hostname', 'edge-01'));
    act(() => result.current.updateField('mac', 'not-a-mac'));
    await act(async () => {
      await result.current.handleSave();
    });

    expect(mockCreateDevice).not.toHaveBeenCalled();
    expect(result.current.fieldErrors.mac?.message).toMatch(/six hex octets/i);
    expect(result.current.message?.type).toBe('error');
  });

  it('blocks save when the hostname is empty', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('mac', '00:1A:2B:3C:4D:5E'));
    await act(async () => {
      await result.current.handleSave();
    });

    expect(mockCreateDevice).not.toHaveBeenCalled();
    expect(result.current.fieldErrors.hostname?.message).toBe('Hostname is required');
  });

  it('saves a valid new device through the API', async () => {
    mockCreateDevice.mockResolvedValueOnce({ device: { hostname: 'edge-01' } });
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('hostname', 'edge-01'));
    act(() => result.current.updateField('mac', '00:1A:2B:3C:4D:5E'));
    await act(async () => {
      await result.current.handleSave();
    });

    expect(mockCreateDevice).toHaveBeenCalledTimes(1);
    expect(mockCreateDevice).toHaveBeenCalledWith(
      expect.objectContaining({ hostname: 'edge-01', mac: '00:1A:2B:3C:4D:5E' }),
    );
    expect(result.current.fieldErrors.hostname).toBeUndefined();
  });

  it('expands the SNMP section up front when arriving via #snmp (Running Devices deep link)', async () => {
    mockLocation = { pathname: '/device-config/new', hash: '#snmp' };
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.expandedSections.has('snmp')).toBe(true);
    expect(result.current.expandedSections.has('basic')).toBe(true);
  });

  it('does not expand the SNMP section without the #snmp hash', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.expandedSections.has('snmp')).toBe(false);
  });
});
