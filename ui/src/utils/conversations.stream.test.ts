import { describe, expect, it } from 'vitest';
import { canFollowStream, getStreamFilter } from './conversations';

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

/**
 * canFollowStream existed twice — once in PacketInspectorPage and once in
 * PcapAnalyzerPage — which is how the first fix for #1494 landed on the live
 * view and left the PCAP view untouched. These pin the shared definition both
 * pages now call.
 */
describe('canFollowStream', () => {
  it('accepts an HTTP conversation carried over TCP', () => {
    expect(canFollowStream(httpOverTcp)).toBe(true);
  });

  it('rejects a packet with no selection', () => {
    expect(canFollowStream(null)).toBe(false);
  });

  it('rejects a frame with no transport ports', () => {
    expect(canFollowStream({ protocol: 'STP', sourceIp: '', destIp: '' } as never)).toBe(false);
  });
});

/**
 * Guards #1497.
 *
 * buildFilterExpression compared the same field to two values inside one
 * conjunction — `tcp.port == 50000 && tcp.port == 80` — which no packet can
 * satisfy, so every conversation filter in the app matched nothing. On CT304,
 * following a conversation in a capture that contained it reported
 * "Showing 0 of 6 packets".
 *
 * The expression must pair each address with its directional port. Asserted
 * through the evaluator rather than by string shape, so it pins that the filter
 * actually selects the conversation.
 */
describe('conversation filter expressions select their own packets', () => {
  const packets = [
    {
      protocol: 'HTTP',
      sourceIp: '10.99.1.10',
      destIp: '10.99.1.20',
      sourcePort: 50000,
      destPort: 80,
      headers: { tcp: {} },
    },
    {
      protocol: 'HTTP',
      sourceIp: '10.99.1.20',
      destIp: '10.99.1.10',
      sourcePort: 80,
      destPort: 50000,
      headers: { tcp: {} },
    },
    {
      protocol: 'HTTP',
      sourceIp: '10.99.9.9',
      destIp: '10.99.1.20',
      sourcePort: 40000,
      destPort: 80,
      headers: { tcp: {} },
    },
  ] as never[];

  it('matches both directions of the conversation and nothing else', async () => {
    const { parse } = await import('./filter/parser');
    const { evaluate } = await import('./filter/evaluator');

    const first = packets[0] as never;
    const filter = getStreamFilter(first);
    expect(filter).not.toBeNull();

    const ast = parse(filter as string);
    const matched = packets.filter((p) => evaluate(ast, p));
    expect(matched).toHaveLength(2);
    expect(matched).toEqual([packets[0], packets[1]]);
  });
});
