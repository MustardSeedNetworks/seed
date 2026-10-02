/**
 * useNetworkDiscoveryAutoScan settings tests (seed#2687).
 *
 * The hook read `options.PortScan.Enabled` and `Enabled`/`AutoScan` from the
 * vulnerability settings, but both endpoints answer with lowercase keys, so
 * every flag came back undefined and no automatic port or vulnerability scan
 * ever ran. These tests feed the bodies the daemon actually sends.
 */

import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  DiscoveredDevice,
  NetworkDiscoveryData,
} from '../components/cards/networkDiscoveryCardTypes';
import type { OptionsResponse } from '../types/generated/options-response';
import { useNetworkDiscoveryAutoScan } from './useNetworkDiscoveryAutoScan';

const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();

vi.mock('../api', () => ({
  api: { post: (path: string, body?: unknown): Promise<unknown> => mockPost(path, body) },
}));

const PORT_SCAN_PATH = '/api/v1/security/discovery/portscan';
const VULN_SCAN_PATH = '/api/v1/security/vulnerabilities/scan';

function discoveryOptions(portScanEnabled: boolean): { options: OptionsResponse } {
  return {
    options: {
      passiveProtocols: { lldp: true, cdp: true, edp: true, ndp: true },
      arpScan: true,
      icmpScan: true,
      portScan: { enabled: portScanEnabled, preset: 'common', tcpPorts: '', udpPorts: '' },
      tcpProbe: { timeoutMs: 1000, workers: 10 },
      traceroute: false,
      snmpQuery: false,
    },
  };
}

// The shape `config.VulnerabilityScanConfig` marshals today.
function vulnSettings(enabled: boolean): Record<string, unknown> {
  return {
    enabled,
    cve_database: 'nvd',
    nvd_api_key: '',
    update_interval: 86400,
    severity_threshold: 'medium',
    max_concurrent: 5,
    auto_scan: enabled,
  };
}

function stubDaemon(portScanEnabled: boolean, vulnEnabled: boolean): void {
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string): Promise<Response> => {
      const body = url.endsWith('/security/discovery/options')
        ? discoveryOptions(portScanEnabled)
        : vulnSettings(vulnEnabled);
      return Promise.resolve(Response.json(body));
    }),
  );
}

const DEVICE = { ip: '192.0.2.10', mac: '00:00:5e:00:53:01', osGuess: 'Linux' } as DiscoveredDevice;

const DISCOVERY: NetworkDiscoveryData = {
  devices: [DEVICE],
  status: { scanning: false } as NetworkDiscoveryData['status'],
};

function expectPosted(path: string, body: unknown): Promise<void> {
  return vi.waitFor(() => expect(mockPost).toHaveBeenCalledWith(path, body));
}

async function settle(): Promise<void> {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
}

function postedPaths(): string[] {
  return mockPost.mock.calls.map(([path]) => path);
}

beforeEach(() => {
  vi.useFakeTimers();
  mockPost.mockResolvedValue({ ip: DEVICE.ip, services: [] });
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

// Each negative case waits for the other scan first, which proves the
// settings were loaded before asserting the disabled one never ran.
describe('useNetworkDiscoveryAutoScan settings', () => {
  it('port-scans a discovered device when port scanning is enabled', async () => {
    stubDaemon(true, false);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectPosted(PORT_SCAN_PATH, expect.objectContaining({ target: DEVICE.ip }));
  });

  it('does not port-scan when port scanning is disabled', async () => {
    stubDaemon(false, true);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectPosted(VULN_SCAN_PATH, { targets: [DEVICE.ip] });
    await settle();
    expect(postedPaths()).not.toContain(PORT_SCAN_PATH);
  });

  it('queues a vulnerability scan when vulnerability auto-scan is enabled', async () => {
    stubDaemon(false, true);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectPosted(VULN_SCAN_PATH, { targets: [DEVICE.ip] });
  });

  it('does not queue a vulnerability scan when vulnerability scanning is disabled', async () => {
    stubDaemon(true, false);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectPosted(PORT_SCAN_PATH, expect.objectContaining({ target: DEVICE.ip }));
    await settle();
    expect(postedPaths()).not.toContain(VULN_SCAN_PATH);
  });
});
