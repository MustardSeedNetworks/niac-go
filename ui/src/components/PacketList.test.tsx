/**
 * PacketList.test.tsx — auto-scroll behavior.
 *
 * Regression coverage for a bug where the "auto-scroll" checkbox only
 * toggled CSS `scroll-behavior` and never actually moved the viewport, so
 * live captures silently stopped following new packets. The list must
 * scroll its container to the newest packet whenever autoScroll is on and
 * packets are appended, and must NOT do so when autoScroll is off.
 */
import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { type Packet, PacketList } from './PacketList';

function makePacket(id: string): Packet {
  return {
    id,
    timestamp: new Date().toISOString(),
    protocol: 'TCP',
    sourceIp: '10.0.0.1',
    destIp: '10.0.0.2',
    sourcePort: 1234,
    destPort: 80,
    size: 64,
    summary: `packet ${id}`,
    rawData: '00112233445566778899aabbccddeeff0011223344',
  };
}

describe('PacketList — auto-scroll', () => {
  it('scrolls the list container to the bottom when a new packet arrives with autoScroll on', () => {
    const { container, rerender } = render(
      <PacketList
        packets={[makePacket('1'), makePacket('2')]}
        selectedPacketId={null}
        onSelectPacket={() => undefined}
        autoScroll={true}
      />,
    );

    const scrollEl = container.querySelector('.overflow-y-auto') as HTMLDivElement;
    expect(scrollEl).not.toBeNull();

    // jsdom doesn't compute layout, so simulate a scrollable container by
    // stubbing scrollHeight and spying on the scrollTop assignment.
    Object.defineProperty(scrollEl, 'scrollHeight', { value: 5000, configurable: true });
    scrollEl.scrollTop = 0;

    rerender(
      <PacketList
        packets={[makePacket('1'), makePacket('2'), makePacket('3')]}
        selectedPacketId={null}
        onSelectPacket={() => undefined}
        autoScroll={true}
      />,
    );

    expect(scrollEl.scrollTop).toBe(5000);
  });

  it('does not force-scroll when autoScroll is off', () => {
    const { container, rerender } = render(
      <PacketList
        packets={[makePacket('1'), makePacket('2')]}
        selectedPacketId={null}
        onSelectPacket={() => undefined}
        autoScroll={false}
      />,
    );

    const scrollEl = container.querySelector('.overflow-y-auto') as HTMLDivElement;
    Object.defineProperty(scrollEl, 'scrollHeight', { value: 5000, configurable: true });
    scrollEl.scrollTop = 42;

    rerender(
      <PacketList
        packets={[makePacket('1'), makePacket('2'), makePacket('3')]}
        selectedPacketId={null}
        onSelectPacket={() => undefined}
        autoScroll={false}
      />,
    );

    expect(scrollEl.scrollTop).toBe(42);
  });
});
