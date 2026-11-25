import { Layers } from 'lucide-react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { SegmentSummary } from '../api/types';
import { DeviceTable } from '../components/DeviceTable';
import { TableSection } from '../components/TableSection';
import { useAppContext } from '../contexts/AppContext';
import { useSegmentsResource } from '../hooks/usePageResources';
import { Alert } from '../shared/Alert';
import { CategoryChip } from '../shared/CategoryChip';
import { EmptyState } from '../shared/EmptyState';
import { Skeleton } from '../shared/Skeleton';

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
export function SegmentsPage() {
  return (
    <div className="grid grid-cols-1 gap-spacious">
      <SegmentsListCard />
    </div>
  );
}

/**
 * Segments List - the VLAN segments from the current config
 */
function SegmentsListCard() {
  const { t } = useTranslation('pages');
  const { sessionId } = useAppContext();
  const { data: segments, loading, error } = useSegmentsResource(sessionId);

  if (loading && !segments) {
    return (
      <div role="status" aria-label={t('segments.title')}>
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }
  if (error) {
    return <Alert tone="error">{error.message}</Alert>;
  }
  if (!segments || segments.length === 0) {
    return (
      <div data-testid="segments-results">
        <EmptyState icon={<Layers className="h-6 w-6" />} title={t('segments.empty')} />
      </div>
    );
  }
  return (
    <div className="stack-xl" data-testid="segments-results">
      {segments.map((segment, index) => (
        <SegmentSection key={`${segment.vlanTag}-${index}`} segment={segment} />
      ))}
    </div>
  );
}

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
    <TableSection
      title={heading}
      data-testid={`segment-${segment.vlanTag}`}
      aside={
        <span data-testid={`segment-count-${segment.vlanTag}`}>
          <CategoryChip
            label={`${segment.devices.length} ${t('segments.deviceNoun', { count: segment.devices.length })}`}
          />
        </span>
      }
    >
      {segment.devices.length === 0 ? (
        <EmptyState title={t('segments.emptySegmentDevices')} />
      ) : (
        <DeviceTable devices={segment.devices} />
      )}
    </TableSection>
  );
});

SegmentSection.displayName = 'SegmentSection';

export default SegmentsPage;
