/**
 * Unauthenticated daemon probes. /__version needs no session and no CSRF
 * token, and must not run the client's 401 refresh/retry (a poller that
 * refreshes a session is not asking whether the daemon is up), so it goes
 * straight to fetch from here, where the raw-fetch gate allows it.
 */

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

const VERSION_PATH = '/__version';

/** Resolves with the parsed body, or rejects on a network error or non-2xx. */
export async function fetchBuildVersionBody(signal: AbortSignal): Promise<unknown> {
  const response = await fetch(`${API_BASE}${VERSION_PATH}`, {
    headers: { Accept: 'application/json' },
    signal,
  });
  if (!response.ok) {
    throw new Error(`status ${response.status}`);
  }
  return response.json();
}

/** Resolves when the daemon answers /__version with a 2xx; rejects otherwise. */
export async function pingDaemon(signal: AbortSignal): Promise<void> {
  const response = await fetch(`${API_BASE}${VERSION_PATH}`, { signal });
  if (!response.ok) {
    throw new Error(`status ${response.status}`);
  }
}
