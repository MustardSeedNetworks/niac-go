import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AttachmentPoliciesResponse, SimulationAttachments } from '../../api/fabric-types';
import '../../i18n';
import { PreflightStep } from './PreflightStep';

const preflightSimulation = vi.fn();
const fetchAttachmentPolicies = vi.fn();
const fetchSimulationAttachments = vi.fn();
vi.mock('../../api/client', () => ({
  preflightSimulation: (payload: unknown) => preflightSimulation(payload),
  fetchAttachmentPolicies: () => fetchAttachmentPolicies(),
  fetchSimulationAttachments: (payload: unknown) => fetchSimulationAttachments(payload),
}));
vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

const request = { interface: 'eth0', configData: 'networks: []\n' } as never;

const policies = (response: AttachmentPoliciesResponse) =>
  fetchAttachmentPolicies.mockResolvedValue(response);
const attachments = (response: SimulationAttachments) =>
  fetchSimulationAttachments.mockResolvedValue(response);

beforeEach(() => {
  vi.clearAllMocks();
  preflightSimulation.mockResolvedValue({
    safe: true,
    topology: {
      binding: {
        attachment: 'cyberscope',
        interface: 'eth0',
        mode: 'access',
        network: 'lab-transit',
        wireTagged: false,
      },
      networks: [],
      interfaces: [],
      routes: [],
      dhcpScopes: [],
    },
    diagnostics: [],
  });
});

/**
 * AP-0 / D1. Preflight's attachment was free text defaulting to the literal
 * `tester` while every generated pack names its attachment `cyberscope`, so
 * the out-of-box path failed with unknown_attachment and nothing in the UI
 * named the valid choice.
 */
describe('PreflightStep binding inputs', () => {
  it('offers the config attachment name without the operator typing it', async () => {
    policies({ policies: [{ interface: 'eth0', mode: 'access', accessVlan: 200 }] });
    attachments({ routed: true, attachments: ['cyberscope'] });

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    const field = await screen.findByTestId('wizard-attachment-name');
    expect(field.tagName).toBe('SELECT');
    expect((field as HTMLSelectElement).value).toBe('cyberscope');
    expect(screen.getByRole('option', { name: 'cyberscope' })).toBeInTheDocument();
  });

  it('sends the config attachment, not a typed default, to preflight and start', async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    policies({ policies: [{ interface: 'eth0', mode: 'access', accessVlan: 200 }] });
    attachments({ routed: true, attachments: ['cyberscope'] });

    render(<PreflightStep request={request} onStart={onStart} />);
    await screen.findByTestId('wizard-attachment-name');
    await user.click(screen.getByTestId('wizard-preflight-check'));
    await waitFor(() => expect(preflightSimulation).toHaveBeenCalled());
    await user.click(screen.getByTestId('wizard-preflight-start'));

    expect(preflightSimulation).toHaveBeenCalledWith(
      expect.objectContaining({
        attachment: 'cyberscope',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
    expect(onStart).toHaveBeenCalledWith(
      expect.objectContaining({ attachment: 'cyberscope', accessVlan: 200 }),
    );
  });

  it('offers only policy-approved modes', async () => {
    policies({
      policies: [
        { interface: 'eth0', mode: 'trunk', allowedVlans: [200, 210] },
        { interface: 'eth1', mode: 'direct' },
      ],
    });
    attachments({ routed: true, attachments: ['cyberscope'] });

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    const mode = (await screen.findByTestId('wizard-attachment-mode')) as HTMLSelectElement;
    expect([...mode.options].map((option) => option.value)).toEqual(['trunk']);
    expect(mode.value).toBe('trunk');
  });

  it('offers only policy-approved VLANs, and none at all in direct mode', async () => {
    policies({
      policies: [
        { interface: 'eth0', mode: 'trunk', allowedVlans: [200, 210] },
        { interface: 'eth0', mode: 'direct' },
      ],
    });
    attachments({ routed: true, attachments: ['cyberscope'] });
    const user = userEvent.setup();

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId('wizard-access-vlan').tagName).toBe('SELECT'));
    const vlan = screen.getByTestId('wizard-access-vlan') as HTMLSelectElement;
    expect([...vlan.options].map((option) => option.value)).toEqual(['200', '210']);

    await user.selectOptions(screen.getByTestId('wizard-attachment-mode'), 'direct');
    expect(screen.queryByTestId('wizard-access-vlan')).not.toBeInTheDocument();
  });

  // A daemon started with no --attachment-policy approves nothing routed. The
  // operator has to be told that, not left to read attachment_policy_denied
  // off a failed start.
  it('says so when no policy approves the selected interface', async () => {
    policies({ policies: [{ interface: 'eth9', mode: 'access', accessVlan: 200 }] });
    attachments({ routed: true, attachments: ['cyberscope'] });

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    expect(await screen.findByTestId('wizard-no-policy-notice')).toBeInTheDocument();
  });

  // A flat scenario binds no attachment and needs no policy in direct or
  // access mode. Demanding a choice it cannot offer would block a start that
  // works today.
  it('asks for no attachment when the scenario is flat', async () => {
    policies({ policies: [] });
    attachments({ routed: false, attachments: [] });

    render(<PreflightStep request={request} onStart={vi.fn()} />);
    await waitFor(() => expect(fetchSimulationAttachments).toHaveBeenCalled());

    expect(screen.queryByTestId('wizard-attachment-name')).not.toBeInTheDocument();
  });

  // Discovery can fail on a config that cannot be read. Falling back to the
  // free-text field keeps the screen usable instead of leaving an empty picker
  // and a disabled button with no explanation.
  it('falls back to a typed attachment when discovery fails', async () => {
    policies({ policies: [{ interface: 'eth0', mode: 'access', accessVlan: 200 }] });
    fetchSimulationAttachments.mockRejectedValue(new Error('Configuration could not be read'));

    render(<PreflightStep request={request} onStart={vi.fn()} />);

    const field = await screen.findByTestId('wizard-attachment-name');
    expect(field.tagName).toBe('INPUT');
  });
});
