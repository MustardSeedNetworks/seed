import { renderHook } from '@testing-library/react';
import type React from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import type { WiFiData } from '../components/cards/WiFiCard';
import type { WiFiResponse } from '../types/generated/wifi-response';
import { useNetworkFetchers } from './useNetworkFetchers';

type CardState = Parameters<typeof useNetworkFetchers>[0] extends {
  setCards: React.Dispatch<React.SetStateAction<infer S>>;
}
  ? S
  : never;

afterEach(() => {
  vi.unstubAllGlobals();
});

it.each<{ response: WiFiResponse; expected: WiFiData }>([
  {
    response: {
      interface: 'en0',
      wireless: true,
      status: 'detailsWithheld',
      reason: 'Location denied',
      remediation: 'Check System Settings',
    },
    expected: {
      status: 'detailsWithheld',
      reason: 'Location denied',
      remediation: 'Check System Settings',
    },
  },
  {
    response: { interface: 'en0', wireless: true, status: 'notAssociated', connected: false },
    expected: { status: 'notAssociated' },
  },
  {
    response: {
      interface: 'en0',
      wireless: true,
      status: 'associated',
      connected: true,
      ssid: 'lab',
      bssid: 'aa:bb:cc:dd:ee:ff',
      signal: -52,
      channel: 44,
      frequency: 5220,
      security: 'WPA2',
    },
    expected: {
      status: 'associated',
      ssid: 'lab',
      bssid: 'aa:bb:cc:dd:ee:ff',
      signal: -52,
      channel: 44,
      frequency: 5220,
      security: 'WPA2',
    },
  },
])(
  'preserves $response.status from the API through the card state',
  async ({ response, expected }) => {
    let cards = { wifi: null } as CardState;
    const setIsWifi = vi.fn();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(response) }),
    );
    const { result } = renderHook(() =>
      useNetworkFetchers({
        currentInterfaceRef: { current: 'en0' },
        setCards: (update) => {
          cards = typeof update === 'function' ? update(cards) : update;
        },
        setCurrentInterface: vi.fn(),
        setInterfaces: vi.fn(),
        setAppVersion: vi.fn(),
        setNetworkDiscovery: vi.fn(),
        setIsWifi,
        userSetWifiModeRef: { current: false },
        networkDiscoveryAbortRef: { current: null },
        prevLinkUpRef: { current: null },
      }),
    );
    await result.current.fetchWifiData();
    expect(cards.wifi).toEqual(expected);
    expect(setIsWifi).toHaveBeenCalledWith(true);
  },
);
