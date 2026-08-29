/**
 * TrunkEdge — the custom topology edge that renders the VLAN/speed middle
 * label, the per-side interface labels, and the hover tooltip.
 *
 * The VLAN run-collapsing is the part worth pinning: "VLANs 10-12,20" and
 * "VLANs 10,11,12,20" are both plausible-looking outputs, and only one is
 * right, so these assert the rendered string rather than that a label exists.
 */

import { render, screen } from '@testing-library/react';
import { Position, ReactFlowProvider } from '@xyflow/react';
import type { ReactNode } from 'react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';

// EdgeLabelRenderer portals into a node that only exists inside a mounted
// <ReactFlow>. Rendering its children inline keeps the labels queryable without
// standing up a whole flow canvas -- everything else (BaseEdge, path geometry)
// stays real.
vi.mock('@xyflow/react', async () => {
  const actual = await vi.importActual<typeof import('@xyflow/react')>('@xyflow/react');
  return {
    ...actual,
    EdgeLabelRenderer: ({ children }: { children: ReactNode }) => <>{children}</>,
  };
});

import i18n from '../../i18n';
import { TrunkEdge } from './TrunkEdge';
import type { LinkEdgeData } from './types';

function renderEdge(data: Partial<LinkEdgeData> = {}) {
  return render(
    <I18nextProvider i18n={i18n}>
      <ReactFlowProvider>
        <svg aria-hidden="true">
          <TrunkEdge
            id="e1"
            source="a"
            target="b"
            sourceX={0}
            sourceY={0}
            targetX={200}
            targetY={100}
            sourcePosition={Position.Bottom}
            targetPosition={Position.Top}
            data={data as LinkEdgeData}
          />
        </svg>
      </ReactFlowProvider>
    </I18nextProvider>,
  );
}

describe('middle label', () => {
  it('renders nothing when there is no vlan or speed', () => {
    renderEdge({});

    expect(screen.queryByText(/VLAN/)).toBeNull();
  });

  it('names a single VLAN in the singular', () => {
    renderEdge({ vlans: [10] });

    expect(screen.getByText(/VLAN 10/)).toBeDefined();
  });

  it('collapses a contiguous run into a range', () => {
    renderEdge({ vlans: [10, 11, 12] });

    expect(screen.getByText(/VLANs 10-12/)).toBeDefined();
  });

  it('keeps non-contiguous VLANs separate and sorts them', () => {
    renderEdge({ vlans: [30, 10, 20] });

    expect(screen.getByText(/VLANs 10,20,30/)).toBeDefined();
  });

  it('mixes runs and singletons', () => {
    renderEdge({ vlans: [10, 11, 12, 20, 30, 31] });

    expect(screen.getByText(/VLANs 10-12,20,30-31/)).toBeDefined();
  });

  it('renders a two-VLAN run as a range, not a pair', () => {
    renderEdge({ vlans: [10, 11] });

    expect(screen.getByText(/VLANs 10-11/)).toBeDefined();
  });

  it('formats speed in G above a gigabit and M below', () => {
    renderEdge({ speed: '1000' });
    expect(screen.getByText(/1G/)).toBeDefined();

    renderEdge({ speed: '100' });
    expect(screen.getByText(/100M/)).toBeDefined();
  });

  it('leaves a dangling separator when the speed is present but unformattable', () => {
    // '0' is a truthy string, so it passes the `if (linkData.speed)` guard and
    // formatSpeed's '' is still joined in -- the label reads 'VLAN 10 · '.
    // Pinned as found; the guard should test the formatted value, not the raw
    // field.
    renderEdge({ vlans: [10], speed: '0' });

    expect(screen.getByText('VLAN 10 ·')).toBeDefined();
  });

  it('joins vlan and speed with a separator', () => {
    renderEdge({ vlans: [10], speed: '1000' });

    expect(screen.getByText(/VLAN 10 · 1G/)).toBeDefined();
  });

  it('hides every label when showLabels is false', () => {
    renderEdge({ vlans: [10], speed: '1000', showLabels: false });

    expect(screen.queryByText(/VLAN 10/)).toBeNull();
  });
});

describe('per-side interface labels', () => {
  it('renders both interface names', () => {
    renderEdge({ sourceInterface: 'Gi0/1', targetInterface: 'Gi0/2' });

    expect(screen.getByText('Gi0/1')).toBeDefined();
    expect(screen.getByText('Gi0/2')).toBeDefined();
  });

  it('renders neither when the interfaces are unknown', () => {
    const { container } = renderEdge({});

    expect(container.textContent).not.toContain('Gi0/');
  });
});

describe('stroke styling', () => {
  it('dashes a discovered edge', () => {
    const { container } = renderEdge({ discovered: true });

    const path = container.querySelector('path.react-flow__edge-path');
    expect(path?.getAttribute('style') ?? '').toContain('6 4');
  });

  it('leaves a configured edge solid', () => {
    const { container } = renderEdge({ discovered: false });

    const path = container.querySelector('path.react-flow__edge-path');
    expect(path?.getAttribute('style') ?? '').not.toContain('6 4');
  });

  it('applies the focus opacity when the edge is dimmed', () => {
    const { container } = renderEdge({ focusOpacity: 0.2 });

    const path = container.querySelector('path.react-flow__edge-path');
    expect(path?.getAttribute('style') ?? '').toContain('0.2');
  });
});

describe('hover tooltip', () => {
  it('is absent until the edge is hovered', () => {
    renderEdge({ speed: '1000', hovered: false });

    expect(screen.queryByText(/Speed/i)).toBeNull();
  });

  it('lists the populated link attributes when hovered', () => {
    renderEdge({
      hovered: true,
      sourceInterface: 'Gi0/1',
      targetInterface: 'Gi0/2',
      vlans: [10, 20],
      speed: '1000',
      duplex: 'full',
      status: 'up',
      linkType: 'trunk',
      utilizationPercent: 42,
    });

    expect(screen.getByText('Gi0/1 ↔ Gi0/2')).toBeDefined();
    expect(screen.getByText('10, 20')).toBeDefined();
    expect(screen.getByText('1G')).toBeDefined();
    expect(screen.getByText('full')).toBeDefined();
    expect(screen.getByText('up')).toBeDefined();
    expect(screen.getByText('trunk')).toBeDefined();
    expect(screen.getByText('42 %')).toBeDefined();
  });

  it('omits utilisation when it is zero or absent', () => {
    // Zero means "not reported", so a "0 %" row would assert something the
    // daemon never said.
    renderEdge({ hovered: true, speed: '1000', utilizationPercent: 0 });

    expect(screen.queryByText('0 %')).toBeNull();
  });

  it('shows an interface row with a placeholder when only one side is known', () => {
    renderEdge({ hovered: true, sourceInterface: 'Gi0/1' });

    expect(screen.getByText('Gi0/1 ↔ ?')).toBeDefined();
  });
});
