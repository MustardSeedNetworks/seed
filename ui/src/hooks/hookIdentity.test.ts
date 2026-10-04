/**
 * Hook callback identity, held by the React Compiler rather than by
 * hand-written useCallback/useMemo (UI-SEED-41 slice 2, #3066).
 *
 * Every function these hooks return must keep its identity across a
 * re-render that changed nothing it reads; callers list them in effect
 * dependencies, and a new identity re-runs the effect (a refetch loop for
 * `refresh`). These tests fail when the compiler is removed from the vitest
 * config, so they measure the compiler, not luck.
 */

import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useAlerts } from './useAlerts';
import { useAppDrawers } from './useAppDrawers';
import { useBonjourBrowse } from './useBonjourBrowse';
import { useChannelGraph } from './useChannelGraph';
import { useDefaults } from './useDefaults';
import { useDiscoveredDevices } from './useDiscoveredDevices';
import { useDriverStats } from './useDriverStats';
import { useGuestNetworkAudit } from './useGuestNetworkAudit';
import { useInsecurePortScan } from './useInsecurePortScan';
import { useNeighbourCache } from './useNeighbourCache';
import { usePlatformCapabilities } from './usePlatformCapabilities';
import { usePollingTargets } from './usePollingTargets';
import { useReports } from './useReports';
import { useTheme } from './useTheme';
import { useTopologyLinks, useTopologyNode, useTopologyNodes } from './useTopology';

const { mockGet, apiMock } = vi.hoisted(() => {
  const get = vi.fn<(path: string) => Promise<unknown>>();
  return {
    mockGet: get,
    apiMock: {
      api: {
        get: (path: string): Promise<unknown> => get(path),
        post: (): Promise<unknown> => Promise.resolve({}),
        put: (): Promise<unknown> => Promise.resolve({}),
        patch: (): Promise<unknown> => Promise.resolve({}),
        delete: (): Promise<unknown> => Promise.resolve({}),
      },
    },
  };
});
vi.mock('../api/client', () => apiMock);
vi.mock('../api', () => apiMock);

beforeEach(() => {
  mockGet.mockReset();
  mockGet.mockResolvedValue({});
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve({ ok: true, json: () => Promise.resolve({ reports: [] }) } as Response),
    ),
  );
});

async function settle(): Promise<void> {
  await act(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));
}

function functionsOf(value: object): [string, unknown][] {
  return Object.entries(value).filter(([, v]) => typeof v === 'function');
}

const cases: [string, () => object][] = [
  ['useAlerts', () => useAlerts()],
  ['useAppDrawers', () => useAppDrawers()],
  ['useBonjourBrowse', () => useBonjourBrowse()],
  ['useChannelGraph', () => useChannelGraph({ isWifi: true, currentInterface: 'wlan0' })],
  ['useDefaults', () => useDefaults()],
  ['useDiscoveredDevices', () => useDiscoveredDevices()],
  ['useDriverStats', () => useDriverStats('eth0')],
  ['useGuestNetworkAudit', () => useGuestNetworkAudit()],
  ['useInsecurePortScan', () => useInsecurePortScan()],
  ['useNeighbourCache', () => useNeighbourCache()],
  ['usePlatformCapabilities', () => usePlatformCapabilities()],
  ['usePollingTargets', () => usePollingTargets()],
  ['useReports', () => useReports()],
  ['useTheme', () => useTheme()],
  ['useTopologyLinks', () => useTopologyLinks()],
  ['useTopologyNode', () => useTopologyNode('n1')],
  ['useTopologyNodes', () => useTopologyNodes()],
];

describe('hook functions keep their identity across an unrelated re-render', () => {
  it.each(cases)('%s', async (_name, hook) => {
    const { result, rerender } = renderHook(hook);
    // Let the mount fetch settle so the re-render below changes nothing.
    await settle();
    const before = functionsOf(result.current);
    expect(before.length).toBeGreaterThan(0);
    rerender();
    for (const [name, fn] of before) {
      expect((result.current as Record<string, unknown>)[name], name).toBe(fn);
    }
  });
});
