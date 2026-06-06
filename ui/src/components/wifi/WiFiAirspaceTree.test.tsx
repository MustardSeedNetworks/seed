import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SSIDGroup } from '../../types/generated/wifi-airspace-response';
import { WiFiAirspaceTree } from './WiFiAirspaceTree';

const group: SSIDGroup = {
  ssid: 'corp',
  hidden: false,
  apCount: 1,
  bssCount: 1,
  stationCount: 1,
  aps: [
    {
      key: 'ap1',
      vendor: 'Cisco',
      bsses: [
        {
          bssid: '00:11:22:33:44:55',
          ssid: 'corp',
          hidden: false,
          band: '5 GHz',
          channel: 36,
          security: 'WPA3',
          standard: '802.11ax (Wi-Fi 6)',
          pmfRequired: true,
          rrmNeighbor: false,
          btmSupported: false,
          ftSupported: false,
          wpsEnabled: false,
          signalDbm: -50,
          beacons: 10,
          lastSeen: '2026-01-01T00:00:00Z',
          stations: [
            {
              mac: 'aa:bb:cc:dd:ee:ff',
              signalDbm: -60,
              frames: 5,
              lastSeen: '2026-01-01T00:00:00Z',
            },
          ],
        },
      ],
    },
  ],
};

describe('WiFiAirspaceTree', () => {
  it('shows an empty state with no SSIDs', () => {
    render(<WiFiAirspaceTree ssids={[]} />);
    expect(screen.getByTestId('wifi-airspace-empty')).toBeInTheDocument();
  });

  it('renders the SSID -> AP -> BSSID -> client hierarchy', () => {
    render(<WiFiAirspaceTree ssids={[group]} />);
    expect(screen.getByTestId('wifi-airspace-tree')).toBeInTheDocument();
    expect(screen.getByText('corp')).toBeInTheDocument();
    expect(screen.getByTestId('wifi-bss')).toHaveTextContent('00:11:22:33:44:55');
    expect(screen.getByTestId('wifi-station')).toHaveTextContent('aa:bb:cc:dd:ee:ff');
  });

  it('labels a cloaked SSID rather than showing a blank name', () => {
    render(<WiFiAirspaceTree ssids={[{ ...group, ssid: '', hidden: true }]} />);
    expect(screen.getByText('(hidden SSID)')).toBeInTheDocument();
  });
});
