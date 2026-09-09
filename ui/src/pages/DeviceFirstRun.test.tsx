import { screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithResources } from '../test/renderWithResources';
import '../i18n';
import { DeviceEditorPage } from './DeviceEditorPage';
import { DeviceListPage } from './DeviceListPage';

const configuration = vi.hoisted(() => ({ loaded: false }));
beforeEach(() => {
  configuration.loaded = false;
});

vi.mock('../api/client', () => ({
  fetchConfigDevices: () =>
    Promise.resolve({ devices: [], totalCount: 0, configurationLoaded: configuration.loaded }),
  fetchDeviceEditorSchema: () => Promise.resolve({ visibleSections: ['basic'] }),
  fetchConfigDevice: vi.fn(),
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
  deleteDevice: vi.fn(),
  deleteDevices: vi.fn(),
  cloneDevice: vi.fn(),
}));
vi.mock('../api/library-client', () => ({ fetchLibraryWalks: () => Promise.resolve([]) }));
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

describe('first-run device authoring', () => {
  it('keeps an empty loaded configuration editable', async () => {
    configuration.loaded = true;
    const router = createMemoryRouter(
      [{ path: '/device-config/:hostname', element: <DeviceEditorPage /> }],
      { initialEntries: ['/device-config/new'] },
    );
    renderWithResources(<RouterProvider router={router} />);
    expect(await screen.findByRole('textbox', { name: /hostname/i })).toBeInTheDocument();
    expect(screen.queryByTestId('create-device-draft')).not.toBeInTheDocument();
  });
  it.each(['/device-config', '/device-config/new'])('offers a saved draft at %s', async (path) => {
    const router = createMemoryRouter(
      [
        { path: '/device-config', element: <DeviceListPage /> },
        { path: '/device-config/:hostname', element: <DeviceEditorPage /> },
      ],
      { initialEntries: [path] },
    );
    renderWithResources(<RouterProvider router={router} />);
    expect(await screen.findByTestId('create-device-draft')).toHaveAttribute(
      'href',
      '/new-simulation',
    );
    expect(screen.queryByRole('textbox', { name: /hostname/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^add device$/i })).not.toBeInTheDocument();
  });
});
