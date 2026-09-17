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
import { api, beginSession, clearCSRFToken, setSessionExpiredCallback } from './client';

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
    clearCSRFToken();
    // These cases all model a client that IS signed in — the client refuses to
    // expire a session it never established (#2643), so say that it holds one.
    beginSession();
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

/**
 * Tests for the CSRF token a retry carries after a silent refresh.
 *
 * The refreshed access token is a different JWT, and the server keys CSRF
 * tokens by the bearer (`csrf.SessionKey`), so a retry that reuses the token
 * minted for the old bearer is answered 403. Treating that 403 as "session
 * expired" logged the operator out of a perfectly valid session — see #2633.
 */
describe('api client retry after refresh', () => {
  let onExpired: ReturnType<typeof vi.fn<() => void>>;
  /** Every settings attempt, in order, with the CSRF token it carried. */
  let attempts: (string | null)[];
  /** How many times a CSRF token was minted. */
  let csrfMints: number;

  /**
   * The server as it behaves for #2633: the first attempt is 401 (expired
   * access token), the refresh succeeds, and the retry is answered by
   * `retryStatus` — 403 when the retry reuses the stale CSRF token.
   *
   * Each `/auth/csrf` call mints a *distinct* token, so a test asserting the
   * retry carried a fresh one cannot pass by comparing one shared constant
   * against itself.
   */
  function mockRefreshThen(retryStatus: number, retryBody: string): void {
    vi.spyOn(globalThis, 'fetch').mockImplementation(
      (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString();
        if (url.includes('/api/v1/auth/refresh')) {
          return Promise.resolve(new Response('{}', { status: 200 }));
        }
        if (url.includes('/api/v1/auth/csrf')) {
          csrfMints++;
          return Promise.resolve(
            new Response(JSON.stringify({ token: `csrf-${csrfMints}` }), { status: 200 }),
          );
        }
        attempts.push(new Headers(init?.headers).get('X-CSRF-Token'));
        return attempts.length === 1
          ? Promise.resolve(new Response('{}', { status: 401 }))
          : Promise.resolve(new Response(retryBody, { status: retryStatus }));
      },
    );
  }

  beforeEach(() => {
    onExpired = vi.fn<() => void>();
    setSessionExpiredCallback(onExpired);
    clearCSRFToken();
    // A signed-in client, as above (#2643).
    beginSession();
    attempts = [];
    csrfMints = 0;
  });

  afterEach(() => {
    setSessionExpiredCallback(null);
    vi.restoreAllMocks();
  });

  it('retries with a CSRF token minted after the refresh', async () => {
    mockRefreshThen(403, JSON.stringify({ error: 'Invalid CSRF token' }));

    await expect(api.post('/api/v1/settings', { theme: 'dark' })).rejects.toThrow(/API error: 403/);

    // Two attempts, and the second carried a token minted after the refresh.
    expect(attempts).toEqual(['csrf-1', 'csrf-2']);
    expect(csrfMints).toBe(2);
    // A 403 on the retry is an ordinary request error, not a dead session.
    expect(onExpired).not.toHaveBeenCalled();
  });

  it('expires the session only when the retry is itself a 401', async () => {
    mockRefreshThen(401, '{}');

    await expect(api.post('/api/v1/settings', { theme: 'dark' })).rejects.toThrow(
      'Session expired',
    );

    expect(onExpired).toHaveBeenCalledTimes(1);
  });
});

/**
 * `GET /auth/csrf` mints with foundation's `Manager.Generate`, which replaces
 * the session's stored token, so a second concurrent mint invalidates the token
 * the first caller is holding. Two mutations that both 401 wake from the same
 * refresh and both need a token — the case the client has to fetch only once.
 */
describe('api client concurrent CSRF mint', () => {
  let csrfMints: number;
  /** The CSRF token each settings attempt carried, in completion order. */
  let attempts: (string | null)[];

  beforeEach(() => {
    clearCSRFToken();
    csrfMints = 0;
    attempts = [];
    const seen = new Map<string, number>();

    vi.spyOn(globalThis, 'fetch').mockImplementation(
      (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString();
        if (url.includes('/api/v1/auth/refresh')) {
          return Promise.resolve(new Response('{}', { status: 200 }));
        }
        if (url.includes('/api/v1/auth/csrf')) {
          csrfMints++;
          return Promise.resolve(
            new Response(JSON.stringify({ token: `csrf-${csrfMints}` }), { status: 200 }),
          );
        }
        attempts.push(new Headers(init?.headers).get('X-CSRF-Token'));
        // Each endpoint's first attempt is the one whose access token expired.
        const n = (seen.get(url) ?? 0) + 1;
        seen.set(url, n);
        return Promise.resolve(new Response('{}', { status: n === 1 ? 401 : 200 }));
      },
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('mints one token for two mutations retrying after the same refresh', async () => {
    await Promise.all([
      api.post('/api/v1/settings', { theme: 'dark' }),
      api.put('/api/v1/alerts/rules/1', { enabled: true }),
    ]);

    // One mint for the first attempts, one for the retries — not one per retry.
    expect(csrfMints).toBe(2);
    // Both retries carried the post-refresh token, so neither was invalidated
    // by the other's mint.
    expect(attempts.slice(2)).toEqual(['csrf-2', 'csrf-2']);
  });
});
