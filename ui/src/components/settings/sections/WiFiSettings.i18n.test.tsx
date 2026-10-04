/**
 * WiFiSettings.i18n.test.tsx — the network list and connect dialog speak the
 * operator's language.
 *
 * .github#100: "Forget", "Cancel" and "Disconnect" are single words, and
 * "Connect to {ssid}" and "Ch {n}" end at an expression, so the shared JSX-text
 * gate skipped all of them and a Spanish operator read them in English. Each
 * case asserts the Spanish text and that the English is gone.
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';
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

beforeEach(async () => {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role: 'operator', isActive: true });
    }
    if (path.includes('/wifi/saved')) {
      return Promise.resolve({ networks: [{ ssid: 'lab-ssid', uuid: 'u1' }] });
    }
    if (path.includes('/wifi/scan')) {
      return Promise.resolve({
        networks: [
          {
            ssid: 'lab-ap',
            bssid: '74:ac:b9:3b:af:40',
            signal: -52,
            channel: 36,
            frequency: 5180,
            security: 'WPA2',
          },
        ],
      });
    }

    return Promise.resolve({});
  });
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('WiFiSettings — Spanish, with no English left behind', () => {
  it('labels the scan list, the connect dialog and the saved networks', async () => {
    render(
      <RoleProvider isAuthenticated={true}>
        <WiFiSettings
          wifiSettings={{ interface: 'wlan0', availableWifi: ['wlan0'], isWireless: true }}
          setWifiSettings={(): void => undefined}
          wifiStatus="idle"
        />
      </RoleProvider>,
    );
    await userEvent.click(await screen.findByRole('button', { name: /wi-?fi/i }));

    expect(await screen.findByText('Can. 36')).toBeInTheDocument();
    expect(screen.getByText(/Redes disponibles/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Olvidar' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Desconectar' })).toBeInTheDocument();

    await userEvent.click(screen.getByText('lab-ap'));
    expect(screen.getByText('Conectar a lab-ap')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancelar' })).toBeInTheDocument();

    for (const english of [
      'Ch 36',
      /Available Networks/,
      'Forget',
      'Disconnect',
      'Connect to lab-ap',
      'Cancel',
    ]) {
      expect(screen.queryByText(english)).not.toBeInTheDocument();
    }
  });
});
