import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { expect, it, vi } from 'vitest';
import { request } from '../api/requestCore';
import '../i18n';
import { SettingsDrawer } from './SettingsDrawer';

vi.mock('../api/client', () => ({
  fetchInterfaces: () => Promise.resolve({ interfaces: [] }),
}));
vi.mock('../api/requestCore', () => ({ request: vi.fn().mockResolvedValue({}) }));
vi.mock('./settings/SimulationSection', () => ({ SimulationSection: () => null }));

it('uses the shell connection state without starting another health poll', async () => {
  render(
    <MemoryRouter>
      <SettingsDrawer
        isOpen
        onClose={vi.fn()}
        connectionStatus="disconnected"
        themeState={{
          theme: 'dark',
          effectiveTheme: 'dark',
          isDark: true,
          setTheme: vi.fn(),
          toggleTheme: vi.fn(),
        }}
      />
    </MemoryRouter>,
  );
  fireEvent.click(screen.getByRole('tab', { name: 'Network' }));
  await screen.findByText('No interfaces found');
  expect(request).not.toHaveBeenCalled();
  expect(screen.getByTestId('connection-status')).toHaveAccessibleName('Backend unreachable');
});
