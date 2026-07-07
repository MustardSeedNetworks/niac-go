/**
 * DeviceNode.tooltip.test.tsx — locks the hover-tooltip / accessible-name fix.
 *
 * The audit flagged that the topology node truncates the device name and shows
 * `+N` overflow for IPs and protocols, so the full data is invisible to users.
 * PR-A surfaces all that data through both `title` (native hover tooltip) and
 * `aria-label` (screen readers). This test fails if either disappears or stops
 * including any one of label / type / status / IPs / protocols.
 */
import { render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { DeviceNode } from './DeviceNode';
import type { DeviceNodeData } from './types';

function renderNode(data: Partial<DeviceNodeData>) {
  // ReactFlow's Handle component requires ReactFlowProvider in context.
  return render(
    <I18nextProvider i18n={i18n}>
      <ReactFlowProvider>
        <DeviceNode
          data={
            {
              label: 'edge-router-01',
              type: 'router',
              status: 'online',
              ips: ['10.0.0.1', '10.0.1.1', '10.0.2.1'],
              protocols: ['BGP', 'OSPF', 'SNMP', 'ARP'],
              ...data,
            } as DeviceNodeData
          }
        />
      </ReactFlowProvider>
    </I18nextProvider>,
  );
}

describe('DeviceNode — hover tooltip + accessible name', () => {
  it('surfaces label, type, status, all IPs, and all protocols', () => {
    renderNode({});
    const btn = screen.getByRole('button');
    const tooltip = btn.getAttribute('title') ?? '';
    const ariaLabel = btn.getAttribute('aria-label') ?? '';
    // title and aria-label carry the same content
    expect(tooltip).toBe(ariaLabel);
    expect(tooltip).toContain('edge-router-01');
    expect(tooltip).toContain('router');
    expect(tooltip).toContain('online');
    expect(tooltip).toContain('10.0.0.1');
    expect(tooltip).toContain('10.0.1.1');
    expect(tooltip).toContain('10.0.2.1');
    expect(tooltip).toContain('BGP');
    expect(tooltip).toContain('OSPF');
    expect(tooltip).toContain('SNMP');
    expect(tooltip).toContain('ARP');
  });

  it('handles missing optional ips/protocols without crashing', () => {
    renderNode({ ips: undefined, protocols: undefined });
    const btn = screen.getByRole('button');
    const tooltip = btn.getAttribute('title') ?? '';
    expect(tooltip).toContain('edge-router-01');
    expect(tooltip).not.toContain('IPs:');
    expect(tooltip).not.toContain('Protocols:');
  });

  it('defaults missing status to "online"', () => {
    renderNode({ status: undefined });
    const tooltip = screen.getByRole('button').getAttribute('title') ?? '';
    expect(tooltip).toContain('online');
  });
});
