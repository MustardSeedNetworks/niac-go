import { ArrowRightLeft, BarChart3, Clock, FileText, Network, Server } from 'lucide-react';
import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapStats as PcapStatsType } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { formatBytes, formatDurationMs, useFormatTime } from '../../utils/format';
import { getProtocolColor } from '../../utils/protocol-colors';

interface PcapStatsProps {
  stats: PcapStatsType | null;
  filename: string | null;
  fileSize: number | null;
}

/**
 * Stat Block Component - displays a single statistic
 */
const StatBlock = memo(
  ({
    icon,
    label,
    value,
    helper,
  }: {
    icon: React.ReactNode;
    label: string;
    value: string | number;
    helper?: string;
  }) => (
    <div className="rounded-lg border border-surface-border bg-bg-base/50 pad">
      <div className="flex items-center gap-compact mb-2">
        {icon}
        <SmallText className="text-text-muted uppercase tracking-wide">{label}</SmallText>
      </div>
      <p className="heading-1 text-text-primary">{value}</p>
      {helper && <SmallText className="text-text-muted">{helper}</SmallText>}
    </div>
  ),
);

StatBlock.displayName = 'StatBlock';

/**
 * Protocol Breakdown Component
 */
const ProtocolBreakdown: FC<{
  protocols: Record<string, number>;
  total: number;
}> = memo(({ protocols, total }) => {
  const { t } = useTranslation('pages');
  // Sort protocols by count (descending)
  const sortedProtocols = Object.entries(protocols)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 10);

  if (sortedProtocols.length === 0) {
    return <SmallText className="text-text-muted">{t('packets.stats.noProtocolData')}</SmallText>;
  }

  return (
    <div className="stack-sm">
      {sortedProtocols.map(([protocol, count]) => {
        const percentage = total > 0 ? (count / total) * 100 : 0;

        return (
          <div key={protocol} className="stack-xs">
            <div className="flex-between">
              <div className="flex items-center gap-compact">
                <Tag colorScheme={getProtocolColor(protocol)} className="text-xs">
                  {protocol}
                </Tag>
                <SmallText className="text-text-muted">{count.toLocaleString()}</SmallText>
              </div>
              <SmallText className="text-text-muted">{percentage.toFixed(1)}%</SmallText>
            </div>
            <div className="h-1.5 w-full rounded-full bg-bg-elevated overflow-hidden">
              <div
                className="h-full rounded-full bg-brand-primary transition-all duration-500"
                style={{ width: `${percentage}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
});

ProtocolBreakdown.displayName = 'ProtocolBreakdown';

/**
 * Top Endpoints Component
 */
const TopEndpoints: FC<{
  sources: Array<{ ip: string; count: number }>;
  destinations: Array<{ ip: string; count: number }>;
}> = memo(({ sources, destinations }) => {
  const { t } = useTranslation('pages');
  return (
    <div className="grid gap-comfortable md:grid-cols-2">
      {/* Top Sources */}
      <div className="stack-sm">
        <div className="flex items-center gap-compact mb-heading">
          <Server className={`${iconSizes.md} text-status-info`} />
          <SmallText className="text-text-muted uppercase tracking-wide font-semibold">
            {t('packets.stats.topSources')}
          </SmallText>
        </div>
        {sources.length === 0 ? (
          <SmallText className="text-text-muted">{t('packets.stats.noSourceData')}</SmallText>
        ) : (
          <div className="stack-xs">
            {sources.slice(0, 5).map((item) => (
              <div
                key={item.ip}
                className="flex-between rounded-lg border border-surface-border bg-bg-base/50 px-3 py-row"
              >
                <span className="font-mono text-sm text-text-primary">{item.ip}</span>
                <Tag colorScheme="blue" className="text-xs">
                  {item.count.toLocaleString()}
                </Tag>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Top Destinations */}
      <div className="stack-sm">
        <div className="flex items-center gap-compact mb-heading">
          <Network className={`${iconSizes.md} text-status-success`} />
          <SmallText className="text-text-muted uppercase tracking-wide font-semibold">
            {t('packets.stats.topDestinations')}
          </SmallText>
        </div>
        {destinations.length === 0 ? (
          <SmallText className="text-text-muted">{t('packets.stats.noDestinationData')}</SmallText>
        ) : (
          <div className="stack-xs">
            {destinations.slice(0, 5).map((item) => (
              <div
                key={item.ip}
                className="flex-between rounded-lg border border-surface-border bg-bg-base/50 px-3 py-row"
              >
                <span className="font-mono text-sm text-text-primary">{item.ip}</span>
                <Tag colorScheme="green" className="text-xs">
                  {item.count.toLocaleString()}
                </Tag>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
});

TopEndpoints.displayName = 'TopEndpoints';

/**
 * PCAP Statistics Component
 *
 * Displays summary statistics from PCAP analysis including:
 * - Total packets and bytes
 * - Time range and duration
 * - Protocol breakdown with percentages
 * - Top source and destination endpoints
 */
export const PcapStats: FC<PcapStatsProps> = memo(({ stats, filename, fileSize }) => {
  const { t } = useTranslation('pages');
  const formatTimestamp = useFormatTime();

  if (!stats) {
    return (
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="py-centered text-center">
          <BarChart3 className={`${iconSizes['3xl']} text-text-disabled mx-auto mb-content`} />
          <p className="text-text-muted">{t('packets.stats.emptyTitle')}</p>
          <SmallText className="text-text-muted">{t('packets.stats.emptyDescription')}</SmallText>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="stack-xl">
      {/* File Info */}
      {filename && (
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent>
            <div className="flex items-center gap-default">
              <FileText className={`${iconSizes.lg} text-brand-accent`} />
              <div>
                <p className="font-medium text-text-primary">{filename}</p>
                {fileSize && (
                  <SmallText className="text-text-muted">{formatBytes(fileSize)}</SmallText>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Summary Statistics */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <H2 className="flex items-center gap-compact mb-0">
            <BarChart3 className={`${iconSizes.lg} text-status-info`} />
            {t('packets.stats.summaryTitle')}
          </H2>

          <div className="grid gap-comfortable sm:grid-cols-2 lg:grid-cols-4">
            <StatBlock
              icon={<ArrowRightLeft className={`${iconSizes.md} text-brand-accent`} />}
              label={t('packets.stats.totalPackets')}
              value={stats.totalPackets.toLocaleString()}
            />
            <StatBlock
              icon={<FileText className={`${iconSizes.md} text-status-info`} />}
              label={t('packets.stats.totalBytes')}
              value={formatBytes(stats.totalBytes)}
              helper={t('packets.stats.totalBytesHelper', {
                value: stats.totalBytes.toLocaleString(),
              })}
            />
            <StatBlock
              icon={<Clock className={`${iconSizes.md} text-status-success`} />}
              label={t('packets.stats.duration')}
              value={formatDurationMs(stats.timeRange.durationMs)}
            />
            <StatBlock
              icon={<Network className={`${iconSizes.md} text-status-warning`} />}
              label={t('packets.stats.protocols')}
              value={Object.keys(stats.protocols).length}
              helper={t('packets.stats.protocolsHelper')}
            />
          </div>
        </CardContent>
      </Card>

      {/* Time Range */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <div className="flex items-center gap-compact">
            <Clock className={`${iconSizes.lg} text-status-success`} />
            <H2>{t('packets.stats.timeRangeTitle')}</H2>
          </div>

          <div className="grid gap-comfortable sm:grid-cols-2">
            <div className="rounded-lg border border-surface-border bg-bg-base/50 pad">
              <SmallText className="text-text-muted uppercase tracking-wide">
                {t('packets.stats.startTime')}
              </SmallText>
              <p className="text-lg font-mono text-text-primary mt-tight">
                {formatTimestamp(stats.timeRange.start)}
              </p>
            </div>
            <div className="rounded-lg border border-surface-border bg-bg-base/50 pad">
              <SmallText className="text-text-muted uppercase tracking-wide">
                {t('packets.stats.endTime')}
              </SmallText>
              <p className="text-lg font-mono text-text-primary mt-tight">
                {formatTimestamp(stats.timeRange.end)}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Protocol Breakdown */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <div className="flex items-center gap-compact">
            <BarChart3 className={`${iconSizes.lg} text-brand-accent`} />
            <H2>{t('packets.stats.protocolBreakdownTitle')}</H2>
          </div>

          <ProtocolBreakdown protocols={stats.protocols} total={stats.totalPackets} />
        </CardContent>
      </Card>

      {/* Top Endpoints */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <div className="flex items-center gap-compact">
            <Network className={`${iconSizes.lg} text-status-info`} />
            <H2>{t('packets.stats.topEndpointsTitle')}</H2>
          </div>

          <TopEndpoints sources={stats.topSources} destinations={stats.topDestinations} />
        </CardContent>
      </Card>
    </div>
  );
});

PcapStats.displayName = 'PcapStats';

export default PcapStats;
