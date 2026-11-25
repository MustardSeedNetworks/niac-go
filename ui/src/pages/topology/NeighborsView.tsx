import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { POLL_INTERVALS } from '../../constants/polling';
import { useAppState } from '../../contexts/AppContext';
import type { TFunction } from '../../i18n';
import { Alert } from '../../shared/Alert';
import { CategoryChip } from '../../shared/CategoryChip';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { SearchInput } from '../../shared/SearchInput';
import { SegmentedControl } from '../../shared/SegmentedControl';
import { Skeleton } from '../../shared/Skeleton';
import { ContainedTable } from '../../ui/ContainedTable';
import { useTableLabels } from '../../ui/tableLabels';

/**
 * NeighborsView renders the live CDP / LLDP / EDP / FDP discovery table —
 * the protocol-filter chip row, the search box, and the per-neighbor
 * table. Used inside TopologyPage as the "Neighbors" tab. The page
 * chrome supplies the title + description; this component renders the
 * filterable list only.
 */

const PROTOCOL_FILTERS = ['all', 'CDP', 'LLDP', 'EDP', 'FDP'] as const;
type ProtocolFilter = (typeof PROTOCOL_FILTERS)[number];

const formatTtl = (ttlNs: number, tCommon: TFunction<'common'>, t: TFunction<'pages'>): string => {
  if (!ttlNs || ttlNs <= 0) return tCommon('format.dash');
  const seconds = Math.round(ttlNs / 1_000_000_000);
  if (seconds < 60) return tCommon('format.uptimeS', { value: seconds });
  const minutes = Math.round(seconds / 60);
  return t('topology.neighbors.ttlMinutes', { value: minutes });
};

const formatRelative = (iso: string, tCommon: TFunction<'common'>): string => {
  const ts = new Date(iso).getTime();
  if (!Number.isFinite(ts)) return iso;
  const deltaSec = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if (deltaSec < 60) return tCommon('format.relativeSecAgo', { value: deltaSec });
  const minutes = Math.round(deltaSec / 60);
  return tCommon('format.relativeMinAgo', { value: minutes });
};

