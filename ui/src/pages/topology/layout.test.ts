/**
 * Tests for topology layout, link parsing and edge styling.
 *
 * These assert produced values — positions, colours, widths, marker sets —
 * rather than counts. The heatmap defects in the sibling product were all
 * cases where the count was right and every value was wrong.
 */

import { describe, expect, it } from 'vitest';
import type { DeviceSummary, TopologyLink } from '../../api/types';
import {
  createEdges,
  getEdgeColor,
  getLinkSpeed,
  getLinkType,
  getVlan,
  layoutNodes,
  UTILIZATION_CRITICAL_COLOR,
  UTILIZATION_HIGH_COLOR,
} from './layout';

const device = (name: string): DeviceSummary => ({ name }) as DeviceSummary;

/** Unwraps the single edge createEdges() produced, so assertions stay direct. */
function onlyEdge(links: TopologyLink[]) {
  const [edge] = createEdges(links);
  if (!edge?.data || !edge.style) throw new Error('createEdges produced no edge');
  return { ...edge, data: edge.data, style: edge.style };
}

const link = (over: Partial<TopologyLink> = {}): TopologyLink =>
  ({ source: 'a', target: 'b', ...over }) as TopologyLink;

describe('getLinkSpeed', () => {
  it('returns nothing for an absent or unparseable label', () => {
    expect(getLinkSpeed()).toBeUndefined();
    expect(getLinkSpeed('')).toBeUndefined();
    expect(getLinkSpeed('uplink')).toBeUndefined();
  });

  it('defaults a bare number to megabits', () => {
    expect(getLinkSpeed('100')).toBe('100');
  });

  it('scales G and T suffixes', () => {
    expect(getLinkSpeed('1G')).toBe('1000');
    expect(getLinkSpeed('10G')).toBe('10000');
    expect(getLinkSpeed('1T')).toBe('1000000');
  });

  it('accepts a lowercase suffix', () => {
    expect(getLinkSpeed('1g')).toBe('1000');
  });

  it('reads the first number out of a longer label', () => {
    expect(getLinkSpeed('trunk 10G uplink')).toBe('10000');
  });

  it('prefers the speed token over a VLAN id earlier in the label', () => {
    // A label that names the VLAN first used to yield the VLAN as the speed,
    // because the match was unanchored: 'vlan 10 1G' parsed as 10 Mbps. Only
    // reachable on API responses that omit the typed speed field.
    expect(getLinkSpeed('vlan 10 1G')).toBe('1000');
    expect(getLinkSpeed('vlan 200 10G uplink')).toBe('10000');
  });

  it('skips a VLAN id when no number in the label carries a unit', () => {
    expect(getLinkSpeed('vlan 10 100')).toBe('100');
  });
});

describe('getLinkType', () => {
  it('returns nothing without a label', () => {
    expect(getLinkType()).toBeUndefined();
    expect(getLinkType('')).toBeUndefined();
  });

  it('detects trunk and lag case-insensitively', () => {
    expect(getLinkType('Trunk')).toBe('trunk');
    expect(getLinkType('LAG 1')).toBe('lag');
    expect(getLinkType('Po1')).toBe('lag');
  });

  it('returns nothing for a plain access link', () => {
    expect(getLinkType('access')).toBeUndefined();
  });

  it('prefers trunk when a label mentions both', () => {
    expect(getLinkType('trunk po1')).toBe('trunk');
  });
});

describe('getVlan', () => {
  it('returns nothing when there is no vlan in the label', () => {
    expect(getVlan()).toBeUndefined();
    expect(getVlan('trunk')).toBeUndefined();
  });

  it('parses vlan with and without a space, any case', () => {
    expect(getVlan('vlan10')).toBe(10);
    expect(getVlan('VLAN 20')).toBe(20);
    expect(getVlan('trunk vlan 300 uplink')).toBe(300);
  });
});

describe('getEdgeColor', () => {
  it('uses the trunk colour for both trunk and lag', () => {
    expect(getEdgeColor({ linkType: 'trunk' })).toBe(getEdgeColor({ linkType: 'lag' }));
  });

  it('falls back to the muted border for an unknown speed', () => {
    expect(getEdgeColor({ speed: '999999' })).toBe('var(--color-border-muted)');
  });

  it('falls back when there is no speed at all', () => {
    expect(getEdgeColor({})).toBe('var(--color-border-muted)');
  });
});

