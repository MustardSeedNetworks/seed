import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { WiFiCard } from './WiFiCard';
import { WifiChannelGraph } from './WiFiChannelGraph';

vi.mock('../../contexts/useSettings', () => ({ useSettings: () => ({ thresholds: {} }) }));
beforeEach(async () => {
  await i18n.changeLanguage('en');
});
afterEach(() => {
  vi.clearAllMocks();
});

it('shows withheld details and remediation without claiming a disconnection', () => {
  render(
    <WiFiCard
      data={{
        status: 'detailsWithheld',
        reason: 'Location Services withheld the network details.',
        remediation: 'Check the connection in System Settings > Wi-Fi.',
      }}
    />,
  );
  expect(screen.getByTestId('wifi-details-withheld')).toHaveTextContent('Wi-Fi details hidden');
  expect(screen.getByText(/Check the connection in System Settings > Wi-Fi/)).toBeVisible();
  expect(screen.queryByText('Disconnected', { exact: false })).not.toBeInTheDocument();
});

it('shows a scan error instead of interpreting it as empty airspace', () => {
  render(
    <WifiChannelGraph
      data={{
        interface: 'en0',
        available: true,
        error: 'Location Services withheld the network details.',
      }}
      visible
    />,
  );
  expect(screen.getByText('Location Services withheld the network details.')).toBeVisible();
  expect(screen.queryByText(/No networks detected/)).not.toBeInTheDocument();
});

it('renders the withheld guidance in Spanish', async () => {
  await i18n.changeLanguage('es');
  render(
    <WiFiCard
      data={{
        status: 'detailsWithheld',
        reason: 'Location denied',
        remediation: 'Check System Settings',
      }}
    />,
  );
  expect(screen.getByTestId('wifi-details-withheld')).toHaveTextContent(
    'Detalles de Wi-Fi ocultos',
  );
  expect(screen.getByTestId('wifi-details-withheld')).toHaveTextContent(
    'Ajustes del Sistema > Wi-Fi',
  );
});

it('renders an observed disconnection explicitly', () => {
  render(<WiFiCard data={{ status: 'notAssociated' }} />);
  expect(screen.getByTestId('wifi-not-associated')).toHaveTextContent('Disconnected');
  expect(screen.queryByTestId('wifi-details-withheld')).not.toBeInTheDocument();
});

it('renders an authorized association with channel and signal', () => {
  render(
    <WiFiCard
      data={{
        status: 'associated',
        ssid: 'lab',
        bssid: 'aa:bb:cc:dd:ee:ff',
        channel: 44,
        frequency: 5220,
        signal: -52,
        security: 'WPA2',
      }}
    />,
  );
  expect(screen.getByTestId('wifi-associated')).toHaveTextContent('lab');
  expect(screen.getByTestId('wifi-associated')).toHaveTextContent('-52 dBm');
  expect(screen.getByTestId('wifi-associated')).toHaveTextContent('44');
});

it('draws each network from the signal and channel width the daemon sends', () => {
  const { container } = render(
    <WifiChannelGraph
      data={{
        interface: 'wlan0',
        available: true,
        data: {
          networks24Ghz: [],
          networks5Ghz: [
            {
              ssid: 'lab',
              bssid: 'aa:bb:cc:dd:ee:ff',
              channel: 36,
              centerFreqMhz: 5180,
              channelWidthMhz: 80,
              signalDbm: -51,
              band: '5GHz',
              isConnected: false,
            },
          ],
          networks6Ghz: [],
          scanTime: '2026-10-12T00:00:00Z',
        },
      }}
    />,
  );
  const curve = container.querySelector('path[d] > title')?.parentElement;
  expect(curve?.getAttribute('d')).toMatch(/^M [\d.]+,/);
  expect(curve?.getAttribute('d')).not.toContain('NaN');
  if (curve) {
    fireEvent.mouseEnter(curve);
  }
  expect(screen.getByText('-51 dBm')).toBeVisible();
  expect(screen.getByText('80 MHz')).toBeVisible();
});
