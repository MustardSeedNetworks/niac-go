import { type FC, memo } from 'react';
import { useTimeDisplay } from '../hooks/useTimeDisplay';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { getProtocolColor } from '../utils/protocol-colors';
import { formatTimeByMode, getTimeDisplayLabel } from '../utils/time-display';

/**
 * Packet data structure for display in the list
 */
export interface Packet {
  id: string;
  timestamp: string;
  protocol: string;
  sourceIp: string;
  destIp: string;
  sourcePort?: number;
  destPort?: number;
  size: number;
  summary: string;
  rawData: string; // Hex encoded raw packet data
  headers?: Record<string, unknown>;
}

interface PacketListProps {
  packets: Packet[];
  selectedPacketId: string | null;
  onSelectPacket: (packet: Packet) => void;
  autoScroll: boolean;
  getRowStyle?: (packet: Packet) => React.CSSProperties | undefined;
}

/**
 * Individual packet row component - memoized for performance
 */
const PacketRow = memo(
  ({
    packet,
    isSelected,
    onClick,
    formattedTime,
    rowStyle,
  }: {
    packet: Packet;
    isSelected: boolean;
    onClick: () => void;
    formattedTime: string;
    rowStyle?: React.CSSProperties;
  }) => (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left pad-xs rounded-lg border transition-colors ${
        isSelected
          ? 'border-brand-primary/50 bg-brand-primary/30'
          : 'border-surface-border bg-bg-base/50 hover:bg-bg-surface/50 hover:border-surface-border'
      }`}
      style={rowStyle}
    >
      <div className="flex items-center gap-compact">
        <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-xs">
          {packet.protocol}
        </Tag>
        <SmallText className="text-text-muted font-mono text-xs">{formattedTime}</SmallText>
      </div>
      <div className="mt-tight text-sm text-text-primary truncate">
        {packet.sourceIp}
        {packet.sourcePort && `:${packet.sourcePort}`}
        <span className="text-text-muted mx-1">-&gt;</span>
        {packet.destIp}
        {packet.destPort && `:${packet.destPort}`}
      </div>
      <SmallText className="text-text-muted truncate block">{packet.summary}</SmallText>
    </button>
  ),
);

PacketRow.displayName = 'PacketRow';

/**
 * Packet List Component
 *
 * Displays a scrollable list of captured packets with filtering support.
 * Clicking a packet selects it for detailed viewing.
 */
export const PacketList: FC<PacketListProps> = memo(
  ({ packets, selectedPacketId, onSelectPacket, autoScroll, getRowStyle }) => {
    const { mode: timeMode, cycleMode: cycleTimeMode } = useTimeDisplay();

    if (packets.length === 0) {
      return (
        <div className="h-full flex-center text-text-muted">
          <div className="text-center">
            <p className="text-sm">No packets to display</p>
            <SmallText>Waiting for packet stream...</SmallText>
          </div>
        </div>
      );
    }

    return (
      <div className="h-full flex flex-col">
        <button
          type="button"
          onClick={cycleTimeMode}
          className="text-xs text-text-muted hover:text-brand-accent mb-tight text-left select-none"
          title="Click to cycle: Absolute / Relative / Delta"
        >
          Mode: {getTimeDisplayLabel(timeMode)}
        </button>
        <div
          className={`flex-1 overflow-y-auto stack-xs pr-2 ${autoScroll ? 'scroll-smooth' : ''}`}
          style={{ scrollBehavior: autoScroll ? 'smooth' : 'auto' }}
        >
          {packets.map((packet, idx) => (
            <PacketRow
              key={packet.id}
              packet={packet}
              isSelected={selectedPacketId === packet.id}
              onClick={() => onSelectPacket(packet)}
              formattedTime={formatTimeByMode(
                packet.timestamp,
                timeMode,
                packets[0]?.timestamp ?? null,
                idx > 0 ? packets[idx - 1].timestamp : null,
              )}
              rowStyle={getRowStyle?.(packet)}
            />
          ))}
        </div>
      </div>
    );
  },
);

PacketList.displayName = 'PacketList';
