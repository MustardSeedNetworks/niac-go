import { type FC, memo } from 'react';
import { Tag, SmallText } from '@krisarmstrong/web-foundation';
import type { Packet } from './PacketList';

interface PacketDetailsProps {
  packet: Packet | null;
}

/**
 * Format bytes to human-readable string
 */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * Format timestamp to full date/time string
 */
function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      fractionalSecondDigits: 3,
      hour12: false,
    });
  } catch {
    return timestamp;
  }
}

/**
 * Get protocol color for tag display
 */
function getProtocolColor(protocol: string): 'blue' | 'green' | 'yellow' | 'purple' | 'gray' | 'red' {
  const colors: Record<string, 'blue' | 'green' | 'yellow' | 'purple' | 'gray' | 'red'> = {
    ARP: 'yellow',
    ICMP: 'blue',
    DNS: 'green',
    TCP: 'purple',
    UDP: 'gray',
    HTTP: 'blue',
    HTTPS: 'green',
    DHCP: 'yellow',
  };
  return colors[protocol.toUpperCase()] || 'gray';
}

/**
 * Detail row component for consistent styling
 */
const DetailRow = memo(({ label, value, mono = false }: { label: string; value: string | number | undefined; mono?: boolean }) => {
  if (value === undefined || value === null || value === '') {
    return null;
  }

  return (
    <div className="flex items-start gap-2 py-1.5 border-b border-white/5 last:border-0">
      <SmallText className="text-gray-400 w-24 flex-shrink-0">{label}</SmallText>
      <span className={`text-sm text-white ${mono ? 'font-mono' : ''}`}>{value}</span>
    </div>
  );
});

DetailRow.displayName = 'DetailRow';

/**
 * Headers section for parsed protocol headers
 */
const HeadersSection = memo(({ headers }: { headers: Record<string, unknown> | undefined }) => {
  if (!headers || Object.keys(headers).length === 0) {
    return null;
  }

  return (
    <div className="mt-4">
      <h4 className="text-sm font-semibold text-gray-300 mb-2">Protocol Headers</h4>
      <div className="rounded-lg bg-gray-950/50 p-3 border border-white/5">
        {Object.entries(headers).map(([key, value]) => (
          <DetailRow
            key={key}
            label={key}
            value={typeof value === 'object' ? JSON.stringify(value) : String(value)}
            mono
          />
        ))}
      </div>
    </div>
  );
});

HeadersSection.displayName = 'HeadersSection';

/**
 * Packet Details Component
 *
 * Displays parsed packet information in a structured format:
 * - Protocol, timestamp, size
 * - Source and destination addresses with ports
 * - Protocol-specific headers when available
 */
export const PacketDetails: FC<PacketDetailsProps> = memo(({ packet }) => {
  if (!packet) {
    return (
      <div className="h-full flex items-center justify-center text-gray-400">
        <p className="text-sm">Select a packet to view details</p>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      {/* Header with protocol tag */}
      <div className="flex items-center justify-between mb-4">
        <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-sm">
          {packet.protocol}
        </Tag>
        <SmallText className="text-gray-400">{formatBytes(packet.size)}</SmallText>
      </div>

      {/* Main details */}
      <div className="space-y-1">
        <DetailRow label="Timestamp" value={formatTimestamp(packet.timestamp)} mono />
        <DetailRow label="Size" value={`${packet.size} bytes`} />

        {/* Source */}
        <div className="pt-2">
          <SmallText className="text-gray-500 uppercase text-xs tracking-wide">Source</SmallText>
        </div>
        <DetailRow
          label="IP Address"
          value={packet.sourceIP}
          mono
        />
        {packet.sourcePort !== undefined && (
          <DetailRow label="Port" value={packet.sourcePort} mono />
        )}

        {/* Destination */}
        <div className="pt-2">
          <SmallText className="text-gray-500 uppercase text-xs tracking-wide">Destination</SmallText>
        </div>
        <DetailRow
          label="IP Address"
          value={packet.destIP}
          mono
        />
        {packet.destPort !== undefined && (
          <DetailRow label="Port" value={packet.destPort} mono />
        )}

        {/* Summary */}
        {packet.summary && (
          <>
            <div className="pt-2">
              <SmallText className="text-gray-500 uppercase text-xs tracking-wide">Summary</SmallText>
            </div>
            <p className="text-sm text-gray-300 py-1.5">{packet.summary}</p>
          </>
        )}
      </div>

      {/* Protocol headers */}
      <HeadersSection headers={packet.headers} />
    </div>
  );
});

PacketDetails.displayName = 'PacketDetails';
