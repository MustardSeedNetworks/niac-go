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

/**
 * Build structured protocol layers from a packet's headers map and metadata.
 */
export function buildProtocolLayers(
  headers: Record<string, unknown> | undefined,
  packet: {
    timestamp?: string;
    protocol?: string;
    sourceIp?: string;
    destIp?: string;
    sourcePort?: number;
    destPort?: number;
    length?: number;
    size?: number;
    number?: number;
  },
): ProtocolLayer[] {
  const layers: ProtocolLayer[] = [];
  const totalLength = packet.length ?? packet.size ?? 0;

  // Frame layer (synthetic)
  layers.push({
    name: 'Frame',
    fields: [
      ...(packet.number !== undefined ? [{ name: 'Number', value: String(packet.number) }] : []),
      { name: 'Length', value: `${totalLength} bytes` },
      ...(packet.timestamp ? [{ name: 'Arrival Time', value: packet.timestamp }] : []),
    ],
  });

  // Ethernet layer
  const eth = headers?.ethernet as Record<string, unknown> | undefined;
  if (eth) {
    layers.push({
      name: 'Ethernet II',
      fields: [
        { name: 'Source MAC', value: String(eth.srcMac ?? eth.src ?? ''), byteStart: 6, byteEnd: 12 },
        { name: 'Destination MAC', value: String(eth.dstMac ?? eth.dst ?? ''), byteStart: 0, byteEnd: 6 },
        ...(eth.etherType ? [{ name: 'EtherType', value: String(eth.etherType), byteStart: 12, byteEnd: 14 }] : []),
      ],
    });
  } else {
    // Infer Ethernet from presence of IPs (always have ethernet in captured packets)
    layers.push({
      name: 'Ethernet II',
      fields: [
        { name: 'Source MAC', value: '(not parsed)', byteStart: 6, byteEnd: 12 },
        { name: 'Destination MAC', value: '(not parsed)', byteStart: 0, byteEnd: 6 },
      ],
    });
  }

  // IPv4/IPv6 layer
  const ipv4 = headers?.ipv4 as Record<string, unknown> | undefined;
  const ipv6 = headers?.ipv6 as Record<string, unknown> | undefined;
  const ip = ipv4 ?? ipv6 ?? (headers?.ip as Record<string, unknown> | undefined);

  if (ip) {
    const version = ipv6 ? 'IPv6' : 'IPv4';
    layers.push({
      name: `Internet Protocol Version ${ipv6 ? '6' : '4'} (${version})`,
      fields: [
        { name: 'Source', value: String(ip.src ?? packet.sourceIp ?? ''), byteStart: 14 + 12, byteEnd: 14 + 16 },
        { name: 'Destination', value: String(ip.dst ?? packet.destIp ?? ''), byteStart: 14 + 16, byteEnd: 14 + 20 },
        ...(ip.ttl !== undefined ? [{ name: 'TTL', value: String(ip.ttl), byteStart: 14 + 8, byteEnd: 14 + 9 }] : []),
        ...(ip.protocol !== undefined ? [{ name: 'Protocol', value: String(ip.protocol) }] : []),
      ],
    });
  } else if (packet.sourceIp && packet.destIp) {
    layers.push({
      name: 'Internet Protocol Version 4 (IPv4)',
      fields: [
        { name: 'Source', value: packet.sourceIp, byteStart: 26, byteEnd: 30 },
        { name: 'Destination', value: packet.destIp, byteStart: 30, byteEnd: 34 },
      ],
    });
  }

  // Transport layer
  const tcp = headers?.tcp as Record<string, unknown> | undefined;
  const udp = headers?.udp as Record<string, unknown> | undefined;
  const icmp = headers?.icmp as Record<string, unknown> | undefined;
  const proto = packet.protocol?.toUpperCase();

  if (tcp || proto === 'TCP') {
    const tcpStart = 34; // typical offset
    layers.push({
      name: 'Transmission Control Protocol (TCP)',
      fields: [
        { name: 'Source Port', value: String(tcp?.srcPort ?? packet.sourcePort ?? ''), byteStart: tcpStart, byteEnd: tcpStart + 2 },
        { name: 'Destination Port', value: String(tcp?.dstPort ?? packet.destPort ?? ''), byteStart: tcpStart + 2, byteEnd: tcpStart + 4 },
        ...(tcp?.seq !== undefined ? [{ name: 'Sequence Number', value: String(tcp.seq), byteStart: tcpStart + 4, byteEnd: tcpStart + 8 }] : []),
        ...(tcp?.ack !== undefined ? [{ name: 'Acknowledgment Number', value: String(tcp.ack), byteStart: tcpStart + 8, byteEnd: tcpStart + 12 }] : []),
        ...(tcp?.flags ? [{ name: 'Flags', value: String(tcp.flags), byteStart: tcpStart + 13, byteEnd: tcpStart + 14 }] : []),
        ...(tcp?.window !== undefined ? [{ name: 'Window Size', value: String(tcp.window) }] : []),
      ],
    });
  } else if (udp || proto === 'UDP') {
    const udpStart = 34;
    layers.push({
      name: 'User Datagram Protocol (UDP)',
      fields: [
        { name: 'Source Port', value: String(udp?.srcPort ?? packet.sourcePort ?? ''), byteStart: udpStart, byteEnd: udpStart + 2 },
        { name: 'Destination Port', value: String(udp?.dstPort ?? packet.destPort ?? ''), byteStart: udpStart + 2, byteEnd: udpStart + 4 },
        ...(udp?.length !== undefined ? [{ name: 'Length', value: String(udp.length), byteStart: udpStart + 4, byteEnd: udpStart + 6 }] : []),
      ],
    });
  } else if (icmp || proto === 'ICMP') {
    layers.push({
      name: 'Internet Control Message Protocol (ICMP)',
      fields: [
        ...(icmp?.type !== undefined ? [{ name: 'Type', value: String(icmp.type), byteStart: 34, byteEnd: 35 }] : []),
        ...(icmp?.code !== undefined ? [{ name: 'Code', value: String(icmp.code), byteStart: 35, byteEnd: 36 }] : []),
        ...(icmp?.id !== undefined ? [{ name: 'Identifier', value: String(icmp.id) }] : []),
        ...(icmp?.seq !== undefined ? [{ name: 'Sequence', value: String(icmp.seq) }] : []),
      ],
    });
  }

  // Application layer
  const dns = headers?.dns as Record<string, unknown> | undefined;
  const arp = headers?.arp as Record<string, unknown> | undefined;

  if (dns || proto === 'DNS') {
    layers.push({
      name: 'Domain Name System (DNS)',
      fields: [
        ...(dns?.id !== undefined ? [{ name: 'Transaction ID', value: String(dns.id) }] : []),
        ...(dns?.qr !== undefined ? [{ name: 'QR', value: dns.qr ? 'Response' : 'Query' }] : []),
        ...(dns?.questions ? [{ name: 'Questions', value: String(dns.questions) }] : []),
        ...(dns?.answers ? [{ name: 'Answers', value: String(dns.answers) }] : []),
      ],
    });
  }

  if (arp || proto === 'ARP') {
    layers.push({
      name: 'Address Resolution Protocol (ARP)',
      fields: [
        ...(arp?.operation !== undefined ? [{ name: 'Opcode', value: String(arp.operation) }] : []),
        ...(arp?.senderMac ? [{ name: 'Sender MAC', value: String(arp.senderMac) }] : []),
        ...(arp?.senderIp ? [{ name: 'Sender IP', value: String(arp.senderIp) }] : []),
        ...(arp?.targetMac ? [{ name: 'Target MAC', value: String(arp.targetMac) }] : []),
        ...(arp?.targetIp ? [{ name: 'Target IP', value: String(arp.targetIp) }] : []),
      ],
    });
  }

  return layers;
}
