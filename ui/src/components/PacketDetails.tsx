import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { formatBytes } from '../utils/format';
import { getProtocolColor } from '../utils/protocol-colors';
import type { Packet } from './PacketList';
import { ProtocolTree } from './ProtocolTree';

interface PacketDetailsProps {
  packet: Packet | null;
  onFieldSelect?: (byteStart: number, byteEnd: number) => void;
}

/**
 * Detail row component for consistent styling
 */
const DetailRow = memo(
  ({
    label,
    value,
    mono = false,
  }: {
    label: string;
    value: string | number | undefined;
    mono?: boolean;
  }) => {
    if (value === undefined || value === null || value === '') {
      return null;
    }

    return (
      <div className="flex items-start gap-2 py-1.5 border-b border-surface-border last:border-0">
        <SmallText className="text-text-muted w-24 flex-shrink-0">{label}</SmallText>
        <span className={`text-sm text-text-primary ${mono ? 'font-mono' : ''}`}>{value}</span>
      </div>
    );
  },
);

DetailRow.displayName = 'DetailRow';

/**
 * Headers section for parsed protocol headers
 */
const HeadersSection = memo(({ headers }: { headers: Record<string, unknown> | undefined }) => {
  const { t } = useTranslation('pages');
  if (!headers || Object.keys(headers).length === 0) {
    return null;
  }

  return (
    <div className="mt-4">
      <h4 className="text-sm font-semibold text-text-secondary mb-2">
        {t('packets.inspector.protocolHeaders')}
      </h4>
      <div className="rounded-lg bg-bg-base/50 p-3 border border-surface-border">
        {Object.entries(headers).map(([key, value]) => (
          <DetailRow
            key={key}
            label={key}
            value={typeof value === 'object' ? JSON.stringify(value) : String(value)}
            mono={true}
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
export const PacketDetails: FC<PacketDetailsProps> = memo(({ packet, onFieldSelect }) => {
  const { t } = useTranslation('pages');
  if (!packet) {
    return (
      <div className="h-full flex items-center justify-center text-text-muted">
        <p className="text-sm">{t('packets.inspector.selectPacketForDetails')}</p>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      {/* Header with protocol tag */}
      <div className="flex items-center justify-between mb-3">
        <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-sm">
          {packet.protocol}
        </Tag>
        <SmallText className="text-text-muted">{formatBytes(packet.size)}</SmallText>
      </div>

      {/* Protocol dissection tree */}
      <ProtocolTree packet={packet} onFieldSelect={onFieldSelect} />

      {/* Summary */}
      {packet.summary && (
        <div className="mt-3 pt-2 border-t border-surface-border">
          <SmallText className="text-text-muted uppercase text-xs tracking-wide">Summary</SmallText>
          <p className="text-sm text-text-secondary py-1">{packet.summary}</p>
        </div>
      )}
    </div>
  );
});

PacketDetails.displayName = 'PacketDetails';
