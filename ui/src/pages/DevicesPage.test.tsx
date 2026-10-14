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
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryDataRouter } from '../test/MemoryDataRouter';
import { renderWithResources as render } from '../test/renderWithResources';
import '../i18n';
import { fetchDevices } from '../api/client';
import { useApiResource } from '../hooks/useApiResource';
import { DevicesPage } from './DevicesPage';

// Runtime reads name their session, so a page rendered on its own has to say
// which scenario it is looking at.
vi.mock('../contexts/AppContext', () => ({
  useAppState: () =>
    useApiResource(() => fetchDevices('test-session'), ['devices', 'test-session']),
}));

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

// Both links are plain anchors, so this file asserts on text and the enclosing
// <a> rather than paying for CodeMirror and for the first *ByRole query. On a
// cold worker those cost about 200 ms and 150 ms (jsdom's getComputedStyle walk
// for the accessibility check), which is what pushed the first test past
// Vitest's 5 s budget on a loaded host (#2524).
vi.mock('../components/config/YamlEditor', () => ({
  YamlEditor: () => null,
}));

describe('DevicesPage', () => {
  it('links to the Device Library for editing', async () => {
    render(
      <MemoryDataRouter>
        <DevicesPage />
      </MemoryDataRouter>,
    );

    const label = await screen.findByText(/edit in device library/i);
    expect(label.closest('a')).toHaveAttribute('href', '/device-config');
  });

  it('links each walk file to the device editor SNMP section', async () => {
    render(
      <MemoryDataRouter>
        <DevicesPage />
      </MemoryDataRouter>,
    );

    const label = await screen.findByText(/use in device editor/i);
    expect(label.closest('a')).toHaveAttribute('href', '/device-config/new#snmp');
  });
});
