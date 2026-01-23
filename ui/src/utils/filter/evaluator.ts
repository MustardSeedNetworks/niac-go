import type { PcapPacket } from '../../api/types';
import type { Packet } from '../../components/PacketList';
import type { ASTNode } from './types';
import { BARE_PROTOCOLS } from './types';

type PacketLike = Packet | PcapPacket;

/**
 * Evaluate a parsed filter AST against a packet.
 * Returns true if the packet matches the filter expression.
 */
export function evaluate(node: ASTNode, packet: PacketLike): boolean {
  switch (node.type) {
    case 'Comparison':
      return evaluateComparison(node.field, node.operator, node.value, packet);

    case 'FieldExists':
      return evaluateFieldExists(node.field, packet);

    case 'Logical':
      if (node.operator === '&&') {
        return evaluate(node.left, packet) && evaluate(node.right, packet);
      }
      return evaluate(node.left, packet) || evaluate(node.right, packet);

    case 'Not':
      return !evaluate(node.operand, packet);

    case 'Group':
      return evaluate(node.expression, packet);

    case 'FieldAccess':
      return evaluateFieldExists(node.field, packet);

    case 'Literal':
      return Boolean(node.value);
  }
}

/**
 * Resolve a field name to the actual packet value.
 */
function resolveField(field: string, packet: PacketLike): string | number | boolean | null {
  const protocol = getProtocol(packet);
  const headers = getHeaders(packet);

  switch (field) {
    case 'ip.src':
      return getSourceIp(packet);
    case 'ip.dst':
      return getDestIp(packet);
    case 'tcp.port':
    case 'udp.port': {
      // Returns source or dest port for matching purposes
      const src = getSourcePort(packet);
      const dst = getDestPort(packet);
      return src ?? dst ?? null;
    }
    case 'tcp.srcport':
    case 'udp.srcport':
      return getSourcePort(packet) ?? null;
    case 'tcp.dstport':
    case 'udp.dstport':
      return getDestPort(packet) ?? null;
    case 'tcp.flags.syn':
      return hasTcpFlag(headers, 'SYN');
    case 'tcp.flags.ack':
      return hasTcpFlag(headers, 'ACK');
    case 'tcp.flags.fin':
      return hasTcpFlag(headers, 'FIN');
    case 'tcp.flags.rst':
      return hasTcpFlag(headers, 'RST');
    case 'tcp.flags.psh':
      return hasTcpFlag(headers, 'PSH');
    case 'eth.src':
      return getEthField(headers, 'src');
    case 'eth.dst':
      return getEthField(headers, 'dst');
    case 'protocol':
      return protocol;
    case 'frame.len':
      return getLength(packet);
    default:
      // Try to resolve from headers map
      return resolveFromHeaders(field, headers);
  }
}

function evaluateComparison(
  field: string,
  operator: string,
  value: string | number,
  packet: PacketLike,
): boolean {
  // Special case: tcp.port/udp.port should check both source and dest
  if (field === 'tcp.port' || field === 'udp.port') {
    const proto = field.split('.')[0].toUpperCase();
    if (getProtocol(packet).toUpperCase() !== proto) {
      return false;
    }
    const srcPort = getSourcePort(packet);
    const dstPort = getDestPort(packet);
    const matchSrc = srcPort !== undefined && compareValues(srcPort, operator, value);
    const matchDst = dstPort !== undefined && compareValues(dstPort, operator, value);
    return matchSrc || matchDst;
  }

  const fieldValue = resolveField(field, packet);
  if (fieldValue === null || fieldValue === undefined) {
    return false;
  }

  return compareValues(fieldValue, operator, value);
}

function compareValues(
  fieldValue: string | number | boolean,
  operator: string,
  value: string | number,
): boolean {
  switch (operator) {
    case '==':
      return String(fieldValue).toLowerCase() === String(value).toLowerCase();
    case '!=':
      return String(fieldValue).toLowerCase() !== String(value).toLowerCase();
    case 'contains':
      return String(fieldValue).toLowerCase().includes(String(value).toLowerCase());
    case '>':
      return Number(fieldValue) > Number(value);
    case '<':
      return Number(fieldValue) < Number(value);
    case '>=':
      return Number(fieldValue) >= Number(value);
    case '<=':
      return Number(fieldValue) <= Number(value);
    default:
      return false;
  }
}

function evaluateFieldExists(field: string, packet: PacketLike): boolean {
  // Bare protocol names check if the packet is that protocol
  if (BARE_PROTOCOLS.includes(field.toLowerCase() as (typeof BARE_PROTOCOLS)[number])) {
    return getProtocol(packet).toLowerCase() === field.toLowerCase();
  }

  // For dotted fields, check if the field resolves to a non-null value
  const value = resolveField(field, packet);
  if (typeof value === 'boolean') {
    return value;
  }
  return value !== null && value !== undefined && value !== '';
}

// -- Packet field accessors (handle both Packet and PcapPacket types) --

function getProtocol(packet: PacketLike): string {
  return packet.protocol ?? '';
}

function getSourceIp(packet: PacketLike): string {
  return packet.sourceIp ?? '';
}

function getDestIp(packet: PacketLike): string {
  return packet.destIp ?? '';
}

function getSourcePort(packet: PacketLike): number | undefined {
  return packet.sourcePort;
}

function getDestPort(packet: PacketLike): number | undefined {
  return packet.destPort;
}

function getHeaders(packet: PacketLike): Record<string, unknown> {
  return (packet.headers as Record<string, unknown>) ?? {};
}

function getLength(packet: PacketLike): number {
  if ('length' in packet && typeof packet.length === 'number') {
    return packet.length;
  }
  if ('size' in packet && typeof packet.size === 'number') {
    return packet.size;
  }
  return 0;
}

function hasTcpFlag(headers: Record<string, unknown>, flag: string): boolean {
  const tcp = headers.tcp as Record<string, unknown> | undefined;
  if (!tcp) return false;

  const flags = tcp.flags;
  if (typeof flags === 'string') {
    return flags.toUpperCase().includes(flag.toUpperCase());
  }
  if (Array.isArray(flags)) {
    return flags.some((f) => String(f).toUpperCase() === flag.toUpperCase());
  }
  return false;
}

function getEthField(headers: Record<string, unknown>, field: 'src' | 'dst'): string | null {
  const eth = headers.ethernet as Record<string, unknown> | undefined;
  if (!eth) return null;

  const key = field === 'src' ? 'srcMac' : 'dstMac';
  const altKey = field === 'src' ? 'src' : 'dst';
  const value = eth[key] ?? eth[altKey];
  return typeof value === 'string' ? value : null;
}

function resolveFromHeaders(field: string, headers: Record<string, unknown>): string | null {
  const parts = field.split('.');
  let current: unknown = headers;

  for (const part of parts) {
    if (current === null || current === undefined || typeof current !== 'object') {
      return null;
    }
    current = (current as Record<string, unknown>)[part];
  }

  if (current === null || current === undefined) {
    return null;
  }
  return String(current);
}