export function NeighborsView() {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const { data: neighbors, loading, error } = useAppState('neighbors');

  const [protocolFilter, setProtocolFilter] = useState<ProtocolFilter>('all');
  const [search, setSearch] = useState('');

  // Deduped + filtered view. Real neighbour tables routinely show the
  // same adjacency announced by multiple protocols (e.g. CDP + LLDP
  // between two Cisco-or-mixed devices); the daemon emits one row per
  // (protocol, chassis, port) so the underlying neighbour table can
  // age each independently, but the human reading the table wants
  // one row per adjacency with the protocol-set rolled up.
  //
  // We collapse on (localDevice, remoteDevice, remotePort) and merge
  // the protocol strings into a comma-separated list. The "TTL" and
  // "Last Seen" columns surface the most recent value across the
  // collapsed group so the row reflects the freshest observation.
  const filtered = useMemo(() => {
    if (!neighbors) return [];
    const q = search.trim().toLowerCase();
    const grouped = new Map<
      string,
      {
        protocols: string[];
        localDevice: string;
        remoteDevice: string;
        remotePort: string;
        remoteChassisId: string;
        managementAddress: string;
        ttl: number;
        lastSeen: string;
        // Track the most recent lastSeen so updates win deterministically.
        lastSeenMs: number;
      }
    >();
    for (const n of neighbors) {
      if (protocolFilter !== 'all' && n.protocol !== protocolFilter) continue;
      if (q) {
        const haystack =
          `${n.localDevice} ${n.remoteDevice} ${n.remoteChassisId} ${n.remotePort}`.toLowerCase();
        if (!haystack.includes(q)) continue;
      }
      const key = `${n.localDevice}|${n.remoteDevice}|${n.remotePort}`;
      const tsMs = new Date(n.lastSeen).getTime();
      const existing = grouped.get(key);
      if (!existing) {
        grouped.set(key, {
          protocols: [n.protocol],
          localDevice: n.localDevice,
          remoteDevice: n.remoteDevice,
          remotePort: n.remotePort,
          remoteChassisId: n.remoteChassisId,
          managementAddress: n.managementAddress,
          ttl: n.ttl,
          lastSeen: n.lastSeen,
          lastSeenMs: Number.isFinite(tsMs) ? tsMs : 0,
        });
        continue;
      }
      if (!existing.protocols.includes(n.protocol)) {
        existing.protocols.push(n.protocol);
        existing.protocols.sort((a, b) => a.localeCompare(b));
      }
      // Most recent wins for the variable fields.
      if (Number.isFinite(tsMs) && tsMs > existing.lastSeenMs) {
        existing.lastSeenMs = tsMs;
        existing.lastSeen = n.lastSeen;
        existing.ttl = n.ttl;
        existing.managementAddress = n.managementAddress || existing.managementAddress;
        existing.remoteChassisId = n.remoteChassisId || existing.remoteChassisId;
      }
    }
    return [...grouped.values()].sort((a, b) => {
      if (a.localDevice !== b.localDevice) return a.localDevice.localeCompare(b.localDevice);
      return a.remoteDevice.localeCompare(b.remoteDevice);
    });
  }, [neighbors, protocolFilter, search]);

  const protocolCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    if (neighbors) {
      for (const n of neighbors) {
        counts[n.protocol] = (counts[n.protocol] ?? 0) + 1;
      }
    }
    return counts;
  }, [neighbors]);

  const labels = useTableLabels<(typeof filtered)[number]>(t('topology.neighbors.tableCaption'));
  const columns: DataTableColumn<(typeof filtered)[number]>[] = [
    {
      key: 'protocol',
      header: t('topology.neighbors.headerProtocol'),
      cell: (n) => (
        <span className="inline-flex gap-1">
          {n.protocols.map((protocol) => (
            <CategoryChip key={protocol} label={protocol} />
          ))}
        </span>
      ),
    },
    {
      key: 'localDevice',
      header: t('topology.neighbors.headerLocalDevice'),
      sortValue: (n) => n.localDevice,
      cell: (n) => <span className="text-text-primary">{n.localDevice}</span>,
    },
    {
      key: 'remoteDevice',
      header: t('topology.neighbors.headerRemoteDevice'),
      sortValue: (n) => n.remoteDevice,
      cell: (n) => <span className="text-text-primary">{n.remoteDevice}</span>,
    },
    {
      key: 'remotePort',
      header: t('topology.neighbors.headerRemotePort'),
      cell: (n) => (
        <span className="text-text-muted">{n.remotePort || tCommon('format.dash')}</span>
      ),
    },
    {
      key: 'chassisId',
      header: t('topology.neighbors.headerChassisId'),
      cell: (n) => (
        <span className="font-mono text-xs text-text-muted">
          {n.remoteChassisId || tCommon('format.dash')}
        </span>
      ),
    },
    {
      key: 'mgmtAddr',
      header: t('topology.neighbors.headerMgmtAddr'),
      cell: (n) => (
        <span className="font-mono text-xs text-text-muted">
          {n.managementAddress || tCommon('format.dash')}
        </span>
      ),
    },
    {
      key: 'ttl',
      header: t('topology.neighbors.headerTtl'),
      cell: (n) => <span className="text-text-muted">{formatTtl(n.ttl, tCommon, t)}</span>,
    },
    {
      key: 'lastSeen',
      header: t('topology.neighbors.headerLastSeen'),
      cell: (n) => <span className="text-text-muted">{formatRelative(n.lastSeen, tCommon)}</span>,
    },
  ];

  return (
    <div className="stack">
      <div className="flex flex-wrap items-center gap-3">
        <SegmentedControl
          label={t('topology.neighbors.filterAllLabel')}
          value={protocolFilter}
          onChange={(value) => {
            const next = PROTOCOL_FILTERS.find((candidate) => candidate === value);
            if (next) setProtocolFilter(next);
          }}
          options={PROTOCOL_FILTERS.map((p) => {
            const count = p === 'all' ? (neighbors?.length ?? 0) : (protocolCounts[p] ?? 0);
            return {
              value: p,
              label: `${p === 'all' ? t('topology.neighbors.filterAllLabel') : p} ${count}`,
            };
          })}
        />
        <SearchInput
          value={search}
          onValueChange={setSearch}
          label={t('topology.neighbors.searchAriaLabel')}
          clearLabel={tCommon('accessibility.clearSearch')}
          placeholder={t('topology.neighbors.searchPlaceholder')}
          className="min-w-48 flex-1"
        />
        <span className="text-xs text-text-muted">
          {t('topology.neighbors.pollingStatus', {
            seconds: POLL_INTERVALS.medium / 1000,
            count: neighbors?.length ?? 0,
          })}
        </span>
      </div>

      {loading && !neighbors && (
        <div role="status" aria-label={t('topology.neighbors.loading')}>
          <Skeleton className="h-32 w-full" />
        </div>
      )}
      {error && (
        <Alert tone="error">{t('topology.neighbors.loadError', { error: error.message })}</Alert>
      )}
      {neighbors && filtered.length === 0 && (
        <EmptyState
          title={
            neighbors.length === 0
              ? t('topology.neighbors.emptyNoData')
              : t('topology.neighbors.emptyFiltered')
          }
        />
      )}
      {filtered.length > 0 && (
        <ContainedTable label={t('topology.neighbors.tableCaption')}>
          <DataTable
            rows={filtered}
            columns={columns}
            getRowKey={(n) => `${n.localDevice}-${n.remoteDevice}-${n.remotePort}`}
            caption={t('topology.neighbors.tableCaption')}
            labels={labels}
            empty={null}
          />
        </ContainedTable>
      )}
    </div>
  );
}
