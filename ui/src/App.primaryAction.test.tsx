/**
 * App.primaryAction.test.tsx — one primary action per page (U5).
 *
 * The rule is a rendering fact, not a source-code fact: a page's primary
 * action can come from any component it renders, so a grep over the page
 * file proves nothing. Button and LinkButton emit data-variant/data-tone,
 * so every route is rendered through the real shell and the primaries are
 * counted in the DOM.
 *
 * Pages are rendered with every API call left pending, which is the state
 * a page is in before its first response — the toolbar is chrome, so the
 * primary must be there before the data is.
 *
 * Every route here is a lazy import of a whole page, resolved for the first
 * time under v8 coverage. testing-library's 1 s default is sized for a
 * mounted component, not for that: on a loaded shared runner the wizard
 * chunk took 1207 ms where the same route's green PR run took 732 ms, and
 * the gate failed main (#1887). The bound below is ~4x that worst case;
 * findBy still returns the moment the header appears.
 */
import { configure, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { ScopeProvider } from './contexts/ScopeContext';

/**
 * Pages whose primary action only exists once their data has arrived —
 * you cannot save alerts you have not loaded — need a response, so those
 * calls are answered. Every other call is left pending rather than given
 * an invented shape: unanswered is a state the page must handle anyway,
 * and it keeps this file from becoming a second fixture library.
 */
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

/**
 * The primary is the one filled button. Tone is not part of the test:
 * /runtime's primary while a simulation runs is Stop, and a destructive
 * primary is red — filled weight is what makes it the primary, not hue.
 */
const PRIMARY = '[data-variant="solid"]';

/**
 * Every route in pageRegistry, and the accessible name of the one primary
 * action it owns — or `none` where the page has no single most-likely
 * action in this state, which PageHeader's own contract says should carry
 * no primary rather than an invented one:
 *
 *   /segments      a read-only view of the authored VLAN segments
 *   /config-diff   compares two files, so its action only exists once
 *                  both are chosen — the initial state has none
 *
 * A path with no entry here fails, so a new route cannot skip the gate.
 */
const none = Symbol('no primary action');
const primaries: Record<string, RegExp | typeof none> = {
  '/': /start a simulation/i,
  '/runtime': /stop simulation/i,
  '/new-simulation': /next/i,
  '/devices': /edit in device library/i,
  '/segments': none,
  '/device-config': /add device/i,
  '/topology': /refresh/i,
  '/alerts': /save alerts/i,
  '/traffic': /inject error/i,
  '/debug': /pause/i,
  '/packets': /pause/i,
  '/config-diff': none,
  '/walk-validator': /validate$/i,
  '/walk-analyzer': /import and review/i,
  '/library/walks': /install bundle/i,
  '/library/pcaps': /install bundle/i,
};

configure({ asyncUtilTimeout: 5000 });

afterEach(() => {
  vi.clearAllMocks();
});

describe('one primary action per page', () => {
  it.each(Object.entries(primaries))('%s has one primary action', async (path, primary) => {
    const { container } = render(
      <MemoryRouter initialEntries={[path]}>
        <ScopeProvider>
          <App />
        </ScopeProvider>
      </MemoryRouter>,
    );
    // The routed page is lazy; wait for its header before counting.
    await screen.findByTestId('page-header-title');
    const expected = primary === none ? 0 : 1;
    await waitFor(() => {
      expect(
        [...container.querySelectorAll(PRIMARY)].map((el) => el.textContent),
        `${path} primaries`,
      ).toHaveLength(expected);
    });
    if (primary !== none) {
      expect(container.querySelector(PRIMARY)).toHaveAccessibleName(primary);
    }
  });
});
