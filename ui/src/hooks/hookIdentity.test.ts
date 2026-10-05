/**
 * Hook callback identity, held by the React Compiler rather than by
 * hand-written useCallback/useMemo (UI-SEED-41 slices 2 to 5 and 10, #3066).
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
import { useAuth } from './useAuth';
import { useBluetoothScan } from './useBluetoothScan';
import { useBonjourBrowse } from './useBonjourBrowse';
import { useCapabilities } from './useCapabilities';
import { useChannelGraph } from './useChannelGraph';
import { useDefaults } from './useDefaults';
import { useDiscoveredDevices } from './useDiscoveredDevices';
import { useDriverStats } from './useDriverStats';
import { useEngineScan } from './useEngineScan';
import { useGuestNetworkAudit } from './useGuestNetworkAudit';
import { useInsecurePortScan } from './useInsecurePortScan';
import { useIperfServerSync } from './useIperfServerSync';
import { useLogs } from './useLogs';
import { useNeighbourCache } from './useNeighbourCache';
import { useNetworkDiscoveryAutoScan } from './useNetworkDiscoveryAutoScan';
import { usePacketCapture } from './usePacketCapture';
import { usePlatformCapabilities } from './usePlatformCapabilities';
import { usePollingTargets } from './usePollingTargets';
import { useReports } from './useReports';
import { useSettingsDrawerLoaders } from './useSettingsDrawerLoaders';
import { useSse } from './useSse';
import { useSubnetSettings } from './useSubnetSettings';
import { useTheme } from './useTheme';
import { useTopologyLinks, useTopologyNode, useTopologyNodes } from './useTopology';
import { useVulnerabilities } from './useVulnerabilities';
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
      beginSession: (): void => undefined,
      clearCSRFToken: (): void => undefined,
    },
  };
});
vi.mock('../api/client', () => apiMock);
vi.mock('../api', () => apiMock);
vi.mock('../contexts/RoleContext', () => ({ useRole: () => ({ canWrite: true }) }));

// The setup.ts EventSource mock cannot be constructed with `new` under
// Vitest 4, so useSse would only ever reach its catch branch.
class FakeEventSource {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 2;
  static opened = 0;
  readyState = FakeEventSource.OPEN;
  onopen: (() => void) | null = null;
  onmessage: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor() {
    FakeEventSource.opened += 1;
  }
  addEventListener(): void {}
  close(): void {
    this.readyState = FakeEventSource.CLOSED;
  }
}

beforeEach(() => {
  FakeEventSource.opened = 0;
  vi.stubGlobal('EventSource', FakeEventSource);
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
  ['useAuth', () => useAuth()],
  ['useBluetoothScan', () => useBluetoothScan()],
  ['useBonjourBrowse', () => useBonjourBrowse()],
  ['useCapabilities', () => useCapabilities(true)],
  ['useChannelGraph', () => useChannelGraph({ isWifi: true, currentInterface: 'wlan0' })],
  ['useDefaults', () => useDefaults()],
  ['useDiscoveredDevices', () => useDiscoveredDevices()],
  ['useDriverStats', () => useDriverStats('eth0')],
  ['useEngineScan', () => useEngineScan()],
  ['useGuestNetworkAudit', () => useGuestNetworkAudit()],
  ['useInsecurePortScan', () => useInsecurePortScan()],
  ['useLogs', () => useLogs()],
  ['useNeighbourCache', () => useNeighbourCache()],
  ['useNetworkDiscoveryAutoScan', () => useNetworkDiscoveryAutoScan(null)],
  ['usePacketCapture', () => usePacketCapture()],
  ['usePlatformCapabilities', () => usePlatformCapabilities()],
  ['usePollingTargets', () => usePollingTargets()],
  ['useReports', () => useReports()],
  ['useSse', () => useSse({ url: '/api/v1/events' })],
  ['useSubnetSettings', () => useSubnetSettings()],
  ['useTheme', () => useTheme()],
  ['useTopologyLinks', () => useTopologyLinks()],
  ['useTopologyNode', () => useTopologyNode('n1')],
  ['useTopologyNodes', () => useTopologyNodes()],
  ['useVulnerabilities', () => useVulnerabilities()],
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

// These hooks list their own callbacks in an effect, so a new identity
// re-runs the effect: a second write to the iperf listener, every settings
// section or the log list fetched again, or the event stream reopened.
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

  it('useSse opens the event stream once', async () => {
    const { result, rerender } = renderHook(() => useSse({ url: '/api/v1/events' }));
    await settle();
    expect(FakeEventSource.opened).toBe(1);
    expect(result.current.status).toBe('connecting');
    rerender();
    await settle();
    expect(FakeEventSource.opened).toBe(1);
  });

  it('useLogs fetches the log list and stats once', async () => {
    const { rerender } = renderHook(() => useLogs());
    await settle();
    expect(fetch).toHaveBeenCalledTimes(2);
    rerender();
    await settle();
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('useSettingsDrawerLoaders loads each section once per open', async () => {
    const set = vi.fn();
    const load = vi.fn(() => Promise.resolve());
    const args = {
      isOpen: true,
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
