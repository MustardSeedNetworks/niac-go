import { type FC, memo, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../api/types';
import { buildProtocolLayers } from '../utils/protocol-layers';
import type { Packet } from './PacketList';
import { ProtocolTreeLayer } from './ProtocolTreeLayer';

interface ProtocolTreeProps {
  packet: Packet | PcapPacket | null;
  onFieldSelect?: (byteStart: number, byteEnd: number) => void;
}

/**
 * Protocol dissection tree view. Replaces the flat PacketDetails for Wireshark-like
 * layered protocol inspection. Clicking fields with byte offsets highlights the hex dump.
 */
export const ProtocolTree: FC<ProtocolTreeProps> = memo(({ packet, onFieldSelect }) => {
  const { t } = useTranslation('pages');
  const layers = useMemo(() => {
    if (!packet) return [];

    const pkt = {
      timestamp: packet.timestamp,
      protocol: packet.protocol,
      sourceIp: packet.sourceIp ?? (packet as PcapPacket).sourceIp,
      destIp: packet.destIp ?? (packet as PcapPacket).destIp,
      sourcePort: packet.sourcePort,
      destPort: packet.destPort,
      length: 'length' in packet ? (packet as PcapPacket).length : undefined,
      size: 'size' in packet ? (packet as Packet).size : undefined,
      number: 'number' in packet ? (packet as PcapPacket).number : undefined,
    };

    return buildProtocolLayers(packet.headers as Record<string, unknown> | undefined, pkt);
  }, [packet]);

  if (!packet) {
    return (
      <div className="h-full flex-center text-text-muted">
        <p className="text-sm">{t('packets.inspector.selectPacketForProtocol')}</p>
      </div>
    );
  }

  if (layers.length === 0) {
    return (
      <div className="h-full flex-center text-text-muted">
        <p className="text-sm">{t('packets.inspector.noProtocolLayers')}</p>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto stack-xs">
      {layers.map((layer) => (
        <ProtocolTreeLayer key={layer.name} layer={layer} onFieldSelect={onFieldSelect} />
      ))}
    </div>
  );
});

ProtocolTree.displayName = 'ProtocolTree';
