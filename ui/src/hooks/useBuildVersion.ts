import { useEffect, useState } from 'react';

/**
 * Build metadata served by the backend's unauthenticated /__version endpoint.
 * Mirrors the lowercase JSON keys defined by the Universal Build Contract;
 * every sibling project (seed/stem/niac) exposes the same shape.
 */
export interface BuildVersion {
  version: string;
  commit: string;
  buildTime: string;
  uiBuildHash: string;
}

const FALLBACK: BuildVersion = {
  version: 'dev',
  commit: 'unknown',
  buildTime: 'unknown',
  uiBuildHash: '',
};

function isBuildVersion(value: unknown): value is BuildVersion {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.version === 'string' &&
    typeof candidate.commit === 'string' &&
    typeof candidate.buildTime === 'string' &&
    typeof candidate.uiBuildHash === 'string'
  );
}

/**
 * Fetches build metadata from /__version once on mount. Falls back to a
 * static "dev" descriptor when the endpoint is unavailable so callers can
 * unconditionally render the version without a loading state.
 */
export function useBuildVersion(): BuildVersion {
  const [data, setData] = useState<BuildVersion>(FALLBACK);
  useEffect(() => {
    let cancelled = false;
    fetch('/__version', { headers: { Accept: 'application/json' } })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error(`status ${response.status}`);
        }
        return response.json();
      })
      .then((body) => {
        if (cancelled) {
          return;
        }
        if (isBuildVersion(body)) {
          setData(body);
        }
      })
      .catch(() => {
        // Swallow: FALLBACK is already in state.
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return data;
}
