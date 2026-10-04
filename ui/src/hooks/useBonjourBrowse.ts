/**
 * useBonjourBrowse — the DNS-SD browse of #364.
 *
 * Deliberately NOT auto-run on mount, unlike the read-only card hooks beside
 * it: a browse sends multicast queries and listens for several seconds, and
 * doing that every time someone opens the page would put traffic on the
 * segment nobody asked for. The operator presses the button.
 */

import { useState } from 'react';

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { BrowseResult } from '../types/generated/bonjour-browse-response';

interface BonjourBrowse {
  result: BrowseResult | null;
  loading: boolean;
  error: string | null;
  browse: () => Promise<void>;
}

export function useBonjourBrowse(): BonjourBrowse {
  const [result, setResult] = useState<BrowseResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const browse = async (): Promise<void> => {
    setLoading(true);
    setError(null);
    await api
      .get<BrowseResult>('/api/v1/discovery/bonjour')
      .then((response) => {
        setResult(response ?? null);
      })
      .catch((err: unknown) => {
        const message =
          err instanceof Error ? err.message : 'Failed to browse for Bonjour services';
        setError(message);
        logger.error(LogComponents.DISCOVERY, 'Failed to browse for Bonjour services', err);
      });
    setLoading(false);
  };

  return { result, loading, error, browse };
}
