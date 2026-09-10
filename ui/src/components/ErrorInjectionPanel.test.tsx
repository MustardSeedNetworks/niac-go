// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

/**
 * ErrorInjectionPanel.test.tsx — locks the ?errorType= deep-link preselect
 * added for the Dashboard's error-type catalog links (PR "Dashboard
 * honesty + shell sim-status"). Clicking a catalog entry on the Dashboard
 * navigates here with the type in the query string; the real injection
 * form must land with that type already selected.
 */

import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { injectError } from '../api/client';
import type { ErrorInjectionInfo } from '../api/types';
import { renderWithResources as render } from '../test/renderWithResources';
import { required } from '../test/required';
import '../i18n';
import { useApiResource } from '../hooks/useApiResource';
import { ErrorInjectionPanel } from './ErrorInjectionPanel';

const fetchErrorTypes = vi.fn<() => Promise<ErrorInjectionInfo>>();
const clearError = vi.fn<(device: string, iface: string, errorType: string) => Promise<unknown>>();

vi.mock('../contexts/AppContext', () => ({
  useAppState: () => useApiResource(fetchErrorTypes, ['errors']),
}));

vi.mock('../api/client', () => ({
  fetchErrorTypes: () => fetchErrorTypes(),
  injectError: vi.fn(),
  clearError: (device: string, iface: string, errorType: string) =>
    clearError(device, iface, errorType),
  clearAllErrors: vi.fn(),
}));

const errorInfo: ErrorInjectionInfo = {
  availableTypes: [
    { type: 'drop', description: 'Drop packets', valueKind: 'number' },
    { type: 'delay', description: 'Delay packets', valueKind: 'number' },
  ],
  info: 'Inject errors on a device interface.',
  targets: [
    {
      device: 'sw-core',
      address: '10.0.0.1',
      interfaces: ['Gi0/1', 'Gi0/2'],
      errorTypes: { 'Gi0/1': ['drop', 'delay'], 'Gi0/2': ['drop', 'delay'] },
    },
    { device: 'sw-edge', address: '10.0.0.2', interfaces: [], errorTypes: {} },
  ],
};

