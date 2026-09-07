/**
 * pageStory — one harness for every page story (U6).
 *
 * Before this, the a11y gate only saw atoms: the two walk pages were the
 * only stories under src/pages, and each hand-rolled its own fetch stub and
 * its own `<main>` wrapper. Sixteen routes cannot each re-derive that, and
 * the wrapper is not optional — a page rendered bare puts its card-section
 * `<header>` elements outside any sectioning content, where they map to
 * `role="banner"`, and axe reports duplicate landmarks that do not exist in
 * the app (U2b found this the hard way). So `pageMeta` owns the wrapper: a
 * story cannot forget it, the way U4 made `htmlFor` required rather than
 * caught after the fact.
 *
 * A story declares only the routes its page reads, under
 * `parameters.api`; the chrome every page needs (scope, CSRF, version,
 * sessions, simulation status) comes from CHROME_ROUTES underneath.
 */
import type { Decorator } from '@storybook/react-vite';
import type { ComponentType } from 'react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { AppProvider } from '../../contexts/AppContext';
import { ScopeProvider } from '../../contexts/ScopeContext';
import {
  type ApiRoutes,
  answeredRequests,
  makeApiStub,
  pendingRequests,
  resetRequestCounters,
  status,
} from './apiStub';
import * as fixtures from './fixtures';

/** Session-scoped resource path, matching api/client's own sessionPath(). */
export const sessionResource = (resource: string) =>
  `/api/v1/sessions/${fixtures.SESSION_ID}/${resource}`;

/**
 * The calls the shell itself makes on every route. `admin` scope matters:
 * ReadOnlyView wraps the whole app in a `disabled` fieldset for a read-only
 * token, and a story rendered under that would show a page whose actions are
 * all inert — not the page a11y is meant to check.
 */
export const CHROME_ROUTES: ApiRoutes = {
  '/api/v1/auth/scope': { scope: 'admin' },
  '/api/v1/csrf-token': { token: 'story' },
  '/api/v1/version': fixtures.version,
  '/api/v1/sessions': fixtures.sessions,
  '/api/v1/simulation': fixtures.simulationStatus,
};

/**
 * Every read path a page can make, answered with the fixtures. Declaring the
 * whole surface once — rather than per story — keeps a story file to the
 * three states it is about, and means a page that grows a new read shows the
 * fixture data instead of the 404 an omitted route would return.
 *
 * Longest key wins, so the narrower paths below sit safely beside their
 * prefixes (`/api/v1/capture/filter` beside `/api/v1/capture`).
 */
export const LOADED_ROUTES: ApiRoutes = {
  ...CHROME_ROUTES,
  '/api/v1/history': fixtures.history,
  '/api/v1/alerts': fixtures.alerts,
  '/api/v1/errors': fixtures.errorTypes,
  '/api/v1/interfaces': fixtures.interfaces,
  '/api/v1/replay': fixtures.replayState,
  '/api/v1/capture': fixtures.captureStatus,
  '/api/v1/capture/filter': { active: false, filter: '' },
  '/api/v1/debug/level': fixtures.debugLevel,
  '/api/v1/templates': fixtures.templates,
  '/api/v1/config': fixtures.configDocument,
  '/api/v1/config/devices': { devices: [], total: 0 },
  '/api/v1/config/schema': {
    $schema: 'https://json-schema.org/draft/2020-12/schema',
    type: 'object',
    title: 'NIAC configuration',
    properties: {},
    required: [],
  },
  '/api/v1/device-schemas': {},
  '/api/v1/synthesize-walk/models': [],
  '/api/v1/library/walks': fixtures.walkFiles,
  '/api/v1/library/pcaps': fixtures.pcapFiles,
  '/api/v1/library/networks': fixtures.libraryNetworks,
  '/api/v1/library/drafts': [],
  '/api/v1/scenario/packs': [],
  '/api/v1/scenario/profiles': [],
  '/api/v1/scenario/profiles/captured': [],
  [sessionResource('stats')]: fixtures.stats,
  [sessionResource('devices')]: fixtures.devices,
  [sessionResource('segments')]: fixtures.segments,
  [sessionResource('neighbors')]: fixtures.neighbors,
  [sessionResource('topology')]: fixtures.topology,
};

/**
 * Nothing running and nothing authored: the state a page is in on a fresh
 * install, which is the one operators actually meet first and the one no
 * page had a story for.
 */
