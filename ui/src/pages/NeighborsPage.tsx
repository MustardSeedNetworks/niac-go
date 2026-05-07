import { type FC, useMemo, useState } from 'react';
import { fetchNeighbors } from '../api/client';
import { useApiResource } from '../hooks/useApiResource';
import { Card, CardContent } from '../ui/Card';

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

export const NeighborsPage: FC = () => {
  const {
    data: neighbors,
    loading,
    error,
  } = useApiResource(() => fetchNeighbors(), [], { intervalMs: NEIGHBOR_POLL_MS });

  const [protocolFilter, setProtocolFilter] = useState<ProtocolFilter>('all');
  const [search, setSearch] = useState('');

  const filtered = useMemo(() => {
    if (!neighbors) return [];
    const q = search.trim().toLowerCase();
    return neighbors.filter((n) => {
      if (protocolFilter !== 'all' && n.protocol !== protocolFilter) return false;
      if (!q) return true;
      return (
        n.localDevice.toLowerCase().includes(q) ||
        n.remoteDevice.toLowerCase().includes(q) ||
        n.remoteChassisId.toLowerCase().includes(q) ||
        n.remotePort.toLowerCase().includes(q)
      );
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
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <header className="flex items-baseline justify-between">
            <div>
              <h1 className="text-2xl font-semibold text-white">Neighbors</h1>
              <p className="text-sm text-gray-400">
                Live discovery (CDP / LLDP / EDP / FDP) collected by the running simulation.
              </p>
            </div>
            <div className="text-xs text-gray-500">
              Polling every {NEIGHBOR_POLL_MS / 1000}s · {neighbors?.length ?? 0} entries
            </div>
          </header>

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
                        ? 'bg-cyan-500/20 text-cyan-200 ring-1 ring-cyan-400/40'
                        : 'bg-gray-800/60 text-gray-300 hover:bg-gray-800'
                    }`}
                  >
                    {p === 'all' ? 'All' : p}
                    <span className="ml-1.5 text-[10px] text-gray-500">{count}</span>
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
              className="ml-auto w-64 rounded border border-white/5 bg-gray-950/60 px-3 py-1.5 text-sm text-gray-100 placeholder-gray-500 focus:border-cyan-400 focus:outline-none"
              aria-label="Filter neighbors"
            />
          </div>
        </CardContent>
      </Card>

      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="p-0">
          {loading && !neighbors && <div className="p-6 text-sm text-gray-400">Loading…</div>}
          {error && (
            <div className="p-6 text-sm text-red-300" role="alert">
              Failed to load neighbors: {error.message}
            </div>
          )}
          {neighbors && filtered.length === 0 && (
            <div className="p-6 text-sm text-gray-500">
              {neighbors.length === 0
                ? 'No neighbors discovered yet. Start a simulation that has CDP/LLDP/EDP/FDP enabled to populate this table.'
                : 'No neighbors match the current filters.'}
            </div>
          )}
          {filtered.length > 0 && (
            <table className="w-full text-sm">
              <thead className="bg-gray-950/40 text-left text-xs uppercase tracking-wider text-gray-400">
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
                {filtered.map((n, idx) => (
                  <tr
                    key={`${n.localDevice}-${n.remoteDevice}-${n.protocol}-${idx}`}
                    className="text-gray-200 hover:bg-gray-950/40"
                  >
                    <td className="px-4 py-2 font-mono text-xs text-cyan-300">{n.protocol}</td>
                    <td className="px-4 py-2">{n.localDevice}</td>
                    <td className="px-4 py-2">{n.remoteDevice}</td>
                    <td className="px-4 py-2 text-gray-400">{n.remotePort || '—'}</td>
                    <td className="px-4 py-2 font-mono text-xs text-gray-400">
                      {n.remoteChassisId || '—'}
                    </td>
                    <td className="px-4 py-2 font-mono text-xs text-gray-400">
                      {n.managementAddress || '—'}
                    </td>
                    <td className="px-4 py-2 text-gray-400">{formatTtl(n.ttl)}</td>
                    <td className="px-4 py-2 text-gray-400">{formatRelative(n.lastSeen)}</td>
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
