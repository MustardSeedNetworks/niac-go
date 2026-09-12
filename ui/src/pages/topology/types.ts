/**
 * Type definitions for the topology visualization
 */

import type { Edge, Node } from '@xyflow/react';

/**
 * Data for device nodes in the topology graph
 */
export interface DeviceNodeData extends Record<string, unknown> {
  /**
   * Which rank the hierarchical layout put this device in, counting from the
   * top.
   *
   * The tier bands used to recover this from the device's y position, which
   * worked only while one rank was one row. A rank too wide for a screen is
   * now wrapped onto several rows, and reading y would report each row as a
   * tier of its own.
   */
  rank?: number;
  label: string;
  type: string;
  ips?: string[];
  protocols?: string[];
  selected?: boolean;
  onClick?: (id: string) => void;
}

/**
 * Data for link edges in the topology graph
 */
export interface LinkEdgeData extends Record<string, unknown> {
  label?: string;
  speed?: string;
  duplex?: string;
  vlan?: number;
  vlans?: number[];
  /** Local interface name on the edge's source device (e.g. "Gi0/1"). */
  sourceInterface?: string;
  /** Local interface name on the edge's target device (e.g. "Gi0/1"). */
  targetInterface?: string;
  /**
   * Where this edge's labels go, in canvas units, decided for the whole graph
   * at once by `placeLabels`.
   *
   * An edge placing its own labels cannot tell that one is landing on a device
   * it has no connection to, or on the label of the edge beside it. A missing
   * entry means the label could not be placed clear of everything else and is
   * not drawn (#2104).
   */
  labelPlacement?: {
    source?: { x: number; y: number };
    middle?: { x: number; y: number };
    target?: { x: number; y: number };
  };
  linkType?: 'trunk' | 'access' | 'lag' | 'standard';
  status?: 'up' | 'down' | 'degraded';
  /** When false, the custom edge component hides all its labels —
   *  driven by the "Show labels" toggle in the topology header. */
  showLabels?: boolean;
  /** True when the link was inferred from runtime LLDP/CDP/EDP/FDP
   *  discovery (not declared in trunk_ports:). TrunkEdge renders
   *  these dashed so the user can tell design from runtime. */
  discovered?: boolean;
  /** Opacity multiplier for the neighbourhood-highlight feature.
   *  1 = fully opaque (default), 0.15 = faded background. Driven by
   *  the selected-node state in TopologyPage. */
  focusOpacity?: number;
  /** True when this edge is currently being hovered. Driven by
   *  ReactFlow's onEdgeMouseEnter / Leave at the page level; the
   *  TrunkEdge component renders the rich tooltip while true. */
  hovered?: boolean;
  /** Link utilization, 0–100. When set, drives a stroke-width and
   *  colour-tint adjustment in createEdges so saturating links pop
   *  visually. Undefined or 0 → default styling. */
  utilizationPercent?: number;
}

/**
 * Full node type for React Flow with device data
 */
export type DeviceNode = Node<DeviceNodeData, 'device'>;

/**
 * Full edge type for React Flow with link data
 */
export type LinkEdge = Edge<LinkEdgeData>;

/**
 * Link speed colors for different network speeds
 */
export const linkSpeedColors: Record<string, string> = {
  '10': 'var(--color-link-10m)',
  '100': 'var(--color-link-100m)',
  '1000': 'var(--color-link-1g)',
  '10000': 'var(--color-link-10g)',
  '25000': 'var(--color-link-25g)',
  '40000': 'var(--color-link-40g)',
  '100000': 'var(--color-link-100g)',
  trunk: 'var(--color-link-trunk)',
};
