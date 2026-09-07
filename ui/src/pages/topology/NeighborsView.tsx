import { type FC, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchNeighbors } from '../../api/client';
import { useAppContext } from '../../contexts/AppContext';
import { useApiResource } from '../../hooks/useApiResource';
import type { TFunction } from '../../i18n';
import { Card, CardContent } from '../../ui/Card';
import { DataTable, type DataTableColumn } from '../../ui/DataTable';

/**
 * NeighborsView renders the live CDP / LLDP / EDP / FDP discovery table —
 * the protocol-filter chip row, the search box, and the per-neighbor
 * table. Used inside TopologyPage as the "Neighbors" tab. The page
 * chrome supplies the title + description; this component renders the
 * filterable list only.
 */

const PROTOCOL_FILTERS = ['all', 'CDP', 'LLDP', 'EDP', 'FDP'] as const;
type ProtocolFilter = (typeof PROTOCOL_FILTERS)[number];

const NEIGHBOR_POLL_MS = 5_000;

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

export const NeighborsView: FC = () => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const { sessionId } = useAppContext();
  const {
    data: neighbors,
    loading,
    error,
  } = useApiResource(() => fetchNeighbors(sessionId ?? ''), [sessionId], {
    intervalMs: NEIGHBOR_POLL_MS,
    enabled: sessionId !== null,
  });

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
        existing.protocols.sort();
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

  const columns: DataTableColumn<(typeof filtered)[number]>[] = [
    {
      key: 'protocol',
      header: t('topology.neighbors.headerProtocol'),
      cellClassName: 'font-mono text-xs text-status-info',
      cell: (n) => n.protocols.join(', '),
    },
    {
      key: 'localDevice',
      header: t('topology.neighbors.headerLocalDevice'),
      cell: (n) => n.localDevice,
    },
    {
      key: 'remoteDevice',
      header: t('topology.neighbors.headerRemoteDevice'),
      cell: (n) => n.remoteDevice,
    },
    {
      key: 'remotePort',
      header: t('topology.neighbors.headerRemotePort'),
      cellClassName: 'text-text-muted',
      cell: (n) => n.remotePort || tCommon('format.dash'),
    },
    {
      key: 'chassisId',
      header: t('topology.neighbors.headerChassisId'),
      cellClassName: 'font-mono text-xs text-text-muted',
      cell: (n) => n.remoteChassisId || tCommon('format.dash'),
    },
    {
      key: 'mgmtAddr',
      header: t('topology.neighbors.headerMgmtAddr'),
      cellClassName: 'font-mono text-xs text-text-muted',
      cell: (n) => n.managementAddress || tCommon('format.dash'),
    },
    {
      key: 'ttl',
      header: t('topology.neighbors.headerTtl'),
      cellClassName: 'text-text-muted',
      cell: (n) => formatTtl(n.ttl, tCommon, t),
    },
    {
      key: 'lastSeen',
      header: t('topology.neighbors.headerLastSeen'),
      cellClassName: 'text-text-muted',
      cell: (n) => formatRelative(n.lastSeen, tCommon),
    },
  ];

  return (
    <div className="stack-lg">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <div className="flex flex-wrap items-center gap-default">
            <div className="flex items-center gap-compact">
              {PROTOCOL_FILTERS.map((p) => {
                const count = p === 'all' ? (neighbors?.length ?? 0) : (protocolCounts[p] ?? 0);
                const active = protocolFilter === p;
                return (
                  <button
                    key={p}
                    type="button"
                    onClick={() => setProtocolFilter(p)}
                    title={
                      p === 'all'
                        ? t('topology.neighbors.filterShowAllTitle')
                        : t('topology.neighbors.filterByProtocolTitle', {
                            protocol: p,
                            count: protocolCounts[p] ?? 0,
                          })
                    }
                    className={`rounded px-3 py-compact text-xs font-medium ${
                      active
                        ? 'bg-status-info/20 text-status-info ring-1 ring-status-info/40'
                        : 'bg-bg-elevated/60 text-text-secondary hover:bg-bg-elevated'
                    }`}
                  >
                    {p === 'all' ? t('topology.neighbors.filterAllLabel') : p}
                    <span className="ml-1.5 text-[10px] text-text-muted">{count}</span>
                  </button>
                );
              })}
            </div>
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('topology.neighbors.searchPlaceholder')}
              title={t('topology.neighbors.searchTitle')}
              className="ml-auto w-64 rounded border border-surface-border bg-bg-base/60 px-3 py-compact-md text-sm text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
              aria-label={t('topology.neighbors.searchAriaLabel')}
            />
            <span className="text-xs text-text-muted">
              {t('topology.neighbors.pollingStatus', {
                seconds: NEIGHBOR_POLL_MS / 1000,
                count: neighbors?.length ?? 0,
              })}
            </span>
          </div>
        </CardContent>
      </Card>

      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="p-0">
          {loading && !neighbors && (
            <div className="pad-lg text-sm text-text-muted">{t('topology.neighbors.loading')}</div>
          )}
          {error && (
            <div className="pad-lg text-sm text-status-error" role="alert">
              {t('topology.neighbors.loadError', { error: error.message })}
            </div>
          )}
          {neighbors && filtered.length === 0 && (
            <div className="pad-lg text-sm text-text-muted">
              {neighbors.length === 0
                ? t('topology.neighbors.emptyNoData')
                : t('topology.neighbors.emptyFiltered')}
            </div>
          )}
          {filtered.length > 0 && (
            <DataTable
              rows={filtered}
              columns={columns}
              getRowKey={(n) => `${n.localDevice}-${n.remoteDevice}-${n.remotePort}`}
              emptyMessage={null}
              rowClassName={() => 'text-text-primary hover:bg-bg-base/40'}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
};
