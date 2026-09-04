/**
 * useReports CSRF tests (seed#2389).
 *
 * generate() and remove() used a raw fetch(), which sends no X-CSRF-Token. The
 * middleware requires one on every mutating route that is not on the exempt
 * list, so both answered 403 and every Generate Report press failed silently.
 * Verified in the real logged-in app:
 *
 *   raw fetch  PUT /api/v1/settings -> 403 {"error":"CSRF token required"}
 *   with token PUT /api/v1/settings -> 200 {"status":"updated"}
 *
 * The api client attaches the token; these assert the calls go through it.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useReports } from './useReports';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();
const mockDelete = vi.fn<(path: string) => Promise<unknown>>();

vi.mock('../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (path: string, body?: unknown): Promise<unknown> => mockPost(path, body),
    delete: (path: string): Promise<unknown> => mockDelete(path),
  },
}));

// refresh() still reads through fetch; stub it so the hook can mount.
const fetchMock = vi.fn(() =>
  Promise.resolve({ ok: true, json: () => Promise.resolve({ reports: [] }) } as Response),
);
vi.stubGlobal('fetch', fetchMock);

afterEach(() => {
  vi.clearAllMocks();
});

describe('useReports mutations', () => {
  it('generates through the api client, so the CSRF token is attached', async () => {
    mockPost.mockResolvedValue({});
    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.generate('summary', 'json');
    });

    expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/generate', {
      type: 'summary',
      format: 'json',
    });
  });

  it('deletes through the api client, so the CSRF token is attached', async () => {
    mockDelete.mockResolvedValue({});
    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.remove('report-1');
    });

    expect(mockDelete).toHaveBeenCalledWith('/api/v1/reports/report-1');
  });

  it('surfaces a failure instead of reporting success', async () => {
    mockPost.mockRejectedValue(new Error('API error: 403'));
    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.generate('summary', 'json');
    });

    expect(result.current.error).toContain('403');
    expect(result.current.generating).toBe(false);
  });
});
