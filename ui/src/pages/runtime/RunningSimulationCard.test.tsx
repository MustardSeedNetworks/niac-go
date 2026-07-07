import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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
  });

  it('surfaces a visible error when downloading the config fails', async () => {
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
});
