/**
 * extractConversations and the conversation duration helper.
 *
 * conversations.stream.test.ts covers the follow-stream filter (#1497). This
 * covers the aggregation: conversations are keyed canonically so A→B and B→A
 * collapse into one row, and the packet/byte/time totals accumulate across
 * both directions. Getting the key wrong produces two half-counted rows that
 * look entirely plausible.
 */

import { describe, expect, it } from 'vitest';
import { extractConversations, getConversationDuration } from './conversations';

interface TestPacket {
  protocol: string;
  sourceIp: string;
  destIp: string;
  sourcePort?: number;
  destPort?: number;
  timestamp: string;
  length?: number;
  size?: number;
}

const packet = (over: Partial<TestPacket> = {}): TestPacket => ({
  protocol: 'TCP',
  sourceIp: '10.0.0.1',
  destIp: '10.0.0.2',
  sourcePort: 1234,
  destPort: 80,
  timestamp: '2026-08-29T10:00:00.000Z',
  length: 100,
  ...over,
});

/** Narrowing helper: the module takes a broader packet union. */
const extract = (packets: TestPacket[]) => extractConversations(packets as never);

describe('conversation keying', () => {
  it('collapses both directions into a single conversation', () => {
    const convs = extract([
      packet(),
      packet({ sourceIp: '10.0.0.2', destIp: '10.0.0.1', sourcePort: 80, destPort: 1234 }),
    ]);

    expect(convs).toHaveLength(1);
    // Both packets counted, not split across two half-rows.
    expect(convs[0]?.packets).toBe(2);
  });

  it('orders the endpoints canonically regardless of direction', () => {
    const forward = extract([packet()]);
    const reverse = extract([
      packet({ sourceIp: '10.0.0.2', destIp: '10.0.0.1', sourcePort: 80, destPort: 1234 }),
    ]);

    expect(forward[0]?.endpointA).toBe(reverse[0]?.endpointA);
    expect(forward[0]?.endpointB).toBe(reverse[0]?.endpointB);
  });

  it('keeps different ports as separate conversations', () => {
    expect(extract([packet({ sourcePort: 1234 }), packet({ sourcePort: 1235 })])).toHaveLength(2);
  });

  it('keeps TCP and UDP apart on the same endpoints', () => {
    expect(extract([packet({ protocol: 'TCP' }), packet({ protocol: 'UDP' })])).toHaveLength(2);
  });

  it('normalises the protocol case', () => {
    expect(extract([packet({ protocol: 'tcp' })])[0]?.protocol).toBe('TCP');
  });
});

describe('packets that are not conversations', () => {
  it('ignores non-TCP/UDP protocols', () => {
    expect(extract([packet({ protocol: 'ICMP' }), packet({ protocol: 'ARP' })])).toEqual([]);
  });

  it('ignores packets with a missing port', () => {
    expect(extract([packet({ sourcePort: undefined })])).toEqual([]);
    expect(extract([packet({ destPort: undefined })])).toEqual([]);
  });

  it('returns nothing for no packets', () => {
    expect(extract([])).toEqual([]);
  });
});

describe('aggregation', () => {
  it('sums bytes using length, falling back to size', () => {
    const convs = extract([packet({ length: 100 }), packet({ length: undefined, size: 40 })]);

    expect(convs[0]?.bytes).toBe(140);
  });

  it('widens the time window in both directions', () => {
    const convs = extract([
      packet({ timestamp: '2026-08-29T10:00:05.000Z' }),
      packet({ timestamp: '2026-08-29T10:00:01.000Z' }),
      packet({ timestamp: '2026-08-29T10:00:09.000Z' }),
    ]);

    expect(convs[0]?.startTime).toBe('2026-08-29T10:00:01.000Z');
    expect(convs[0]?.endTime).toBe('2026-08-29T10:00:09.000Z');
  });

  it('sorts conversations by packet count, busiest first', () => {
    const convs = extract([
      packet({ sourcePort: 1000 }),
      packet({ sourcePort: 2000 }),
      packet({ sourcePort: 2000 }),
    ]);

    expect(convs[0]?.packets).toBe(2);
    expect(convs[1]?.packets).toBe(1);
  });
});

describe('getConversationDuration', () => {
  it('reports the span in seconds', () => {
    const [conv] = extract([
      packet({ timestamp: '2026-08-29T10:00:00.000Z' }),
      packet({ timestamp: '2026-08-29T10:00:02.500Z' }),
    ]);

    expect(getConversationDuration(conv as never)).toBe(2.5);
  });

  it('reports zero for a single-packet conversation', () => {
    const [conv] = extract([packet()]);

    expect(getConversationDuration(conv as never)).toBe(0);
  });

  it('never reports a negative duration', () => {
    expect(
      getConversationDuration({
        startTime: '2026-08-29T10:00:05.000Z',
        endTime: '2026-08-29T10:00:00.000Z',
      } as never),
    ).toBe(0);
  });
});
