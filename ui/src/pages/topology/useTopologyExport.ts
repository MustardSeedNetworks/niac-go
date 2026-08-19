/**
 * Topology export.
 *
 * Four formats over two sources: JSON and PNG capture what is on canvas, DOT
 * and GraphML come from the daemon's own rendering so the output reflects what
 * the running simulation sees rather than what this browser laid out. Grouped
 * here because they share one error surface and one download mechanism, and
 * because TopologyPage is over the file-size red flag without them.
 */

import { useCallback, useState } from 'react';
import { exportTopology } from '../../api/client';
import type { DeviceNode, LinkEdge } from './types';

/** Filename stem; the caller-visible date makes exports self-identifying. */
function fileName(extension: string): string {
  return `niac-topology-${new Date().toISOString().slice(0, 10)}.${extension}`;
}

/** download hands the browser a blob or data URL under a chosen name. */
function download(href: string, extension: string): void {
  const link = document.createElement('a');
  link.href = href;
  link.download = fileName(extension);
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

/** downloadBlob wraps content in a blob, downloads it, and releases the URL. */
function downloadBlob(content: string, mimeType: string, extension: string): void {
  const url = URL.createObjectURL(new Blob([content], { type: mimeType }));
  download(url, extension);
  URL.revokeObjectURL(url);
}

export interface TopologyExport {
  /** Last export failure, or null. Cleared at the start of each attempt. */
  error: string | null;
  exportJSON: () => void;
  exportDOT: () => void;
  exportGraphML: () => void;
  exportPNG: () => void;
}

export function useTopologyExport(nodes: DeviceNode[], edges: LinkEdge[]): TopologyExport {
  const [error, setError] = useState<string | null>(null);

  const exportJSON = useCallback(() => {
    const snapshot = {
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
    downloadBlob(JSON.stringify(snapshot, null, 2), 'application/json', 'json');
  }, [nodes, edges]);

  // Server-side export: the daemon renders the topology it is actually
  // simulating, suitable for feeding into Graphviz / yEd / gephi.
  const exportServerFormat = useCallback(
    async (format: 'dot' | 'graphml', extension: string, mimeType: string) => {
      setError(null);
      try {
        downloadBlob(await exportTopology(format), mimeType, extension);
      } catch (err) {
        setError((err as Error).message);
      }
    },
    [],
  );

  const exportDOT = useCallback(
    () => void exportServerFormat('dot', 'dot', 'text/vnd.graphviz'),
    [exportServerFormat],
  );
  const exportGraphML = useCallback(
    () => void exportServerFormat('graphml', 'graphml', 'application/xml'),
    [exportServerFormat],
  );

  // html-to-image walks the canvas DOM, inlines computed styles and
  // rasterises. 2× pixel ratio so the image stays sharp on Retina/4K.
  const exportPNG = useCallback(async () => {
    setError(null);
    const root = document.querySelector<HTMLElement>('.react-flow');
    if (!root) {
      setError('Could not find the topology canvas to export.');
      return;
    }
    try {
      const { toPng } = await import('html-to-image');
      const dataUrl = await toPng(root, {
        pixelRatio: 2,
        backgroundColor: '#0a0a0a',
        cacheBust: true,
        // Skip the ReactFlow controls + minimap chrome — viewers want the
        // diagram, not the UI scaffolding.
        filter: (node) =>
          !(node instanceof HTMLElement) || !node.classList?.contains?.('react-flow__panel'),
      });
      download(dataUrl, 'png');
    } catch (err) {
      setError((err as Error).message || 'Failed to export topology as PNG.');
    }
  }, []);

  return {
    error,
    exportJSON,
    exportDOT,
    exportGraphML,
    exportPNG: () => void exportPNG(),
  };
}
