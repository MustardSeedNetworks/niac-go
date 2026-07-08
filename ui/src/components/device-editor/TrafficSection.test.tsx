/**
 * TrafficSection.test.tsx
 *
 * The traffic mechanisms (ARP announcements, periodic pings, random
 * traffic) each collapse independently so a user can focus on one
 * mechanism's fields without the others' state changing.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import type { Device } from '../../api/types';
import { TrafficSection } from './TrafficSection';

const baseDevice: Device = {
  hostname: 'edge-01',
  mac: '00:1A:2B:3C:4D:5E',
  type: 'server',
  traffic: {
    enabled: true,
    arpAnnouncements: { enabled: true, interval: 60 },
    periodicPings: { enabled: true, interval: 30, payloadSize: 56 },
    randomTraffic: { enabled: false, interval: 60, packetCount: 5, patterns: [] },
  },
};

describe('TrafficSection', () => {
  it('expands the ARP subsection by default and keeps the others collapsed', () => {
    render(
      <TrafficSection
        device={baseDevice}
        isExpanded={true}
        onToggle={vi.fn()}
        onUpdate={vi.fn()}
      />,
    );

    expect(screen.getByTestId('traffic-subsection-arp-header')).toHaveAttribute(
      'aria-expanded',
      'true',
    );
    expect(screen.getByTestId('traffic-subsection-pings-header')).toHaveAttribute(
      'aria-expanded',
      'false',
    );
    expect(screen.getByTestId('traffic-subsection-random-header')).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  it('toggles each subsection independently', async () => {
    const user = userEvent.setup();
    render(
      <TrafficSection
        device={baseDevice}
        isExpanded={true}
        onToggle={vi.fn()}
        onUpdate={vi.fn()}
      />,
    );

    const arpHeader = screen.getByTestId('traffic-subsection-arp-header');
    const pingsHeader = screen.getByTestId('traffic-subsection-pings-header');

    // Expand pings — ARP's own state must not change.
    await user.click(pingsHeader);
    expect(pingsHeader).toHaveAttribute('aria-expanded', 'true');
    expect(arpHeader).toHaveAttribute('aria-expanded', 'true');

    // Collapse ARP — pings must stay expanded.
    await user.click(arpHeader);
    expect(arpHeader).toHaveAttribute('aria-expanded', 'false');
    expect(pingsHeader).toHaveAttribute('aria-expanded', 'true');
  });

  it('hides subsection fields while collapsed but keeps the enable switch visible', async () => {
    const user = userEvent.setup();
    render(
      <TrafficSection
        device={baseDevice}
        isExpanded={true}
        onToggle={vi.fn()}
        onUpdate={vi.fn()}
      />,
    );

    // Pings subsection starts collapsed — its interval field is not rendered.
    expect(screen.queryByText('Payload Size (bytes)')).toBeNull();

    await user.click(screen.getByTestId('traffic-subsection-pings-header'));
    expect(screen.getByText('Payload Size (bytes)')).toBeInTheDocument();
  });
});
