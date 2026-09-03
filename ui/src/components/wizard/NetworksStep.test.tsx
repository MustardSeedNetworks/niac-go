import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { NetworksStep } from './NetworksStep';
import { parseNetworkModel } from './network-addressing';

const config = `# clinic branch office
networks:
  - name: clinic-lan
    subnet: 10.20.0.0/24

devices:
  - name: clinic-rtr-01
    type: router
    mac: "00:1A:2B:20:00:20"
  - name: clinic-srv-01
    type: server
    mac: "00:1A:2B:20:00:21"
`;

const Harness = ({ initial = config }: { initial?: string }) => {
  const [content, setContent] = useState(initial);
  return (
    <>
      <NetworksStep content={content} onChange={setContent} />
      <pre data-testid="content">{content}</pre>
    </>
  );
};

const currentContent = () => screen.getByTestId('content').textContent ?? '';

describe('NetworksStep', () => {
  it('adds a network without disturbing the rest of the config', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByTestId('networks-add'));

    const model = parseNetworkModel(currentContent());
    expect(model.networks).toHaveLength(2);
    expect(currentContent()).toContain('# clinic branch office');
    expect(currentContent()).toContain('mac: "00:1A:2B:20:00:20"');
  });

  it('removes the section entirely when the last network goes', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByTestId('networks-remove-0'));

    // `networks:` with nothing under it is an explicit empty list, which is a
    // different config from having no networks at all.
    expect(currentContent()).not.toContain('networks:');
    expect(parseNetworkModel(currentContent()).devices).toHaveLength(2);
  });

  it('auto-assigns an address inside the chosen network', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.selectOptions(screen.getByTestId('addressing-network-clinic-rtr-01'), 'clinic-lan');

    const model = parseNetworkModel(currentContent());
    expect(model.devices[0]?.address).toBe('10.20.0.1/24');
    expect(model.devices[0]?.network).toBe('clinic-lan');
  });

  it('never hands the same address to two devices', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByTestId('addressing-assign-all'));

    const model = parseNetworkModel(currentContent());
    const addresses = model.devices.map((entry) => entry.address);
    expect(addresses).toEqual(['10.20.0.1/24', '10.20.0.2/24']);
    expect(new Set(addresses).size).toBe(addresses.length);
  });

  it('cannot add an attachment before a network exists', async () => {
    const user = userEvent.setup();
    render(<Harness initial={'devices: []\n'} />);

    // An attachment names a network; offering the control first would only
    // author `connect:` with nothing to point at.
    expect(screen.getByTestId('attachments-add')).toBeDisabled();
    await user.click(screen.getByTestId('networks-add'));
    expect(screen.getByTestId('attachments-add')).toBeEnabled();
  });
});
