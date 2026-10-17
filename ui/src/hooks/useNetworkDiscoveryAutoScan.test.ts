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

import { COMMON_PORTS } from '../components/cards/NetworkDiscoveryCardHelpers';
import type {
  DiscoveredDevice,
  NetworkDiscoveryData,
} from '../components/cards/networkDiscoveryCardTypes';
import type { CreateJobRequest } from '../types/generated/create-job-request';
import type { OptionsResponse } from '../types/generated/options-response';
import type { VulnerabilitySettings } from '../types/generated/vulnerability-settings';
import { useNetworkDiscoveryAutoScan } from './useNetworkDiscoveryAutoScan';

const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();
const mockSubmitJob = vi.fn<(req: CreateJobRequest) => Promise<unknown>>();

vi.mock('../api', () => ({
  api: { post: (path: string, body?: unknown): Promise<unknown> => mockPost(path, body) },
}));

vi.mock('../lib/jobsClient', () => ({
  submitJob: (req: CreateJobRequest): Promise<unknown> => mockSubmitJob(req),
}));

const PORT_SCAN_PATH = '/api/v1/security/discovery/portscan';

function discoveryOptions(portScanEnabled: boolean): { options: OptionsResponse } {
  return {
    options: {
      passiveProtocols: { lldp: true, cdp: true, edp: true, ndp: true },
      arpScan: true,
      icmpScan: true,
      portScan: { enabled: portScanEnabled, preset: 'common', tcpPorts: '' },
      traceroute: false,
      snmpQuery: false,
    },
  };
}

function vulnSettings(enabled: boolean): VulnerabilitySettings {
  return {
    enabled,
    cveDatabase: 'nvd',
    hasNvdApiKey: false,
    updateInterval: 86400,
    severityThreshold: 'medium',
    maxConcurrent: 5,
    autoScan: enabled,
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

// The scan names the one device it is for. The hook used to post
// {targets:[ip]}, which the daemon did not read, so every trigger scanned
// every device (#2960).
function expectVulnScanQueued(): Promise<void> {
  return vi.waitFor(() =>
    expect(mockSubmitJob).toHaveBeenCalledWith({ kind: 'vuln-scan', params: { ip: DEVICE.ip } }),
  );
}

beforeEach(() => {
  vi.useFakeTimers();
  mockPost.mockResolvedValue({ ip: DEVICE.ip, services: [], scanTimeMs: 0 });
  mockSubmitJob.mockResolvedValue({ id: 'job-1', kind: 'vuln-scan', state: 'queued', progress: 0 });
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

  // The daemon decodes the request strictly. The hook also sent a
  // `timeout` the request does not have, so every deep scan answered 400.
  it('posts only fields the port-scan request accepts', async () => {
    stubDaemon(true, false);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectPosted(PORT_SCAN_PATH, { target: DEVICE.ip, ports: COMMON_PORTS });
  });

  it('does not port-scan when port scanning is disabled', async () => {
    stubDaemon(false, true);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectVulnScanQueued();
    await settle();
    expect(postedPaths()).not.toContain(PORT_SCAN_PATH);
  });

  it('queues a vulnerability scan when vulnerability auto-scan is enabled', async () => {
    stubDaemon(false, true);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectVulnScanQueued();
  });

  it('does not queue a vulnerability scan when vulnerability scanning is disabled', async () => {
    stubDaemon(true, false);
    renderHook(() => useNetworkDiscoveryAutoScan(DISCOVERY));
    await expectPosted(PORT_SCAN_PATH, expect.objectContaining({ target: DEVICE.ip }));
    await settle();
    expect(mockSubmitJob).not.toHaveBeenCalled();
  });
});

const SEVEN_DEVICES = Array.from(
  { length: 7 },
  (_, i) => ({ ip: `192.0.2.${i + 1}`, mac: `00:00:5e:00:53:0${i + 1}` }) as DiscoveredDevice,
);

function portScannedIps(): string[] {
  return mockPost.mock.calls
    .filter(([path]) => path === PORT_SCAN_PATH)
    .map(([, body]) => (body as { target: string }).target);
}

function vulnScannedIps(): string[] {
  return mockSubmitJob.mock.calls.map(([req]) => (req.params as { ip: string }).ip);
}

function expectEachDeviceOnce(ips: string[]): void {
  expect(ips).toHaveLength(SEVEN_DEVICES.length);
  expect(new Set(ips)).toEqual(new Set(SEVEN_DEVICES.map((d) => d.ip)));
}

// The settings arrive through fetch and body parsing, which fake timers do
// not drive, so wait for the first scan before stepping the clock.
function firstScan(scanned: () => string[]): Promise<void> {
  return vi.waitFor(() => expect(scanned().length).toBeGreaterThan(0));
}

// One act per second, so React commits each render, and runs each effect
// cleanup, between timers as it does in the browser.
async function stepSeconds(seconds: number): Promise<void> {
  for (let s = 0; s < seconds; s++) {
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
  }
}

// The hook marked every device scanned before its timers fired, and each
// re-run of the effect (a scan starting, or a discovery poll) cleared the
// pending timers, so only the first batch of three was ever scanned (#3092).
describe('useNetworkDiscoveryAutoScan reaches every device', () => {
  const data: NetworkDiscoveryData = { devices: SEVEN_DEVICES, status: DISCOVERY.status };

  it('port-scans every discovered device once', async () => {
    stubDaemon(true, false);
    renderHook(() => useNetworkDiscoveryAutoScan(data));
    await firstScan(portScannedIps);
    await stepSeconds(10);
    expectEachDeviceOnce(portScannedIps());
  });

  it('runs at most three port scans at once', async () => {
    stubDaemon(true, false);
    const pending: (() => void)[] = [];
    mockPost.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.push(() => resolve({ ip: '', services: [], scanTimeMs: 0 }));
        }),
    );
    renderHook(() => useNetworkDiscoveryAutoScan(data));
    await firstScan(portScannedIps);
    await settle();
    expect(portScannedIps()).toHaveLength(3);

    await act(async () => {
      pending.shift()?.();
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(portScannedIps()).toHaveLength(4);
  });

  it('keeps port-scanning when discovery polls mid-run', async () => {
    stubDaemon(true, false);
    const { rerender } = renderHook(({ d }) => useNetworkDiscoveryAutoScan(d), {
      initialProps: { d: data },
    });
    await firstScan(portScannedIps);
    for (let poll = 0; poll < 10; poll++) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });
      rerender({ d: { ...data, devices: [...SEVEN_DEVICES] } });
    }
    expectEachDeviceOnce(portScannedIps());
  });

  it('queues a vulnerability scan for every device when discovery polls mid-run', async () => {
    stubDaemon(false, true);
    const fingerprinted = SEVEN_DEVICES.map((d) => ({ ...d, osGuess: 'Linux' }));
    const vulnData = { ...data, devices: fingerprinted };
    const { rerender } = renderHook(({ d }) => useNetworkDiscoveryAutoScan(d), {
      initialProps: { d: vulnData },
    });
    await firstScan(vulnScannedIps);
    for (let poll = 0; poll < 10; poll++) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(400);
      });
      rerender({ d: { ...vulnData, devices: [...fingerprinted] } });
    }
    expectEachDeviceOnce(vulnScannedIps());
  });
});
