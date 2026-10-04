/**
 * Hook callback identity, held by the React Compiler rather than by
 * hand-written useCallback/useMemo (UI-SEED-41 slices 2 and 3, #3066).
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
import { useAlertWebhookSettings } from './useAlertWebhookSettings';
import { useAppDrawers } from './useAppDrawers';
import { useBonjourBrowse } from './useBonjourBrowse';
import { useCapabilities } from './useCapabilities';
import { useChannelGraph } from './useChannelGraph';
import { useDefaults } from './useDefaults';
import { useDiscoveredDevices } from './useDiscoveredDevices';
import { useDriverStats } from './useDriverStats';
import { useGuestNetworkAudit } from './useGuestNetworkAudit';
import { useInsecurePortScan } from './useInsecurePortScan';
import { useIperfServerSync } from './useIperfServerSync';
import { useNeighbourCache } from './useNeighbourCache';
import { usePlatformCapabilities } from './usePlatformCapabilities';
import { usePollingTargets } from './usePollingTargets';
import { useReports } from './useReports';
import { useSettingsDrawerLoaders } from './useSettingsDrawerLoaders';
import { useSubnetSettings } from './useSubnetSettings';
import { useTheme } from './useTheme';
import { useTopologyLinks, useTopologyNode, useTopologyNodes } from './useTopology';
import { useVulnerabilitySettings } from './useVulnerabilitySettings';

const { mockGet, mockPost, apiMock } = vi.hoisted(() => {
  const get = vi.fn<(path: string) => Promise<unknown>>();
  const post = vi.fn<(path: string) => Promise<unknown>>();
  return {
    mockGet: get,
    mockPost: post,
    apiMock: {
      api: {
        get: (path: string): Promise<unknown> => get(path),
        post: (path: string): Promise<unknown> => post(path),
        put: (): Promise<unknown> => Promise.resolve({}),
        patch: (): Promise<unknown> => Promise.resolve({}),
        delete: (): Promise<unknown> => Promise.resolve({}),
      },
    },
  };
});
vi.mock('../api/client', () => apiMock);
vi.mock('../api', () => apiMock);
vi.mock('../contexts/RoleContext', () => ({ useRole: () => ({ canWrite: true }) }));

beforeEach(() => {
  mockGet.mockReset();
  mockGet.mockResolvedValue({});
  mockPost.mockReset();
  mockPost.mockResolvedValue({});
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
  ['useAlertWebhookSettings', () => useAlertWebhookSettings()],
  ['useAppDrawers', () => useAppDrawers()],
  ['useBonjourBrowse', () => useBonjourBrowse()],
  ['useCapabilities', () => useCapabilities()],
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
  ['useSubnetSettings', () => useSubnetSettings()],
  ['useTheme', () => useTheme()],
  ['useTopologyLinks', () => useTopologyLinks()],
  ['useTopologyNode', () => useTopologyNode('n1')],
  ['useTopologyNodes', () => useTopologyNodes()],
  ['useVulnerabilitySettings', () => useVulnerabilitySettings()],
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

// These two hooks list their own callbacks in an effect, so a new identity
// re-runs the effect: a second write to the iperf listener, or every settings
// section fetched again.
describe('hook effects do not re-run on an unrelated re-render', () => {
  it('useIperfServerSync starts the listener once', async () => {
    mockGet.mockResolvedValue({ running: false, port: 5201, pid: 0 });
    const onStatus = vi.fn();
    const { rerender } = renderHook(() =>
      useIperfServerSync({ installed: true, enableServer: true, serverPort: 5201, onStatus }),
    );
    await settle();
    expect(mockPost).toHaveBeenCalledTimes(1);
    rerender();
    await settle();
    expect(mockPost).toHaveBeenCalledTimes(1);
  });

  it('useSettingsDrawerLoaders loads each section once per open', async () => {
    const ref = (): { current: boolean } => ({ current: false });
    const initRefs = {
      initialLoadRef: ref(),
      thresholdsInitRef: ref(),
      testsInitRef: ref(),
      wifiInitRef: ref(),
      linkInitRef: ref(),
      cableTestInitRef: ref(),
      networkDiscoveryInitRef: ref(),
      snmpInitRef: ref(),
      vulnInitRef: ref(),
    };
    const set = vi.fn();
    const load = vi.fn(() => Promise.resolve());
    const args = {
      isOpen: true,
      initRefs,
      setThresholds: set,
      setIpSettings: set,
      setDnsInput: set,
      setTestsSettings: set,
      setIperfSuggestions: set,
      setIperfSuggestionsStatus: set,
      setIperfSuggestionsError: set,
      setWifiSettings: set,
      setNetworkDiscoverySettings: set,
      setSnmpSettings: set,
      setLinkSettings: set,
      setCableTestSettings: set,
      setLogPreview: set,
      setLogLoading: set,
      setLogError: set,
      fetchSubnets: load,
      fetchVulnSettings: load,
    };
    const { result, rerender } = renderHook(() => useSettingsDrawerLoaders(args));
    await settle();
    const fetches = vi.mocked(fetch).mock.calls.length;
    expect(fetches).toBeGreaterThan(0);
    const { fetchIperfSuggestions, fetchLogPreview } = result.current;
    rerender();
    await settle();
    expect(vi.mocked(fetch).mock.calls.length).toBe(fetches);
    expect(result.current.fetchIperfSuggestions).toBe(fetchIperfSuggestions);
    expect(result.current.fetchLogPreview).toBe(fetchLogPreview);
  });
});