describe('layoutNodes', () => {
  it('returns nothing for no devices', () => {
    expect(layoutNodes([], [])).toEqual([]);
  });

  it('falls back to the grid when there are no links', () => {
    const nodes = layoutNodes([device('a'), device('b')], []);

    expect(nodes.map((n) => n.id)).toEqual(['a', 'b']);
    // Two devices give a 2-wide grid, so both sit on the same row.
    expect(nodes[0]?.position.y).toBe(nodes[1]?.position.y);
    expect(nodes[1]?.position.x).toBeGreaterThan(nodes[0]?.position.x ?? 0);
  });

  it('wraps the grid at ceil(sqrt(n)) columns', () => {
    const nodes = layoutNodes([device('a'), device('b'), device('c'), device('d')], []);

    // 4 devices -> 2 columns, so the third device starts a new row.
    expect(nodes[2]?.position.y).toBeGreaterThan(nodes[0]?.position.y ?? 0);
    expect(nodes[2]?.position.x).toBe(nodes[0]?.position.x);
  });

  it('uses the grid when explicitly asked, even with links present', () => {
    const links = [link()];
    const grid = layoutNodes([device('a'), device('b')], links, 'grid');
    const noLinks = layoutNodes([device('a'), device('b')], []);

    expect(grid.map((n) => n.position)).toEqual(noLinks.map((n) => n.position));
  });

  it('places linked devices hierarchically rather than on a grid row', () => {
    const nodes = layoutNodes([device('a'), device('b')], [link()], 'hierarchical');

    expect(nodes).toHaveLength(2);
    // A directed a->b link ranks b below a.
    const a = nodes.find((n) => n.id === 'a');
    const b = nodes.find((n) => n.id === 'b');
    expect(b?.position.y).toBeGreaterThan(a?.position.y ?? 0);
  });
});

describe('createEdges', () => {
  it('prefers the daemon-supplied fields over label parsing', () => {
    const edge = onlyEdge([
      link({ label: 'vlan 10 1G access', speed: '40000', linkType: 'trunk', vlans: [99] }),
    ]);

    expect(edge.data.speed).toBe('40000');
    expect(edge.data.linkType).toBe('trunk');
    expect(edge.data.vlan).toBe(99);
  });

  it('falls back to parsing the label when the structured fields are absent', () => {
    const edge = onlyEdge([link({ label: 'trunk 1G' })]);

    expect(edge.data.speed).toBe('1000');
    expect(edge.data.linkType).toBe('trunk');
  });

  it('falls back to the label when vlans is present but empty', () => {
    const edge = onlyEdge([link({ label: 'vlan 7', vlans: [] })]);

    expect(edge.data.vlan).toBe(7);
  });

  it('gives trunk and lag links arrows at both ends and animates them', () => {
    const trunk = onlyEdge([link({ linkType: 'trunk' })]);
    const access = onlyEdge([link({ linkType: undefined, label: 'access' })]);

    expect(trunk.animated).toBe(true);
    expect(trunk.markerStart).toBeDefined();
    expect(trunk.style.strokeWidth).toBe(3);

    expect(access.animated).toBe(false);
    expect(access.markerStart).toBeUndefined();
    expect(access.style.strokeWidth).toBe(2);
  });

  it('builds a unique id per link, including parallel links', () => {
    const edges = createEdges([link(), link()]);

    expect(edges[0]?.id).toBe('e-a-b-0');
    expect(edges[1]?.id).toBe('e-a-b-1');
  });

  it('carries the per-side interface and vlan hydration through to the edge data', () => {
    const edge = onlyEdge([
      link({
        sourceInterface: 'Gi0/1',
        targetInterface: 'Gi0/2',
        vlans: [10, 20],
        discovered: true,
        utilizationPercent: 30,
      }),
    ]);

    expect(edge.data.sourceInterface).toBe('Gi0/1');
    expect(edge.data.targetInterface).toBe('Gi0/2');
    expect(edge.data.vlans).toEqual([10, 20]);
    expect(edge.data.discovered).toBe(true);
  });
});

describe('utilisation styling', () => {
  /** Reads the rendered stroke/width for a given utilisation. */
  function styleFor(utilizationPercent: number | undefined) {
    const edge = onlyEdge([link({ label: 'access', utilizationPercent })]);
    return { stroke: edge.style.stroke, width: edge.style.strokeWidth };
  }

  it('leaves an unknown or idle link at the base width and colour', () => {
    expect(styleFor(undefined).width).toBe(2);
    expect(styleFor(0).width).toBe(2);
    expect(styleFor(undefined).stroke).toBe('var(--color-border-muted)');
  });

  it('keeps the base width below 25 percent', () => {
    expect(styleFor(24).width).toBe(2);
  });

  it('thickens to 3 px from 25 percent without recolouring', () => {
    expect(styleFor(25).width).toBe(3);
    expect(styleFor(59).width).toBe(3);
    expect(styleFor(59).stroke).toBe('var(--color-border-muted)');
  });

  it('turns amber at 60 percent', () => {
    expect(styleFor(60).stroke).toBe(UTILIZATION_HIGH_COLOR);
    expect(styleFor(60).width).toBe(4);
    expect(styleFor(84).stroke).toBe(UTILIZATION_HIGH_COLOR);
  });

  it('turns red at 85 percent', () => {
    expect(styleFor(85).stroke).toBe(UTILIZATION_CRITICAL_COLOR);
    expect(styleFor(85).width).toBe(5);
    expect(styleFor(100).stroke).toBe(UTILIZATION_CRITICAL_COLOR);
  });

  it('never narrows a trunk below its 3 px base', () => {
    // A lightly loaded trunk must not render thinner than an idle one.
    const edge = onlyEdge([link({ linkType: 'trunk', utilizationPercent: 10 })]);
    expect(edge.style.strokeWidth).toBe(3);
  });
});
