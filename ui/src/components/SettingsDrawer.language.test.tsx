import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { SettingsDrawer } from './SettingsDrawer';

vi.mock('../api/client', () => ({
  fetchInterfaces: () => Promise.resolve({ interfaces: [] }),
}));
vi.mock('./settings/SimulationSection', () => ({ SimulationSection: () => null }));

afterEach(async () => {
  await i18n.changeLanguage('en');
  window.localStorage.removeItem('niac-language');
});

it('switches the interface language and remembers the choice', async () => {
  render(
    <MemoryRouter>
      <SettingsDrawer
        isOpen
        onClose={vi.fn()}
        connectionStatus="connected"
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
  fireEvent.click(screen.getByRole('tab', { name: 'Appearance' }));

  const select = screen.getByTestId('language-select');
  expect(select).toHaveValue('en');
  expect(screen.getByRole('option', { name: 'Español' })).toBeInTheDocument();

  fireEvent.change(select, { target: { value: 'es' } });

  await waitFor(() => expect(i18n.language).toBe('es'));
  expect(await screen.findByTestId('language-select')).toHaveValue('es');
  expect(screen.getByText('Idioma')).toBeVisible();
  expect(window.localStorage.getItem('niac-language')).toBe('es');
  expect(document.documentElement.lang).toBe('es');
});
