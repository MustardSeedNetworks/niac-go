import type { Edge, Node } from '@xyflow/react';

// ============================================================================
// Types - Using proper React Flow generic patterns
// ============================================================================

export interface DeviceNodeData extends Record<string, unknown> {
  label: string;
  type: string;
  ips?: string[];
  protocols?: string[];
  status?: 'online' | 'offline' | 'warning';
  selected?: boolean;
  onClick?: (id: string) => void;
}

export interface LinkEdgeData extends Record<string, unknown> {
  label?: string;
  speed?: string;
  duplex?: string;
  vlan?: number;
  linkType?: 'trunk' | 'access' | 'lag' | 'standard';
  status?: 'up' | 'down' | 'degraded';
}

// Full node and edge types for React Flow
export type DeviceNode = Node<DeviceNodeData, 'device'>;
export type LinkEdge = Edge<LinkEdgeData>;

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
