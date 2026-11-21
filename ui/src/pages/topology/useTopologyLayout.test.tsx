import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { DeviceSummary, NeighborRecord, TopologyGraph } from '../../api/types';
import type { TopologyFilter } from './graph';
import * as layoutModule from './layout';
import { useTopologyLayout } from './useTopologyLayout';

vi.mock('./layout', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./layout')>();
  return { ...actual, layoutNodes: vi.fn(actual.layoutNodes) };
});

const devices: DeviceSummary[] = [
  { name: 'COS-A', type: 'switch', ips: [], protocols: [], properties: { site: 'COS' } },
  { name: 'COS-B', type: 'switch', ips: [], protocols: [], properties: { site: 'COS' } },
  { name: 'EVT-A', type: 'switch', ips: [], protocols: [], properties: { site: 'EVT' } },
];
const topology: TopologyGraph = {
  nodes: devices.map((d) => ({ name: d.name, type: d.type })),
  links: [{ source: 'COS-A', target: 'COS-B', label: 'Gi0/1' }],
};
const neighbors: NeighborRecord[] = [];
const allSites: TopologyFilter = {
  hidden: new Set(),
  types: new Set(),
  search: '',
  site: null,
};

const layoutRuns = () => vi.mocked(layoutModule.layoutNodes).mock.calls.length;

function render(initial: Partial<Parameters<typeof useTopologyLayout>[0]> = {}) {
  return renderHook((props) => useTopologyLayout(props), {
    initialProps: {
      devices,
      topology,
      neighbors,
      filter: allSites,
      layoutMode: 'hierarchical' as const,
      revision: 0,
      ...initial,
    },
  });
}

describe('useTopologyLayout', () => {
  beforeEach(() => {
    vi.mocked(layoutModule.layoutNodes).mockClear();
  });

  it('lays the graph out once, and again for none of an unrelated re-render', () => {
    const { result, rerender } = render();
    const first = result.current;

    rerender({
      devices,
      topology,
      neighbors,
      filter: allSites,
      layoutMode: 'hierarchical',
      revision: 0,
    });
    rerender({
      devices,
      topology,
      neighbors,
      filter: allSites,
      layoutMode: 'hierarchical',
      revision: 0,
    });

    expect(layoutRuns()).toBe(1);
    expect(result.current).toBe(first);
  });

  it('lays out again when a filter, the layout mode or the revision changes', () => {
    const { result, rerender } = render();
    const base = { devices, topology, neighbors, revision: 0 } as const;

    rerender({ ...base, filter: { ...allSites, site: 'COS' }, layoutMode: 'hierarchical' });
    expect(layoutRuns()).toBe(2);
    expect(result.current?.nodes.map((n) => n.id).sort()).toEqual(['COS-A', 'COS-B']);

    rerender({ ...base, filter: { ...allSites, site: 'COS' }, layoutMode: 'grid' });
    expect(layoutRuns()).toBe(3);

    rerender({ ...base, filter: { ...allSites, site: 'COS' }, layoutMode: 'grid', revision: 1 });
    expect(layoutRuns()).toBe(4);
  });

  it('has no layout until both devices and topology have loaded', () => {
    expect(render({ topology: null }).result.current).toBeNull();
    expect(render({ devices: null }).result.current).toBeNull();
    expect(layoutRuns()).toBe(0);
  });
});
