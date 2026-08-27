/**
 * Pins the seam between the reports API and what ReportsCard renders.
 *
 * The card's stories hand-write their props and the Go handler tests assert on
 * Go structs, so without this nothing covers the step in between: a field
 * rename or a status-value change would leave both sides green while the card
 * rendered wrong (#2154, same gap as the LLDP/SwitchCard seam in #2147).
 *
 * The payloads are the shape internal/api/handlers_reports.go emits.
 */
import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useReports } from './useReports';

function reportsBody(overrides: Record<string, unknown> = {}) {
  return {
    reports: [
      {
        id: 'rep-1',
        name: 'Executive Report - 2026-08-27',
        type: 'executive',
        format: 'pdf',
        status: 'complete',
        fileSize: 20481,
        createdAt: '2026-08-27T10:00:00Z',
        completedAt: '2026-08-27T10:00:05Z',
        expiresAt: '2026-09-27T10:00:00Z',
        ...overrides,
      },
    ],
  };
}

function mockFetch(responses: Array<{ ok: boolean; status?: number; body?: unknown }>) {
  const fn = vi.fn();
  for (const r of responses) {
    fn.mockResolvedValueOnce({
      ok: r.ok,
      status: r.status ?? (r.ok ? 200 : 500),
      json: () => Promise.resolve(r.body ?? {}),
    });
  }
  vi.stubGlobal('fetch', fn);
  return fn;
}

/** Returns the nth fetch call, failing loudly if it was never made. */
function nthCall(fetchMock: ReturnType<typeof vi.fn>, n: number): [string, RequestInit] {
  const call = fetchMock.mock.calls[n];
  if (!call) {
    throw new Error(`expected at least ${n + 1} fetch calls, saw ${fetchMock.mock.calls.length}`);
  }
  return [String(call[0]), (call[1] ?? {}) as RequestInit];
}

describe('useReports', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('maps the API payload onto the fields the card renders', async () => {
    mockFetch([{ ok: true, body: reportsBody() }]);

    const { result } = renderHook(() => useReports());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBeNull();
    expect(result.current.reports).toHaveLength(1);
    expect(result.current.reports[0]).toMatchObject({
      id: 'rep-1',
      name: 'Executive Report - 2026-08-27',
      format: 'pdf',
      status: 'complete',
    });
  });

  it('requests the collection endpoint with credentials', async () => {
    const fetchMock = mockFetch([{ ok: true, body: { reports: [] } }]);

    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/reports', { credentials: 'include' });
  });

  // 402 is the licence gate and 401 the auth boundary. Reporting an empty list
  // for either would tell the reader there are no reports, which is a lie.
  it.each([401, 402, 500])(
    'surfaces a %d as an error rather than an empty list',
    async (status) => {
      mockFetch([{ ok: false, status }]);

      const { result } = renderHook(() => useReports());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.error).toContain(String(status));
      expect(result.current.reports).toEqual([]);
    },
  );

  it('tolerates a malformed body without throwing', async () => {
    mockFetch([{ ok: true, body: { reports: 'not-an-array' } }]);

    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.reports).toEqual([]);
  });

  // The endpoint answers 202 with a pending snapshot, so the hook must re-read
  // rather than trust it: the row would otherwise sit at "pending" forever.
  it('re-reads the collection after generating', async () => {
    const fetchMock = mockFetch([
      { ok: true, body: { reports: [] } },
      { ok: true, status: 202, body: { id: 'rep-2', status: 'pending' } },
      { ok: true, body: reportsBody({ id: 'rep-2', status: 'generating' }) },
    ]);

    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.generate('executive', 'pdf');
    });

    const [url, init] = nthCall(fetchMock, 1);
    expect(url).toBe('/api/v1/reports/generate');
    expect(init).toMatchObject({ method: 'POST', credentials: 'include' });
    expect(JSON.parse(String(init.body))).toEqual({ type: 'executive', format: 'pdf' });

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(result.current.reports[0]).toMatchObject({ id: 'rep-2', status: 'generating' });
  });

  it('deletes by id and re-reads', async () => {
    const fetchMock = mockFetch([
      { ok: true, body: reportsBody() },
      { ok: true, status: 204 },
      { ok: true, body: { reports: [] } },
    ]);

    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.remove('rep-1');
    });

    const [url, init] = nthCall(fetchMock, 1);
    expect(url).toBe('/api/v1/reports/rep-1');
    expect(init).toMatchObject({ method: 'DELETE' });
    expect(result.current.reports).toEqual([]);
  });

  it('encodes the id so a hostile value cannot escape the path', async () => {
    const fetchMock = mockFetch([
      { ok: true, body: { reports: [] } },
      { ok: true, status: 204 },
      { ok: true, body: { reports: [] } },
    ]);

    const { result } = renderHook(() => useReports());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.remove('../../etc/passwd');
    });

    expect(nthCall(fetchMock, 1)[0]).toBe('/api/v1/reports/..%2F..%2Fetc%2Fpasswd');
  });
});
