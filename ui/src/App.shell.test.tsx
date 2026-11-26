/**
 * App.shell.test.tsx — shell behaviours that live in App itself: the unknown
 * route and where the sidebar's version comes from.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { ScopeProvider } from './contexts/ScopeContext';
import { MemoryDataRouter } from './test/MemoryDataRouter';

// Every daemon read stays pending except /api/v1/version, which reports a
// version the sidebar must NOT show: the sidebar reads /__version.
vi.mock('./api/client', async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return Object.fromEntries(
    Object.entries(actual).map(([name, value]) => [
      name,
      typeof value === 'function' && name.startsWith('fetch')
        ? vi.fn(() =>
            name === 'fetchVersion'
              ? Promise.resolve({ version: '9.9.9-api' })
              : new Promise(() => {}),
          )
        : value,
    ]),
  );
});
vi.mock('./api/requestCore', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  deduplicatedGet: vi.fn(async () => ({ scope: 'admin' })),
}));

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset().mockImplementation((input: RequestInfo | URL) => {
    if (String(input) === '/__version') {
      return Promise.resolve(
        Response.json({
          version: '1.2.3-build',
          commit: 'abc',
          buildTime: 'now',
          uiBuildHash: 'h',
        }),
      );
    }
    return new Promise<Response>(() => {});
  });
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function renderApp(path: string) {
  return render(
    <MemoryDataRouter initialEntries={[path]}>
      <ScopeProvider>
        <App />
      </ScopeProvider>
    </MemoryDataRouter>,
  );
}

describe('App shell', () => {
  it('shows the not-found panel for an unknown route instead of redirecting', async () => {
    renderApp('/definitely/not/a/page');

    expect(await screen.findByTestId('not-found')).toHaveTextContent('NIAC has no page');
    expect(screen.getByRole('link', { name: 'Back to the dashboard' })).toHaveAttribute(
      'href',
      '/',
    );
  });

  it('takes the sidebar version from /__version, like the help drawer', async () => {
    renderApp('/definitely/not/a/page');

    await waitFor(() =>
      expect(screen.getAllByTestId('rail-version')[0]).toHaveTextContent('v1.2.3'),
    );
    expect(screen.queryByText('9.9.9-api')).not.toBeInTheDocument();
  });
});
