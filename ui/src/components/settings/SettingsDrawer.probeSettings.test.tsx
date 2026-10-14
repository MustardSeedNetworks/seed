/**
 * SettingsDrawer probe settings (S6-4 slice 40): an edit anywhere in the probe
 * settings saves every endpoint list back unchanged.
 *
 * The server replaces each list with what the PUT carries. The drawer used to
 * load and send only DNS, ping, TCP, UDP and HTTP, so changing the DNS test
 * hostname deleted every RTSP, DICOM, HL7, FHIR, SQL, file share, LDAP, LTI,
 * OPC-UA and Modbus target, and an HTTP endpoint's body match with it.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import type { ReactElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ProfileProvider } from '../../contexts/profileContext';
import { RoleProvider } from '../../contexts/RoleContext';
import { SETTINGS_RESPONSE } from '../../test/settingsResponse';
import type { TestsSettingsResponse } from '../../types/generated/tests-settings-response';
import { SettingsDrawer } from './SettingsDrawer';

const PROBES_PATH = '/api/v1/telemetry/probes/settings';

const STORED: TestsSettingsResponse = {
  dnsHostname: 'google.com',
  dnsServers: [{ address: '9.9.9.9', enabled: true }],
  pingTargets: [],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [
    {
      name: 'Portal',
      url: 'https://portal.example',
      expectedStatus: 200,
      enabled: true,
      bodyMatch: 'Welcome',
      checkSecurityHeaders: true,
    },
  ],
  rtspEndpoints: [{ name: 'Lobby camera', url: 'rtsp://cam.example:554/live', enabled: true }],
  dicomEndpoints: [
    {
      name: 'PACS',
      host: 'pacs.example',
      port: 104,
      calledAe: 'PACS-SCP',
      callingAe: 'SEED-SCU',
      enabled: true,
    },
  ],
  hl7Endpoints: [],
  fhirEndpoints: [],
  sqlEndpoints: [],
  fileShareEndpoints: [
    { name: 'Scans', protocol: 'smb', host: 'files.example', share: 'scans', enabled: true },
  ],
  ldapEndpoints: [],
  ltiEndpoints: [],
  opcuaEndpoints: [],
  modbusEndpoints: [
    {
      name: 'PLC',
      host: 'plc.example',
      port: 502,
      unitId: 1,
      testRegister: 0,
      enabled: false,
    },
  ],
  speedtest: { serverId: '', autoRunOnLink: false },
  iperf: { autoRunOnLink: false },
  runPerformance: true,
  runSpeedtest: true,
  runIperf: true,
  runDiscovery: true,
};

const { mockApi, mockPut } = vi.hoisted(() => {
  const put = vi.fn<(path: string, body: unknown) => Promise<unknown>>(() => Promise.resolve({}));
  return {
    mockPut: put,
    mockApi: {
      get: (path: string): Promise<unknown> =>
        Promise.resolve(
          path.includes('/users/me') ? { username: 'u', role: 'operator', isActive: true } : {},
        ),
      post: (): Promise<unknown> => Promise.resolve({}),
      put: (path: string, body: unknown): Promise<unknown> => put(path, body),
      patch: (): Promise<unknown> => Promise.resolve({}),
      delete: (): Promise<unknown> => Promise.resolve({}),
    },
  };
});
vi.mock('../../api/client', () => ({ api: mockApi }));
vi.mock('../../api', () => ({ api: mockApi }));
vi.mock('../../contexts/LicenseContext', () => ({
  useLicense: (): { status: { features: string[] } } => ({ status: { features: [] } }),
}));

async function tick(ms: number): Promise<void> {
  await act(() => vi.advanceTimersByTimeAsync(ms));
}

beforeEach(() => {
  vi.useFakeTimers();
  mockPut.mockClear();
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string) => {
      const body = url.endsWith(PROBES_PATH)
        ? STORED
        : url.endsWith('/api/v1/settings')
          ? SETTINGS_RESPONSE
          : {};
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200 }));
    }),
  );
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('SettingsDrawer probe settings', () => {
  it('saves every stored endpoint list back when the DNS hostname changes', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const node: ReactElement = (
      <QueryClientProvider client={queryClient}>
        <ProfileProvider>
          <RoleProvider isAuthenticated={true}>
            <SettingsDrawer isOpen={true} onClose={(): void => undefined} version="1.0.0" />
          </RoleProvider>
        </ProfileProvider>
      </QueryClientProvider>
    );
    render(node);
    await tick(0);
    const section = within(screen.getByTestId('dns-settings-section'));
    fireEvent.click(section.getByRole('button', { expanded: false }));
    await tick(0);

    fireEvent.change(screen.getByLabelText('Test Hostname'), {
      target: { value: 'example.org' },
    });
    await tick(800);

    const saves = mockPut.mock.calls.filter(([path]) => path === PROBES_PATH);
    expect(saves.map(([, body]) => body)).toEqual([{ ...STORED, dnsHostname: 'example.org' }]);
  });
});
