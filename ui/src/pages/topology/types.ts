/**
 * Type definitions for the topology visualization
 */

import type { Edge, Node } from '@xyflow/react';

/**
 * Data for device nodes in the topology graph
 */
export interface DeviceNodeData extends Record<string, unknown> {
  label: string;
  type: string;
  ips?: string[];
  protocols?: string[];
  status?: 'online' | 'offline' | 'warning';
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
  linkType?: 'trunk' | 'access' | 'lag' | 'standard';
  status?: 'up' | 'down' | 'degraded';
  /** When false, the custom edge component hides all its labels —
   *  driven by the "Show labels" toggle in the topology header. */
  showLabels?: boolean;
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
