import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { NetworksStep } from './NetworksStep';

/**
 * AP-1. An attachment can name a pool of free switch ports instead of a whole
 * network, which is the only form that can put a tester on a specific port. The
 * wizard must be able to author it: a scenario field the UI cannot write is one
 * an operator can only get by hand-editing YAML (owner decision 2026-09-02).
 */
const config = `networks:
  - name: med-data
    subnet: 10.51.210.0/24
    virtual_vlan: 210

attachments:
  - name: cyberscope
    connect: med-data

devices:
  - name: MED-ACC-SW01
    type: switch
    mac: "00:1A:2B:20:00:20"
    interfaces:
      - name: GigabitEthernet1/0/20
        vlans: [210]
      - name: GigabitEthernet1/0/21
        vlans: [210]
`;

const Harness = () => {
  const [content, setContent] = useState(config);
  return (
    <>
      <NetworksStep content={content} onChange={setContent} />
      <pre data-testid="content">{content}</pre>
    </>
  );
};

const currentContent = () => screen.getByTestId('content').textContent ?? '';

describe('NetworksStep attachment pools', () => {
  it('switches an attachment from a network to a pool of ports', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.selectOptions(screen.getByTestId('attachment-form-0'), 'ports');
    await user.selectOptions(screen.getByTestId('attachment-device-0'), 'MED-ACC-SW01');
    await user.click(screen.getByTestId('attachment-port-0-GigabitEthernet1/0/20'));

    expect(currentContent()).toContain('at:');
    expect(currentContent()).toContain('device: MED-ACC-SW01');
    expect(currentContent()).toContain('- GigabitEthernet1/0/20');
    expect(currentContent()).not.toContain('connect: med-data');
  });

  it('pins a client MAC to one port of the pool', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.selectOptions(screen.getByTestId('attachment-form-0'), 'ports');
    await user.selectOptions(screen.getByTestId('attachment-device-0'), 'MED-ACC-SW01');
    await user.click(screen.getByTestId('attachment-port-0-GigabitEthernet1/0/21'));
    await user.click(screen.getByTestId('attachment-pin-add-0'));
    await user.type(screen.getByTestId('attachment-pin-mac-0-0'), '00:c0:17:aa:bb:cc');

    const content = currentContent();
    expect(content).toContain('pins:');
    expect(content).toContain('mac: "00:c0:17:aa:bb:cc"');
    expect(content).toContain('interface: GigabitEthernet1/0/21');
  });

  // The port VLAN is scenario-internal: it decides which network a client lands
  // on, and is nothing to do with the VLAN the NIAC host is cabled on. Showing
  // it read-only is what keeps the two from being confused.
  it('shows each port VLAN without offering to change it', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.selectOptions(screen.getByTestId('attachment-form-0'), 'ports');
    await user.selectOptions(screen.getByTestId('attachment-device-0'), 'MED-ACC-SW01');

    const port = screen.getByTestId('attachment-port-row-0-GigabitEthernet1/0/20');
    expect(port).toHaveTextContent('210');
    expect(port.querySelector('input[type="number"]')).toBeNull();
  });

  it('keeps the network form working', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByTestId('attachments-add'));

    expect(currentContent()).toContain('connect: med-data');
    expect(screen.getByTestId('attachment-form-1')).toHaveValue('network');
  });
});
