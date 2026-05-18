import { type FC, useMemo, useState } from 'react';
import { fetchNeighbors } from '../../api/client';
import { useApiResource } from '../../hooks/useApiResource';
import { Card, CardContent } from '../../ui/Card';

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

const formatTtl = (ttlNs: number): string => {
  if (!ttlNs || ttlNs <= 0) return '—';
  const seconds = Math.round(ttlNs / 1_000_000_000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.round(seconds / 60);
  return `${minutes}m`;
};

const formatRelative = (iso: string): string => {
  const ts = new Date(iso).getTime();
  if (!Number.isFinite(ts)) return iso;
  const deltaSec = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if (deltaSec < 60) return `${deltaSec}s ago`;
  const minutes = Math.round(deltaSec / 60);
  return `${minutes}m ago`;
};

export const NeighborsView: FC = () => {
  const {
    data: neighbors,
    loading,
    error,
  } = useApiResource(() => fetchNeighbors(), [], { intervalMs: NEIGHBOR_POLL_MS });

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

  return (
    <div className="space-y-4">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2">
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
                        ? 'Show all discovery protocols'
                        : `Filter to ${p} entries only (${protocolCounts[p] ?? 0} now)`
                    }
                    className={`rounded px-3 py-1 text-xs font-medium ${
                      active
                        ? 'bg-status-info/20 text-status-info ring-1 ring-cyan-400/40'
                        : 'bg-bg-elevated/60 text-text-secondary hover:bg-bg-elevated'
                    }`}
                  >
                    {p === 'all' ? 'All' : p}
                    <span className="ml-1.5 text-[10px] text-text-muted">{count}</span>
                  </button>
                );
              })}
            </div>
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Filter by device or chassis ID…"
              title="Substring match on local device, remote device, chassis ID, or remote port (case-insensitive)."
              className="ml-auto w-64 rounded border border-surface-border bg-bg-base/60 px-3 py-1.5 text-sm text-text-primary placeholder-gray-500 focus:border-status-info focus:outline-none"
              aria-label="Filter neighbors"
            />
            <span className="text-xs text-text-muted">
              Polling every {NEIGHBOR_POLL_MS / 1000}s · {neighbors?.length ?? 0} entries
            </span>
          </div>
        </CardContent>
      </Card>

      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="p-0">
          {loading && !neighbors && <div className="p-6 text-sm text-text-muted">Loading…</div>}
          {error && (
            <div className="p-6 text-sm text-status-error" role="alert">
              Failed to load neighbors: {error.message}
            </div>
          )}
          {neighbors && filtered.length === 0 && (
            <div className="p-6 text-sm text-text-muted">
              {neighbors.length === 0
                ? 'No neighbors discovered yet. Start a simulation that has CDP/LLDP/EDP/FDP enabled to populate this table.'
                : 'No neighbors match the current filters.'}
            </div>
          )}
          {filtered.length > 0 && (
            <table className="w-full text-sm">
              <thead className="bg-bg-base/40 text-left text-xs uppercase tracking-wider text-text-muted">
                <tr>
                  <th className="px-4 py-2">Protocol</th>
                  <th className="px-4 py-2">Local Device</th>
                  <th className="px-4 py-2">Remote Device</th>
                  <th className="px-4 py-2">Remote Port</th>
                  <th className="px-4 py-2">Chassis ID</th>
                  <th className="px-4 py-2">Mgmt Addr</th>
                  <th className="px-4 py-2">TTL</th>
                  <th className="px-4 py-2">Last Seen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {filtered.map((n) => (
                  <tr
                    key={`${n.localDevice}-${n.remoteDevice}-${n.remotePort}`}
                    className="text-text-primary hover:bg-bg-base/40"
                  >
                    <td className="px-4 py-2 font-mono text-xs text-status-info">
                      {n.protocols.join(', ')}
                    </td>
                    <td className="px-4 py-2">{n.localDevice}</td>
                    <td className="px-4 py-2">{n.remoteDevice}</td>
                    <td className="px-4 py-2 text-text-muted">{n.remotePort || '—'}</td>
                    <td className="px-4 py-2 font-mono text-xs text-text-muted">
                      {n.remoteChassisId || '—'}
                    </td>
                    <td className="px-4 py-2 font-mono text-xs text-text-muted">
                      {n.managementAddress || '—'}
                    </td>
                    <td className="px-4 py-2 text-text-muted">{formatTtl(n.ttl)}</td>
                    <td className="px-4 py-2 text-text-muted">{formatRelative(n.lastSeen)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
