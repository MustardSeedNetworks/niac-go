import {
  addEdge,
  Background,
  BackgroundVariant,
  type Connection,
  Controls,
  type Edge,
  MarkerType,
  MiniMap,
  type Node,
  type NodeTypes,
  Panel,
  ReactFlow,
  useEdgesState,
  useNodesState,
} from "@xyflow/react";
import { type FC, memo, useCallback, useEffect, useState } from "react";
import "@xyflow/react/dist/style.css";
import { Download, Eye, EyeOff, Layers, Network, RefreshCw, Wifi } from "lucide-react";
import { fetchDevices, fetchNeighbors, fetchTopology } from "../api/client";
import type { DeviceSummary, TopologyLink } from "../api/types";
import {
  topologyDeviceColors as deviceColors,
  topologyDeviceIcons as deviceIcons,
} from "../constants/deviceTypes";
import { useApiResource } from "../hooks/useApiResource";
import { Button, Card, CardContent, H2, SmallText, Tag } from "../ui";

// ============================================================================
// Types - Using proper React Flow generic patterns
// ============================================================================

interface DeviceNodeData extends Record<string, unknown> {
  label: string;
  type: string;
  ips?: string[];
  protocols?: string[];
  status?: "online" | "offline" | "warning";
  selected?: boolean;
  onClick?: (id: string) => void;
}

interface LinkEdgeData extends Record<string, unknown> {
  label?: string;
  speed?: string;
  duplex?: string;
  vlan?: number;
  linkType?: "trunk" | "access" | "lag" | "standard";
  status?: "up" | "down" | "degraded";
}

// Full node and edge types for React Flow
type DeviceNode = Node<DeviceNodeData, "device">;
type LinkEdge = Edge<LinkEdgeData>;

const linkSpeedColors: Record<string, string> = {
  "10": "var(--color-link-10m)",
  "100": "var(--color-link-100m)",
  "1000": "var(--color-link-1g)",
  "10000": "var(--color-link-10g)",
  "25000": "var(--color-link-25g)",
  "40000": "var(--color-link-40g)",
  "100000": "var(--color-link-100g)",
  trunk: "var(--color-link-trunk)",
};

// ============================================================================
// Custom Device Node Component
// ============================================================================

const DeviceNode: FC<{ data: DeviceNodeData; selected?: boolean }> = memo(({ data, selected }) => {
  const deviceType = (data.type as string)?.toLowerCase() || "unknown";
  const Icon = deviceIcons[deviceType] || deviceIcons.unknown;
  const color = deviceColors[deviceType] || deviceColors.unknown;

  const statusColor = {
    online: "bg-green-500",
    offline: "bg-gray-500",
    warning: "bg-yellow-500",
  }[data.status || "online"];

  return (
    <div
      className={`
        relative px-4 py-3 rounded-xl border-2 transition-all duration-200
        ${selected ? "ring-2 ring-brand-500 ring-offset-2 ring-offset-gray-900" : ""}
        hover:shadow-lg hover:shadow-black/30
      `}
      style={{
        backgroundColor: "var(--color-bg-elevated)",
        borderColor: selected ? color : "var(--color-border-muted)",
        minWidth: "140px",
      }}
      onClick={() => data.onClick?.(data.label)}
    >
      {/* Status indicator */}
      <div
        className={`absolute -top-1 -right-1 w-3 h-3 rounded-full ${statusColor} border-2 border-gray-900`}
      />

      {/* Icon and name */}
      <div className="flex items-center gap-3">
        <div
          className="p-2 rounded-lg"
          style={{ backgroundColor: `color-mix(in srgb, ${color} 20%, transparent)` }}
        >
          <Icon className={`w-5 h-5`} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="font-semibold text-white text-sm truncate">{data.label}</div>
          <div className="text-xs text-gray-400 capitalize">{data.type}</div>
        </div>
      </div>

      {/* IPs */}
      {data.ips && data.ips.length > 0 && (
        <div className="mt-2 pt-2 border-t border-white/10">
          <div className="text-xs font-mono text-gray-400 truncate">
            {data.ips[0]}
            {data.ips.length > 1 && <span className="text-gray-500"> +{data.ips.length - 1}</span>}
          </div>
        </div>
      )}

      {/* Protocols */}
      {data.protocols && data.protocols.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {data.protocols.slice(0, 3).map((proto) => (
            <span
              key={proto}
              className="px-1.5 py-0.5 text-[10px] rounded-md bg-white/10 text-gray-300"
            >
              {proto}
            </span>
          ))}
          {data.protocols.length > 3 && (
            <span className="px-1.5 py-0.5 text-[10px] rounded-md bg-white/5 text-gray-500">
              +{data.protocols.length - 3}
            </span>
          )}
        </div>
      )}
    </div>
  );
});

