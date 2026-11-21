import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useBuildVersion } from './useBuildVersion';

const mockFetch = vi.fn();

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch);
});
afterEach(() => {
  mockFetch.mockReset();
  vi.unstubAllGlobals();
});

const build = {
  version: '1.2.3',
  commit: 'abc1234',
  buildTime: '2026-11-21T00:00:00Z',
  uiBuildHash: 'deadbeef',
  tlsFingerprint: 'ff',
};

describe('useBuildVersion', () => {
  it('reports the running daemon version from /__version', async () => {
    mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(build) });
    const { result } = renderHook(useBuildVersion);
    expect(result.current.version).toBe('dev');
    await waitFor(() => expect(result.current.version).toBe('1.2.3'));
    expect(result.current.uiBuildHash).toBe('deadbeef');
    expect(mockFetch.mock.calls[0]?.[0]).toBe('/__version');
  });

  it('keeps the dev fallback on a non-OK response', async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 503, json: () => Promise.resolve(build) });
    const { result } = renderHook(useBuildVersion);
    await waitFor(() => expect(mockFetch).toHaveBeenCalled());
    expect(result.current.version).toBe('dev');
  });

  it('keeps the dev fallback when the body is not build metadata', async () => {
    mockFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({ version: 7 }) });
    const { result } = renderHook(useBuildVersion);
    await waitFor(() => expect(mockFetch).toHaveBeenCalled());
    expect(result.current.version).toBe('dev');
  });

  it('keeps the dev fallback when the request fails, and aborts on unmount', async () => {
    mockFetch.mockRejectedValue(new TypeError('Failed to fetch'));
    const { result, unmount } = renderHook(useBuildVersion);
    await waitFor(() => expect(mockFetch).toHaveBeenCalled());
    const signal = mockFetch.mock.calls[0]?.[1]?.signal as AbortSignal;
    expect(result.current.version).toBe('dev');
    unmount();
    expect(signal.aborted).toBe(true);
  });
});
