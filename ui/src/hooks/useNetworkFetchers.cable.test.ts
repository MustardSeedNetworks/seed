/**
 * The cable read honours the platform capability report (#2691). Where the
 * platform has no TDR API, GET /telemetry/cable answers 501, and every poll of
 * it put an error in the browser console.
 */
import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useNetworkFetchers } from './useNetworkFetchers';

function cableReads(fetchMock: ReturnType<typeof vi.fn>): number {
  return fetchMock.mock.calls.filter(([url]) => String(url).includes('/telemetry/cable')).length;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe.each([
  { cableSupported: false, reads: 0 },
  { cableSupported: true, reads: 1 },
])('fetchCableData with cableSupported=$cableSupported', ({ cableSupported, reads }) => {
  it(`reads the cable route ${reads} time(s)`, async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ supported: true, status: 'ok', faults: [] }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const { result } = renderHook(() =>
      useNetworkFetchers({
        currentInterfaceRef: { current: 'eth0' },
        setCards: vi.fn(),
        setCurrentInterface: vi.fn(),
        setInterfaces: vi.fn(),
        setAppVersion: vi.fn(),
        setNetworkDiscovery: vi.fn(),
        setIsWifi: vi.fn(),
        userSetWifiModeRef: { current: false },
        networkDiscoveryAbortRef: { current: null },
        prevLinkUpRef: { current: null },
        cableSupported,
      }),
    );

    await result.current.fetchCableData();

    expect(cableReads(fetchMock)).toBe(reads);
  });
});
