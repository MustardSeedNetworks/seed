/**
 * A page load that finds a live cookie session must still report a genuine
 * later expiry.
 *
 * The client refuses to expire a session it never established (#2643), and
 * `login()` is not the only way one comes to exist: reloading with a valid
 * refresh cookie leaves the user authenticated without login() ever running.
 * The mount probe therefore calls `beginSession()` when `/api/v1/status`
 * answers ok. Without that call the guard sees generation 0 for the whole
 * page load and silently swallows the expiry banner for every reloaded tab.
 *
 * Its own file because the generation counter is module state: a
 * `beginSession()` anywhere earlier in a shared registry makes this vacuous.
 */

import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api, setSessionExpiredCallback } from '../api';
import { useAuth } from './useAuth';

describe('useAuth mount probe on a live cookie session', () => {
  let onExpired: ReturnType<typeof vi.fn<() => void>>;

  beforeEach(() => {
    onExpired = vi.fn<() => void>();
    // The cookie is still good, so the probe is answered; everything else —
    // including the refresh a 401 triggers — is not.
    vi.spyOn(globalThis, 'fetch').mockImplementation((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString();
      return Promise.resolve(
        new Response('{}', { status: url.includes('/api/v1/status') ? 200 : 401 }),
      );
    });
  });

  afterEach(() => {
    setSessionExpiredCallback(null);
    vi.restoreAllMocks();
  });

  it('expires the session when the cookie later goes stale', async () => {
    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    // Registered after the probe so the probe's own traffic cannot satisfy it.
    setSessionExpiredCallback(onExpired);

    await expect(api.get('/api/v1/profiles')).rejects.toThrow('Session expired');

    expect(onExpired).toHaveBeenCalledTimes(1);
  });
});
