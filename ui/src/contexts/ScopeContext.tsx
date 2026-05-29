/**
 * ScopeContext — current bearer-token scope and UI gating helpers.
 *
 * Fetches GET /api/v1/auth/scope once at app boot, caches the result,
 * and exposes it via `useScope()`. UI components use this to render
 * scope-aware controls so a read-only token doesn't see (and click)
 * write buttons that would 403 against #743's scope enforcement.
 *
 * niac's tiers (from internal/api/token_store.go):
 *   read-only  < read-write  < admin
 * The auth() middleware stashes the resolved scope on the request
 * context; handleAuthScope echoes it. On bypass paths (loopback no-
 * token, NIAC_AUTH_DISABLED) the backend returns "admin" so dev
 * sessions aren't suddenly locked out of admin controls.
 *
 * Fail-closed: on fetch error or while the initial request is in
 * flight, `canWrite` and `isAdmin` are false so write/admin controls
 * hide rather than flash visible.
 */

import {
  createContext,
  type ReactElement,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';
import { deduplicatedGet } from '../api/requestCore';

/** Allowed scope values, mirroring TokenScope.String() on the server. */
export type Scope = 'read-only' | 'read-write' | 'admin';

const SCOPE_RANK: Record<Scope, number> = {
  'read-only': 1,
  'read-write': 2,
  admin: 3,
};

interface AuthScopeResponse {
  scope: Scope;
}

interface ScopeContextValue {
  /** Latest /auth/scope fetch, or null while loading / on fetch error. */
  scope: Scope | null;
  /** True while the initial fetch is in flight. */
  loading: boolean;
  /** Last fetch error message, or null if none. */
  error: string | null;
  /** Force a re-fetch (e.g. after rotating tokens via niac CLI). */
  refresh: () => Promise<void>;
  /**
   * True iff the current token may perform state-changing operations
   * (read-write or admin). Fail-closed while loading / on error.
   */
  canWrite: boolean;
  /**
   * True iff the current token is admin-scoped. Use for admin-only UI
   * surfaces such as /api/v1/config/import (#743).
   */
  isAdmin: boolean;
}

const ScopeContext = createContext<ScopeContextValue | null>(null);

interface ScopeProviderProps {
  children: ReactNode;
}

export function ScopeProvider({ children }: ScopeProviderProps): ReactElement {
  const [scope, setScope] = useState<Scope | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const fresh = await deduplicatedGet<AuthScopeResponse>('/api/v1/auth/scope');
      setScope(fresh.scope);
    } catch (err) {
      setScope(null);
      setError(err instanceof Error ? err.message : 'Failed to load token scope');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const canWrite = scope !== null && SCOPE_RANK[scope] >= SCOPE_RANK['read-write'];
  const isAdmin = scope === 'admin';

  return (
    <ScopeContext.Provider value={{ scope, loading, error, refresh, canWrite, isAdmin }}>
      {children}
    </ScopeContext.Provider>
  );
}

/**
 * useScope returns the active token scope state. Throws if called
 * outside a `<ScopeProvider>` — that always indicates a wiring bug.
 */
export function useScope(): ScopeContextValue {
  const ctx = useContext(ScopeContext);
  if (!ctx) {
    throw new Error('useScope() must be called inside <ScopeProvider>');
  }
  return ctx;
}
