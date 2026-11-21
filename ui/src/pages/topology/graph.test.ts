import { describe, expect, it } from 'vitest';
import type { DeviceSummary, NeighborRecord, TopologyLink } from '../../api/types';
import { buildLayout, filterGraph, mergeLinks, neighbourhood, siteOf, sitesOf } from './graph';

const device = (name: string, site?: string, type = 'switch'): DeviceSummary => ({
  name,
  type,
  ips: [],
  protocols: [],
  ...(site ? { properties: { site } } : {}),
});
const link = (source: string, target: string, over: Partial<TopologyLink> = {}): TopologyLink => ({
  source,
  target,
  label: '',
  ...over,
});
const neighbor = (localDevice: string, remoteDevice: string): NeighborRecord => ({
  protocol: 'LLDP',
  localDevice,
  remoteDevice,
  remotePort: '',
  remoteChassisId: '',
  description: '',
  capabilities: [],
  managementAddress: '',
  lastSeen: '',
  ttl: 0,
});
const noFilter = { hidden: new Set<string>(), types: new Set<string>(), search: '', site: null };

describe('mergeLinks', () => {
  it('adds a discovered link only for neighbour pairs the config does not declare', () => {
    const merged = mergeLinks(
      [link('a', 'b')],
      [neighbor('b', 'a'), neighbor('a', 'c'), neighbor('c', 'a')],
    );

    expect(merged.map((l) => [l.source, l.target, l.discovered])).toEqual([
      ['a', 'b', undefined],
      ['a', 'c', true],
    ]);
    expect(merged[1]?.label).toBe('LLDP discovery');
  });
});

describe('sites', () => {
  it('reads the site a pack stamps on a device', () => {
    expect(siteOf(device('a', 'COS'))).toBe('COS');
    expect(siteOf(device('a'))).toBeUndefined();
  });

  it('lists each site once, sorted, and ignores devices without one', () => {
    expect(
      sitesOf([device('a', 'EVT'), device('b', 'COS'), device('c', 'EVT'), device('d')]),
    ).toEqual(['COS', 'EVT']);
  });
});

describe('filterGraph', () => {
  const devices = [
    device('COS-CORE-01', 'COS', 'router'),
    device('COS-ACC-01', 'COS'),
    device('EVT-ACC-01', 'EVT'),
  ];
  const links = [link('COS-CORE-01', 'COS-ACC-01'), link('COS-ACC-01', 'EVT-ACC-01')];

  it('keeps one site and drops the links that leave it', () => {
    const shown = filterGraph(devices, links, { ...noFilter, site: 'COS' });

    expect(shown.devices.map((d) => d.name)).toEqual(['COS-CORE-01', 'COS-ACC-01']);
    expect(shown.links).toEqual([links[0]]);
  });

  it('combines site, type, search and hidden devices', () => {
    expect(
      filterGraph(devices, links, { ...noFilter, types: new Set(['router']) }).devices,
    ).toHaveLength(1);
    expect(filterGraph(devices, links, { ...noFilter, search: ' acc-01 ' }).devices).toHaveLength(
      2,
    );
    expect(
      filterGraph(devices, links, {
        ...noFilter,
        hidden: new Set(['EVT-ACC-01']),
        search: 'acc',
      }).devices.map((d) => d.name),
    ).toEqual(['COS-ACC-01']);
  });
});

describe('neighbourhood', () => {
  it('is the node and its direct peers on either end of a link', () => {
    const peers = neighbourhood([link('a', 'b'), link('c', 'a'), link('c', 'd')], 'a');

    expect([...peers].sort((a, b) => a.localeCompare(b))).toEqual(['a', 'b', 'c']);
  });
});

describe('buildLayout', () => {
  it('lays out, creates and places every edge of the graph it is given', () => {
    const devices = [device('a'), device('b'), device('c')];
    const links = [link('a', 'b', { sourceInterface: 'Gi0/1' }), link('b', 'c')];

    const layout = buildLayout(devices, links, 'hierarchical');

    expect(layout.nodes.map((n) => n.id).sort((a, b) => a.localeCompare(b))).toEqual([
      'a',
      'b',
      'c',
    ]);
    expect(layout.edges).toHaveLength(2);
    expect(layout.placements.size).toBe(2);
    expect(layout.links).toBe(links);
  });
});
