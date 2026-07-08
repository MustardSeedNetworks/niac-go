import { type FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
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

  const [expandedLayers, setExpandedLayers] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    setExpandedLayers(
      new Set(layers.filter((layer) => layer.expanded ?? true).map((layer) => layer.name)),
    );
  }, [layers]);

  const toggleLayer = useCallback((name: string) => {
    setExpandedLayers((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  const expandAll = useCallback(() => {
    setExpandedLayers(new Set(layers.map((layer) => layer.name)));
  }, [layers]);

  const collapseAll = useCallback(() => {
    setExpandedLayers(new Set());
  }, []);

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
      <div className="flex items-center justify-end gap-tight pb-1">
        <button
          type="button"
          data-testid="protocol-tree-expand-all"
          onClick={expandAll}
          className="text-xs text-text-muted hover:bg-surface-hover rounded px-1 py-0.5"
        >
          {t('packets.inspector.expandAll')}
        </button>
        <button
          type="button"
          data-testid="protocol-tree-collapse-all"
          onClick={collapseAll}
          className="text-xs text-text-muted hover:bg-surface-hover rounded px-1 py-0.5"
        >
          {t('packets.inspector.collapseAll')}
        </button>
      </div>
      {layers.map((layer) => (
        <ProtocolTreeLayer
          key={layer.name}
          layer={layer}
          isExpanded={expandedLayers.has(layer.name)}
          onToggle={toggleLayer}
          onFieldSelect={onFieldSelect}
        />
      ))}
    </div>
  );
});

ProtocolTree.displayName = 'ProtocolTree';
