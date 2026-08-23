import { describe, expect, it } from 'vitest';
import { evaluate } from './evaluator';
import { parse } from './parser';

/**
 * Guards the bare-protocol half of #1497.
 *
 * `protocol` carries the application protocol once one is recognised, so an
 * HTTP packet is labelled HTTP even though it is carried over TCP. A bare
 * `tcp` atom matched only the label, so it missed every HTTP or DNS packet —
 * and the generated conversation filters, which all begin with `tcp &&`,
 * matched nothing at all.
 *
 * Wireshark's `tcp` matches HTTP traffic; this makes ours agree, using the
 * transport headers the analyzer already emits.
 */
const matches = (expr: string, packet: unknown): boolean => evaluate(parse(expr), packet as never);

const httpOverTcp = {
  protocol: 'HTTP',
  sourceIp: '10.99.1.10',
  destIp: '10.99.1.20',
  sourcePort: 50000,
  destPort: 80,
  headers: { tcp: { srcPort: 50000, dstPort: 80 } },
};

describe('bare protocol atoms', () => {
  it('matches an HTTP packet with tcp, because it is carried over TCP', () => {
    expect(matches('tcp', httpOverTcp)).toBe(true);
  });

  it('still matches on the label when there are no transport headers', () => {
    expect(matches('cdp', { protocol: 'CDP' })).toBe(true);
    expect(matches('stp', { protocol: 'STP' })).toBe(true);
  });

  it('does not match a transport the packet does not carry', () => {
    expect(matches('udp', httpOverTcp)).toBe(false);
    expect(matches('tcp', { protocol: 'STP' })).toBe(false);
  });

  it('still matches the application label itself', () => {
    expect(matches('http', httpOverTcp)).toBe(true);
  });
});
