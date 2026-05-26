/**
 * LicenseContext — active license tier + feature flags.
 *
 * Fetches GET /api/v1/license once at app boot, caches the result,
 * and exposes it via `useLicense()`. UI components use this to
 * decide whether to render a feature (`<RequireFeature>`) or
 * disable it with an upgrade hint (`<TierGate>`).
 *
 * No license is a valid state — unlicensed Free is the default.
 * Components that read the hook before fetch completes get
 * `loading=true` and should render placeholder content rather than
 * flashing paid features.
 *
 * On a 402 response from a gated API call, call `refresh()` so the
 * UI picks up any tier change made via the `niac license` CLI
 * without forcing a page reload.
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

/** Shape of GET /api/v1/license — mirrors LicenseStatusResponse on the server. */
export interface LicenseStatus {
  tier: string;
  isActivated: boolean;
  isTrialMode: boolean;
  trialDaysRemaining: number;
  features: string[];
  /**
   * False on dev / test builds where no license manager is wired
   * in. UIs should treat this as "all features available, hide
   * upgrade hints" so local development isn't cluttered with
   * paid-tier prompts.
   */
  licenseEnforced: boolean;
}

interface LicenseContextValue {
  /** Latest fetch, or null while loading / on fetch error. */
  status: LicenseStatus | null;
  /** True while the initial fetch is in flight. */
  loading: boolean;
  /** Last fetch error message, or null. */
  error: string | null;
  /** Re-fetch the license state — call after CLI activation. */
  refresh: () => Promise<void>;
  /**
   * True iff the active license includes the named feature.
   * - When the server reports `licenseEnforced=false` (dev / test),
   *   all features are reported as present so local UI works.
   * - During loading, returns false so gated UI hides rather than
   *   flashing on boot.
   */
  hasFeature: (feature: string) => boolean;
}

const LicenseContext = createContext<LicenseContextValue | null>(null);

interface LicenseProviderProps {
  children: ReactNode;
}

export function LicenseProvider({ children }: LicenseProviderProps): ReactElement {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const fresh = await deduplicatedGet<LicenseStatus>('/api/v1/license');
      setStatus(fresh);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load license');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const hasFeature = useCallback(
    (feature: string): boolean => {
      if (!status) {
        return false;
      }
      if (!status.licenseEnforced) {
        return true;
      }
      return status.features.includes(feature);
    },
    [status],
  );

  return (
    <LicenseContext.Provider value={{ status, loading, error, refresh, hasFeature }}>
      {children}
    </LicenseContext.Provider>
  );
}

/**
 * useLicense returns the active license state. Throws if called
 * outside `<LicenseProvider>` — that always indicates a wiring bug.
 */
export function useLicense(): LicenseContextValue {
  const ctx = useContext(LicenseContext);
  if (!ctx) {
    throw new Error('useLicense() must be called inside <LicenseProvider>');
  }
  return ctx;
}
