/**
 * useDeviceScan error-surfacing tests (seed#2394).
 *
 * A failed scan used to log, clear `scanning`, and spread `...prev` -- the
 * card reverted to its previous contents with no way to tell "the scan
 * failed" from "the scan found nothing". These assert the hook now reports
 * the failure via its returned `scanError`, except for a session-expiry
 * 401, which the global session-expired flow (api client's
 * onSessionExpired callback) is already handling -- a second, generic
 * error banner on a card about to unmount would be redundant noise.
 */

import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { NetworkDiscoveryData } from '../components/cards/NetworkDiscoveryCard';
import { useDeviceScan } from './useDeviceScan';

const mockPost = vi.fn<(path: string) => Promise<unknown>>();
const mockGet = vi.fn<(path: string) => Promise<unknown>>();

const { MockSessionExpiredError } = vi.hoisted(() => ({
  MockSessionExpiredError: class extends Error {
    constructor() {
      super('Session expired');
      this.name = 'SessionExpiredError';
    }
  },
}));

vi.mock('../api', () => ({
  api: {
    post: (path: string): Promise<unknown> => mockPost(path),
    get: (path: string): Promise<unknown> => mockGet(path),
  },
  SessionExpiredError: MockSessionExpiredError,
}));

afterEach(() => {
  vi.clearAllMocks();
});

const samplePrev: NetworkDiscoveryData = {
  devices: [],
  status: {
    scanning: false,
    deviceCount: 0,
    lastScan: '',
    subnet: '',
    localIP: '',
    interface: '',
  },
};

describe('useDeviceScan error surfacing', () => {
  it.each([
    ['a generic 500', new Error('boom'), true],
    ['a network failure', new TypeError('Failed to fetch'), true],
  ])(
    'sets scanError when the scan request fails with %s',
    async (_label, rejection, wantsError) => {
      mockPost.mockRejectedValueOnce(rejection);
      const setNetworkDiscovery = vi.fn();
      const fetchNetworkDiscovery = vi.fn().mockResolvedValue(undefined);

      const { result } = renderHook(() =>
        useDeviceScan({ fetchNetworkDiscovery, setNetworkDiscovery }),
      );

      await act(async () => {
        await result.current.triggerDeviceScan();
      });

      expect(result.current.scanError).toBe(wantsError);

      // The card must not keep showing "scanning" once the attempt has failed.
      const lastUpdater = setNetworkDiscovery.mock.calls.at(-1)?.[0] as (
        prev: NetworkDiscoveryData | null,
      ) => NetworkDiscoveryData | null;
      expect(lastUpdater(samplePrev)?.status.scanning).toBe(false);
    },
  );

  it('does not surface a generic error for a session-expiry 401 -- the global session-expired flow already handles it', async () => {
    mockPost.mockRejectedValueOnce(new MockSessionExpiredError());
    const setNetworkDiscovery = vi.fn();
    const fetchNetworkDiscovery = vi.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useDeviceScan({ fetchNetworkDiscovery, setNetworkDiscovery }),
    );

    await act(async () => {
      await result.current.triggerDeviceScan();
    });

    expect(result.current.scanError).toBe(false);

    const lastUpdater = setNetworkDiscovery.mock.calls.at(-1)?.[0] as (
      prev: NetworkDiscoveryData | null,
    ) => NetworkDiscoveryData | null;
    expect(lastUpdater(samplePrev)?.status.scanning).toBe(false);
  });

  it('clears a stale error at the start of a new scan attempt', async () => {
    mockPost.mockRejectedValueOnce(new Error('boom'));
    const setNetworkDiscovery = vi.fn();
    const fetchNetworkDiscovery = vi.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useDeviceScan({ fetchNetworkDiscovery, setNetworkDiscovery }),
    );

    await act(async () => {
      await result.current.triggerDeviceScan();
    });
    expect(result.current.scanError).toBe(true);

    mockPost.mockResolvedValueOnce(undefined);
    mockGet.mockResolvedValue({ scanning: false });

    await act(async () => {
      await result.current.triggerDeviceScan();
    });
    expect(result.current.scanError).toBe(false);
  });
});
