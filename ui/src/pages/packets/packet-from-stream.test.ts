import { describe, expect, it } from 'vitest';
import { buildProtocolLayers } from '../../utils/protocol-layers';
import { packetFromStreamEvent } from './packet-from-stream';

/**
 * Guards #1458.
 *
 * A CDP frame is LLC-encapsulated and has no IP layer at all, yet the inspector
 * rendered "Internet Protocol Version 4 (IPv4) / Source: Unknown / Destination:
 * Unknown" under it on CT304. The mapping coerced absent addresses to the
 * literal string 'Unknown' — a display placeholder leaking into the data model —
 * and buildIpLayer's fallback branch tests truthiness, which 'Unknown' passes.
 *
 * These assert the mapping and the layer builder together, because neither half
 * is wrong on its own: the bug only exists where the sentinel meets the branch.
 */

/** A CDP frame exactly as internal/api/sse/packet_observer.go emits it. */
const cdpEvent = {
  timestamp: '2026-08-23T14:30:15.000Z',
  protocol: 'CDP',
  size: 132,
  raw_data: '01000ccccccc00000c4a080200760aaa',
  headers: {
    ethernet: {
      srcMac: '00:00:0c:4a:08:02',
      dstMac: '01:00:0c:cc:cc:cc',
      etherType: 'LLC',
    },
  },
};

describe('packetFromStreamEvent', () => {
  it('leaves addresses absent on a frame that has no IP layer', () => {
    const packet = packetFromStreamEvent(cdpEvent, '2026-08-23T14:30:15.000Z', 'p1');

    expect(packet.sourceIp, "'Unknown' is a display placeholder, not an address").toBeUndefined();
    expect(packet.destIp).toBeUndefined();
  });

  it('does not fabricate an IPv4 layer for a CDP frame', () => {
    const packet = packetFromStreamEvent(cdpEvent, '2026-08-23T14:30:15.000Z', 'p1');
    const layers = buildProtocolLayers(packet.headers, packet);

    expect(layers.map((l) => l.name)).not.toContain('Internet Protocol Version 4 (IPv4)');
    expect(layers.map((l) => l.name)).toContain('Ethernet II');
  });

  it('still renders the IP layer when the frame really carries one', () => {
    const packet = packetFromStreamEvent(
      {
        protocol: 'UDP',
        size: 74,
        source_ip: '10.44.10.5',
        dest_ip: '10.44.10.1',
        headers: { ethernet: { srcMac: 'aa:bb:cc:dd:ee:ff', dstMac: '11:22:33:44:55:66' } },
      },
      '2026-08-23T14:30:15.000Z',
      'p2',
    );
    const layers = buildProtocolLayers(packet.headers, packet);

    expect(packet.sourceIp).toBe('10.44.10.5');
    const ip = layers.find((l) => l.name.startsWith('Internet Protocol'));
    expect(ip?.fields.find((f) => f.name === 'Source')?.value).toBe('10.44.10.5');
  });
});
