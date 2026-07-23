import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { useUIStore } from '../../stores/ui-store';
import { ToastContainer } from '../../ui/ToastContainer';
import { RunningSimulationCard } from './RunningSimulationCard';

const fetchConfig = vi.fn();

vi.mock('../../api/client', () => ({
  fetchConfig: () => fetchConfig(),
}));

const simStatus = {
  interface: 'eth0',
  configName: 'demo.yaml',
  deviceCount: 3,
  uptimeSeconds: 120,
};

describe('RunningSimulationCard', () => {
  beforeEach(() => {
    fetchConfig.mockReset();
    useUIStore.getState().clearNotifications();
  });

  it('surfaces a download failure as a toast, not a bespoke inline banner', async () => {
    const user = userEvent.setup();
    fetchConfig.mockRejectedValue(new Error('daemon unreachable'));

    render(
      <MemoryRouter>
        <RunningSimulationCard
          simStatus={simStatus}
          stopping={false}
          onStop={vi.fn()}
          message={null}
        />
        <ToastContainer />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole('button', { name: 'Download YAML' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('daemon unreachable');
  });

  it('does not show an error after a successful download', async () => {
    const user = userEvent.setup();
    fetchConfig.mockResolvedValue({ content: 'devices: []', filename: 'niac-config.yaml' });

    // jsdom doesn't implement createObjectURL/revokeObjectURL
    URL.createObjectURL = vi.fn(() => 'blob:mock');
    URL.revokeObjectURL = vi.fn();

    render(
      <MemoryRouter>
        <RunningSimulationCard
          simStatus={simStatus}
          stopping={false}
          onStop={vi.fn()}
          message={null}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole('button', { name: 'Download YAML' }));

    expect(fetchConfig).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('shows the physical attachment, virtual networks, routes, and counters', () => {
    render(
      <MemoryRouter>
        <RunningSimulationCard
          simStatus={{
            ...simStatus,
            fabric: {
              topology: {
                binding: {
                  attachment: 'tester',
                  interface: 'eth0',
                  mode: 'access',
                  physicalVlan: 200,
                  network: 'access',
                  wireTagged: false,
                },
                networks: [{ name: 'servers', prefix: '10.20.0.0/24', virtualVlan: 20 }],
                interfaces: [],
                routes: [
                  {
                    device: 'edge',
                    destination: '10.20.0.0/24',
                    via: 'inside',
                    connected: true,
                  },
                ],
                dhcpScopes: [],
              },
              forwarded: 12,
              drops: 3,
              received: 20,
              transmitted: 10,
            },
          }}
          stopping={false}
          onStop={vi.fn()}
          message={null}
        />
      </MemoryRouter>,
    );

    expect(screen.getByText(/Attachment: tester/)).toBeInTheDocument();
    expect(screen.getByText(/Physical VLAN:.*200/)).toBeInTheDocument();
    expect(screen.getByText(/Forwarded: 12/)).toBeInTheDocument();
    expect(screen.getByText(/Drops: 3/)).toBeInTheDocument();
    expect(screen.getByText(/servers.*10\.20\.0\.0\/24.*Virtual VLAN:.*20/)).toBeInTheDocument();
    expect(screen.getByText(/edge:.*10\.20\.0\.0\/24.*inside/)).toBeInTheDocument();
  });
});