DeviceNode.displayName = "DeviceNode";

// ============================================================================
// Node Types
// ============================================================================

const nodeTypes: NodeTypes = {
  device: DeviceNode,
};

// ============================================================================
// Layout Algorithm (Simple Force-Directed)
// ============================================================================

function layoutNodes(devices: DeviceSummary[], links: TopologyLink[]): DeviceNode[] {
  const nodeMap = new Map<string, { x: number; y: number }>();

  // Create adjacency list
  const adjacency = new Map<string, Set<string>>();
  for (const device of devices) {
    adjacency.set(device.name, new Set());
  }
  for (const link of links) {
    adjacency.get(link.source)?.add(link.target);
    adjacency.get(link.target)?.add(link.source);
  }

  // Calculate degree for each node
  const degrees = new Map<string, number>();
  for (const [name, neighbors] of adjacency) {
    degrees.set(name, neighbors.size);
  }

  // Sort by degree (most connected first)
  const sorted = [...devices].sort(
    (a, b) => (degrees.get(b.name) || 0) - (degrees.get(a.name) || 0),
  );

  // Layout in concentric circles based on connectivity
  const centerX = 400;
  const centerY = 300;
  let currentRadius = 0;
  let angleOffset = 0;
  let nodesInRing = 1;
  let ringIndex = 0;

  for (let i = 0; i < sorted.length; i++) {
    const device = sorted[i];

    if (ringIndex >= nodesInRing) {
      ringIndex = 0;
      currentRadius += 180;
      angleOffset += Math.PI / 6; // Stagger rings
      nodesInRing = Math.max(1, Math.floor((2 * Math.PI * currentRadius) / 200));
    }

    const angle = (2 * Math.PI * ringIndex) / nodesInRing + angleOffset;
    const x = centerX + currentRadius * Math.cos(angle);
    const y = centerY + currentRadius * Math.sin(angle);

    nodeMap.set(device.name, { x, y });
    ringIndex++;
  }

  // Create nodes with layout positions
  return devices.map((device, index) => {
    const pos = nodeMap.get(device.name) || {
      x: 100 + (index % 5) * 200,
      y: 100 + Math.floor(index / 5) * 150,
    };
    return {
      id: device.name,
      type: "device",
      position: pos,
      data: {
        label: device.name,
        type: device.type || "unknown",
        ips: device.ips,
        protocols: device.protocols,
        status: "online",
      } as DeviceNodeData,
    };
  });
}

function createEdges(links: TopologyLink[]): LinkEdge[] {
  return links.map((link, index) => {
    // Parse link label for speed/type info
    const data: LinkEdgeData = {
      label: link.label,
    };

    // Try to extract speed from label
    const speedMatch = link.label?.match(/(\d+)([MGT])?/i);
    if (speedMatch) {
      const num = parseInt(speedMatch[1], 10);
      const unit = speedMatch[2]?.toUpperCase() || "M";
      const multiplier = unit === "G" ? 1000 : unit === "T" ? 1000000 : 1;
      data.speed = String(num * multiplier);
    }

    // Check for trunk/lag indicators
    if (link.label?.toLowerCase().includes("trunk")) {
      data.linkType = "trunk";
    } else if (
      link.label?.toLowerCase().includes("lag") ||
      link.label?.toLowerCase().includes("po")
    ) {
      data.linkType = "lag";
    }

    // Extract VLAN if present
    const vlanMatch = link.label?.match(/vlan\s*(\d+)/i);
    if (vlanMatch) {
      data.vlan = parseInt(vlanMatch[1], 10);
    }

    // Determine edge color
    let edgeColor = "var(--color-border-muted)";
    if (data.linkType === "trunk" || data.linkType === "lag") {
      edgeColor = linkSpeedColors.trunk;
    } else if (data.speed && linkSpeedColors[data.speed]) {
      edgeColor = linkSpeedColors[data.speed];
    }

    return {
      id: `e-${link.source}-${link.target}-${index}`,
      source: link.source,
      target: link.target,
      type: "smoothstep",
      animated: data.linkType === "trunk" || data.linkType === "lag",
      label: link.label,
      labelBgPadding: [8, 4] as [number, number],
      labelBgBorderRadius: 4,
      labelBgStyle: {
        fill: "var(--color-bg-overlay)",
        fillOpacity: 0.9,
      },
      labelStyle: {
        fill: "var(--color-text-secondary)",
        fontSize: 10,
        fontWeight: 500,
      },
      style: {
        stroke: edgeColor,
        strokeWidth: data.linkType === "trunk" || data.linkType === "lag" ? 3 : 2,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: edgeColor,
        width: 15,
        height: 15,
      },
      data,
    };
  });
}

