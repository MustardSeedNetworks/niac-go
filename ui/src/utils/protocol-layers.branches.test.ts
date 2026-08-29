/**
 * Branch coverage for the packet-inspector's protocol layer builder.
 *
 * protocol-layers.test.ts covers the common shapes. This covers the optional
 * fields and the fallbacks: nearly every field is conditionally emitted, so the
 * failure mode is a field silently missing from the inspector rather than an
 * error, and the byte offsets are what drive the hex-pane highlighting.
 */

import { describe, expect, it } from 'vitest';
import { buildProtocolLayers, computeHeaderBoundary } from './protocol-layers';

type Headers = Record<string, unknown>;

/** The layer with the given name, or a failure if the builder omitted it. */
function layer(layers: ReturnType<typeof buildProtocolLayers>, match: string) {
  const found = layers.find((l) => l.name.includes(match));
  if (!found)
    throw new Error(`no layer matching ${match} in ${layers.map((l) => l.name).join(', ')}`);
  return found;
}

/** Field names present on a layer. */
function fieldNames(layers: ReturnType<typeof buildProtocolLayers>, match: string): string[] {
  return layer(layers, match).fields.map((f) => f.name);
}

describe('frame layer', () => {
  it('prefers length over size and omits the optional fields', () => {
    const layers = buildProtocolLayers(undefined, { length: 128, size: 64 });

    expect(fieldNames(layers, 'Frame')).toEqual(['Length']);
    expect(layer(layers, 'Frame').fields[0]?.value).toBe('128 bytes');
  });

  it('falls back to size, then to zero', () => {
    expect(layer(buildProtocolLayers(undefined, { size: 64 }), 'Frame').fields[0]?.value).toBe(
      '64 bytes',
    );
    expect(layer(buildProtocolLayers(undefined, {}), 'Frame').fields[0]?.value).toBe('0 bytes');
  });

  it('includes the number and arrival time when present', () => {
    const layers = buildProtocolLayers(undefined, {
      number: 7,
      timestamp: '2026-08-29T10:00:00Z',
      length: 64,
    });

    expect(fieldNames(layers, 'Frame')).toEqual(['Number', 'Length', 'Arrival Time']);
  });
});

describe('ethernet layer', () => {
  it('reads the canonical srcMac/dstMac keys', () => {
    const headers: Headers = { ethernet: { srcMac: 'aa:00', dstMac: 'bb:00' } };
    const eth = layer(buildProtocolLayers(headers, {}), 'Ethernet');

    expect(eth.fields[0]?.value).toBe('aa:00');
    expect(eth.fields[1]?.value).toBe('bb:00');
  });

  it('falls back to the short src/dst keys', () => {
    const eth = layer(buildProtocolLayers({ ethernet: { src: 'a', dst: 'b' } }, {}), 'Ethernet');

    expect(eth.fields[0]?.value).toBe('a');
    expect(eth.fields[1]?.value).toBe('b');
  });

  it('emits EtherType only when it is present', () => {
    expect(fieldNames(buildProtocolLayers({ ethernet: {} }, {}), 'Ethernet')).not.toContain(
      'EtherType',
    );
    expect(
      fieldNames(buildProtocolLayers({ ethernet: { etherType: '0x0800' } }, {}), 'Ethernet'),
    ).toContain('EtherType');
  });

  it('renders a not-parsed placeholder when there is no ethernet header', () => {
    const eth = layer(buildProtocolLayers(undefined, {}), 'Ethernet');

    expect(eth.fields[0]?.value).toBe('(not parsed)');
    // The byte offsets still hold, so the hex pane highlights the right range.
    expect(eth.fields[0]?.byteStart).toBe(6);
    expect(eth.fields[0]?.byteEnd).toBe(12);
  });
});

describe('ip layer', () => {
  it('labels IPv6 distinctly from IPv4', () => {
    const v6 = buildProtocolLayers({ ipv6: { src: '::1', dst: '::2' } }, {});
    expect(layer(v6, 'Internet Protocol').name).toContain('6');

    const v4 = buildProtocolLayers({ ipv4: { src: '10.0.0.1', dst: '10.0.0.2' } }, {});
    expect(layer(v4, 'Internet Protocol').name).toContain('4');
  });

  it('emits TTL and protocol only when present', () => {
    const bare = buildProtocolLayers({ ipv4: { src: '10.0.0.1', dst: '10.0.0.2' } }, {});
    expect(fieldNames(bare, 'Internet Protocol')).toEqual(['Source', 'Destination']);

    const full = buildProtocolLayers(
      { ipv4: { src: '10.0.0.1', dst: '10.0.0.2', ttl: 64, protocol: 6 } },
      {},
    );
    expect(fieldNames(full, 'Internet Protocol')).toEqual([
      'Source',
      'Destination',
      'TTL',
      'Protocol',
    ]);
  });

  it('falls back to the packet metadata when there is no ip header', () => {
    const layers = buildProtocolLayers(undefined, { sourceIp: '10.0.0.1', destIp: '10.0.0.2' });

    expect(layer(layers, 'Internet Protocol').fields[0]?.value).toBe('10.0.0.1');
  });

  it('omits the ip layer entirely when nothing supplies addresses', () => {
    const layers = buildProtocolLayers(undefined, {});

    expect(layers.some((l) => l.name.includes('Internet Protocol'))).toBe(false);
  });
});

