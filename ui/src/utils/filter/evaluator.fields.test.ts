/**
 * Field resolution, comparison and boolean-structure coverage for the display
 * filter evaluator.
 *
 * evaluator.test.ts pins the bare-protocol behaviour from #1497; this covers
 * the rest of the surface — every operator, every resolvable field, and the
 * cases where a field is absent, since "no such field" and "field is false"
 * must not collapse into the same answer.
 */

import { describe, expect, it } from 'vitest';
import { evaluate } from './evaluator';
import { parse } from './parser';

const matches = (expr: string, packet: unknown): boolean => evaluate(parse(expr), packet as never);

const tcpPacket = {
  protocol: 'TCP',
  sourceIp: '10.0.0.1',
  destIp: '10.0.0.2',
  sourcePort: 1234,
  destPort: 80,
  length: 512,
  headers: {
    tcp: { flags: 'SYN,ACK' },
    ethernet: { srcMac: 'aa:bb:cc:dd:ee:01', dstMac: 'aa:bb:cc:dd:ee:02' },
  },
};

describe('address and protocol fields', () => {
  it('matches on source and destination IP', () => {
    expect(matches('ip.src == 10.0.0.1', tcpPacket)).toBe(true);
    expect(matches('ip.dst == 10.0.0.2', tcpPacket)).toBe(true);
    expect(matches('ip.src == 10.0.0.9', tcpPacket)).toBe(false);
  });

  it('matches on the protocol label case-insensitively', () => {
    expect(matches('protocol == tcp', tcpPacket)).toBe(true);
    expect(matches('protocol == TCP', tcpPacket)).toBe(true);
  });

  it('treats a packet with no addresses as non-matching rather than throwing', () => {
    // getSourceIp falls back to '', which evaluateFieldExists reads as absent.
    expect(matches('ip.src == 10.0.0.1', { protocol: 'TCP' })).toBe(false);
    expect(matches('ip.src', { protocol: 'TCP' })).toBe(false);
  });
});

describe('port fields', () => {
  it('tcp.port matches either endpoint', () => {
    expect(matches('tcp.port == 1234', tcpPacket)).toBe(true);
    expect(matches('tcp.port == 80', tcpPacket)).toBe(true);
    expect(matches('tcp.port == 443', tcpPacket)).toBe(false);
  });

  it('tcp.port does not match a packet of another transport', () => {
    // The transport guard is what stops `udp.port == 80` matching TCP:80.
    expect(matches('udp.port == 80', tcpPacket)).toBe(false);
  });

  it('udp.port matches a UDP packet on either endpoint', () => {
    const udp = { protocol: 'UDP', sourcePort: 53, destPort: 40000 };
    expect(matches('udp.port == 53', udp)).toBe(true);
    expect(matches('udp.port == 40000', udp)).toBe(true);
  });

  it('distinguishes source from destination ports', () => {
    expect(matches('tcp.srcport == 1234', tcpPacket)).toBe(true);
    expect(matches('tcp.dstport == 1234', tcpPacket)).toBe(false);
    expect(matches('tcp.dstport == 80', tcpPacket)).toBe(true);
  });

  it('does not match a port comparison when the packet has no ports', () => {
    expect(matches('tcp.port == 80', { protocol: 'TCP' })).toBe(false);
    expect(matches('tcp.srcport == 80', { protocol: 'TCP' })).toBe(false);
  });
});

describe('tcp flags', () => {
  it('reads flags from a comma-separated string', () => {
    expect(matches('tcp.flags.syn', tcpPacket)).toBe(true);
    expect(matches('tcp.flags.ack', tcpPacket)).toBe(true);
    expect(matches('tcp.flags.fin', tcpPacket)).toBe(false);
    expect(matches('tcp.flags.rst', tcpPacket)).toBe(false);
    expect(matches('tcp.flags.psh', tcpPacket)).toBe(false);
  });

  it('reads flags from an array', () => {
    const p = { protocol: 'TCP', headers: { tcp: { flags: ['FIN', 'ACK'] } } };
    expect(matches('tcp.flags.fin', p)).toBe(true);
    expect(matches('tcp.flags.syn', p)).toBe(false);
  });

  it('reports no flags when tcp headers are missing or flags are an unexpected shape', () => {
    expect(matches('tcp.flags.syn', { protocol: 'TCP', headers: {} })).toBe(false);
    expect(matches('tcp.flags.syn', { protocol: 'TCP', headers: { tcp: { flags: 42 } } })).toBe(
      false,
    );
  });
});

