import type { Packet } from '../../components/PacketList';

/**
 * Map a live SSE packet payload onto the Packet shape the inspector renders.
 *
 * SSE payloads bypass the shared toCamelCase converter in client.ts, so every
 * field falls back to its snake_case spelling.
 */
export function packetFromStreamEvent(
  incoming: Record<string, unknown>,
  fallbackTimestamp: string,
  id: string,
): Packet {
  return {
    id,
    timestamp: (incoming.timestamp as string) || fallbackTimestamp,
    protocol: (incoming.protocol as string) || 'Unknown',
    // Absent, not 'Unknown': a frame with no IP layer has no address, and a
    // display placeholder here made buildIpLayer fabricate an IPv4 layer for it.
    sourceIp: (incoming.sourceIp as string) || (incoming.source_ip as string) || undefined,
    destIp: (incoming.destIp as string) || (incoming.dest_ip as string) || undefined,
    sourcePort:
      (incoming.sourcePort as number | undefined) ?? (incoming.source_port as number | undefined),
    destPort:
      (incoming.destPort as number | undefined) ?? (incoming.dest_port as number | undefined),
    size: (incoming.size as number) || 0,
    summary: (incoming.summary as string) || '',
    rawData:
      (incoming.rawData as string) ||
      (incoming.raw_data as string) ||
      (incoming.payload as string) ||
      '',
    headers: incoming.headers as Record<string, unknown> | undefined,
    physicalVlan:
      (incoming.physicalVlan as number | undefined) ??
      (incoming.physical_vlan as number | undefined),
    ingressNetwork:
      (incoming.ingressNetwork as string | undefined) ??
      (incoming.ingress_network as string | undefined),
    routeDecision:
      (incoming.routeDecision as string | undefined) ??
      (incoming.route_decision as string | undefined),
    hop: incoming.hop as string | undefined,
    egressNetwork:
      (incoming.egressNetwork as string | undefined) ??
      (incoming.egress_network as string | undefined),
    egressRejectionReason:
      (incoming.egressRejectionReason as string | undefined) ??
      (incoming.egress_rejection_reason as string | undefined),
  };
}
