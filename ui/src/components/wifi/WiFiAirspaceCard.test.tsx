/**
 * Below Pro the airspace endpoint strips every station and station count and
 * sets `clientsWithheld` (seed#2889). The card must state the tier boundary
 * rather than render the zeroed counts as an empty airspace.
 */

import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { WiFiAirspaceResponse } from '../../types/generated/wifi-airspace-response';
import { WiFiAirspaceCard } from './WiFiAirspaceCard';

let response: WiFiAirspaceResponse;

vi.mock('../../hooks/useWifiVisibility', () => ({
  useWifiAirspace: () => ({ data: response, isLoading: false, isError: false }),
}));

function airspace(clientsWithheld: boolean): WiFiAirspaceResponse {
  const stations = clientsWithheld
    ? []
    : [{ mac: 'aa:bb:cc:dd:ee:ff', signalDbm: -60, frames: 5, lastSeen: '2026-01-01T00:00:00Z' }];
  return {
    clientsWithheld,
    status: {
      captureActive: false,
      ssids: 1,
      aps: 1,
      bsses: 1,
      stations: stations.length,
      anomalies: 0,
    },
    ssids: [
      {
        ssid: 'corp',
        hidden: false,
        apCount: 1,
        bssCount: 1,
        stationCount: stations.length,
        aps: [
          {
            key: 'ap1',
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
                channelWidthMhz: 80,
                channelUtil: 0,
                advertisedStations: 0,
                hasBssLoad: false,
                signalDbm: -50,
                beacons: 10,
                recentDeauths: 0,
                lastSeen: '2026-01-01T00:00:00Z',
                stations,
              },
            ],
          },
        ],
      },
    ],
  };
}

describe('WiFiAirspaceCard', () => {
  beforeEach(() => {
    response = airspace(false);
  });

  it('shows client counts and stations when the tier includes them', () => {
    render(<WiFiAirspaceCard />);
    expect(screen.queryByTestId('wifi-clients-withheld')).not.toBeInTheDocument();
    expect(screen.getByTestId('wifi-capture-status')).toHaveTextContent('1Clients');
    expect(screen.getByTestId('wifi-ssid-group')).toHaveTextContent('1 clients');
    expect(screen.getByTestId('wifi-station')).toHaveTextContent('aa:bb:cc:dd:ee:ff');
  });

  it('states the tier boundary instead of zero clients when they are withheld', () => {
    response = airspace(true);
    render(<WiFiAirspaceCard />);
    expect(screen.getByTestId('wifi-clients-withheld')).toHaveTextContent(
      'Clients require the Pro tier.',
    );
    expect(screen.getByTestId('wifi-capture-status')).not.toHaveTextContent(/\dClients/);
    expect(screen.getByTestId('wifi-ssid-group')).not.toHaveTextContent(/clients/i);
    expect(screen.getByTestId('wifi-bss')).toHaveTextContent('00:11:22:33:44:55');
  });

  it('describes a map built from scans as well as capture', () => {
    render(<WiFiAirspaceCard />);
    expect(screen.getByText(/Wi-Fi scans/)).toBeInTheDocument();
  });
});
