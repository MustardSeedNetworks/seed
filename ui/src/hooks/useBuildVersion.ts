import { useEffect, useState } from 'react';
import { fetchBuildVersionBody } from '../api/buildVersion';

/**
 * Build metadata served by the daemon's unauthenticated /__version endpoint.
 * The lowercase keys come from the Universal Build Contract; seed, stem and
 * niac all return this shape (seed adds tlsFingerprint, which the UI ignores).
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
 * "dev" descriptor when the endpoint is unavailable, so callers can render
 * the version without a loading state.
 */
export function useBuildVersion(): BuildVersion {
  const [data, setData] = useState<BuildVersion>(FALLBACK);
  useEffect(() => {
    const controller = new AbortController();
    fetchBuildVersionBody(controller.signal)
      .then((body: unknown) => {
        if (isBuildVersion(body)) {
          setData(body);
        }
      })
      .catch(() => {
        // FALLBACK is already in state.
      });
    return () => controller.abort();
  }, []);
  return data;
}
