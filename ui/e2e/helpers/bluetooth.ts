import type { Page } from '@playwright/test';

// A synthetic bluetooth-scan job result with the decoded fields BT.1 added
// (companyName / serviceNames). Returned terminal on the submit response so the
// flow needs no SSE timing dance — the hook captures the result directly.
const BT_SCAN_RESULT = {
  devices: [
    {
      id: 'bt-dev-1',
      address: 'AA:BB:CC:DD:EE:01',
      name: 'AirPods Pro',
      alias: '',
      vendor: 'Apple',
      isConnected: true,
      type: 'ble',
      deviceClass: '',
      appearance: 0,
      rssi: -47,
      txPower: 0,
      estDistanceM: 0.8,
      isConnectable: true,
      serviceNames: ['Battery'],
      companyName: 'Apple',
      isAuthorized: false,
      isTrusted: true,
      isPaired: true,
      isBlocked: false,
      firstSeen: '2026-06-05T00:00:00Z',
      lastSeen: '2026-06-05T00:00:00Z',
    },
    {
      id: 'bt-dev-2',
      address: 'AA:BB:CC:DD:EE:02',
      name: 'Fitbit Charge',
      alias: '',
      vendor: 'Fitbit',
      isConnected: false,
      type: 'ble',
      deviceClass: '',
      appearance: 0,
      rssi: -71,
      txPower: 0,
      estDistanceM: 4.2,
      isConnectable: true,
      serviceNames: ['Heart Rate'],
      companyName: '',
      isAuthorized: false,
      isTrusted: false,
      isPaired: false,
      isBlocked: false,
      firstSeen: '2026-06-05T00:00:00Z',
      lastSeen: '2026-06-05T00:00:00Z',
    },
  ],
  adapterName: 'hci0',
  scanType: 'dual',
  scanTime: '2026-06-05T00:00:00Z',
  scanDurationMs: 5000,
  stats: {
    totalDevices: 2,
    classicDevices: 0,
    bleDevices: 2,
    dualDevices: 0,
    connectedDevices: 1,
    authorizedCount: 0,
    unauthorizedCount: 2,
    devicesByClass: {},
    vendorBreakdown: { Apple: 1, Fitbit: 1 },
    lastScanTime: '2026-06-05T00:00:00Z',
  },
};

// Mock POST /api/v1/jobs (the scan submit) to return an already-succeeded
// bluetooth-scan job carrying the result. The trailing-segment regex matches
// /api/v1/jobs but NOT /api/v1/jobs/events (the SSE stream), which is left to
// the real backend.
export async function mockBluetoothScanJob(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/jobs(\?.*)?$/, (route) => {
    if (route.request().method() !== 'POST') {
      return route.fallback();
    }
    return route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'bt-job-e2e',
        kind: 'bluetooth-scan',
        state: 'succeeded',
        progress: 1,
        result: BT_SCAN_RESULT,
      }),
    });
  });
}
