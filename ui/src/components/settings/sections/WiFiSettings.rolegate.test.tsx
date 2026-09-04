/**
 * WiFiSettings role-gating tests (#1254).
 *
 * connect, disconnect and forget all hit routes the backend registers with
 * `minRole: op`, so a viewer's click can only ever return 403. These assert the
 * controls say so before the click rather than after it.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactElement, ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import { WiFiSettings } from './WiFiSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));

function asUser(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }
    if (path.includes('/wifi/saved')) {
      return Promise.resolve({ networks: [{ ssid: 'lab-ssid', uuid: 'u1' }] });
    }

    return Promise.resolve({});
  });
}

function renderSettings(): ReactElement {
  const node = (
    <RoleProvider isAuthenticated={true}>
      <WiFiSettings
        wifiSettings={{ interface: 'wlan0', availableWifi: ['wlan0'], isWireless: true }}
        setWifiSettings={(): void => undefined}
        wifiStatus="idle"
      />
    </RoleProvider>
  );
  render(node as ReactNode as ReactElement);

  return node;
}

/** The section is a CollapsibleSection and starts closed, so nothing inside it
 *  is in the DOM until the header is clicked. */
async function openSection(): Promise<void> {
  const header = await screen.findByRole('button', { name: /wi-?fi/i });
  await userEvent.click(header);
}

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('WiFiSettings — role gating', () => {
  it('disables Forget for a viewer and says why', async () => {
    asUser('viewer');
    renderSettings();
    await openSection();

    const forget = await screen.findByRole('button', { name: 'Forget' });
    expect(forget).toBeDisabled();
    expect(forget).toHaveAttribute(
      'title',
      'Read-only — operator role required to change the Wi-Fi connection',
    );
  });

  it('leaves Forget usable for an operator', async () => {
    asUser('operator');
    renderSettings();
    await openSection();

    const forget = await screen.findByRole('button', { name: 'Forget' });
    await waitFor(() => {
      expect(forget).not.toBeDisabled();
    });
    expect(forget).not.toHaveAttribute('title');
  });
});
