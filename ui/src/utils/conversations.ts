import type { PcapPacket } from '../api/types';
import type { Packet } from '../components/PacketList';

/**
 * A conversation is a bidirectional flow between two endpoints (IP:port pairs)
 * over a specific protocol (TCP or UDP).
 */
export interface Conversation {
  id: string;
  endpointA: string; // ip:port
  endpointB: string; // ip:port
  protocol: string;
  packets: number;
  bytes: number;
  startTime: string;
  endTime: string;
  /** Display filter expression to show only this conversation's packets */
  filterExpression: string;
}

type PacketLike = Packet | PcapPacket;

function getPacketSize(packet: PacketLike): number {
  if ('size' in packet) return packet.size;
  if ('length' in packet) return packet.length;
  return 0;
}

/**
 * Generate a canonical conversation key from a packet.
 * The key is sorted so that A:portA <-> B:portB and B:portB <-> A:portA
 * produce the same key.
 */
function getConversationKey(packet: PacketLike): string | null {
  const proto = packet.protocol.toUpperCase();
  if (proto !== 'TCP' && proto !== 'UDP') {
    return null;
  }

  if (!packet.sourcePort || !packet.destPort) {
    return null;
  }

  const epA = `${packet.sourceIp}:${packet.sourcePort}`;
  const epB = `${packet.destIp}:${packet.destPort}`;

  // Sort endpoints for canonical key
  const [first, second] = epA < epB ? [epA, epB] : [epB, epA];
  return `${proto}|${first}|${second}`;
}

/**
 * Build a display filter expression for a specific conversation.
 */
function buildFilterExpression(protocol: string, endpointA: string, endpointB: string): string {
  const [ipA, portA] = splitEndpoint(endpointA);
  const [ipB, portB] = splitEndpoint(endpointB);
  const proto = protocol.toLowerCase();

  return (
    `${proto} && ` +
    `((ip.src == ${ipA} && ${proto}.port == ${portA} && ip.dst == ${ipB} && ${proto}.port == ${portB}) || ` +
    `(ip.src == ${ipB} && ${proto}.port == ${portB} && ip.dst == ${ipA} && ${proto}.port == ${portA}))`
  );
}

function splitEndpoint(endpoint: string): [string, string] {
  const lastColon = endpoint.lastIndexOf(':');
  return [endpoint.substring(0, lastColon), endpoint.substring(lastColon + 1)];
}

/**
 * Extract all TCP/UDP conversations from a list of packets.
 * Returns conversations sorted by packet count (descending).
 */
export function extractConversations(packets: PacketLike[]): Conversation[] {
  const convMap = new Map<
    string,
    {
      endpointA: string;
      endpointB: string;
      protocol: string;
      packets: number;
      bytes: number;
      startTime: string;
      endTime: string;
    }
  >();

  for (const packet of packets) {
    const key = getConversationKey(packet);
    if (!key) continue;

    const proto = packet.protocol.toUpperCase();
    const epA = `${packet.sourceIp}:${packet.sourcePort}`;
    const epB = `${packet.destIp}:${packet.destPort}`;
    const [first, second] = epA < epB ? [epA, epB] : [epB, epA];

    const existing = convMap.get(key);
    if (existing) {
      existing.packets += 1;
      existing.bytes += getPacketSize(packet);
      if (packet.timestamp < existing.startTime) {
        existing.startTime = packet.timestamp;
      }
      if (packet.timestamp > existing.endTime) {
        existing.endTime = packet.timestamp;
      }
    } else {
      convMap.set(key, {
        endpointA: first,
        endpointB: second,
        protocol: proto,
        packets: 1,
        bytes: getPacketSize(packet),
        startTime: packet.timestamp,
        endTime: packet.timestamp,
      });
    }
  }

  const conversations: Conversation[] = [];
  for (const [id, conv] of convMap) {
    conversations.push({
      id,
      endpointA: conv.endpointA,
      endpointB: conv.endpointB,
      protocol: conv.protocol,
      packets: conv.packets,
      bytes: conv.bytes,
      startTime: conv.startTime,
      endTime: conv.endTime,
      filterExpression: buildFilterExpression(conv.protocol, conv.endpointA, conv.endpointB),
    });
  }

  // Sort by packet count descending
  conversations.sort((a, b) => b.packets - a.packets);
  return conversations;
}

/**
 * The transport a packet's stream rides on, or null when it has none.
 *
 * `protocol` carries the *application* protocol once one is recognised — HTTP,
 * DNS — so matching it against 'TCP'/'UDP' left Follow Stream permanently
 * disabled for exactly the conversations worth following (#1494). The transport
 * is still available in the headers the analyzer emits; fall back to the label
 * only when they are absent.
 */
export function streamTransport(packet: PacketLike): 'TCP' | 'UDP' | null {
  const headers = packet.headers as Record<string, unknown> | undefined;
  if (headers?.tcp) {
    return 'TCP';
  }
  if (headers?.udp) {
    return 'UDP';
  }
  const proto = packet.protocol.toUpperCase();
  return proto === 'TCP' || proto === 'UDP' ? proto : null;
}

/**
 * Get the conversation filter expression for a specific packet.
 * Returns null if the packet doesn't belong to a TCP/UDP conversation.
 */
export function getStreamFilter(packet: PacketLike): string | null {
  const proto = streamTransport(packet);
  if (!proto) {
    return null;
  }
  if (!packet.sourcePort || !packet.destPort) {
    return null;
  }

  const epA = `${packet.sourceIp}:${packet.sourcePort}`;
  const epB = `${packet.destIp}:${packet.destPort}`;
  const [first, second] = epA < epB ? [epA, epB] : [epB, epA];

  return buildFilterExpression(proto, first, second);
}

/**
 * Calculate conversation duration in seconds.
 */
export function getConversationDuration(conv: Conversation): number {
  const start = new Date(conv.startTime).getTime();
  const end = new Date(conv.endTime).getTime();
  return Math.max(0, (end - start) / 1000);
}
