/**
 * DevicesPage.test.tsx — Running Devices naming-collision cross-links (PR 2a).
 *
 * Two contracts pinned here:
 *   1. The device list card links to the Device Library (`/device-config`)
 *      so the read-only running view isn't a dead end for editing.
 *   2. Each SNMP walk file listed under "Available SNMP walks" links to the
 *      device editor's SNMP section (`/device-config/new#snmp`) in addition
 *      to the existing "Copy name" button, so the name is actually usable.
 */
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import '../i18n';
import { DevicesPage } from './DevicesPage';

vi.mock('../api/client', () => ({
  fetchDevices: () => Promise.resolve([]),
  fetchConfig: () =>
    Promise.resolve({
      content: 'devices: []',
      path: '/tmp/config.yaml',
      modifiedAt: '2026-01-01T00:00:00Z',
      sizeBytes: 12,
    }),
  updateConfig: vi.fn(),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryWalks: () =>
    Promise.resolve([{ name: 'cisco/cisco-c3900-01.walk', source: 'bundled' }]),
}));

describe('DevicesPage', () => {
  it('links to the Device Library for editing', async () => {
    render(
      <MemoryRouter>
        <DevicesPage />
      </MemoryRouter>,
    );

    const link = await screen.findByRole('link', { name: /edit in device library/i });
    expect(link).toHaveAttribute('href', '/device-config');
  });

  it('links each walk file to the device editor SNMP section', async () => {
    render(
      <MemoryRouter>
        <DevicesPage />
      </MemoryRouter>,
    );

    const link = await screen.findByRole('link', { name: /use in device editor/i });
    expect(link).toHaveAttribute('href', '/device-config/new#snmp');
  });
});
