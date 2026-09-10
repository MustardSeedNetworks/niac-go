import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, expect, it, vi } from 'vitest';
import '../i18n';
import { TrafficInjectionPage } from '../pages/TrafficInjectionPage';

vi.mock('./ReplayControlPanel', () => ({ ReplayControlPanel: () => null }));

const state = vi.hoisted(() => ({
  disabled: false,
  inject: vi.fn(),
  clear: vi.fn(),
  refetch: vi.fn(),
  data: {
    availableDeviceTypes: [
      { type: 'Captive Portal', description: 'Redirect HTTP', maxValue: 1, valueKind: 'toggle' },
      { type: 'CPU Utilization', description: 'CPU load', maxValue: 100, valueKind: 'percent' },
      { type: 'Latency', description: 'Delay echoes', maxValue: 60000, valueKind: 'milliseconds' },
      {
        type: 'Duplicate DHCP Offer',
        description: 'Offer a peer-owned address',
        valueKind: 'address',
      },
    ],
    deviceTargets: [
      {
        device: 'gateway',
        errorTypes: ['Captive Portal', 'CPU Utilization', 'Latency', 'Duplicate DHCP Offer'],
      },
      { device: 'client', errorTypes: ['Latency'] },
    ],
    activeDeviceErrors: {
      gateway: {
        'Captive Portal': { value: 1 },
        'CPU Utilization': { value: 87 },
        'Duplicate DHCP Offer': { address: '192.0.2.20' },
      },
    },
  },
}));
vi.mock('../contexts/AppContext', () => ({
  useAppState: () => ({ data: state.data, refetch: state.refetch }),
}));
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: state.disabled }),
}));
vi.mock('../api/client', () => ({ injectError: state.inject, clearError: state.clear }));
beforeEach(() => {
  vi.clearAllMocks();
  state.disabled = false;
  state.inject.mockReset();
  state.clear.mockReset();
});

it('submits and clears an address fault without a numeric value', async () => {
  render(
    <MemoryRouter>
      <TrafficInjectionPage />
    </MemoryRouter>,
  );
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'gateway' } });
  fireEvent.change(screen.getByLabelText('Device fault'), {
    target: { value: 'Duplicate DHCP Offer' },
  });
  expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument();
  expect(screen.getByTestId('apply-device-fault')).toBeDisabled();
  fireEvent.change(screen.getByLabelText('Conflict IPv4 address'), {
    target: { value: '224.0.0.1' },
  });
  expect(screen.getByTestId('apply-device-fault')).toBeDisabled();
  fireEvent.change(screen.getByLabelText('Conflict IPv4 address'), {
    target: { value: '192.0.2.20' },
  });
  fireEvent.click(screen.getByTestId('apply-device-fault'));
  await waitFor(() =>
    expect(state.inject).toHaveBeenCalledWith({
      device: 'gateway',
      interface: '',
      errorType: 'Duplicate DHCP Offer',
      address: '192.0.2.20',
    }),
  );
  expect(screen.getByText('192.0.2.20')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Clear Duplicate DHCP Offer on gateway' }));
  await waitFor(() =>
    expect(state.clear).toHaveBeenCalledWith('gateway', '', 'Duplicate DHCP Offer'),
  );
});

it.each([
  ['CPU Utilization', 100],
  ['Latency', 60000],
] as const)('submits the authored %s value instead of a default rate', async (type, value) => {
  render(
    <MemoryRouter>
      <TrafficInjectionPage />
    </MemoryRouter>,
  );
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'gateway' } });
  fireEvent.change(screen.getByLabelText('Device fault'), { target: { value: type } });
  fireEvent.change(screen.getByRole('spinbutton'), { target: { value: String(value) } });
  fireEvent.click(screen.getByRole('button', { name: 'Apply device fault' }));
  await waitFor(() =>
    expect(state.inject).toHaveBeenCalledWith({
      device: 'gateway',
      interface: '',
      errorType: type,
      value,
    }),
  );
});

it('keeps faults visible to viewers but blocks apply and clear', () => {
  state.disabled = true;
  render(
    <MemoryRouter>
      <TrafficInjectionPage />
    </MemoryRouter>,
  );
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'gateway' } });
  fireEvent.change(screen.getByLabelText('Device fault'), { target: { value: 'Captive Portal' } });
  expect(screen.getByRole('button', { name: 'Apply device fault' })).toBeDisabled();
  const clear = screen.getByRole('button', { name: 'Clear Captive Portal on gateway' });
  expect(clear).toBeDisabled();
  fireEvent.click(clear);
  expect(state.clear).not.toHaveBeenCalled();
  expect(screen.getByText('Armed')).toBeVisible();
});

it('surfaces a rejected device fault without claiming success', async () => {
  state.inject.mockRejectedValue(new Error('service unavailable'));
  render(
    <MemoryRouter>
      <TrafficInjectionPage />
    </MemoryRouter>,
  );
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'gateway' } });
  fireEvent.change(screen.getByLabelText('Device fault'), { target: { value: 'Captive Portal' } });
  fireEvent.click(screen.getByRole('button', { name: 'Apply device fault' }));
  expect(await screen.findByRole('alert')).toHaveTextContent('service unavailable');
  expect(state.refetch).not.toHaveBeenCalled();
});

it('arms a device portal with no interface or rate and clears only that device fault', async () => {
  render(
    <MemoryRouter>
      <TrafficInjectionPage />
    </MemoryRouter>,
  );
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'gateway' } });
  fireEvent.change(screen.getByLabelText('Device fault'), { target: { value: 'Captive Portal' } });
  expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Apply device fault' }));
  await waitFor(() =>
    expect(state.inject).toHaveBeenCalledWith({
      device: 'gateway',
      interface: '',
      errorType: 'Captive Portal',
      value: 1,
    }),
  );
  expect(screen.getByText('Armed')).toBeVisible();
  expect(screen.getByText('87%')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Clear Captive Portal on gateway' }));
  await waitFor(() => expect(state.clear).toHaveBeenCalledWith('gateway', '', 'Captive Portal'));
});

it('uses catalog bounds and removes an unsupported fault when the device changes', async () => {
  render(
    <MemoryRouter>
      <TrafficInjectionPage />
    </MemoryRouter>,
  );
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'gateway' } });
  fireEvent.change(screen.getByLabelText('Device fault'), { target: { value: 'CPU Utilization' } });
  const input = screen.getByRole('spinbutton');
  expect(input).toHaveAttribute('max', '100');
  fireEvent.change(input, { target: { value: '101' } });
  expect(screen.getByRole('button', { name: 'Apply device fault' })).toBeDisabled();
  fireEvent.change(screen.getByLabelText('Device fault target'), { target: { value: 'client' } });
  expect(screen.queryByRole('option', { name: 'Captive Portal' })).not.toBeInTheDocument();
  expect(screen.getByLabelText('Device fault')).toHaveValue('');
  fireEvent.change(screen.getByLabelText('Device fault'), { target: { value: 'Latency' } });
  expect(screen.getByRole('spinbutton')).toHaveAttribute('max', '60000');
});
