/**
 * useDeviceEditor.test.tsx — pins the editor's document model (P1b-2).
 *
 * The hook holds the authored YAML document, not the camelCase projection, and
 * saves it verbatim through `rawYaml`. These tests pin that contract at the
 * edges that used to lose authoring: an edit reaches the document, an invalid
 * identity is blocked with an inline error before the round trip, and a save
 * sends the serialized document rather than a rebuilt object.
 */
import { act, renderHook, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useDeviceEditor } from './useDeviceEditor';

const mockNavigate = vi.fn();
let mockLocation: { pathname: string; hash: string } = {
  pathname: '/device-config/new',
  hash: '',
};
vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
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
  fetchConfigDevice: () => Promise.resolve(null),
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

/** A device whose identity is valid, so a test can get past the save gate. */
async function renderNamedDevice() {
  const rendered = render();
  await waitFor(() => expect(rendered.result.current.loading).toBe(false));
  act(() => rendered.result.current.updateField('name', 'edge-01'));
  act(() => rendered.result.current.updateField('mac', '00:1A:2B:3C:4D:5E'));
  return rendered;
}

describe('useDeviceEditor', () => {
  it('flows updateField changes into the authored document', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('name', 'edge-01'));
    expect(result.current.device.name).toBe('edge-01');
  });

  it('writes a whole generated section as one authored block', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('snmp_agent', { enabled: true, community: 'msn_public' }));

    expect(result.current.yaml).toContain('community: msn_public');
  });

  it('blocks save on an invalid MAC and surfaces an inline field error', async () => {
    const { result } = await renderNamedDevice();

    act(() => result.current.updateField('mac', 'not-a-mac'));
    act(() => {
      void result.current.handleSave();
    });

    await waitFor(() => expect(result.current.fieldErrors.mac).toMatch(/six hex octets/i));
    expect(mockCreateDevice).not.toHaveBeenCalled();
    expect(result.current.message?.type).toBe('error');
  });

  it('blocks save when the name is empty', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.updateField('mac', '00:1A:2B:3C:4D:5E'));
    act(() => {
      void result.current.handleSave();
    });

    await waitFor(() => expect(result.current.fieldErrors.name).toBe('Name is required'));
    expect(mockCreateDevice).not.toHaveBeenCalled();
  });

  // The daemon rejects both-or-neither (ErrDeviceMACSourceConflict). The form
  // makes it unrepresentable; this pins the check behind it, because a document
  // loaded from YAML can still arrive in the illegal state.
  it('blocks save when a device carries both a MAC and a vendor', async () => {
    const { result } = await renderNamedDevice();

    act(() => result.current.updateField('vendor', 'cisco'));
    act(() => {
      void result.current.handleSave();
    });

    await waitFor(() =>
      expect(result.current.fieldErrors.mac).toMatch(/not both and not neither/i),
    );
    expect(mockCreateDevice).not.toHaveBeenCalled();
  });

  it('saves a new device as the serialized document, not a rebuilt object', async () => {
    mockCreateDevice.mockResolvedValueOnce({ success: true });
    const { result } = await renderNamedDevice();

    act(() => result.current.updateField('ips', ['10.20.0.1']));
    act(() => {
      void result.current.handleSave();
    });

    await waitFor(() => expect(mockCreateDevice).toHaveBeenCalledTimes(1));
    const [hostname, rawYaml] = mockCreateDevice.mock.calls[0] as [string, string];
    expect(hostname).toBe('edge-01');
    expect(rawYaml).toContain('name: edge-01');
    expect(rawYaml).toContain('mac: 00:1A:2B:3C:4D:5E');
    expect(rawYaml).toContain('- 10.20.0.1');
    expect(result.current.fieldErrors.name).toBeUndefined();
  });

  it('expands the SNMP section up front when arriving via #snmp (Running Devices deep link)', async () => {
    mockLocation = { pathname: '/device-config/new', hash: '#snmp' };
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.expandedSections.has('snmp_agent')).toBe(true);
    expect(result.current.expandedSections.has('basic')).toBe(true);
  });

  it('does not expand the SNMP section without the #snmp hash', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.expandedSections.has('snmp_agent')).toBe(false);
  });

  // Every section renders; the per-type schema only decides what comes first.
  // Hiding one would make its fields unreachable while the authoring-parity
  // gate still counted them bound.
  it('orders relevant sections first without dropping any', async () => {
    const { result } = render();
    await waitFor(() => expect(result.current.loading).toBe(false));

    const { DEVICE_SECTIONS } = await import('./generated/sections.generated');
    expect(result.current.sections).toHaveLength(DEVICE_SECTIONS.length);
  });
});
