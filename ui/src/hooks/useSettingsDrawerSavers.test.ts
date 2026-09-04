/**
 * useSettingsDrawerSavers CSRF tests (seed#2389).
 *
 * Every saver here used a raw fetch(), which sends no X-CSRF-Token, so the
 * middleware answered 403 "CSRF token required" and no setting in the drawer
 * could be saved. Verified in the real logged-in app:
 *
 *   raw fetch  PUT /api/v1/settings -> 403 {"error":"CSRF token required"}
 *   with token PUT /api/v1/settings -> 200 {"status":"updated"}
 *
 * Nothing noticed because ui/e2e/settings.spec.ts asserts only that sections
 * render, and the savers swallowed the failure into a status pill.
 */

import { act, renderHook } from '@testing-library/react';
import { createRef } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useSettingsDrawerSavers } from './useSettingsDrawerSavers';

const mockPut = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();

vi.mock('../api', () => ({
  api: { put: (path: string, body?: unknown): Promise<unknown> => mockPut(path, body) },
}));

afterEach(() => {
  vi.clearAllMocks();
});

function setup(): {
  result: { current: ReturnType<typeof useSettingsDrawerSavers> };
  statuses: string[];
} {
  const statuses: string[] = [];
  const record = (status: string): void => {
    statuses.push(status);
  };
  const { result } = renderHook(() =>
    useSettingsDrawerSavers({
      thresholds: { gateway: { good: 1, warning: 2 } } as never,
      setThresholdsStatus: record as never,
      testsSettings: {} as never,
      setTestsStatus: record as never,
      testsSettingsChangedRef: createRef<boolean>() as never,
      wifiSettings: {} as never,
      setWifiStatus: record as never,
      linkSettings: { mode: 'auto', availableModes: [] } as never,
      setLinkStatus: record as never,
      cableTestSettings: { enabled: true } as never,
      setCableTestStatus: record as never,
      networkDiscoverySettings: {} as never,
      setNetworkDiscoveryStatus: record as never,
      snmpSettings: {} as never,
      setSnmpStatus: record as never,
    }),
  );

  return { result, statuses };
}

describe('useSettingsDrawerSavers', () => {
  it('saves through the api client, so the CSRF token is attached', async () => {
    mockPut.mockResolvedValue({});
    const { result } = setup();

    await act(async () => {
      await result.current.saveThresholds();
    });

    expect(mockPut).toHaveBeenCalledWith('/api/v1/settings', {
      thresholds: { gateway: { good: 1, warning: 2 } },
    });
  });

  it('routes every saver at its own endpoint', async () => {
    mockPut.mockResolvedValue({});
    const { result } = setup();

    await act(async () => {
      await result.current.saveLinkSettings();
      await result.current.saveCableTestSettings();
      await result.current.saveSnmpSettings();
    });

    const paths = mockPut.mock.calls.map(([path]) => path);
    expect(paths).toContain('/api/v1/settings/link');
    expect(paths).toContain('/api/v1/settings/cable');
    expect(paths).toContain('/api/v1/telemetry/snmp/settings');
  });

  it('reports error rather than saved when the write is refused', async () => {
    mockPut.mockRejectedValue(new Error('API error: 403: CSRF token required'));
    const { result, statuses } = setup();

    await act(async () => {
      await result.current.saveThresholds();
    });

    expect(statuses).toContain('error');
    expect(statuses).not.toContain('saved');
  });
});
