import { describe, expect, it } from 'vitest';
import { getStreamFilter } from './conversations';

/**
 * Guards #1494.
 *
 * The analyzer sets `protocol` to the *application* protocol — HTTP, DNS — so
 * the transport is not in that field. getStreamFilter matched it exactly
 * against 'TCP'/'UDP', which meant Follow Stream could never enable for the
 * conversations anyone actually wants to follow. Driven on CT304: every packet
 * of a six-packet HTTP capture left the button disabled.
 *
 * The transport is not lost, only unused — the analyzer emits headers.tcp and
 * headers.udp alongside the application label.
 */
const httpOverTcp = {
  protocol: 'HTTP',
  sourceIp: '10.99.1.10',
  destIp: '10.99.1.20',
  sourcePort: 50000,
  destPort: 80,
  headers: { tcp: { srcPort: 50000, dstPort: 80 } },
} as never;

describe('getStreamFilter', () => {
  it('follows an HTTP conversation using its TCP transport', () => {
    const filter = getStreamFilter(httpOverTcp);
    expect(filter).not.toBeNull();
    expect(filter).toContain('tcp');
    expect(filter).toContain('10.99.1.10');
    expect(filter).toContain('10.99.1.20');
  });

  it('follows a DNS conversation using its UDP transport', () => {
    const filter = getStreamFilter({
      protocol: 'DNS',
      sourceIp: '10.0.0.5',
      destIp: '10.0.0.53',
      sourcePort: 40000,
      destPort: 53,
      headers: { udp: { srcPort: 40000, dstPort: 53 } },
    } as never);
    expect(filter).not.toBeNull();
    expect(filter).toContain('udp');
  });

  it('still follows a packet labelled plainly as TCP', () => {
    const filter = getStreamFilter({
      protocol: 'TCP',
      sourceIp: '10.0.0.1',
      destIp: '10.0.0.2',
      sourcePort: 1234,
      destPort: 80,
    } as never);
    expect(filter).toContain('tcp');
  });

  it('refuses a packet with no transport ports', () => {
    expect(getStreamFilter({ protocol: 'STP', sourceIp: '', destIp: '' } as never)).toBeNull();
  });

  it('refuses ICMP, which has no stream to follow', () => {
    expect(
      getStreamFilter({
        protocol: 'ICMP',
        sourceIp: '10.0.0.1',
        destIp: '10.0.0.2',
        headers: { icmp: {} },
      } as never),
    ).toBeNull();
  });
});
