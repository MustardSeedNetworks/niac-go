/**
 * protocol-layers.test.ts — header/payload boundary derivation.
 *
 * Regression coverage for a bug where the hex dump viewer always used a
 * hardcoded Ethernet-only header length (14 bytes), so IP/TCP/UDP header
 * bytes rendered as "Payload". computeHeaderBoundary must derive the real
 * boundary from the byte ranges already computed by buildProtocolLayers.
 */
import { describe, expect, it } from 'vitest';
import { buildProtocolLayers, computeHeaderBoundary } from './protocol-layers';

describe('computeHeaderBoundary', () => {
  it('falls back to the Ethernet-only default when no layer has byte metadata', () => {
    const layers = buildProtocolLayers(undefined, { protocol: 'Unknown' });
    expect(computeHeaderBoundary(layers)).toBe(14);
  });

  it('extends the boundary through the IP header when no transport layer is present', () => {
    const layers = buildProtocolLayers(undefined, {
      protocol: 'Unknown',
      sourceIp: '10.0.0.1',
      destIp: '10.0.0.2',
    });
    // Unparsed IP layer's Destination field is the last byte-annotated
    // field, ending at byte 34.
    expect(computeHeaderBoundary(layers)).toBe(34);
  });

  it('extends the boundary through a parsed TCP header', () => {
    const headers = {
      ipv4: { src: '10.0.0.1', dst: '10.0.0.2', ttl: 64 },
      tcp: { srcPort: 1234, dstPort: 80, seq: 1, ack: 2, flags: 'SYN', window: 65535 },
    };
    const layers = buildProtocolLayers(headers, {
      protocol: 'TCP',
      sourceIp: '10.0.0.1',
      destIp: '10.0.0.2',
      sourcePort: 1234,
      destPort: 80,
    });
    // TCP flags field is the last byte-annotated field: s=34, byteEnd = 34+14 = 48.
    expect(computeHeaderBoundary(layers)).toBe(48);
  });

  it('extends the boundary through a parsed UDP header', () => {
    const headers = {
      ipv4: { src: '10.0.0.1', dst: '10.0.0.2' },
      udp: { srcPort: 53, dstPort: 5353, length: 40 },
    };
    const layers = buildProtocolLayers(headers, {
      protocol: 'UDP',
      sourceIp: '10.0.0.1',
      destIp: '10.0.0.2',
      sourcePort: 53,
      destPort: 5353,
    });
    // UDP length field ends at s+6 = 40.
    expect(computeHeaderBoundary(layers)).toBe(40);
  });

  it('does not extend the boundary for application-layer protocols like DNS', () => {
    const headers = {
      ipv4: { src: '10.0.0.1', dst: '10.0.0.2' },
      udp: { srcPort: 53, dstPort: 5353 },
      dns: { id: 1, qr: true, questions: 1, answers: 1 },
    };
    const layers = buildProtocolLayers(headers, {
      protocol: 'DNS',
      sourceIp: '10.0.0.1',
      destIp: '10.0.0.2',
      sourcePort: 53,
      destPort: 5353,
    });
    // DNS fields carry no byte ranges, so the boundary stays at the end of
    // the UDP header (s+4 = 38).
    expect(computeHeaderBoundary(layers)).toBe(38);
  });
});
