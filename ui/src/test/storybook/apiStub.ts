/**
 * apiStub — the fetch stub every page story runs against.
 *
 * Pages reach the daemon through one seam: api/requestCore builds a relative
 * URL and calls `fetch`. Both contexts the shell mounts (AppContext's polls,
 * ScopeContext's scope probe) go through the same seam, so replacing `fetch`
 * is enough to put a page into any state — no per-module mocking, which is
 * unavailable in Storybook anyway.
 *
 * A route key is matched as a substring of the request URL, longest key
 * first, so `/api/v1/simulation/preflight` wins over `/api/v1/simulation`
 * regardless of declaration order. An unmatched request is a 404 rather than
 * an invented body: a story that silently answers a call it never declared
 * hides the page's real behaviour.
 */

/**
 * A route answers with a JSON body (200) or, via `status()`, with a chosen
 * status code. The two are told apart by a brand rather than by looking for
 * a `status` key: several of this API's real bodies have one
 * (StandaloneCaptureStatus, the walk validation results), and a body that
 * silently became a status code would be a very quiet bug.
 */
const STATUS_ROUTE = Symbol('stub-status-route');

export type StubStatus = { readonly [STATUS_ROUTE]: true; status: number; body?: unknown };

/** An explicit status code — `status(500, { error })` for the Error stories. */
export const status = (code: number, body?: unknown): StubStatus => ({
  [STATUS_ROUTE]: true,
  status: code,
  body,
});

export type ApiRoutes = Record<string, unknown>;

const isExplicit = (r: unknown): r is StubStatus =>
  typeof r === 'object' && r !== null && STATUS_ROUTE in r;

const json = (body: unknown, status: number): Response =>
  new Response(status === 204 ? null : JSON.stringify(body ?? null), {
    status,
    headers: { 'content-type': 'application/json' },
  });

/**
 * makeApiStub returns a `fetch` implementation answering `routes`.
 * Keys are URL substrings; the longest matching key wins.
 */
export function makeApiStub(routes: ApiRoutes): typeof fetch {
  const keys = Object.keys(routes).sort((a, b) => b.length - a.length);
  return ((input: RequestInfo | URL) => {
    const url = String(input instanceof Request ? input.url : input);
    const key = keys.find((k) => url.includes(k));
    if (key === undefined) {
      return Promise.resolve(json({ error: `no story route for ${url}` }, 404));
    }
    const route = routes[key];
    return Promise.resolve(isExplicit(route) ? json(route.body, route.status) : json(route, 200));
  }) as typeof fetch;
}
