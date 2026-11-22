/**
 * The device list at scenario-pack scale: `enterprise-scale` runs 543 devices,
 * and each view mounts a window of them rather than a row per device.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { Device } from '../../api/types';
import { DeviceListProvider } from '../../contexts/DeviceListContext';
import '../../i18n';
import { DeviceCardView } from './DeviceCardView';
import { DeviceTableView } from './DeviceTableView';

const devices: Device[] = Array.from({ length: 543 }, (_, i) => ({
  hostname: `host-${String(i).padStart(3, '0')}`,
  mac: `02:00:00:00:${(i >> 8).toString(16).padStart(2, '0')}:${(i & 255).toString(16).padStart(2, '0')}`,
  ip: `10.0.${i >> 8}.${i & 255}`,
  type: 'switch',
  protocols: ['lldp', 'snmp'],
}));

function renderView(view: 'table' | 'cards', selected: string[] = []) {
  return render(
    <DeviceListProvider
      devices={devices}
      selectedDevices={new Set(selected)}
      onSelectDevice={vi.fn()}
      onSelectAll={vi.fn()}
      onEdit={vi.fn()}
      onClone={vi.fn()}
      onDelete={vi.fn()}
      getDeviceProtocols={(device) => device.protocols ?? []}
    >
      {view === 'table' ? <DeviceTableView /> : <DeviceCardView />}
    </DeviceListProvider>,
  );
}

describe('device list at 543 devices', () => {
  it('table view mounts a window of rows, starting at the first device', () => {
    renderView('table');

    const rows = screen.getAllByTestId('device-list-row');
    expect(rows.length).toBeLessThan(40);
    expect(screen.getByText('host-000')).toBeInTheDocument();
    expect(screen.queryByText('host-542')).not.toBeInTheDocument();
  });

  it('table view keeps the select-all checkbox bound to the whole list', () => {
    renderView(
      'table',
      devices.map((d) => d.hostname),
    );

    expect(screen.getByRole('checkbox', { name: /select all/i })).toBeChecked();
  });

  it('card view mounts a window of cards, one column at jsdom width', () => {
    renderView('cards');

    const cards = screen.getAllByTestId('device-list-card');
    expect(cards.length).toBeLessThan(40);
    expect(screen.getByText('host-000')).toBeInTheDocument();
    expect(screen.queryByText('host-542')).not.toBeInTheDocument();
  });

  it('card view packs a row of cards per column the viewport allows', () => {
    const widths = new Set([640, 1024]);
    const matchMedia = window.matchMedia;
    window.matchMedia = ((query: string) => ({
      ...matchMedia(query),
      matches: widths.has(Number(/(\d+)px/.exec(query)?.[1])),
    })) as typeof window.matchMedia;
    try {
      renderView('cards');

      const [firstRow] = screen.getAllByRole('listitem');
      expect(firstRow?.querySelectorAll('[data-testid="device-list-card"]')).toHaveLength(3);
    } finally {
      window.matchMedia = matchMedia;
    }
  });
});
