/**
 * useTopologyExport — the topology export menu's four formats.
 *
 * JSON is built in the browser, DOT and GraphML come from the daemon, and PNG
 * rasterises the canvas. Each has its own failure path, and a failed export
 * that silently does nothing is the worst outcome, so the error surface is
 * asserted for every one.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { DeviceNode, LinkEdge } from './types';
import { useTopologyExport } from './useTopologyExport';

const exportTopology = vi.fn();
vi.mock('../../api/client', () => ({
  exportTopology: (format: string) => exportTopology(format),
}));

const toPng = vi.fn();
vi.mock('html-to-image', () => ({ toPng: (...args: unknown[]) => toPng(...args) }));

/** Records what each anchor download was handed. */
let downloads: { href: string; filename: string }[];
let revoked: string[];

const nodes = [
  {
    id: 'sw1',
    position: { x: 1, y: 2 },
    data: { type: 'switch', ips: ['10.0.0.1'], protocols: ['SNMP'] },
  },
] as unknown as DeviceNode[];

const edges = [
  { source: 'sw1', target: 'sw2', label: 'trunk', data: { speed: '1000' } },
] as unknown as LinkEdge[];

beforeEach(() => {
  downloads = [];
  revoked = [];
  exportTopology.mockReset();
  toPng.mockReset();

  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: () => 'blob:mock',
    revokeObjectURL: (url: string) => revoked.push(url),
  });

  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    downloads.push({ href: this.href, filename: this.download });
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  document.body.innerHTML = '';
});

describe('JSON export', () => {
  it('downloads a .json snapshot and releases the object URL', () => {
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportJSON();
    });

    expect(downloads).toHaveLength(1);
    expect(downloads[0]?.filename).toMatch(/\.json$/);
    expect(revoked).toEqual(['blob:mock']);
  });

  it('does not need the daemon', () => {
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportJSON();
    });

    expect(exportTopology).not.toHaveBeenCalled();
  });
});

describe('server-rendered formats', () => {
  it('requests DOT and downloads it with the right extension', async () => {
    exportTopology.mockResolvedValue('digraph {}');
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportDOT();
    });

    expect(exportTopology).toHaveBeenCalledWith('dot');
    await waitFor(() => expect(downloads[0]?.filename).toMatch(/\.dot$/));
  });

  it('requests GraphML and downloads it with the right extension', async () => {
    exportTopology.mockResolvedValue('<graphml/>');
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportGraphML();
    });

    expect(exportTopology).toHaveBeenCalledWith('graphml');
    await waitFor(() => expect(downloads[0]?.filename).toMatch(/\.graphml$/));
  });

  it('surfaces a daemon failure instead of failing silently', async () => {
    exportTopology.mockRejectedValue(new Error('daemon said no'));
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportDOT();
    });

    await waitFor(() => expect(result.current.error).toBe('daemon said no'));
    expect(downloads).toHaveLength(0);
  });

  it('clears a previous error at the start of the next attempt', async () => {
    exportTopology.mockRejectedValueOnce(new Error('first failed'));
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportDOT();
    });
    await waitFor(() => expect(result.current.error).toBe('first failed'));

    exportTopology.mockResolvedValue('digraph {}');
    act(() => {
      result.current.exportDOT();
    });

    await waitFor(() => expect(result.current.error).toBeNull());
  });
});

describe('PNG export', () => {
  /** Stands in for the ReactFlow canvas the exporter looks for. */
  function mountCanvas(): void {
    const root = document.createElement('div');
    root.className = 'react-flow';
    document.body.appendChild(root);
  }

  it('reports a missing canvas rather than doing nothing', async () => {
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportPNG();
    });

    await waitFor(() => expect(result.current.error).toMatch(/canvas/i));
    expect(toPng).not.toHaveBeenCalled();
  });

  it('rasterises the canvas and downloads a .png', async () => {
    mountCanvas();
    toPng.mockResolvedValue('data:image/png;base64,AAA');
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportPNG();
    });

    await waitFor(() => expect(downloads[0]?.filename).toMatch(/\.png$/));
    expect(result.current.error).toBeNull();
  });

  it('renders at 2x and excludes the ReactFlow chrome', async () => {
    mountCanvas();
    toPng.mockResolvedValue('data:image/png;base64,AAA');
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportPNG();
    });

    await waitFor(() => expect(toPng).toHaveBeenCalled());
    const options = toPng.mock.calls[0]?.[1] as {
      pixelRatio: number;
      filter: (n: unknown) => boolean;
    };
    expect(options.pixelRatio).toBe(2);

    // The panel chrome (controls, minimap) is filtered out; the diagram is not.
    const panel = document.createElement('div');
    panel.className = 'react-flow__panel';
    expect(options.filter(panel)).toBe(false);

    const diagram = document.createElement('div');
    expect(options.filter(diagram)).toBe(true);

    // A non-element node has no classList and must be kept, not crash.
    expect(options.filter(document.createTextNode('x'))).toBe(true);
  });

  it('surfaces a rasterisation failure', async () => {
    mountCanvas();
    toPng.mockRejectedValue(new Error('tainted canvas'));
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportPNG();
    });

    await waitFor(() => expect(result.current.error).toBe('tainted canvas'));
  });

  it('falls back to a generic message when the failure has none', async () => {
    mountCanvas();
    toPng.mockRejectedValue(new Error(''));
    const { result } = renderHook(() => useTopologyExport(nodes, edges));

    act(() => {
      result.current.exportPNG();
    });

    await waitFor(() => expect(result.current.error).toMatch(/Failed to export/));
  });
});
