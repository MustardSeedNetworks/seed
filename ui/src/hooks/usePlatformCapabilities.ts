/**
 * usePlatformCapabilities — what this operating system can do (#750).
 *
 * The report comes from /api/v1/status, which internal/capabilities fills and
 * which also generates HARDWARE.md's matrix (#749) — so the UI, the API and the
 * document cannot disagree.
 *
 * Deliberately separate from useCapabilities, which reports *privileges*
 * (whether this process can open a raw socket), and from useLicense, which
 * reports *tier*. Three questions with three remedies: install differently,
 * run with more privilege, buy a licence. Conflating them would be worse than
 * the silence this replaces.
 */

import { useCallback, useEffect, useState } from 'react';

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';

/** Mirrors capabilities.Level. */
export type CapabilityLevel = 'full' | 'partial' | 'limited' | 'none';

/** Mirrors capabilities.Entry. */
export interface PlatformCapability {
  capability: string;
  title: string;
  level: CapabilityLevel;
  note?: string;
}

interface StatusWithCapabilities {
  capabilities?: PlatformCapability[];
}

interface PlatformCapabilities {
  /** Every capability, in matrix order. */
  all: PlatformCapability[];
  /** Look one up. Unknown names report `none` — a capability the backend does
   *  not describe is not one this UI should assume works. */
  levelOf: (capability: string) => CapabilityLevel;
  /** True when the platform supports it at all. */
  supported: (capability: string) => boolean;
  /** Everything below full support, for the warnings banner. */
  degraded: PlatformCapability[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function usePlatformCapabilities(): PlatformCapabilities {
  const [all, setAll] = useState<PlatformCapability[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const status = await api.get<StatusWithCapabilities>('/api/v1/status');
      setAll(status?.capabilities ?? []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to read platform capabilities';
      setError(message);
      logger.error(LogComponents.SYSTEM, 'Failed to read platform capabilities', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh().catch(() => undefined);
  }, [refresh]);

  const levelOf = useCallback(
    (capability: string): CapabilityLevel =>
      all.find((entry) => entry.capability === capability)?.level ?? 'none',
    [all],
  );

  return {
    all,
    levelOf,
    supported: useCallback(
      (capability: string): boolean => levelOf(capability) !== 'none',
      [levelOf],
    ),
    degraded: all.filter((entry) => entry.level !== 'full'),
    loading,
    error,
    refresh,
  };
}
