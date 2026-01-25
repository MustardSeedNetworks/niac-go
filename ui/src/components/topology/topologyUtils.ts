import { MarkerType } from '@xyflow/react';
import type { DeviceSummary, TopologyLink } from '../../api/types';
import type { DeviceNode, DeviceNodeData, LinkEdge, LinkEdgeData } from './topologyTypes';
import { linkSpeedColors } from './topologyTypes';

// ============================================================================
// Layout Algorithm (Simple Force-Directed)
// ============================================================================

export function layoutNodes(devices: DeviceSummary[], links: TopologyLink[]): DeviceNode[] {
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

  for (const device of sorted) {
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
      type: 'device',
      position: pos,
      data: {
        label: device.name,
        type: device.type || 'unknown',
        ips: device.ips,
        protocols: device.protocols,
        status: 'online',
      } as DeviceNodeData,
    };
  });
}

// ============================================================================
// Link Parsing Utilities
// ============================================================================

export const getLinkSpeed = (label?: string): string | undefined => {
  if (!label) {
    return;
  }
  const speedMatch = /(\d+)([MGT])?/i.exec(label);
  if (!speedMatch) {
    return;
  }
  const num = Number.parseInt(speedMatch[1], 10);
  const unit = speedMatch[2]?.toUpperCase() || 'M';
  const multiplier = unit === 'G' ? 1000 : unit === 'T' ? 1000000 : 1;
  return String(num * multiplier);
};

export const getLinkType = (label?: string): LinkEdgeData['linkType'] | undefined => {
  const normalized = label?.toLowerCase();
  if (!normalized) {
    return;
  }
  if (normalized.includes('trunk')) {
    return 'trunk';
  }
  if (normalized.includes('lag') || normalized.includes('po')) {
    return 'lag';
  }
  return;
};

export const getVlan = (label?: string): number | undefined => {
  if (!label) {
    return;
  }
  const vlanMatch = /vlan\s*(\d+)/i.exec(label);
  if (!vlanMatch) {
    return;
  }
  return Number.parseInt(vlanMatch[1], 10);
};

export const getEdgeColor = (data: LinkEdgeData): string => {
  if (data.linkType === 'trunk' || data.linkType === 'lag') {
    return linkSpeedColors.trunk;
  }
  if (data.speed && linkSpeedColors[data.speed]) {
    return linkSpeedColors[data.speed];
  }
  return 'var(--color-border-muted)';
};

// ============================================================================
// Edge Creation
// ============================================================================

export function createEdges(links: TopologyLink[]): LinkEdge[] {
  return links.map((link, index) => {
    // Parse link label for speed/type info
    const data: LinkEdgeData = {
      label: link.label,
    };

    data.speed = getLinkSpeed(link.label);
    data.linkType = getLinkType(link.label);
    data.vlan = getVlan(link.label);

    const edgeColor = getEdgeColor(data);

    return {
      id: `e-${link.source}-${link.target}-${index}`,
      source: link.source,
      target: link.target,
      type: 'smoothstep',
      animated: data.linkType === 'trunk' || data.linkType === 'lag',
      label: link.label,
      labelBgPadding: [8, 4] as [number, number],
      labelBgBorderRadius: 4,
      labelBgStyle: {
        fill: 'var(--color-bg-overlay)',
        fillOpacity: 0.9,
      },
      labelStyle: {
        fill: 'var(--color-text-secondary)',
        fontSize: 10,
        fontWeight: 500,
      },
      style: {
        stroke: edgeColor,
        strokeWidth: data.linkType === 'trunk' || data.linkType === 'lag' ? 3 : 2,
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
// Export Utilities
// ============================================================================

export function exportTopologyAsJson(nodes: DeviceNode[], edges: LinkEdge[]): void {
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
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `niac-topology-${new Date().toISOString().slice(0, 10)}.json`;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  URL.revokeObjectURL(url);
}
