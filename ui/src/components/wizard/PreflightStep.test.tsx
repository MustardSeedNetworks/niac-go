import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../../i18n';
import type { SimulationPreflightReport } from '../../api/fabric-types';
import { PreflightStep } from './PreflightStep';

const preflightSimulation = vi.fn();

vi.mock('../../api/client', () => ({
  preflightSimulation: (payload: unknown) => preflightSimulation(payload),
}));

const safeReport: SimulationPreflightReport = {
  safe: true,
  topology: {
    binding: {
      attachment: 'tester',
      interface: 'eth0',
      mode: 'access',
      accessVlan: 200,
      network: 'lab-access',
      wireTagged: false,
    },
    networks: [],
    interfaces: [],
    routes: [],
    dhcpScopes: [],
  },
  diagnostics: [],
};

beforeEach(() => {
  vi.clearAllMocks();
  preflightSimulation.mockResolvedValue(safeReport);
});

describe('PreflightStep', () => {
  it('checks access mode with VLAN 200 by default and starts only after a safe report', async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    render(
      <PreflightStep
        request={{ interface: 'eth0', configData: 'devices: []\n' }}
        onStart={onStart}
      />,
    );

    expect(screen.getByTestId('wizard-preflight-start')).toBeDisabled();
    await user.click(screen.getByTestId('wizard-preflight-check'));

    await waitFor(() => expect(preflightSimulation).toHaveBeenCalledTimes(1));
    expect(preflightSimulation).toHaveBeenCalledWith(
      expect.objectContaining({
        interface: 'eth0',
        attachment: 'tester',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
    expect(screen.getByTestId('wizard-preflight-start')).not.toBeDisabled();

    await user.click(screen.getByTestId('wizard-preflight-start'));
    expect(onStart).toHaveBeenCalledWith(expect.objectContaining({ accessVlan: 200 }));
  });

  it('accepts direct attachment without a VLAN and blocks start on diagnostics', async () => {
    const user = userEvent.setup();
    preflightSimulation.mockResolvedValue({
      ...safeReport,
      safe: false,
      diagnostics: [{ code: 'unknown_attachment', field: 'attachment', message: 'not found' }],
    });
    render(<PreflightStep request={{ interface: 'eth0' }} onStart={vi.fn()} />);

    await user.selectOptions(screen.getByTestId('wizard-attachment-mode'), 'direct');
    expect(screen.queryByTestId('wizard-access-vlan')).not.toBeInTheDocument();
    expect(screen.getByTestId('wizard-preflight-check')).toBeDisabled();
    await user.click(screen.getByTestId('wizard-dedicated-interface'));
    await user.click(screen.getByTestId('wizard-preflight-check'));

    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('not found'));
    expect(preflightSimulation).toHaveBeenCalledWith(
      expect.objectContaining({ attachmentMode: 'direct', dedicated: true }),
    );
    expect(preflightSimulation.mock.calls[0][0]).not.toHaveProperty('accessVlan');
    expect(screen.getByTestId('wizard-preflight-start')).toBeDisabled();
  });

  it('does not approve a changed payload when an older preflight resolves late', async () => {
    const user = userEvent.setup();
    let resolveFirst: (report: SimulationPreflightReport) => void = () => undefined;
    preflightSimulation.mockReturnValueOnce(
      new Promise<SimulationPreflightReport>((resolve) => {
        resolveFirst = resolve;
      }),
    );
    const onStart = vi.fn();
    render(<PreflightStep request={{ interface: 'eth0' }} onStart={onStart} />);

    await user.click(screen.getByTestId('wizard-preflight-check'));
    await user.clear(screen.getByTestId('wizard-access-vlan'));
    await user.type(screen.getByTestId('wizard-access-vlan'), '2');
    resolveFirst(safeReport);

    await waitFor(() => expect(preflightSimulation).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId('wizard-preflight-start')).toBeDisabled();
    await user.click(screen.getByTestId('wizard-preflight-check'));
    await waitFor(() => expect(screen.getByTestId('wizard-preflight-start')).not.toBeDisabled());
    await user.click(screen.getByTestId('wizard-preflight-start'));

    expect(onStart).toHaveBeenCalledWith(expect.objectContaining({ accessVlan: 2 }));
  });
});
