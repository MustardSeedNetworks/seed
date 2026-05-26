/**
 * LicenseContext - Active license tier & feature flags
 *
 * Fetches GET /api/v1/license once at app boot, caches the result, and
 * exposes it via `useLicense()`. UI components use this to decide
 * whether to render a feature (`<RequireFeature>`) or disable it with
 * an upgrade hint (`<TierGate>`).
 *
 * Server returns FeatureGateResponse-shaped JSON; on a 402 from any
 * gated API call the relevant component can additionally refresh
 * state via `refresh()`.
 *
 * No license is a valid state — the unlicensed (Free) user is the
 * default. Components that read this hook before fetch completes get
 * `null` and should render a loading state or treat the user as Free.
 */

import { createContext, type ReactNode, useCallback, useContext, useEffect, useState } from 'react';
import { api } from '../api/client';

/**
 * Shape of GET /api/v1/license (see handlers_api_tokens.go
 * LicenseStatusResponse). When the backend adds a feature to the
 * response (e.g. expiry, features array), extend this interface in
 * lockstep.
 */
export interface LicenseStatus {
  tier: string;
  tierValue: number;
  isTrialMode: boolean;
  trialDaysLeft?: number;
  canMintTokens: boolean;
  activated: boolean;
  /**
   * Features granted by the active license. Mirrors keygen's
   * productCatalog. UI gates use HasFeature() over this slice.
   *
   * Backend currently returns canMintTokens as a convenience flag but
   * not the full feature list — this field is reserved for the
   * upcoming Phase D-3 follow-up that extends the endpoint.
   */
  features?: string[];
}

interface LicenseContextValue {
  /** Latest license fetch, or null while loading / on fetch error. */
  status: LicenseStatus | null;
  /** True while the initial fetch is in flight. */
  loading: boolean;
  /** Last fetch error message, or null if none. */
  error: string | null;
  /** Force a re-fetch (e.g. after activating a key via CLI). */
  refresh: () => Promise<void>;
  /**
   * Convenience: true iff the active license includes the named
   * feature. Treats `status === null` (loading / unlicensed) as false
   * so gated UI hides rather than flashes during boot.
   */
  hasFeature: (feature: string) => boolean;
}

const LicenseContext = createContext<LicenseContextValue | null>(null);

interface LicenseProviderProps {
  children: ReactNode;
}

export function LicenseProvider({ children }: LicenseProviderProps): React.ReactElement {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const fresh = await api.get<LicenseStatus>('/api/v1/license');
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
    (feature: string): boolean => Boolean(status?.features?.includes(feature)),
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
 * outside a `<LicenseProvider>` — that always indicates a wiring bug.
 */
export function useLicense(): LicenseContextValue {
  const ctx = useContext(LicenseContext);
  if (!ctx) {
    throw new Error('useLicense() must be called inside <LicenseProvider>');
  }
  return ctx;
}
