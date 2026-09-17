/**
 * Tests for the 401 a request collects *before* this client ever held a
 * session.
 *
 * `sessionGeneration` starts at 0 and `beginSession()` only runs once the
 * client is authenticated, so on a never-authenticated page load the #2204
 * staleness guard (`issuedGeneration === sessionGeneration`) compares 0 to 0
 * and passes. Every unauthenticated fetch — `ProfileProvider` sits above
 * `<App>` in `main.tsx` and queries `/api/v1/profiles` on mount — therefore
 * fired the session-expired callback, and a first-time visitor was told
 * "Session expired. Please sign in again." before ever signing in (#2643).
 *
 * This lives in its own file because the generation counter is module state:
 * a test that a session was never established cannot share a module registry
 * with one that establishes one.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api, beginSession, clearCSRFToken, setSessionExpiredCallback } from './client';

/** A 401 with a failing refresh — the path that reaches onSessionExpired. */
function mockUnauthorizedWithFailedRefresh(): void {
  vi.spyOn(globalThis, 'fetch').mockImplementation(() =>
    Promise.resolve(new Response('{}', { status: 401 })),
  );
}

describe('api client before any session exists', () => {
  let onExpired: ReturnType<typeof vi.fn<() => void>>;

  beforeEach(() => {
    onExpired = vi.fn<() => void>();
    setSessionExpiredCallback(onExpired);
    clearCSRFToken();
    mockUnauthorizedWithFailedRefresh();
  });

  afterEach(() => {
    setSessionExpiredCallback(null);
    vi.restoreAllMocks();
  });

  it('does not expire a session the client never established', async () => {
    // No beginSession() has run in this module registry: nothing is signed in,
    // so this 401 is the ordinary unauthenticated answer, not an expiry.
    await expect(api.get('/api/v1/profiles')).rejects.toThrow('Session expired');

    expect(onExpired).not.toHaveBeenCalled();
  });

  it('expires the session once one has been established', async () => {
    beginSession();

    await expect(api.get('/api/v1/profiles')).rejects.toThrow('Session expired');

    expect(onExpired).toHaveBeenCalledTimes(1);
  });
});
