import { cleanup, renderHook } from '@testing-library/react';
import { act } from 'react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { useConnectionStatus } from './useConnectionStatus';

const mockFetch = vi.fn();

function stalledUntilAborted(_url: string, init?: RequestInit): Promise<Response> {
  return new Promise((_resolve, reject) => {
    init?.signal?.addEventListener('abort', () =>
      reject(new DOMException('Aborted', 'AbortError')),
    );
  });
}

async function advance(ms: number): Promise<void> {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal('fetch', mockFetch);
  mockFetch.mockReset().mockResolvedValue({ ok: true });
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

it('polls /__version once per interval and reports recovery', async () => {
  mockFetch.mockRejectedValueOnce(new TypeError('Failed to fetch'));
  const { result } = renderHook(useConnectionStatus);
  expect(result.current).toBe('checking');
  await advance(0);
  expect(result.current).toBe('disconnected');
  expect(mockFetch).toHaveBeenCalledTimes(1);
  expect(mockFetch.mock.calls[0]?.[0]).toBe('/__version');
  await advance(15_000);
  expect(result.current).toBe('connected');
  expect(mockFetch).toHaveBeenCalledTimes(2);
});

it('treats a non-OK response as unreachable', async () => {
  mockFetch.mockResolvedValue({ ok: false, status: 502 });
  const { result } = renderHook(useConnectionStatus);
  await advance(0);
  expect(result.current).toBe('disconnected');
});

it('times out a stalled request and recovers on the next check', async () => {
  mockFetch.mockImplementationOnce(stalledUntilAborted);
  const { result } = renderHook(useConnectionStatus);
  const signal = mockFetch.mock.calls[0]?.[1]?.signal as AbortSignal;
  expect(result.current).toBe('checking');
  await advance(5_000);
  expect(signal.aborted).toBe(true);
  expect(result.current).toBe('disconnected');
  await advance(10_000);
  expect(result.current).toBe('connected');
});

it('aborts its pending request and clears every timer on unmount', async () => {
  mockFetch.mockImplementation(stalledUntilAborted);
  const { unmount } = renderHook(useConnectionStatus);
  const signal = mockFetch.mock.calls[0]?.[1]?.signal as AbortSignal;
  unmount();
  expect(signal.aborted).toBe(true);
  await advance(30_000);
  expect(mockFetch).toHaveBeenCalledTimes(1);
  expect(vi.getTimerCount()).toBe(0);
});
