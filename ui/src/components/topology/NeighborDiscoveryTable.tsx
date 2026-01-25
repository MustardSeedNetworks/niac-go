import { Wifi } from 'lucide-react';
import type { FC } from 'react';
import type { NeighborRecord } from '../../api/types';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2 } from '../../ui/Typography';

// ============================================================================
// Neighbor Discovery Table
// ============================================================================

interface NeighborDiscoveryTableProps {
  neighbors: NeighborRecord[];
}

export const NeighborDiscoveryTable: FC<NeighborDiscoveryTableProps> = ({ neighbors }) => {
  if (!neighbors || neighbors.length === 0) {
    return null;
  }

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent>
        <H2 className="mb-4 flex items-center gap-2">
          <Wifi className="w-5 h-5 text-pink-400" />
          Discovered Neighbors
        </H2>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-white/10 text-sm">
            <thead className="bg-gray-950/60 text-xs uppercase tracking-wide text-gray-400">
              <tr>
                <th className="px-4 py-3 text-left">Local Device</th>
                <th className="px-4 py-3 text-left">Remote Device</th>
                <th className="px-4 py-3 text-left">Protocol</th>
                <th className="px-4 py-3 text-left">Remote Port</th>
                <th className="px-4 py-3 text-left">Management IP</th>
                <th className="px-4 py-3 text-left">TTL</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5 text-gray-300">
              {neighbors.map((neighbor, idx) => (
                <tr
                  key={`${neighbor.localDevice}-${neighbor.remoteDevice}-${idx}`}
                  className="hover:bg-white/5"
                >
                  <td className="px-4 py-3 font-semibold text-white">{neighbor.localDevice}</td>
                  <td className="px-4 py-3 text-white">{neighbor.remoteDevice}</td>
                  <td className="px-4 py-3">
                    <Tag
                      colorScheme={
                        neighbor.protocol === 'LLDP'
                          ? 'purple'
                          : neighbor.protocol === 'CDP'
                            ? 'blue'
                            : neighbor.protocol === 'EDP'
                              ? 'green'
                              : 'gray'
                      }
                    >
                      {neighbor.protocol}
                    </Tag>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{neighbor.remotePort || '-'}</td>
                  <td className="px-4 py-3 font-mono text-xs text-blue-300">
                    {neighbor.managementAddress || '-'}
                  </td>
                  <td className="px-4 py-3 text-gray-400">{neighbor.ttl}s</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
};
