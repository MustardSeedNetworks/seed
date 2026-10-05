/**
 * SettingsDrawer auto-save (#2994): an edit is saved however soon after
 * opening it is made, and a loaded value is never saved however late it lands.
 *
 * The drawer used to ignore every change for 500 ms after opening. An edit
 * inside that window was dropped, which is how the discovery-timing E2E lost
 * its save when a busy runner made the user fast relative to the timer, and a
 * load landing after it was written straight back to the daemon.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import type { ReactElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ProfileProvider } from '../../contexts/profileContext';
import { RoleProvider } from '../../contexts/RoleContext';
import { SettingsDrawer } from './SettingsDrawer';

const SETTINGS_PATH = '/api/v1/security/devices/settings';

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

/** The stored discovery settings, served when `release` is called. */
function deferDiscoveryLoad(): { release: () => void } {
  let release = (): void => undefined;
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string) => {
      const body = url.endsWith(SETTINGS_PATH) ? { timing: { rescanIntervalMs: 120_000 } } : {};
      const response = new Response(JSON.stringify(body), { status: 200 });
      return url.endsWith(SETTINGS_PATH) ? held.then(() => response) : Promise.resolve(response);
    }),
  );
  return { release: (): void => release() };
}

async function tick(ms: number): Promise<void> {
  await act(() => vi.advanceTimersByTimeAsync(ms));
}

async function openDiscovery(): Promise<HTMLInputElement> {
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
  const section = within(screen.getByTestId('discovery-settings-section'));
  fireEvent.click(section.getByRole('button', { name: /^Discovery/, expanded: false }));
  await tick(0);
  return screen.getByTestId('discovery-rescan-interval');
}

function discoveryPuts(): unknown[] {
  return mockPut.mock.calls.filter(([path]) => path === SETTINGS_PATH).map(([, body]) => body);
}

beforeEach(() => {
  vi.useFakeTimers();
  mockPut.mockClear();
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('SettingsDrawer auto-save', () => {
  it('saves an edit made as the drawer opens', async () => {
    deferDiscoveryLoad().release();
    const rescan = await openDiscovery();
    expect(rescan).toHaveValue(2);

    fireEvent.change(rescan, { target: { value: '7' } });
    await tick(800);

    expect(discoveryPuts()).toEqual([
      expect.objectContaining({ timing: { rescanIntervalMs: 420_000 } }),
    ]);
  });

  it('never writes back a load that lands late', async () => {
    const load = deferDiscoveryLoad();
    const rescan = await openDiscovery();

    await tick(2000);
    load.release();
    await tick(0);
    expect(rescan).toHaveValue(2);
    await tick(5000);

    expect(discoveryPuts()).toEqual([]);
  });
});
