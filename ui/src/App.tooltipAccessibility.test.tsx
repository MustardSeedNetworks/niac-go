import { act, configure, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { ScopeProvider } from './contexts/ScopeContext';
import { expectNoAxeViolations } from './test/a11y';
import { MemoryDataRouter } from './test/MemoryDataRouter';

const { mockClient } = vi.hoisted(() => {
  const responses: Record<string, unknown> = {
    // The shell resolves a session first; every per-session call is
    // disabled until it has one, so without this the pages render nothing.
    fetchSessions: [{ sessionId: 'sim-1', deviceCount: 1, interface: 'eth0' }],
    fetchSimulationStatus: {
      sessionId: 'sim-1',
      running: true,
      deviceCount: 1,
      uptimeSeconds: 42,
      interface: 'eth0',
    },
    fetchAlerts: { packetsThreshold: 0, webhookUrl: '' },
    fetchDevices: [{ name: 'sw1', type: 'switch', ips: ['10.0.0.1'], protocols: ['snmp'] }],
    fetchUsableInterfaces: { interfaces: [{ name: 'eth0', addresses: ['10.0.0.9'] }] },
    fetchSegments: [],
    fetchDebugLevel: { level: 'info', defaultLevel: 'basic' },
    fetchTemplates: [
      { name: 'basic-router', description: 'A basic router', deviceCount: 1, type: 'router' },
    ],
    fetchLibraryNetworks: [],
    fetchLibraryWalks: [
      {
        name: 'router.walk',
        sizeBytes: 200,
        source: 'starter',
        modifiedAt: '2026-09-15T00:00:00Z',
        edited: false,
      },
    ],
    fetchLibraryPcaps: [],
  };
  return {
    // Only the fetch* reads are stubbed. Replacing every export would
    // also replace pure helpers — defaultScenarioRequest() is one, and a
    // promise where the wizard expects its default state crashes the page.
    mockClient: (actual: Record<string, unknown>) =>
      Object.fromEntries(
        Object.entries(actual).map(([name, value]) => [
          name,
          typeof value === 'function' && name.startsWith('fetch')
            ? vi.fn(() =>
                name in responses ? Promise.resolve(responses[name]) : new Promise(() => {}),
              )
            : value,
        ]),
      ),
  };
});

vi.mock('./api/client', async (importOriginal) =>
  mockClient(await importOriginal<Record<string, unknown>>()),
);
// The shell's scope probe: admin, so no surface is hidden behind a
// read-only fieldset and every page renders the actions it really has.
vi.mock('./api/requestCore', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  deduplicatedGet: vi.fn(async () => ({ scope: 'admin' })),
}));
vi.mock('./api/library-client', async (importOriginal) =>
  mockClient(await importOriginal<Record<string, unknown>>()),
);
vi.mock('./api/scenario-client', async (importOriginal) =>
  mockClient(await importOriginal<Record<string, unknown>>()),
);
vi.mock('./api/walk-profile-client', async (importOriginal) =>
  mockClient(await importOriginal<Record<string, unknown>>()),
);
vi.mock('./api/capture', async (importOriginal) =>
  mockClient(await importOriginal<Record<string, unknown>>()),
);

function repeatsOwnText(el: Element): boolean {
  const title = el.getAttribute('title')?.trim() ?? '';
  return title !== '' && title === (el.textContent ?? '').trim();
}

function describe_(el: Element): string {
  const id = el.getAttribute('data-testid');
  return `<${el.tagName.toLowerCase()}${id ? ` data-testid="${id}"` : ''} title="${el.getAttribute('title')}">`;
}

configure({ asyncUtilTimeout: 5000 });

afterEach(() => {
  vi.clearAllMocks();
});

const routes = [
  '/',
  '/runtime',
  '/new-simulation',
  '/devices',
  '/segments',
  '/device-config',
  '/topology',
  '/alerts',
  '/traffic',
  '/debug',
  '/packets',
  '/config-diff',
  '/walk-validator',
  '/walk-analyzer',
  '/library/walks',
  '/library/pcaps',
];

describe('tooltips and control names', () => {
  it.each(routes)('%s uses native title only to repeat visible text', async (path) => {
    const { container } = render(
      <MemoryDataRouter initialEntries={[path]}>
        <ScopeProvider>
          <App />
        </ScopeProvider>
      </MemoryDataRouter>,
    );
    await screen.findByTestId('page-header-title');
    await Promise.resolve(act(async () => {}));

    const stranded = [...container.querySelectorAll('[title]')].filter((el) => !repeatsOwnText(el));
    expect(stranded.map(describe_), `${path}: title on an element no keyboard can reach`).toEqual(
      [],
    );
    await expectNoAxeViolations(container, {
      runOnly: {
        type: 'rule',
        values: [
          'button-name',
          'link-name',
          'aria-allowed-attr',
          'aria-valid-attr',
          'aria-valid-attr-value',
          'aria-required-attr',
        ],
      },
    });
  });

  it.each(routes)('%s connects each tooltip to a named control', async (path) => {
    const { container } = render(
      <MemoryDataRouter initialEntries={[path]}>
        <ScopeProvider>
          <App />
        </ScopeProvider>
      </MemoryDataRouter>,
    );
    await screen.findByTestId('page-header-title');
    await Promise.resolve(act(async () => {}));

    const descriptions = [...document.querySelectorAll('[role="tooltip"]')];
    expect(descriptions.length).toBeGreaterThan(0);
    for (const description of descriptions) {
      const triggers = [...container.querySelectorAll('[aria-describedby]')].filter((element) =>
        element.getAttribute('aria-describedby')?.split(/\s+/).includes(description.id),
      );
      expect(
        triggers.length,
        `${path}: unconnected tooltip ${description.textContent}`,
      ).toBeGreaterThan(0);
      for (const trigger of triggers) {
        expect(['BUTTON', 'INPUT', 'SELECT', 'TEXTAREA', 'A']).toContain(trigger.tagName);
        expect(trigger).toHaveAccessibleName();
        expect(trigger).toHaveAccessibleDescription();
      }
    }
  });
});
