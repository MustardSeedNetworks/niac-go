import { Layers } from 'lucide-react';
import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchSegments } from '../api/client';
import type { SegmentSummary } from '../api/types';
import { DeviceTable } from '../components/DeviceTable';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useAppContext } from '../contexts/AppContext';
import { useApiResource } from '../hooks/useApiResource';
import { BaseCard } from '../ui/BaseCard';
import { Tag } from '../ui/Tag';
import { H3, SmallText } from '../ui/Typography';

/**
 * Segments Page
 *
 * Read-only view of the multi-VLAN segments the active config describes
 * (ADR 0008): each segment groups the devices isolated on one VLAN tag,
 * using the exact same device shape as the Running Devices page. A flat
 * (non-segmented) config still reports one "Untagged" segment wrapping
 * every device, so this page always has something to show once a
 * simulation is loaded.
 */
export const SegmentsPage: FC = () => (
  <div className="grid gap-spacious">
    <SegmentsListCard />
  </div>
);

/**
 * Segments List Card - Shows VLAN segments from current config
 */
const SegmentsListCard: FC = () => {
  const { t } = useTranslation('pages');
  const { sessionId } = useAppContext();
  const {
    data: segments,
    loading,
    error,
  } = useApiResource(() => fetchSegments(sessionId ?? ''), [sessionId], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: sessionId !== null,
  });

  return (
    <BaseCard<SegmentSummary[]>
      title={t('segments.title')}
      subtitle={t('segments.description')}
      icon={<Layers className={`${iconSizes.lg} text-status-info`} />}
      data={segments}
      loading={loading && !segments}
      error={error?.message}
      getStatus={(d) => (d.length > 0 ? 'success' : 'unknown')}
      emptyMessage={t('segments.empty')}
    >
      {(data) =>
        data.length === 0 ? (
          <div
            className="rounded-xl border border-surface-border bg-bg-base/50 pad-xl text-center text-text-muted"
            data-testid="segments-results"
          >
            {t('segments.empty')}
          </div>
        ) : (
          <div className="stack" data-testid="segments-results">
            {data.map((segment, index) => (
              <SegmentSection key={`${segment.vlanTag}-${index}`} segment={segment} />
            ))}
          </div>
        )
      }
    </BaseCard>
  );
};

/**
 * One VLAN segment: header (VLAN tag or "Untagged") + device count, then
 * the grouped device table.
 */
const SegmentSection = memo(({ segment }: { segment: SegmentSummary }) => {
  const { t } = useTranslation('pages');
  const heading = segment.untagged
    ? t('segments.untaggedHeader')
    : t('segments.vlanHeader', { tag: segment.vlanTag });

  return (
    <div
      className="rounded-xl border border-surface-border bg-bg-surface/40 pad-lg stack-sm"
      data-testid={`segment-${segment.vlanTag}`}
    >
      <div className="flex-between">
        <H3>{heading}</H3>
        {/* The count changes as the simulation runs, so it is a figure; the
            noun beside it is prose and must not be monospaced with it. */}
        <Tag colorScheme="blue">
          <span className="figure" data-testid={`segment-count-${segment.vlanTag}`}>
            {segment.devices.length}
          </span>{' '}
          {t('segments.deviceNoun', { count: segment.devices.length })}
        </Tag>
      </div>
      {segment.devices.length === 0 ? (
        <SmallText className="text-text-muted">{t('segments.emptySegmentDevices')}</SmallText>
      ) : (
        <DeviceTable devices={segment.devices} />
      )}
    </div>
  );
});

SegmentSection.displayName = 'SegmentSection';

export default SegmentsPage;