describe('ErrorInjectionPanel', () => {
  beforeEach(() => {
    vi.mocked(injectError).mockReset();
    fetchErrorTypes.mockReset().mockResolvedValue(errorInfo);
    clearError.mockReset().mockResolvedValue({});
  });

  it.each(['0', '32'])(
    'submits /%s as a typed mask without a numeric fault value',
    async (prefix) => {
      fetchErrorTypes.mockResolvedValue({
        ...errorInfo,
        availableTypes: [{ type: 'Bad Subnet Mask', description: 'Mask', valueKind: 'prefix' }],
        targets: [
          { device: 'client', interfaces: ['eth0'], errorTypes: { eth0: ['Bad Subnet Mask'] } },
        ],
      });
      render(
        <MemoryRouter>
          <ErrorInjectionPanel />
        </MemoryRouter>,
      );
      await screen.findByRole('option', { name: 'client' });
      fireEvent.change(screen.getByLabelText('Device'), { target: { value: 'client' } });
      fireEvent.change(screen.getByLabelText('Interface'), { target: { value: 'eth0' } });
      fireEvent.change(screen.getByLabelText('Error Type'), {
        target: { value: 'Bad Subnet Mask' },
      });
      const input = screen.getByTestId('interface-fault-prefix');
      const submit = screen.getByTestId('apply-interface-fault');
      expect(submit).toBeDisabled();
      for (const value of ['-1', '33', '1.5', '']) {
        fireEvent.change(input, { target: { value } });
        expect(submit).toBeDisabled();
      }
      expect(injectError).not.toHaveBeenCalled();
      fireEvent.change(input, { target: { value: prefix } });
      expect(submit).toBeEnabled();
      fireEvent.click(submit);
      await waitFor(() =>
        expect(injectError).toHaveBeenCalledWith({
          device: 'client',
          interface: 'eth0',
          errorType: 'Bad Subnet Mask',
          prefixBits: Number(prefix),
        }),
      );
    },
  );

  it('renders /0 as active and clears only the selected mask', async () => {
    fetchErrorTypes.mockResolvedValue({
      ...errorInfo,
      activeErrors: {
        client: { eth0: { 'Bad Subnet Mask': { prefixBits: 0 }, 'FCS Errors': { value: 20 } } },
      },
    });
    render(
      <MemoryRouter>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );
    expect(await screen.findByText('/0')).toBeInTheDocument();
    expect(screen.getByText('20/s')).toBeInTheDocument();
    const buttons = screen.getAllByRole('button', { name: /Clear error on/ });
    fireEvent.click(required(buttons[0], 'mask clear button'));
    await waitFor(() =>
      expect(clearError).toHaveBeenCalledWith('client', 'eth0', 'Bad Subnet Mask'),
    );
    expect(injectError).not.toHaveBeenCalled();
  });

  it('submits an IPv4 conflict without a numeric value and rejects invalid addresses', async () => {
    fetchErrorTypes.mockResolvedValue({
      ...errorInfo,
      availableTypes: [
        ...errorInfo.availableTypes,
        { type: 'Duplicate IP', description: 'Conflict', valueKind: 'address' },
      ],
      targets: [{ device: 'client', interfaces: ['eth0'], errorTypes: { eth0: ['Duplicate IP'] } }],
    });
    render(
      <MemoryRouter>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );
    await screen.findByRole('option', { name: 'client' });
    fireEvent.change(screen.getByLabelText('Device'), { target: { value: 'client' } });
    fireEvent.change(screen.getByLabelText('Interface'), { target: { value: 'eth0' } });
    fireEvent.change(screen.getByLabelText('Error Type'), { target: { value: 'Duplicate IP' } });
    const input = screen.getByLabelText(/IPv4/);
    const submit = screen.getByRole('button', { name: 'Inject Error' });
    fireEvent.change(input, { target: { value: '224.0.0.1' } });
    expect(submit).toBeDisabled();
    expect(injectError).not.toHaveBeenCalled();
    fireEvent.change(input, { target: { value: '192.0.2.20' } });
    expect(submit).toBeEnabled();
    fireEvent.click(submit);
    await waitFor(() =>
      expect(injectError).toHaveBeenCalledWith({
        device: 'client',
        interface: 'eth0',
        errorType: 'Duplicate IP',
        address: '192.0.2.20',
      }),
    );
  });

  it('surfaces a failed fault catalog read', async () => {
    fetchErrorTypes.mockRejectedValue(new Error('catalog unavailable'));
    render(
      <MemoryRouter>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );
    expect(await screen.findByRole('alert')).toHaveTextContent('catalog unavailable');
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
    await screen.findByRole('option', { name: /sw-core/ });
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
    await screen.findByRole('option', { name: /sw-edge/ });
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
          'Gi0/1': { 'FCS Errors': { value: 25 }, 'Packet Discards': { value: 40 } },
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

  it('renders an addressed interface fault and clears only that named fault', async () => {
    fetchErrorTypes.mockResolvedValue({
      ...errorInfo,
      activeErrors: {
        'sw-core': {
          'Gi0/1': {
            'Duplicate IP': { address: '192.0.2.20' },
            'High Utilization': { value: 70 },
          },
        },
      },
    });
    render(
      <MemoryRouter>
        <ErrorInjectionPanel />
      </MemoryRouter>,
    );
    expect(await screen.findByText('192.0.2.20')).toBeInTheDocument();
    expect(screen.getByText('70%')).toBeInTheDocument();
    const buttons = await screen.findAllByRole('button', { name: /Clear error on/ });
    fireEvent.click(required(buttons[0], 'address fault clear'));
    await waitFor(() =>
      expect(clearError).toHaveBeenCalledWith('sw-core', 'Gi0/1', 'Duplicate IP'),
    );
  });
});
