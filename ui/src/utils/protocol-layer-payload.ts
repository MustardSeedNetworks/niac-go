import type { PacketMeta, ProtocolLayer } from './protocol-layers';

export function buildTransportLayer(
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
  return {
    name: 'Transmission Control Protocol (TCP)',
    fields: [
      {
        name: 'Source Port',
        value: String(tcp?.srcPort ?? packet.sourcePort ?? ''),
      },
      {
        name: 'Destination Port',
        value: String(tcp?.dstPort ?? packet.destPort ?? ''),
      },
      ...(tcp?.seq !== undefined ? [{ name: 'Sequence Number', value: String(tcp.seq) }] : []),
      ...(tcp?.ack !== undefined
        ? [
            {
              name: 'Acknowledgment Number',
              value: String(tcp.ack),
            },
          ]
        : []),
      ...(tcp?.flags ? [{ name: 'Flags', value: String(tcp.flags) }] : []),
      ...(tcp?.window !== undefined ? [{ name: 'Window Size', value: String(tcp.window) }] : []),
    ],
  };
}

function buildUdpLayer(
  udp: Record<string, unknown> | undefined,
  packet: PacketMeta,
): ProtocolLayer {
  return {
    name: 'User Datagram Protocol (UDP)',
    fields: [
      {
        name: 'Source Port',
        value: String(udp?.srcPort ?? packet.sourcePort ?? ''),
      },
      {
        name: 'Destination Port',
        value: String(udp?.dstPort ?? packet.destPort ?? ''),
      },
      ...(udp?.length !== undefined ? [{ name: 'Length', value: String(udp.length) }] : []),
    ],
  };
}

function buildIcmpLayer(icmp: Record<string, unknown> | undefined): ProtocolLayer {
  return {
    name: 'Internet Control Message Protocol (ICMP)',
    fields: [
      ...(icmp?.type !== undefined ? [{ name: 'Type', value: String(icmp.type) }] : []),
      ...(icmp?.code !== undefined ? [{ name: 'Code', value: String(icmp.code) }] : []),
      ...(icmp?.id !== undefined ? [{ name: 'Identifier', value: String(icmp.id) }] : []),
      ...(icmp?.seq !== undefined ? [{ name: 'Sequence', value: String(icmp.seq) }] : []),
    ],
  };
}

export function buildApplicationLayers(
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
