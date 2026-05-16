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
import { Download, Layers, Network, Radar, RefreshCw } from 'lucide-react';
import { fetchDevices, fetchNeighbors, fetchTopology } from '../api/client';
import type { DeviceSummary } from '../api/types';
import { topologyDeviceColors as deviceColors } from '../constants/device-types';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, SmallText } from '../ui/Typography';
import {
  createEdges,
  DeviceDetailsPanel,
  DeviceNode,
  type DeviceNodeData,
  type DeviceNodeType,
  type LinkEdge,
  layoutNodes,
  NeighborsView,
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

// Where we stash user-dragged node positions so they survive page
// reloads. Keyed by device name → {x, y}. SSR-safe — no-ops if window
// is undefined.
const TOPOLOGY_POSITIONS_KEY = 'niac.topology.positions';

function readSavedPositions(): Record<string, { x: number; y: number }> {
  if (typeof window === 'undefined') return {};
  try {
    const raw = window.localStorage.getItem(TOPOLOGY_POSITIONS_KEY);
    if (!raw) return {};
    return JSON.parse(raw) as Record<string, { x: number; y: number }>;
  } catch {
    return {};
  }
}

function writeSavedPositions(positions: Record<string, { x: number; y: number }>) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(TOPOLOGY_POSITIONS_KEY, JSON.stringify(positions));
  } catch {
    // Quota / privacy mode — silently skip; positions will reset on reload.
  }
}

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
  // Minimap defaults off — it's distracting for small graphs and most
  // users will pan with the mouse anyway. Toggle on via the header
  // button when working with larger topologies.
  const [showMinimap, setShowMinimap] = useState(false);
  const [view, setView] = useState<'graph' | 'neighbors'>('graph');

  // Build graph from API data, merging trunk port links with neighbor-discovered links
  useEffect(() => {
    if (!(devices && topology)) {
      return;
    }

    // The daemon returns null arrays when no simulation is loaded; coerce
    // so spread / map / Set initialization don't crash with TypeError.
    const topologyLinks = topology.links ?? [];

    // Start with trunk port links from config
    const allLinks = [...topologyLinks];

    // Merge neighbor-discovered edges that aren't already represented by trunk ports
    if (neighbors && neighbors.length > 0) {
      const existingEdges = new Set(
        topologyLinks.map((l) => [l.source, l.target].sort().join('|')),
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

    // Preserve user-dragged positions across the 15s data poll. For each
    // device that's already on canvas, keep its current position rather
    // than blowing it away with the freshly-computed layout. Brand-new
    // devices that arrived since last render get the fresh layout slot.
    // Also pulls any previously-saved positions out of localStorage so
    // drags survive a page reload.
    setNodes((current) => {
      const positionByName = new Map<string, { x: number; y: number }>();
      for (const node of current) {
        positionByName.set(node.id, node.position);
      }
      const stored = readSavedPositions();
      return layoutedNodes.map((node) => {
        const existing = positionByName.get(node.id);
        const saved = stored[node.id];
        const position = existing ?? saved ?? node.position;
        return { ...node, position };
      });
    });
    setEdges(layoutedEdges);
  }, [devices, topology, neighbors, setNodes, setEdges]);

  // Persist node positions when the user stops dragging so layouts
  // survive page reloads. Keyed by device name; stored as a flat
  // {name: {x, y}} map in localStorage.
  const handleNodeDragStop = useCallback(() => {
    const positions: Record<string, { x: number; y: number }> = {};
    for (const node of nodes) {
      positions[node.id] = { x: node.position.x, y: node.position.y };
    }
    writeSavedPositions(positions);
  }, [nodes]);

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
                <H2>Network Topology</H2>
                <SmallText className="text-gray-400">
                  {devices?.length || 0} devices | {topology?.links?.length || 0} connections
                </SmallText>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <div
                className="inline-flex rounded-lg border border-white/10 bg-gray-950/40 p-0.5"
                role="tablist"
                aria-label="Topology view"
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={view === 'graph'}
                  onClick={() => setView('graph')}
                  className={`flex items-center gap-1.5 rounded px-3 py-1 text-xs font-medium transition-colors ${
                    view === 'graph'
                      ? 'bg-cyan-500/20 text-cyan-100'
                      : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
                  }`}
                >
                  <Network className="w-3.5 h-3.5" />
                  Graph
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={view === 'neighbors'}
                  onClick={() => setView('neighbors')}
                  className={`flex items-center gap-1.5 rounded px-3 py-1 text-xs font-medium transition-colors ${
                    view === 'neighbors'
                      ? 'bg-cyan-500/20 text-cyan-100'
                      : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
                  }`}
                >
                  <Radar className="w-3.5 h-3.5" />
                  Neighbors
                </button>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRefresh}
                leftIcon={<RefreshCw className="w-4 h-4" />}
                disabled={loading}
              >
                Refresh
              </Button>
              {view === 'graph' && (
                <>
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
                </>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Topology Visualization — sized to fill the available viewport
          minus the page chrome (header card + breadcrumbs + page header
          ≈ 240 px on this layout). Gives much more room for the graph
          on standard 1080p+ displays. */}
      {view === 'graph' && (
        <Card className="border-white/5 bg-gray-900/70 overflow-hidden">
          <div className="h-[calc(100vh-260px)] min-h-[520px] relative">
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
                      <code className="text-yellow-100">port_channels:</code> entries to your YAML
                      to visualise connections. Live LLDP/CDP/EDP/FDP neighbours are on the{' '}
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
                  onNodeDragStop={handleNodeDragStop}
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

                  {/* Legend Panel — only meaningful when there are edges
                      to colour. With zero edges the link-speed key is
                      just noise. */}
                  {edges.length > 0 && (
                    <Panel position="top-left">
                      <TopologyLegend
                        show={showLegend}
                        onToggle={() => setShowLegend(!showLegend)}
                      />
                    </Panel>
                  )}

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
      )}

      {/* Neighbors view — replaces the standalone /neighbors page */}
      {view === 'neighbors' && <NeighborsView />}
    </div>
  );
};

export default TopologyPage;
