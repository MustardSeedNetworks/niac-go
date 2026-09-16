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
  it('surfaces label, type, all IPs, and all protocols', () => {
    renderNode({});
    const btn = screen.getByRole('button');
    const tooltip = screen.getByRole('tooltip', { hidden: true }).textContent ?? '';
    const ariaLabel = btn.getAttribute('aria-label') ?? '';
    expect(tooltip).toBe(ariaLabel);
    expect(btn).toHaveAccessibleDescription(tooltip);
    expect(tooltip).toContain('edge-router-01');
    expect(tooltip).toContain('router');
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
    const tooltip = screen.getByRole('tooltip', { hidden: true }).textContent ?? '';
    expect(tooltip).toContain('edge-router-01');
    expect(tooltip).not.toContain('IPs:');
    expect(tooltip).not.toContain('Protocols:');
  });

  it('never claims a reachability state the daemon does not report', () => {
    // This replaces a test that asserted the opposite — it was named
    // 'defaults missing status to "online"' and encoded the defect as intent.
    renderNode({});
    const tooltip = screen.getByRole('tooltip', { hidden: true }).textContent ?? '';

    expect(tooltip).not.toContain('online');
    expect(tooltip).not.toContain('offline');
    expect(tooltip).not.toContain('warning');
  });
});
