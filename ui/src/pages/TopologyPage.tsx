import {
  Background,
  BackgroundVariant,
  type Connection,
  Controls,
  MiniMap,
  type Node,
  type NodeTypes,
  Panel,
  ReactFlow,
  useEdgesState,
  useNodesState,
} from '@xyflow/react';
import { type FC, useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import '@xyflow/react/dist/style.css';
import { Download, Layers, Network, RefreshCw, Wifi } from 'lucide-react';
import { fetchDevices, fetchNeighbors, fetchTopology } from '../api/client';
import type { DeviceSummary } from '../api/types';
import { topologyDeviceColors as deviceColors } from '../constants/device-types';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
import {
  createEdges,
  DeviceDetailsPanel,
  DeviceNode,
  type DeviceNodeData,
  type DeviceNodeType,
  type LinkEdge,
  layoutNodes,
  TopologyLegend,
} from './topology';

/**
 * nodeTypes is the ReactFlow lookup that maps a node.type string onto
 * a React component. We register exactly one — every node is a device.
 * Kept here (not in topology/) because nodeTypes is consumed only by
 * the page-level ReactFlow instance and including it in the barrel
 * would force everyone who imports a single helper to pull DeviceNode.
 */
const nodeTypes: NodeTypes = {
  device: DeviceNode,
};

/**
 * TopologyPage renders the network graph with the device-node and
 * link-edge presenters defined in ./topology. The page itself owns
 * data fetching (topology + devices + neighbors), state for the
 * legend / minimap / selected device, and the export action.
 */
export const TopologyPage: FC = () => {
  const navigate = useNavigate();

  // Fetch topology data from the API with periodic polling
  const {
    data: topology,
    loading: topologyLoading,
    refetch: refetchTopology,
  } = useApiResource(fetchTopology, [], { intervalMs: 15000 });
  const {
    data: devices,
    loading: devicesLoading,
    refetch: refetchDevices,
  } = useApiResource(fetchDevices, [], { intervalMs: 15000 });
  const { data: neighbors, refetch: refetchNeighbors } = useApiResource(fetchNeighbors, [], {
    intervalMs: 15000,
  });

  const [nodes, setNodes, onNodesChange] = useNodesState<DeviceNodeType>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<LinkEdge>([]);
  const [selectedDevice, setSelectedDevice] = useState<DeviceSummary | null>(null);
  const [showLegend, setShowLegend] = useState(true);
  const [showMinimap, setShowMinimap] = useState(true);

  // Build graph from API data, merging trunk port links with neighbor-discovered links
  useEffect(() => {
    if (!(devices && topology)) {
      return;
    }

    // Start with trunk port links from config
    const allLinks = [...topology.links];

    // Merge neighbor-discovered edges that aren't already represented by trunk ports
    if (neighbors && neighbors.length > 0) {
      const existingEdges = new Set(
        topology.links.map((l) => [l.source, l.target].sort().join('|')),
      );

      for (const neighbor of neighbors) {
        const key = [neighbor.localDevice, neighbor.remoteDevice].sort().join('|');
        if (!existingEdges.has(key)) {
          existingEdges.add(key);
          allLinks.push({
            source: neighbor.localDevice,
            target: neighbor.remoteDevice,
            label: `${neighbor.protocol} discovery`,
          });
        }
      }
    }

    const layoutedNodes = layoutNodes(devices, allLinks);
    const layoutedEdges = createEdges(allLinks);

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);
  }, [devices, topology, neighbors, setNodes, setEdges]);

  // Handle node selection
  const handleNodeClick = useCallback(
    (deviceName: string) => {
      const device = devices?.find((d) => d.name === deviceName);
      setSelectedDevice(device || null);
    },
    [devices],
  );

  // Update node data with click handler
  useEffect(() => {
    setNodes((nds) =>
      nds.map((node) => ({
        ...node,
        data: {
          ...node.data,
          onClick: handleNodeClick,
        },
      })),
    );
  }, [handleNodeClick, setNodes]);

  // onConnect is a no-op: edges come from the backend and are not user-editable
  const onConnect = useCallback((_params: Connection) => {
    // Intentionally empty - topology edges are derived from device configs
    // and neighbor discovery; user-drawn edges would not persist.
  }, []);

  // Export topology as JSON
  const handleExport = useCallback(() => {
    const exportData = {
      nodes: nodes.map((n) => ({
        name: n.id,
        type: n.data.type,
        ips: n.data.ips,
        protocols: n.data.protocols,
        position: n.position,
      })),
      edges: edges.map((e) => ({
        source: e.source,
        target: e.target,
        label: e.label,
        data: e.data,
      })),
    };
    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `niac-topology-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [nodes, edges]);

  // Refresh data by refetching from the API
  const handleRefresh = useCallback(() => {
    refetchTopology();
    refetchDevices();
    refetchNeighbors();
  }, [refetchTopology, refetchDevices, refetchNeighbors]);

  const loading = topologyLoading || devicesLoading;

  // Minimap node color function
  const getMinimapNodeColor = useCallback((node: Node) => {
    const nodeData = node.data as DeviceNodeData;
    const nodeType = typeof nodeData?.type === 'string' ? nodeData.type.toLowerCase() : 'unknown';
    return deviceColors[nodeType] || deviceColors.unknown;
  }, []);

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
                disabled={nodes.length === 0}
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
          {loading ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="flex flex-col items-center gap-3">
                <RefreshCw className="w-8 h-8 text-brand-400 animate-spin" />
                <SmallText className="text-gray-400">Loading topology...</SmallText>
              </div>
            </div>
          ) : nodes.length === 0 ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <Network className="w-16 h-16 text-gray-600 mx-auto mb-4" />
                <p className="text-gray-400 mb-2">No topology data available</p>
                <SmallText className="text-gray-500">
                  Configure devices with trunk ports or port-channels to visualize connections
                </SmallText>
              </div>
            </div>
          ) : (
            <>
              {edges.length === 0 && (
                // z-50 wins over ReactFlow's internal Panel chrome and
                // pointer-events-none lets pan/zoom drag through the
                // banner so it doesn't trap clicks on the canvas below.
                <div className="pointer-events-none absolute top-0 left-0 right-0 z-50 bg-yellow-900/60 border-b border-yellow-500/40 px-4 py-2 text-center backdrop-blur-sm">
                  <SmallText className="text-yellow-200">
                    Devices loaded, but the running config has no declared topology links. Add{' '}
                    <code className="text-yellow-100">trunk_ports:</code> or{' '}
                    <code className="text-yellow-100">port_channels:</code> entries to your YAML to
                    visualise connections. Live LLDP/CDP/EDP/FDP neighbours are on the{' '}
                    <strong>Neighbors</strong> page.
                  </SmallText>
                </div>
              )}
              <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                nodeTypes={nodeTypes}
                fitView={true}
                fitViewOptions={{ padding: 0.2 }}
                minZoom={0.2}
                maxZoom={2}
                defaultEdgeOptions={{
                  type: 'smoothstep',
                }}
                proOptions={{ hideAttribution: true }}
              >
                <Background
                  variant={BackgroundVariant.Dots}
                  gap={20}
                  size={1}
                  color="rgba(255, 255, 255, 0.05)"
                />
                <Controls showZoom={true} showFitView={true} showInteractive={false} />
                {showMinimap && (
                  <MiniMap
                    nodeColor={getMinimapNodeColor}
                    maskColor="rgba(0, 0, 0, 0.8)"
                    pannable={true}
                    zoomable={true}
                  />
                )}

                {/* Legend Panel */}
                <Panel position="top-left">
                  <TopologyLegend show={showLegend} onToggle={() => setShowLegend(!showLegend)} />
                </Panel>

                {/* Selected Device Panel */}
                {selectedDevice && (
                  <DeviceDetailsPanel
                    device={selectedDevice}
                    onClose={() => setSelectedDevice(null)}
                    onEdit={(device) => {
                      navigate(`/device-config/${device.name}`);
                    }}
                  />
                )}
              </ReactFlow>
            </>
          )}
        </div>
      </Card>

      {/* Neighbor Discovery Table */}
      {neighbors && neighbors.length > 0 && (
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
                      <td className="px-4 py-3 font-mono text-xs">{neighbor.remotePort || '—'}</td>
                      <td className="px-4 py-3 font-mono text-xs text-blue-300">
                        {neighbor.managementAddress || '—'}
                      </td>
                      <td className="px-4 py-3 text-gray-400">{neighbor.ttl}s</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};

export default TopologyPage;
