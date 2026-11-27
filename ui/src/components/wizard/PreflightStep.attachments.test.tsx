import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/errors';
import '../../i18n';
import { PreflightStep } from './PreflightStep';

const fetchSimulationAttachments = vi.fn();
vi.mock('../../api/client', () => ({
  preflightSimulation: vi.fn(),
  fetchAttachmentPolicies: () => Promise.resolve({ policies: [] }),
  fetchSimulationAttachments: (payload: unknown) => fetchSimulationAttachments(payload),
}));

vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

const request = { interface: 'lo0', scenarioName: 'modern-hospital' } as never;

// The attachment lookup failing used to be swallowed: a routed scenario fell
// back to an empty attachment, Run preflight stayed disabled, and the page
// said nothing (#3154).
describe('PreflightStep attachment lookup failure', () => {
  it('shows why the scenario could not be read, with the server detail', async () => {
    fetchSimulationAttachments.mockRejectedValue(
      new ApiError('Configuration could not be read', 400, 'config_load_failed', [
        { issue: 'load configuration: device MED-WAN-R01: walk file not found: x.walk' },
      ]),
    );

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    const error = await screen.findByTestId('wizard-attachments-error');
    expect(error).toHaveTextContent('Configuration could not be read');
    expect(error).toHaveTextContent('walk file not found: x.walk');
    expect(screen.getByTestId('wizard-preflight-start')).toBeDisabled();
  });

  it('shows no error when the lookup succeeds', async () => {
    fetchSimulationAttachments.mockResolvedValue({ routed: false, attachments: [] });

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    await screen.findByTestId('wizard-preflight-check');
    expect(screen.queryByTestId('wizard-attachments-error')).not.toBeInTheDocument();
  });
});
