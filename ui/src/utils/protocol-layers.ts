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
}

interface PacketMeta {
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
  const ipLayer = buildIpLayer(headers, packet);
  if (ipLayer) layers.push(ipLayer);
  const transportLayer = buildTransportLayer(headers, packet);
  if (transportLayer) layers.push(transportLayer);
  layers.push(...buildApplicationLayers(headers, packet.protocol));

  return layers;
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
          byteStart: 6,
          byteEnd: 12,
        },
        {
          name: 'Destination MAC',
          value: String(eth.dstMac ?? eth.dst ?? ''),
          byteStart: 0,
          byteEnd: 6,
        },
        ...(eth.etherType
          ? [{ name: 'EtherType', value: String(eth.etherType), byteStart: 12, byteEnd: 14 }]
          : []),
      ],
    };
  }
  return {
    name: 'Ethernet II',
    fields: [
      { name: 'Source MAC', value: '(not parsed)', byteStart: 6, byteEnd: 12 },
      { name: 'Destination MAC', value: '(not parsed)', byteStart: 0, byteEnd: 6 },
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

  if (ip) {
    const version = ipv6 ? 'IPv6' : 'IPv4';
    return {
      name: `Internet Protocol Version ${ipv6 ? '6' : '4'} (${version})`,
      fields: [
        {
          name: 'Source',
          value: String(ip.src ?? packet.sourceIp ?? ''),
          byteStart: 26,
          byteEnd: 30,
        },
        {
          name: 'Destination',
          value: String(ip.dst ?? packet.destIp ?? ''),
          byteStart: 30,
          byteEnd: 34,
        },
        ...(ip.ttl !== undefined
          ? [{ name: 'TTL', value: String(ip.ttl), byteStart: 22, byteEnd: 23 }]
          : []),
        ...(ip.protocol !== undefined ? [{ name: 'Protocol', value: String(ip.protocol) }] : []),
      ],
    };
  }

  if (packet.sourceIp && packet.destIp) {
    return {
      name: 'Internet Protocol Version 4 (IPv4)',
      fields: [
        { name: 'Source', value: packet.sourceIp, byteStart: 26, byteEnd: 30 },
        { name: 'Destination', value: packet.destIp, byteStart: 30, byteEnd: 34 },
      ],
    };
  }

  return null;
}

function buildTransportLayer(
  headers: Record<string, unknown> | undefined,
  packet: PacketMeta,
): ProtocolLayer | null {
  const proto = packet.protocol?.toUpperCase();
  const tcp = headers?.tcp as Record<string, unknown> | undefined;
  const udp = headers?.udp as Record<string, unknown> | undefined;
  const icmp = headers?.icmp as Record<string, unknown> | undefined;

  if (tcp || proto === 'TCP') return buildTcpLayer(tcp, packet);
  if (udp || proto === 'UDP') return buildUdpLayer(udp, packet);
  if (icmp || proto === 'ICMP') return buildIcmpLayer(icmp);
  return null;
}

function buildTcpLayer(
  tcp: Record<string, unknown> | undefined,
  packet: PacketMeta,
): ProtocolLayer {
  const s = 34; // typical TCP header offset
  return {
    name: 'Transmission Control Protocol (TCP)',
    fields: [
      {
        name: 'Source Port',
        value: String(tcp?.srcPort ?? packet.sourcePort ?? ''),
        byteStart: s,
        byteEnd: s + 2,
      },
      {
        name: 'Destination Port',
        value: String(tcp?.dstPort ?? packet.destPort ?? ''),
        byteStart: s + 2,
        byteEnd: s + 4,
      },
      ...(tcp?.seq !== undefined
        ? [{ name: 'Sequence Number', value: String(tcp.seq), byteStart: s + 4, byteEnd: s + 8 }]
        : []),
      ...(tcp?.ack !== undefined
        ? [
            {
              name: 'Acknowledgment Number',
              value: String(tcp.ack),
              byteStart: s + 8,
              byteEnd: s + 12,
            },
          ]
        : []),
      ...(tcp?.flags
        ? [{ name: 'Flags', value: String(tcp.flags), byteStart: s + 13, byteEnd: s + 14 }]
        : []),
      ...(tcp?.window !== undefined ? [{ name: 'Window Size', value: String(tcp.window) }] : []),
    ],
  };
}

function buildUdpLayer(
  udp: Record<string, unknown> | undefined,
  packet: PacketMeta,
): ProtocolLayer {
  const s = 34;
  return {
    name: 'User Datagram Protocol (UDP)',
    fields: [
      {
        name: 'Source Port',
        value: String(udp?.srcPort ?? packet.sourcePort ?? ''),
        byteStart: s,
        byteEnd: s + 2,
      },
      {
        name: 'Destination Port',
        value: String(udp?.dstPort ?? packet.destPort ?? ''),
        byteStart: s + 2,
        byteEnd: s + 4,
      },
      ...(udp?.length !== undefined
        ? [{ name: 'Length', value: String(udp.length), byteStart: s + 4, byteEnd: s + 6 }]
        : []),
    ],
  };
}

function buildIcmpLayer(icmp: Record<string, unknown> | undefined): ProtocolLayer {
  return {
    name: 'Internet Control Message Protocol (ICMP)',
    fields: [
      ...(icmp?.type !== undefined
        ? [{ name: 'Type', value: String(icmp.type), byteStart: 34, byteEnd: 35 }]
        : []),
      ...(icmp?.code !== undefined
        ? [{ name: 'Code', value: String(icmp.code), byteStart: 35, byteEnd: 36 }]
        : []),
      ...(icmp?.id !== undefined ? [{ name: 'Identifier', value: String(icmp.id) }] : []),
      ...(icmp?.seq !== undefined ? [{ name: 'Sequence', value: String(icmp.seq) }] : []),
    ],
  };
}

function buildApplicationLayers(
  headers: Record<string, unknown> | undefined,
  protocol: string | undefined,
): ProtocolLayer[] {
  const layers: ProtocolLayer[] = [];
  const proto = protocol?.toUpperCase();
  const dns = headers?.dns as Record<string, unknown> | undefined;
  const arp = headers?.arp as Record<string, unknown> | undefined;

  if (dns || proto === 'DNS') {
    layers.push(buildDnsLayer(dns));
  }
  if (arp || proto === 'ARP') {
    layers.push(buildArpLayer(arp));
  }
  return layers;
}

function buildDnsLayer(dns: Record<string, unknown> | undefined): ProtocolLayer {
  return {
    name: 'Domain Name System (DNS)',
    fields: [
      ...(dns?.id !== undefined ? [{ name: 'Transaction ID', value: String(dns.id) }] : []),
      ...(dns?.qr !== undefined ? [{ name: 'QR', value: dns.qr ? 'Response' : 'Query' }] : []),
      ...(dns?.questions ? [{ name: 'Questions', value: String(dns.questions) }] : []),
      ...(dns?.answers ? [{ name: 'Answers', value: String(dns.answers) }] : []),
    ],
  };
}

function buildArpLayer(arp: Record<string, unknown> | undefined): ProtocolLayer {
  return {
    name: 'Address Resolution Protocol (ARP)',
    fields: [
      ...(arp?.operation !== undefined ? [{ name: 'Opcode', value: String(arp.operation) }] : []),
      ...(arp?.senderMac ? [{ name: 'Sender MAC', value: String(arp.senderMac) }] : []),
      ...(arp?.senderIp ? [{ name: 'Sender IP', value: String(arp.senderIp) }] : []),
      ...(arp?.targetMac ? [{ name: 'Target MAC', value: String(arp.targetMac) }] : []),
      ...(arp?.targetIp ? [{ name: 'Target IP', value: String(arp.targetIp) }] : []),
    ],
  };
}
