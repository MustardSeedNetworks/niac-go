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
import { type ApiRoutes, makeApiStub, status } from './apiStub';
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
 */
export const withFailure = (path: string): ApiRoutes => ({
  ...LOADED_ROUTES,
  [path]: status(500, { error: 'simulated daemon failure' }),
});

const apiDecorator: Decorator = (Story, context) => {
  const routes = (context.parameters.api ?? {}) as ApiRoutes;
  globalThis.fetch = makeApiStub({ ...CHROME_ROUTES, ...routes });
  return <Story />;
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