// ============================================================================
// Topology Legend Component
// ============================================================================

const TopologyLegend: FC<{ show: boolean; onToggle: () => void }> = ({ show, onToggle }) => {
  if (!show) {
    return (
      <button
        onClick={onToggle}
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-800/90 border border-white/10 text-sm text-gray-300 hover:bg-gray-700/90 transition-colors"
      >
        <Eye className="w-4 h-4" />
        Show Legend
      </button>
    );
  }

  return (
    <div className="bg-gray-800/95 backdrop-blur-sm border border-white/10 rounded-xl p-4 min-w-[200px]">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm font-semibold text-white">Legend</span>
        <button onClick={onToggle} className="text-gray-400 hover:text-white">
          <EyeOff className="w-4 h-4" />
        </button>
      </div>

      <div className="space-y-4">
        {/* Device Types */}
        <div>
          <div className="text-xs text-gray-400 uppercase tracking-wide mb-2">Device Types</div>
          <div className="space-y-1.5">
            {[
              { type: "router", label: "Router" },
              { type: "switch", label: "Switch" },
              { type: "firewall", label: "Firewall" },
              { type: "server", label: "Server" },
              { type: "workstation", label: "Workstation" },
              { type: "access-point", label: "Access Point" },
            ].map(({ type, label }) => {
              const Icon = deviceIcons[type] || Network;
              const color = deviceColors[type];
              return (
                <div key={type} className="flex items-center gap-2">
                  <div className="w-4 h-4 flex items-center justify-center">
                    <Icon className="w-4 h-4 text-current" />
                  </div>
                  <span className="text-xs text-gray-300">{label}</span>
                  <div
                    className="w-2 h-2 rounded-full ml-auto"
                    style={{ backgroundColor: color }}
                  />
                </div>
              );
            })}
          </div>
        </div>

        {/* Link Speeds */}
        <div>
          <div className="text-xs text-gray-400 uppercase tracking-wide mb-2">Link Speeds</div>
          <div className="space-y-1.5">
            {[
              { label: "100M", color: "var(--color-link-100m)" },
              { label: "1G", color: "var(--color-link-1g)" },
              { label: "10G", color: "var(--color-link-10g)" },
              { label: "Trunk/LAG", color: "var(--color-link-trunk)" },
            ].map(({ label, color }) => (
              <div key={label} className="flex items-center gap-2">
                <div className="w-6 h-0.5 rounded" style={{ backgroundColor: color }} />
                <span className="text-xs text-gray-300">{label}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// Device Details Panel
// ============================================================================

const DeviceDetailsPanel: FC<{
  device: DeviceSummary | null;
  onClose: () => void;
  onEdit?: (device: DeviceSummary) => void;
}> = ({ device, onClose, onEdit }) => {
  if (!device) return null;

  const deviceType = device.type?.toLowerCase() || "unknown";
  const Icon = deviceIcons[deviceType] || deviceIcons.unknown;
  const color = deviceColors[deviceType] || deviceColors.unknown;

  return (
    <div className="absolute top-4 right-4 w-80 bg-gray-800/95 backdrop-blur-sm border border-white/10 rounded-xl p-4 shadow-2xl z-50">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className="p-2 rounded-lg"
            style={{ backgroundColor: `color-mix(in srgb, ${color} 20%, transparent)` }}
          >
            <Icon className="w-6 h-6" />
          </div>
          <div>
            <h3 className="font-semibold text-white">{device.name}</h3>
            <p className="text-sm text-gray-400 capitalize">{device.type}</p>
          </div>
        </div>
        <button onClick={onClose} className="text-gray-400 hover:text-white text-xl">
          &times;
        </button>
      </div>

      {device.ips && device.ips.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-gray-400 uppercase tracking-wide mb-1">IP Addresses</div>
          <div className="space-y-1">
            {device.ips.map((ip) => (
              <code key={ip} className="block text-sm text-blue-300 font-mono">
                {ip}
              </code>
            ))}
          </div>
        </div>
      )}

      {device.protocols && device.protocols.length > 0 && (
        <div className="mb-4">
          <div className="text-xs text-gray-400 uppercase tracking-wide mb-1">Protocols</div>
          <div className="flex flex-wrap gap-1">
            {device.protocols.map((proto) => (
              <Tag key={proto} colorScheme="purple" className="text-xs">
                {proto}
              </Tag>
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-2">
        <Button size="sm" tone="violet" onClick={() => onEdit?.(device)} className="flex-1">
          Edit Device
        </Button>
        <Button size="sm" variant="outline" onClick={onClose}>
          Close
        </Button>
      </div>
    </div>
  );
};

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
  const { data: neighbors } = useApiResource(fetchNeighbors, [], { intervalMs: 15000 });

  const [nodes, setNodes, onNodesChange] = useNodesState<DeviceNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<LinkEdge>([]);
  const [selectedDevice, setSelectedDevice] = useState<DeviceSummary | null>(null);
  const [showLegend, setShowLegend] = useState(true);
  const [showMinimap, setShowMinimap] = useState(true);

  // Build graph from API data
  useEffect(() => {
    if (!devices || !topology) return;

    const layoutedNodes = layoutNodes(devices, topology.links);
    const layoutedEdges = createEdges(topology.links);

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);
  }, [devices, topology, setNodes, setEdges]);

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

  // Handle new connections (for editing)
  const onConnect = useCallback(
    (params: Connection) => {
      const newEdge: LinkEdge = {
        ...params,
        id: `e-${params.source}-${params.target}-new`,
        source: params.source || "",
        target: params.target || "",
        type: "smoothstep",
        style: { stroke: "var(--color-link-1g)", strokeWidth: 2 },
        data: { linkType: "standard" } as LinkEdgeData,
      };
      setEdges((eds) => addEdge(newEdge, eds) as LinkEdge[]);
    },
    [setEdges],
  );

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
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `niac-topology-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [nodes, edges]);

  // Refresh data
  const handleRefresh = useCallback(() => {
    window.location.reload();
  }, []);

  const loading = topologyLoading || devicesLoading;

  // Minimap node color function
  const getMinimapNodeColor = useCallback((node: Node) => {
    const nodeData = node.data as DeviceNodeData;
    const nodeType = typeof nodeData?.type === "string" ? nodeData.type.toLowerCase() : "unknown";
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
                variant={showMinimap ? "outline" : "ghost"}
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
            <ReactFlow
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              nodeTypes={nodeTypes}
              fitView
              fitViewOptions={{ padding: 0.2 }}
              minZoom={0.2}
              maxZoom={2}
              defaultEdgeOptions={{
                type: "smoothstep",
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
                  pannable
                  zoomable
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
                    window.location.href = `/device-config/${device.name}`;
                  }}
                />
              )}
            </ReactFlow>
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
                      key={`${neighbor.LocalDevice}-${neighbor.RemoteDevice}-${idx}`}
                      className="hover:bg-white/5"
                    >
                      <td className="px-4 py-3 font-semibold text-white">{neighbor.LocalDevice}</td>
                      <td className="px-4 py-3 text-white">{neighbor.RemoteDevice}</td>
                      <td className="px-4 py-3">
                        <Tag
                          colorScheme={
                            neighbor.Protocol === "LLDP"
                              ? "purple"
                              : neighbor.Protocol === "CDP"
                                ? "blue"
                                : neighbor.Protocol === "EDP"
                                  ? "green"
                                  : "gray"
                          }
                        >
                          {neighbor.Protocol}
                        </Tag>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs">{neighbor.RemotePort || "—"}</td>
                      <td className="px-4 py-3 font-mono text-xs text-blue-300">
                        {neighbor.ManagementAddress || "—"}
                      </td>
                      <td className="px-4 py-3 text-gray-400">{neighbor.TTL}s</td>
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
