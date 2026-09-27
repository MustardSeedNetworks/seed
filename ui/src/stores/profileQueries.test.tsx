/**
 * The active-profile query treats "no active profile yet" as a normal state.
 * It used to recognise that by `message.includes('404')`, which also matched
 * any server reason carrying those digits; it now reads `ApiError.status`
 * (#763).
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { beginSession, clearCSRFToken } from '../api';
import { useActiveProfileQuery } from './profileQueries';
import { useProfileStore } from './profileStore';

/** Answers every active-profile GET with `status` and `body`; counts them. */
function respondWith(status: number, body: object): { calls: () => number } {
  let calls = 0;
  vi.spyOn(globalThis, 'fetch').mockImplementation((input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString();
    if (url.includes('/api/v1/profiles/active')) {
      calls++;
    }
    // The refresh a 401 triggers fails, so the 401 case ends as an expiry.
    return Promise.resolve(new Response(JSON.stringify(body), { status }));
  });
  return { calls: () => calls };
}

function renderActiveProfileQuery(): void {
  const client = new QueryClient({ defaultOptions: { queries: { retryDelay: 0 } } });
  const wrapper = ({ children }: { children: ReactNode }): ReactNode => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  renderHook(() => useActiveProfileQuery(), { wrapper });
}

describe('useActiveProfileQuery errors', () => {
  beforeEach(() => {
    clearCSRFToken();
    beginSession();
    useProfileStore.setState({ error: null, isSettingsLoaded: false });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('treats a 404 as no active profile: one request, no error', async () => {
    const server = respondWith(404, { error: 'No active profile', code: 'NOT_FOUND' });

    renderActiveProfileQuery();

    await waitFor(() => expect(useProfileStore.getState().isSettingsLoaded).toBe(true));
    expect(server.calls()).toBe(1);
    expect(useProfileStore.getState().error).toBeNull();
  });

  it('reports a 500 whose reason mentions 404, after retrying it', async () => {
    const server = respondWith(500, { error: 'upstream 404 from store', code: 'INTERNAL_ERROR' });

    renderActiveProfileQuery();

    await waitFor(() =>
      expect(useProfileStore.getState().error).toBe('API error: 500: upstream 404 from store'),
    );
    expect(server.calls()).toBe(4);
  });

  it('leaves an expired session to the session flow', async () => {
    respondWith(401, { error: 'Unauthorized', code: 'UNAUTHORIZED' });

    renderActiveProfileQuery();

    await waitFor(() => expect(useProfileStore.getState().isSettingsLoaded).toBe(true));
    expect(useProfileStore.getState().error).toBeNull();
  });
});
