import { Download, Layers, Network, RefreshCw } from 'lucide-react';
import { type FC, useCallback, useState } from 'react';
import { fetchDevices, fetchNeighbors, fetchTopology } from '../api/client';
import type { DeviceSummary } from '../api/types';
import { NeighborDiscoveryTable } from '../components/topology/NeighborDiscoveryTable';
import { TopologyCanvas } from '../components/topology/TopologyCanvas';
import type { DeviceNode, LinkEdge } from '../components/topology/topologyTypes';
import { exportTopologyAsJson } from '../components/topology/topologyUtils';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, SmallText } from '../ui/Typography';

// ============================================================================
// Main Topology Page Component
// ============================================================================

export const TopologyPage: FC = () => {
  const { data: topology, loading: topologyLoading } = useApiResource(fetchTopology, [], {
    intervalMs: 15000,
  });
  const { data: devices, loading: devicesLoading } = useApiResource(fetchDevices, [], {
    intervalMs: 15000,
  });
  const { data: neighbors } = useApiResource(fetchNeighbors, [], {
    intervalMs: 15000,
  });

  const [selectedDevice, setSelectedDevice] = useState<DeviceSummary | null>(null);
  const [showLegend, setShowLegend] = useState(true);
  const [showMinimap, setShowMinimap] = useState(true);
  const [currentNodes, setCurrentNodes] = useState<DeviceNode[]>([]);
  const [currentEdges, setCurrentEdges] = useState<LinkEdge[]>([]);

  // Export topology as JSON
  const handleExport = useCallback(() => {
    exportTopologyAsJson(currentNodes, currentEdges);
  }, [currentNodes, currentEdges]);

  // Refresh data
  const handleRefresh = useCallback(() => {
    window.location.reload();
  }, []);

  const loading = topologyLoading || devicesLoading;

  return (
    <div className="space-y-6">
      {/* Header Card */}
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-cyan-500/20">
                <Network className="w-6 h-6 text-cyan-400" />
              </div>
              <div>
                <H2 className="mb-0">Network Topology</H2>
                <SmallText className="text-gray-400">
                  {devices?.length || 0} devices | {topology?.links.length || 0} connections
                </SmallText>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRefresh}
                leftIcon={<RefreshCw className="w-4 h-4" />}
                disabled={loading}
              >
                Refresh
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleExport}
                leftIcon={<Download className="w-4 h-4" />}
                disabled={currentNodes.length === 0}
              >
                Export
              </Button>
              <Button
                variant={showMinimap ? 'outline' : 'ghost'}
                size="sm"
                onClick={() => setShowMinimap(!showMinimap)}
                leftIcon={<Layers className="w-4 h-4" />}
              >
                Minimap
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Topology Visualization */}
      <Card className="border-white/5 bg-gray-900/70 overflow-hidden">
        <div className="h-[600px] relative">
          <TopologyCanvas
            devices={devices}
            topology={topology}
            loading={loading}
            showMinimap={showMinimap}
            showLegend={showLegend}
            onToggleLegend={() => setShowLegend(!showLegend)}
            selectedDevice={selectedDevice}
            onSelectDevice={setSelectedDevice}
            onNodesChange={setCurrentNodes}
            onEdgesChange={setCurrentEdges}
          />
        </div>
      </Card>

      {/* Neighbor Discovery Table */}
      {neighbors && neighbors.length > 0 && <NeighborDiscoveryTable neighbors={neighbors} />}
    </div>
  );
};

export default TopologyPage;
