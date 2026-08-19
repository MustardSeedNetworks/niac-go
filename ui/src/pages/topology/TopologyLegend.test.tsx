/**
 * TopologyLegend.test.tsx — PR 3f-2: the legend must document every
 * visual encoding the graph actually renders (TrunkEdge.tsx line
 * style/color/arrowheads, layout.ts utilization tiers, DeviceNode.tsx
 * status dots, and the focus/dimming behavior wired in TopologyPage).
 * This locks the new entries so a future edit to the legend can't
 * silently drop one of them.
 */
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { TopologyLegend } from './TopologyLegend';

function renderLegend() {
  return render(
    <I18nextProvider i18n={i18n}>
      <TopologyLegend show onToggle={() => undefined} />
    </I18nextProvider>,
  );
}

describe('TopologyLegend — complete encoding coverage', () => {
  it('does not advertise node states the data layer cannot produce', () => {
    // Was 'documents node status dot colors'. The legend showed Online /
    // Offline / Warning swatches while every node was hardcoded 'online' and
    // DeviceSummary carried no status field at all — so two of the three
    // states were unreachable and the third was fabricated (niac-go#1352).
    // A legend that documents a non-existent encoding is what made the green
    // read as meaningful.
    renderLegend();

    expect(screen.queryByText('Online')).not.toBeInTheDocument();
    expect(screen.queryByText('Offline')).not.toBeInTheDocument();
    expect(screen.queryByText('Warning')).not.toBeInTheDocument();
  });

  it('documents discovered (dashed) vs declared (solid) edge line style (TrunkEdge.tsx)', () => {
    renderLegend();
    expect(screen.getByText('Line Style')).toBeInTheDocument();
    expect(screen.getByText('Declared (config)')).toBeInTheDocument();
    expect(screen.getByText('Discovered (LLDP/CDP)')).toBeInTheDocument();
  });

  it('documents the amber/red utilization tiers (layout.ts utilizationStyle)', () => {
    renderLegend();
    expect(screen.getByText('Utilization')).toBeInTheDocument();
    expect(screen.getByText('High (60–84%)')).toBeInTheDocument();
    expect(screen.getByText('Critical (85%+)')).toBeInTheDocument();
  });

  it('documents single vs double arrowhead directionality (layout.ts bidirectional)', () => {
    renderLegend();
    expect(screen.getByText('Direction')).toBeInTheDocument();
    expect(screen.getByText('Single (access link)')).toBeInTheDocument();
    expect(screen.getByText('Bidirectional (trunk/LAG)')).toBeInTheDocument();
  });

  it('documents the focus/dimming hint (TopologyPage nodeOpacity/edgeOpacity)', () => {
    renderLegend();
    expect(
      screen.getByText('Click a node to highlight its neighbors — unrelated devices dim.'),
    ).toBeInTheDocument();
  });

  it('keeps the pre-existing device type and link speed groups intact', () => {
    renderLegend();
    expect(screen.getByText('Device Types')).toBeInTheDocument();
    expect(screen.getByText('Router')).toBeInTheDocument();
    expect(screen.getByText('Link Speeds')).toBeInTheDocument();
    expect(screen.getByText('Trunk/LAG')).toBeInTheDocument();
  });

  it('groups the legend under Nodes and Edges headings', () => {
    renderLegend();
    expect(screen.getByText('Nodes')).toBeInTheDocument();
    expect(screen.getByText('Edges')).toBeInTheDocument();
  });
});
