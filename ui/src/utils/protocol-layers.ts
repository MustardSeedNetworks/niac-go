import { applyByteRanges, type PacketByteRange } from './protocol-byte-ranges';
import { buildApplicationLayers, buildTransportLayer } from './protocol-layer-payload';

export interface ProtocolField {
  name: string;
  value: string;
  byteStart?: number;
  byteEnd?: number;
}

export interface ProtocolLayer {
  name: string;
  fields: ProtocolField[];
  expanded?: boolean;
  byteEnd?: number;
}

export interface PacketMeta {
  byteRanges?: PacketByteRange[];
  timestamp?: string;
  protocol?: string;
  sourceIp?: string;
  destIp?: string;
  sourcePort?: number;
  destPort?: number;
  length?: number;
  size?: number;
  number?: number;
}

/**
 * Build structured protocol layers from a packet's headers map and metadata.
 */
export function buildProtocolLayers(
  headers: Record<string, unknown> | undefined,
  packet: PacketMeta,
): ProtocolLayer[] {
  const layers: ProtocolLayer[] = [];

  layers.push(buildFrameLayer(packet));
  layers.push(buildEthernetLayer(headers));
  const vlan = headers?.dot1q as Record<string, unknown> | undefined;
  if (vlan)
    layers.push({
      name: '802.1Q',
      fields: [
        { name: 'VLAN', value: String(vlan.vlanId) },
        { name: 'Priority', value: String(vlan.priority) },
      ],
    });
  const ipLayer = buildIpLayer(headers, packet);
  if (ipLayer) layers.push(ipLayer);
  const transportLayer = buildTransportLayer(headers, packet);
  if (transportLayer) layers.push(transportLayer);
  layers.push(...buildApplicationLayers(headers, packet.protocol));
  applyByteRanges(layers, packet.byteRanges);

  return layers;
}

/**
 * Derive the header/payload byte boundary from a packet's protocol layers.
 *
 * Complete decoded layer ranges include options and extension headers. Without
 * decoder metadata, no bytes are claimed as parsed headers.
 */
export function computeHeaderBoundary(layers: ProtocolLayer[]): number {
  let boundary = 0;
  for (const layer of layers) {
    boundary = Math.max(boundary, layer.byteEnd ?? 0);
    for (const field of layer.fields) {
      if (field.byteEnd !== undefined && field.byteEnd > boundary) {
        boundary = field.byteEnd;
      }
    }
  }
  return boundary;
}

function buildFrameLayer(packet: PacketMeta): ProtocolLayer {
  const totalLength = packet.length ?? packet.size ?? 0;
  return {
    name: 'Frame',
    fields: [
      ...(packet.number !== undefined ? [{ name: 'Number', value: String(packet.number) }] : []),
      { name: 'Length', value: `${totalLength} bytes` },
      ...(packet.timestamp ? [{ name: 'Arrival Time', value: packet.timestamp }] : []),
    ],
  };
}

function buildEthernetLayer(headers: Record<string, unknown> | undefined): ProtocolLayer {
  const eth = headers?.ethernet as Record<string, unknown> | undefined;
  if (eth) {
    return {
      name: 'Ethernet II',
      fields: [
        {
          name: 'Source MAC',
          value: String(eth.srcMac ?? eth.src ?? ''),
        },
        {
          name: 'Destination MAC',
          value: String(eth.dstMac ?? eth.dst ?? ''),
        },
        ...(eth.etherType ? [{ name: 'EtherType', value: String(eth.etherType) }] : []),
      ],
    };
  }
  return {
    name: 'Ethernet II',
    fields: [
      { name: 'Source MAC', value: '(not parsed)' },
      { name: 'Destination MAC', value: '(not parsed)' },
    ],
  };
}

function buildIpLayer(
  headers: Record<string, unknown> | undefined,
  packet: PacketMeta,
): ProtocolLayer | null {
  const ipv4 = headers?.ipv4 as Record<string, unknown> | undefined;
  const ipv6 = headers?.ipv6 as Record<string, unknown> | undefined;
  const ip = ipv4 ?? ipv6 ?? (headers?.ip as Record<string, unknown> | undefined);

  // Only render an IP layer when the packet actually has one. This used to fall
  // back to packet.sourceIp, which PacketInspectorPage defaults to the literal
  // "Unknown" — so an STP BPDU, which has no IP layer at all, was shown as
  // "Internet Protocol Version 4 (IPv4) / Source: Unknown" (D16).
  const hasAddresses = Boolean(ip?.src ?? ip?.dst);
  if (ip && hasAddresses) {
    const version = ipv6 ? 'IPv6' : 'IPv4';
    return {
      name: `Internet Protocol Version ${ipv6 ? '6' : '4'} (${version})`,
      fields: [
        {
          name: 'Source',
          value: String(ip.src ?? packet.sourceIp ?? ''),
        },
        {
          name: 'Destination',
          value: String(ip.dst ?? packet.destIp ?? ''),
        },
        ...(ip.ttl !== undefined ? [{ name: 'TTL', value: String(ip.ttl) }] : []),
        ...(ip.hopLimit !== undefined ? [{ name: 'Hop Limit', value: String(ip.hopLimit) }] : []),
        ...(ip.nextHeader !== undefined
          ? [{ name: 'Next Header', value: String(ip.nextHeader) }]
          : []),
        ...(ip.protocol !== undefined ? [{ name: 'Protocol', value: String(ip.protocol) }] : []),
      ],
    };
  }

  if (packet.sourceIp && packet.destIp) {
    return {
      name: 'Internet Protocol Version 4 (IPv4)',
      fields: [
        { name: 'Source', value: packet.sourceIp },
        { name: 'Destination', value: packet.destIp },
      ],
    };
  }

  return null;
}
