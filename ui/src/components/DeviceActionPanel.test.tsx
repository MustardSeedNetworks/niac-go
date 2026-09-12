import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, expect, it, vi } from 'vitest';
import '../i18n';
import { DeviceActionPanel } from './DeviceActionPanel';

const state = vi.hoisted(() => ({
  disabled: false,
  execute: vi.fn(),
  refetch: vi.fn(),
  data: {
    availableActions: [
      { type: 'reboot', description: 'Restart the device' },
      { type: 'stp_topology_change', description: 'Signal a topology change' },
    ],
    // A device with no STP cannot publish a topology change, so the daemon
    // does not offer it there. The panel must follow the daemon rather than
    // list every action against every device.
    actionTargets: [
      { device: 'core-sw-01', actions: ['reboot', 'stp_topology_change'] },
      { device: 'clinic-pc-04', actions: ['reboot'] },
    ],
  },
}));
vi.mock('../contexts/AppContext', () => ({
  useAppState: () => ({ data: state.data, refetch: state.refetch }),
}));
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: state.disabled }),
}));
vi.mock('../api/client', () => ({ executeDeviceAction: state.execute }));

beforeEach(() => {
  vi.clearAllMocks();
  state.disabled = false;
  state.execute.mockReset();
});

const renderPanel = () =>
  render(
    <MemoryRouter>
      <DeviceActionPanel />
    </MemoryRouter>,
  );

it('runs a reboot against the selected device', async () => {
  renderPanel();

  fireEvent.change(screen.getByLabelText('Device action target'), {
    target: { value: 'core-sw-01' },
  });
  fireEvent.change(screen.getByLabelText('Device action'), { target: { value: 'reboot' } });
  fireEvent.click(screen.getByTestId('run-device-action'));

  await waitFor(() => expect(state.execute).toHaveBeenCalledWith('core-sw-01', 'reboot'));
  expect(screen.getByText('Ran reboot on core-sw-01')).toBeVisible();
});

it('offers only the actions the daemon says the device can publish', () => {
  renderPanel();

  fireEvent.change(screen.getByLabelText('Device action target'), {
    target: { value: 'clinic-pc-04' },
  });

  const options = Array.from(screen.getByLabelText('Device action').querySelectorAll('option')).map(
    (option) => option.getAttribute('value'),
  );
  expect(options).toEqual(['', 'reboot']);
});

it('cannot run before an action is chosen', () => {
  renderPanel();

  fireEvent.change(screen.getByLabelText('Device action target'), {
    target: { value: 'core-sw-01' },
  });

  expect(screen.getByTestId('run-device-action')).toBeDisabled();
});
