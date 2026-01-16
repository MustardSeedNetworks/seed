// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * useDefaults Hook
 *
 * Fetches and caches default settings from the backend.
 * The backend is the single source of truth for all default values.
 *
 * Note: Settings are now managed by ProfileContext. This hook is kept for
 * backward compatibility but components should prefer using useProfileContext()
 * which provides settings merged with defaults.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";
import type { DefaultSettings } from "../types/defaults";

// ============================================================================
// Cache for defaults (shared across all hook instances)
// ============================================================================

let cachedDefaults: DefaultSettings | null = null;
let fetchPromise: Promise<DefaultSettings> | null = null;

// ============================================================================
// Hook Interface
// ============================================================================

interface UseDefaultsResult {
  defaults: DefaultSettings | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

// ============================================================================
// Hook Implementation
// ============================================================================

/**
 * Hook to fetch and cache default settings from the backend.
 * Uses a shared cache to avoid redundant API calls across components.
 *
 * Prefer using useProfileContext() which provides settings already merged
 * with defaults. This hook is for accessing raw backend defaults only.
 */
export function useDefaults(): UseDefaultsResult {
  const [defaults, setDefaults] = useState<DefaultSettings | null>(cachedDefaults);
  const [isLoading, setIsLoading] = useState(!cachedDefaults);
  const [error, setError] = useState<Error | null>(null);
  const isMountedRef = useRef(true);

  const fetchDefaults = useCallback(async () => {
    // If already cached, use cached value
    if (cachedDefaults) {
      setDefaults(cachedDefaults);
      setIsLoading(false);
      return;
    }

    // If already fetching, wait for that promise
    const existingPromise: Promise<DefaultSettings> | null = fetchPromise;
    // biome-ignore lint/nursery/noMisusedPromises: checking if promise exists, not its resolved value
    if (existingPromise) {
      try {
        // biome-ignore lint/nursery/useAwaitThenable: existingPromise is a Promise<DefaultSettings>
        const result: DefaultSettings = await existingPromise;
        if (isMountedRef.current) {
          setDefaults(result);
          setIsLoading(false);
        }
      } catch (err) {
        if (isMountedRef.current) {
          setError(err instanceof Error ? err : new Error(String(err)));
          setIsLoading(false);
        }
      }
      return;
    }

    // Start new fetch
    setIsLoading(true);
    setError(null);

    const newPromise: Promise<DefaultSettings> = api.get<DefaultSettings>(
      "/api/v1/settings/defaults",
    );
    fetchPromise = newPromise;

    try {
      const result: DefaultSettings = await newPromise;
      cachedDefaults = result;
      if (isMountedRef.current) {
        setDefaults(result);
        setIsLoading(false);
      }
    } catch (err) {
      logger.warn(LogComponents.Config, "Failed to fetch defaults from backend", err);
      if (isMountedRef.current) {
        setError(err instanceof Error ? err : new Error(String(err)));
        // Don't set fallback - let caller handle missing defaults
        setIsLoading(false);
      }
    } finally {
      fetchPromise = null;
    }
  }, []);

  const refetch = useCallback(async () => {
    cachedDefaults = null;
    fetchPromise = null;
    await fetchDefaults();
  }, [fetchDefaults]);

  useEffect(() => {
    isMountedRef.current = true;
    fetchDefaults().catch(() => undefined);
    return (): void => {
      isMountedRef.current = false;
    };
  }, [fetchDefaults]);

  return { defaults, isLoading, error, refetch };
}

/**
 * Get cached defaults synchronously.
 * Returns null if not yet loaded.
 */
export function getDefaultsSync(): DefaultSettings | null {
  return cachedDefaults;
}

/**
 * Clear the defaults cache (useful for testing).
 */
export function clearDefaultsCache(): void {
  cachedDefaults = null;
  fetchPromise = null;
}
