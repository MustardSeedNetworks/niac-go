import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '../test/a11y';
import { InfoPopover } from './InfoPopover';

describe('InfoPopover', () => {
  it('is closed by default and opens on click, revealing its content', async () => {
    const user = userEvent.setup();
    render(
      <InfoPopover label="About BPF" title="BPF">
        Berkeley Packet Filter — a kernel-level expression that drops packets before capture.
      </InfoPopover>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();

    const trigger = screen.getByRole('button', { name: 'About BPF' });
    expect(trigger).toHaveAttribute('aria-expanded', 'false');

    await user.click(trigger);

    expect(trigger).toHaveAttribute('aria-expanded', 'true');
    const panel = screen.getByRole('dialog');
    expect(panel).toBeInTheDocument();
    expect(screen.getByText(/Berkeley Packet Filter/)).toBeInTheDocument();
  });

  it('closes on Escape and returns focus to the trigger', async () => {
    const user = userEvent.setup();
    render(
      <InfoPopover label="About OID" title="OID">
        An object identifier addressing a value in an SNMP MIB tree.
      </InfoPopover>,
    );

    const trigger = screen.getByRole('button', { name: 'About OID' });
    await user.click(trigger);
    expect(screen.getByRole('dialog')).toBeInTheDocument();

    await user.keyboard('{Escape}');

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('closes on outside click', async () => {
    const user = userEvent.setup();
    render(
      <div>
        <InfoPopover label="About PCAP" title="PCAP">
          Packet capture — a saved recording of network traffic.
        </InfoPopover>
        <button type="button">outside</button>
      </div>,
    );

    await user.click(screen.getByRole('button', { name: 'About PCAP' }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'outside' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('has no axe violations open or closed', async () => {
    const { container, rerender } = render(
      <InfoPopover label="About 5-tuple" title="5-tuple">
        The protocol, source IP:port, and destination IP:port that identify a conversation.
      </InfoPopover>,
    );
    await expectNoAxeViolations(container);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: 'About 5-tuple' }));
    rerender(
      <InfoPopover label="About 5-tuple" title="5-tuple">
        The protocol, source IP:port, and destination IP:port that identify a conversation.
      </InfoPopover>,
    );
    await expectNoAxeViolations(container);
  });
});