describe('ethernet fields', () => {
  it('reads the canonical srcMac/dstMac keys', () => {
    expect(matches('eth.src == aa:bb:cc:dd:ee:01', tcpPacket)).toBe(true);
    expect(matches('eth.dst == aa:bb:cc:dd:ee:02', tcpPacket)).toBe(true);
  });

  it('falls back to the short src/dst keys', () => {
    const p = { protocol: 'TCP', headers: { ethernet: { src: 'a', dst: 'b' } } };
    expect(matches('eth.src == a', p)).toBe(true);
    expect(matches('eth.dst == b', p)).toBe(true);
  });

  it('resolves to nothing when ethernet headers are absent or non-string', () => {
    expect(matches('eth.src == a', { protocol: 'TCP', headers: {} })).toBe(false);
    expect(matches('eth.src == 1', { protocol: 'TCP', headers: { ethernet: { srcMac: 1 } } })).toBe(
      false,
    );
  });
});

describe('frame.len', () => {
  it('reads `length` when present', () => {
    expect(matches('frame.len == 512', tcpPacket)).toBe(true);
  });

  it('reads `size` for packet shapes that use it instead', () => {
    expect(matches('frame.len == 64', { protocol: 'TCP', size: 64 })).toBe(true);
  });

  it('reports zero when neither is present', () => {
    expect(matches('frame.len == 0', { protocol: 'TCP' })).toBe(true);
  });
});

describe('comparison operators', () => {
  it('supports equality and inequality, case-insensitively', () => {
    expect(matches('ip.src == 10.0.0.1', tcpPacket)).toBe(true);
    expect(matches('ip.src != 10.0.0.1', tcpPacket)).toBe(false);
    expect(matches('ip.src != 10.0.0.9', tcpPacket)).toBe(true);
  });

  it('supports contains', () => {
    expect(matches('ip.src contains 10.0', tcpPacket)).toBe(true);
    expect(matches('ip.src contains 192', tcpPacket)).toBe(false);
  });

  it('supports the numeric comparisons', () => {
    expect(matches('frame.len > 100', tcpPacket)).toBe(true);
    expect(matches('frame.len > 512', tcpPacket)).toBe(false);
    expect(matches('frame.len < 1000', tcpPacket)).toBe(true);
    expect(matches('frame.len >= 512', tcpPacket)).toBe(true);
    expect(matches('frame.len <= 512', tcpPacket)).toBe(true);
    expect(matches('frame.len <= 100', tcpPacket)).toBe(false);
  });
});

describe('boolean structure', () => {
  it('evaluates AND', () => {
    expect(matches('tcp && ip.src == 10.0.0.1', tcpPacket)).toBe(true);
    expect(matches('tcp && ip.src == 10.0.0.9', tcpPacket)).toBe(false);
  });

  it('evaluates OR', () => {
    expect(matches('ip.src == 10.0.0.9 || ip.dst == 10.0.0.2', tcpPacket)).toBe(true);
    expect(matches('ip.src == 10.0.0.9 || ip.dst == 10.0.0.9', tcpPacket)).toBe(false);
  });

  it('evaluates NOT', () => {
    expect(matches('!(ip.src == 10.0.0.9)', tcpPacket)).toBe(true);
    expect(matches('!(ip.src == 10.0.0.1)', tcpPacket)).toBe(false);
  });

  it('honours grouping over precedence', () => {
    // Without the parentheses the && would bind first.
    expect(
      matches('(ip.src == 10.0.0.9 || ip.dst == 10.0.0.2) && frame.len == 512', tcpPacket),
    ).toBe(true);
  });
});

describe('generic header traversal', () => {
  it('resolves a dotted path into the headers map', () => {
    const p = { protocol: 'DNS', headers: { dns: { qname: 'example.com' } } };
    expect(matches('dns.qname == example.com', p)).toBe(true);
    expect(matches('dns.qname contains example', p)).toBe(true);
  });

  it('does not match when the path runs off the end of the object', () => {
    const p = { protocol: 'DNS', headers: { dns: { qname: 'example.com' } } };
    expect(matches('dns.qname.extra', p)).toBe(false);
    expect(matches('dns.missing', p)).toBe(false);
    expect(matches('nothing.here', p)).toBe(false);
  });

  it('treats an existing nested field as present', () => {
    const p = { protocol: 'DNS', headers: { dns: { qname: 'example.com' } } };
    expect(matches('dns.qname', p)).toBe(true);
  });

  it('treats an empty-string field as absent', () => {
    const p = { protocol: 'DNS', headers: { dns: { qname: '' } } };
    expect(matches('dns.qname', p)).toBe(false);
  });
});
