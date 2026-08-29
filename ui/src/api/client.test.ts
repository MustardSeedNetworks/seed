/**
 * Tests for the API client's session-expiry handling.
 *
 * The case that matters here is a 401 that arrives *late*: a request issued
 * under one session, whose response lands after the user has already logged in
 * again. Expiring the session on that response logs out the session that
 * replaced it — see #2204, where it reset the login form to
 * "Session expired. Please log in again." immediately after a successful login.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api, beginSession, setSessionExpiredCallback } from './client';

/** A 401 with a failing refresh — the path that reaches onSessionExpired. */
function mockUnauthorizedWithFailedRefresh(): void {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString();
    // The refresh attempt fails the way the server reports an absent cookie.
    if (url.includes('/api/v1/auth/refresh')) {
      return Promise.resolve(new Response('{}', { status: 401 }));
    }
    return Promise.resolve(new Response('{}', { status: 401 }));
  });
}

describe('api client session expiry', () => {
  let onExpired: ReturnType<typeof vi.fn<() => void>>;

  beforeEach(() => {
    onExpired = vi.fn<() => void>();
    setSessionExpiredCallback(onExpired);
    mockUnauthorizedWithFailedRefresh();
  });

  afterEach(() => {
    setSessionExpiredCallback(null);
    vi.restoreAllMocks();
  });

  it('expires the session when the 401 belongs to the current session', async () => {
    await expect(api.get('/api/v1/status')).rejects.toThrow('Session expired');

    expect(onExpired).toHaveBeenCalledTimes(1);
  });

  it('does not expire a newer session when a stale 401 arrives after re-login', async () => {
    // The request is issued under the current session...
    const inFlight = api.get('/api/v1/status');

    // ...but the user logs in again before its 401 is handled.
    beginSession();

    await expect(inFlight).rejects.toThrow('Session expired');

    // The 401 belongs to the previous session. Acting on it would log out the
    // session that just replaced it.
    expect(onExpired).not.toHaveBeenCalled();
  });
});
