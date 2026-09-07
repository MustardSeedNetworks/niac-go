/**
 * apiStub — the fetch stub every page story runs against.
 *
 * Pages reach the daemon through one seam: api/requestCore builds a relative
 * URL and calls `fetch`. Both contexts the shell mounts (AppContext's polls,
 * ScopeContext's scope probe) go through the same seam, so replacing `fetch`
 * is enough to put a page into any state — no per-module mocking, which is
 * unavailable in Storybook anyway.
 *
 * A route key matches the request's PATH exactly, query string ignored.
 * Substring matching was the obvious choice and was wrong: pages fetch
 * session-scoped resources before the session id resolves, so
 * `/api/v1/sessions//topology` matched the `/api/v1/sessions` key and the
 * topology arrived as the list of sessions — a page showing invented data in
 * the story that exists to show it has none. Unmatched is a 404, which is
 * what the daemon answers for that path anyway.
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
 * Request bookkeeping, so a story can wait for the state it claims to show.
 *
 * This is the difference between a story that is named Loaded and one that
 * *is* loaded: the a11y addon runs axe as soon as mount and play settle, and
 * a page whose data has not landed yet is still showing its skeleton. On the
 * topology page the error state took over a second to appear, so an Error
 * story with no play had axe checking a spinner.
 */
let pending = 0;
let answered = 0;

/** Requests the stub has been given but not yet answered. */
export const pendingRequests = (): number => pending;

/** Requests the stub has answered since the counters were last reset. */
export const answeredRequests = (): number => answered;

export const resetRequestCounters = (): void => {
  pending = 0;
  answered = 0;
};

/**
 * makeApiStub returns a `fetch` implementation answering `routes`.
 * Keys are URL substrings; the longest matching key wins.
 */
export function makeApiStub(routes: ApiRoutes): typeof fetch {
  return ((input: RequestInfo | URL) => {
    const url = String(input instanceof Request ? input.url : input);
    // Relative paths are what api/requestCore builds; URL needs a base.
    const path = new URL(url, 'http://story.invalid').pathname;
    const key = path in routes ? path : undefined;
    const route = key === undefined ? undefined : routes[key];
    const response =
      key === undefined
        ? json({ error: `no story route for ${path}` }, 404)
        : isExplicit(route)
          ? json(route.body, route.status)
          : json(route, 200);

    // Answered on a macrotask rather than synchronously: a promise that is
    // already resolved settles inside the same microtask checkpoint as the
    // caller, so `pending` would never be observed above zero and the wait
    // below would pass before anything had been requested.
    pending += 1;
    return new Promise<Response>((resolve) => {
      setTimeout(() => {
        pending -= 1;
        answered += 1;
        resolve(response);
      }, 0);
    });
  }) as typeof fetch;
}