export const EMPTY_ROUTES: ApiRoutes = {
  ...LOADED_ROUTES,
  '/api/v1/sessions': [],
  '/api/v1/simulation': fixtures.simulationIdle,
  '/api/v1/history': [],
  '/api/v1/templates': [],
  '/api/v1/library/walks': [],
  '/api/v1/library/pcaps': [],
  '/api/v1/library/networks': [],
  '/api/v1/errors': { info: '', availableTypes: [], targets: [] },
  '/api/v1/alerts': { packetsThreshold: 0, webhookUrl: '' },
  [sessionResource('devices')]: [],
  [sessionResource('segments')]: [],
  [sessionResource('neighbors')]: [],
  [sessionResource('topology')]: { nodes: [], links: [] },
};

/**
 * The loaded state with one read failing — the third story every page owes.
 * U1 gave every page a shared error state; nothing rendered it, so nothing
 * proved it appears.
 *
 * 404 rather than 500 deliberately. requestCore retries any 5xx with
 * exponential backoff (1 s, 2 s, 4 s), and during those sleeps there is no
 * request outstanding — so a 500 both delays the page's error state past any
 * reasonable story timeout and gives `settled` a false quiet window to
 * return in. A 4xx is not retried, so the page reaches the same error state
 * immediately, which is the state these stories are about.
 */
export const withFailure = (path: string): ApiRoutes => ({
  ...LOADED_ROUTES,
  [path]: status(404, { error: 'simulated read failure' }),
});

const apiDecorator: Decorator = (Story, context) => {
  const routes = (context.parameters.api ?? {}) as ApiRoutes;
  resetRequestCounters();
  globalThis.fetch = makeApiStub({ ...CHROME_ROUTES, ...routes });
  return <Story />;
};

const SETTLE_TIMEOUT_MS = 10_000;
const SETTLE_QUIET_MS = 150;
const SETTLE_POLL_MS = 25;

/**
 * settled — the play function every page story runs before axe does.
 *
 * The a11y addon checks the DOM as soon as mount and play settle, so without
 * this a story named Loaded is checked while it is still a skeleton, and one
 * named Error while it is still a spinner: on the topology page the error
 * state took 1-2 s to appear, well past testing-library's 1 s default. That
 * is the fleet's "green but checking nothing" shape — the gate runs, reports
 * pass, and never sees the state the story exists to cover.
 *
 * One quiet moment is not enough. A page fires its session-scoped reads
 * before AppContext has resolved the session id, so there is a real gap
 * between that first round and the refetch that follows — the topology page
 * settled, then failed, in two rounds. So the wait is for the answered count
 * to hold still across a window, which catches the whole cascade. The window
 * is far shorter than the 2 s poll, so a polling page still has quiet
 * stretches to find.
 */
export const settled = () => async (): Promise<void> => {
  const deadline = Date.now() + SETTLE_TIMEOUT_MS;
  let lastAnswered = -1;
  let quietSince = 0;
  while (Date.now() < deadline) {
    const answered = answeredRequests();
    if (pendingRequests() === 0 && answered > 0 && answered === lastAnswered) {
      if (quietSince === 0) quietSince = Date.now();
      if (Date.now() - quietSince >= SETTLE_QUIET_MS) return;
    } else {
      quietSince = 0;
    }
    lastAnswered = answered;
    await new Promise((resolve) => setTimeout(resolve, SETTLE_POLL_MS));
  }
  throw new Error(
    `page never settled: ${answeredRequests()} answered, ` +
      `${pendingRequests()} still pending after ${SETTLE_TIMEOUT_MS} ms`,
  );
};

/**
 * The shell renders every route inside `<main>` (ui/src/ui/Sidebar.tsx) with
 * a router above it. Both are landmark/behaviour context, not decoration:
 * without the router any page using useNavigate throws, and without `<main>`
 * axe sees banners the app does not have.
 */
const shellDecorator: Decorator = (Story, context) => {
  // A page reading useParams (the device editor's hostname) needs a matched
  // Route above it, not just a history entry — outside one, useParams is
  // empty and the page silently renders its "new" state instead.
  const pattern = context.parameters.routePattern as string | undefined;
  const page = <Story />;
  return (
    <MemoryRouter initialEntries={[(context.parameters.route as string) ?? '/']}>
      <ScopeProvider>
        <AppProvider>
          <main>{pattern ? <Routes>{<Route path={pattern} element={page} />}</Routes> : page}</main>
        </AppProvider>
      </ScopeProvider>
    </MemoryRouter>
  );
};

/**
 * pageMeta builds the Meta for one page's story file. `title` is the
 * page component's name so the Pages/* tree matches pageRegistry.
 */
export function pageMeta<T extends ComponentType<Record<string, never>>>(
  name: string,
  component: T,
) {
  return {
    title: `Pages/${name}`,
    component,
    // Order matters: the fetch stub must be installed before the providers
    // mount, because ScopeProvider and AppProvider fetch on their first
    // effect. Storybook applies decorators innermost-first, so the API
    // decorator is listed last to end up outside the shell.
    decorators: [shellDecorator, apiDecorator],
  };
}
