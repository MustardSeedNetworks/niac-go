import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/errors';
import '../../i18n';
import { PreflightStep } from './PreflightStep';

const preflightSimulation = vi.fn();
vi.mock('../../api/client', () => ({
  preflightSimulation: (payload: unknown) => preflightSimulation(payload),
}));

/**
 * Guards #1472.
 *
 * #1461 made the server enumerate its validation errors, and it does — on
 * CT304 a failing preflight returns one detail per error, each with the field
 * path that names the offending device. PreflightStep narrowed the rejection to
 * `Error` and kept only `message`, so the one screen whose job is diagnosis
 * showed a bare "Simulation preflight failed".
 *
 * #1429 (D3) threaded details through the *toast* path; the wizard renders this
 * failure as an inline banner and never reaches that code.
 */
const request = { interface: 'eth0', configData: 'devices: []\n' } as never;

describe('PreflightStep validation failures', () => {
  it('lists every detail the server sent, with its field', async () => {
    const user = userEvent.setup();
    preflightSimulation.mockRejectedValue(
      new ApiError('Simulation preflight failed', 400, 'preflight_failed', [
        {
          field: 'devices[0].snmp_agent.community',
          issue: 'SNMPv1/v2c requires an explicit community',
        },
        {
          field: 'devices[1].snmp_agent.community',
          issue: 'SNMPv1/v2c requires an explicit community',
        },
      ]),
    );

    render(<PreflightStep request={request} onStart={vi.fn()} />);
    await user.click(screen.getByTestId('wizard-preflight-check'));

    expect(await screen.findAllByText(/SNMPv1\/v2c requires an explicit community/)).toHaveLength(
      2,
    );
    expect(screen.getByText(/devices\[0\]\.snmp_agent\.community/)).toBeInTheDocument();
    expect(screen.getByText(/devices\[1\]\.snmp_agent\.community/)).toBeInTheDocument();
  });

  it('still shows the message when the server sent no details', async () => {
    const user = userEvent.setup();
    preflightSimulation.mockRejectedValue(
      new ApiError('Simulation preflight failed', 400, 'preflight_failed', []),
    );

    render(<PreflightStep request={request} onStart={vi.fn()} />);
    await user.click(screen.getByTestId('wizard-preflight-check'));

    expect(await screen.findByText('Simulation preflight failed')).toBeInTheDocument();
  });
});
