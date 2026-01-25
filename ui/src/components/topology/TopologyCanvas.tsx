import {
  addEdge,
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
import '@xyflow/react/dist/style.css';
import { Network, RefreshCw } from 'lucide-react';
import { type FC, useCallback, useEffect } from 'react';
import type { DeviceSummary, TopologyGraph } from '../../api/types';
import { topologyDeviceColors as deviceColors } from '../../constants/device-types';
import { SmallText } from '../../ui/Typography';
import { DeviceDetailsPanel } from './DeviceDetailsPanel';
import { TopologyLegend } from './TopologyLegend';
import { TopologyNode } from './TopologyNode';
import type { DeviceNode, DeviceNodeData, LinkEdge, LinkEdgeData } from './topologyTypes';
import { createEdges, layoutNodes } from './topologyUtils';

// ============================================================================
// Node Types
// ============================================================================

const nodeTypes: NodeTypes = {
  device: TopologyNode,
};

// ============================================================================
// Topology Canvas Props
// ============================================================================

interface TopologyCanvasProps {
  devices: DeviceSummary[] | null | undefined;
  topology: TopologyGraph | null | undefined;
  loading: boolean;
  showMinimap: boolean;
  showLegend: boolean;
  onToggleLegend: () => void;
  selectedDevice: DeviceSummary | null;
  onSelectDevice: (device: DeviceSummary | null) => void;
  onNodesChange?: (nodes: DeviceNode[]) => void;
  onEdgesChange?: (edges: LinkEdge[]) => void;
}

// ============================================================================
// Topology Canvas Component
// ============================================================================

export const TopologyCanvas: FC<TopologyCanvasProps> = ({
  devices,
  topology,
  loading,
  showMinimap,
  showLegend,
  onToggleLegend,
  selectedDevice,
  onSelectDevice,
  onNodesChange: onNodesChangeCallback,
  onEdgesChange: onEdgesChangeCallback,
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState<DeviceNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<LinkEdge>([]);

  // Build graph from API data
  useEffect(() => {
    if (!(devices && topology)) {
      return;
    }

    const layoutedNodes = layoutNodes(devices, topology.links);
    const layoutedEdges = createEdges(topology.links);

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);

    // Notify parent of node/edge changes
    onNodesChangeCallback?.(layoutedNodes);
    onEdgesChangeCallback?.(layoutedEdges);
  }, [devices, topology, setNodes, setEdges, onNodesChangeCallback, onEdgesChangeCallback]);

  // Handle node selection
  const handleNodeClick = useCallback(
    (deviceName: string) => {
      const device = devices?.find((d) => d.name === deviceName);
      onSelectDevice(device || null);
    },
    [devices, onSelectDevice],
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

  // Handle new connections (for editing)
  const onConnect = useCallback(
    (params: Connection) => {
      const newEdge: LinkEdge = {
        ...params,
        id: `e-${params.source}-${params.target}-new`,
        source: params.source || '',
        target: params.target || '',
        type: 'smoothstep',
        style: { stroke: 'var(--color-link-1g)', strokeWidth: 2 },
        data: { linkType: 'standard' } as LinkEdgeData,
      };
      setEdges((eds) => addEdge(newEdge, eds) as LinkEdge[]);
    },
    [setEdges],
  );

  // Minimap node color function
  const getMinimapNodeColor = useCallback((node: Node) => {
    const nodeData = node.data as DeviceNodeData;
    const nodeType = typeof nodeData?.type === 'string' ? nodeData.type.toLowerCase() : 'unknown';
    return deviceColors[nodeType] || deviceColors.unknown;
  }, []);

  if (loading) {
    return (
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <RefreshCw className="w-8 h-8 text-brand-400 animate-spin" />
          <SmallText className="text-gray-400">Loading topology...</SmallText>
        </div>
      </div>
    );
  }

  if (nodes.length === 0) {
    return (
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="text-center">
          <Network className="w-16 h-16 text-gray-600 mx-auto mb-4" />
          <p className="text-gray-400 mb-2">No topology data available</p>
          <SmallText className="text-gray-500">
            Configure devices with trunk ports or port-channels to visualize connections
          </SmallText>
        </div>
      </div>
    );
  }

  return (
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
        <TopologyLegend show={showLegend} onToggle={onToggleLegend} />
      </Panel>

      {/* Selected Device Panel */}
      {selectedDevice && (
        <DeviceDetailsPanel
          device={selectedDevice}
          onClose={() => onSelectDevice(null)}
          onEdit={(device) => {
            window.location.href = `/device-config/${device.name}`;
          }}
        />
      )}
    </ReactFlow>
  );
};