describe('transport layer', () => {
  it('builds TCP from the protocol label alone', () => {
    const layers = buildProtocolLayers(undefined, { protocol: 'TCP' });

    expect(layer(layers, 'Transmission Control').fields.length).toBeGreaterThan(0);
  });

  it('emits the optional TCP fields only when present', () => {
    const bare = buildProtocolLayers({ tcp: {} }, {});
    expect(fieldNames(bare, 'Transmission Control')).toEqual(['Source Port', 'Destination Port']);

    const full = buildProtocolLayers({ tcp: { seq: 1, ack: 2, flags: 'SYN', window: 512 } }, {});
    expect(fieldNames(full, 'Transmission Control')).toContain('Sequence Number');
    expect(fieldNames(full, 'Transmission Control')).toContain('Acknowledgment Number');
    expect(fieldNames(full, 'Transmission Control')).toContain('Flags');
    expect(fieldNames(full, 'Transmission Control')).toContain('Window Size');
  });

  it('builds UDP, with length only when reported', () => {
    expect(fieldNames(buildProtocolLayers({ udp: {} }, {}), 'User Datagram')).not.toContain(
      'Length',
    );
    expect(fieldNames(buildProtocolLayers({ udp: { length: 40 } }, {}), 'User Datagram')).toContain(
      'Length',
    );
  });

  it('builds ICMP, with each optional field gated separately', () => {
    expect(fieldNames(buildProtocolLayers({ icmp: {} }, {}), 'Internet Control')).toEqual([]);

    const full = buildProtocolLayers({ icmp: { type: 8, code: 0, id: 1, seq: 2 } }, {});
    expect(fieldNames(full, 'Internet Control')).toEqual([
      'Type',
      'Code',
      'Identifier',
      'Sequence',
    ]);
  });

  it('emits a type of 0 rather than treating it as absent', () => {
    // ICMP type 0 is Echo Reply; a truthiness check would drop it.
    expect(fieldNames(buildProtocolLayers({ icmp: { type: 0 } }, {}), 'Internet Control')).toEqual([
      'Type',
    ]);
  });

  it('omits the transport layer when nothing identifies one', () => {
    const layers = buildProtocolLayers(undefined, {});

    expect(layers.some((l) => l.name.includes('Protocol (TCP)'))).toBe(false);
  });
});

describe('application layers', () => {
  it('adds DNS from the header or the protocol label', () => {
    expect(buildProtocolLayers({ dns: {} }, {}).some((l) => l.name.includes('Domain Name'))).toBe(
      true,
    );
    expect(
      buildProtocolLayers(undefined, { protocol: 'DNS' }).some((l) =>
        l.name.includes('Domain Name'),
      ),
    ).toBe(true);
  });

  it('emits DNS fields only when reported, and renders QR as a word', () => {
    expect(fieldNames(buildProtocolLayers({ dns: {} }, {}), 'Domain Name')).toEqual([]);

    const query = buildProtocolLayers({ dns: { id: 1, qr: false } }, {});
    expect(layer(query, 'Domain Name').fields[1]?.value).toBe('Query');

    const response = buildProtocolLayers({ dns: { qr: true, questions: 1, answers: 2 } }, {});
    expect(layer(response, 'Domain Name').fields[0]?.value).toBe('Response');
  });

  it('adds ARP from the header or the protocol label', () => {
    const layers = buildProtocolLayers(
      { arp: { operation: 1, senderMac: 'aa:00', senderIp: '10.0.0.1' } },
      {},
    );

    expect(fieldNames(layers, 'Address Resolution')).toContain('Opcode');
    expect(fieldNames(layers, 'Address Resolution')).toContain('Sender MAC');
  });

  it('adds no application layer for an unremarkable packet', () => {
    const layers = buildProtocolLayers({ tcp: {} }, { protocol: 'TCP' });

    expect(layers.some((l) => l.name.includes('Domain Name'))).toBe(false);
    expect(layers.some((l) => l.name.includes('Address Resolution'))).toBe(false);
  });
});

describe('computeHeaderBoundary', () => {
  it('clamps to the 14-byte Ethernet minimum', () => {
    expect(computeHeaderBoundary([])).toBe(14);
  });

  it('returns the highest byteEnd across every layer', () => {
    const layers = buildProtocolLayers(
      { ethernet: { srcMac: 'a', dstMac: 'b', etherType: '0x0800' }, ipv4: { src: 'x', dst: 'y' } },
      {},
    );

    // The IPv4 destination ends at byte 34, past the Ethernet header.
    expect(computeHeaderBoundary(layers)).toBe(34);
  });

  it('ignores fields with no byte range', () => {
    expect(computeHeaderBoundary([{ name: 'X', fields: [{ name: 'a', value: 'b' }] }])).toBe(14);
  });
});
