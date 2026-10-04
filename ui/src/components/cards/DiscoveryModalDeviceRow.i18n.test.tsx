/**
 * DiscoveryModalDeviceRow.i18n.test.tsx — the expanded device detail and the
 * channel graph speak the operator's language.
 *
 * .github#100: every LLDP, CDP, SNMP and inventory label here is a single word
 * ("System:", "Uptime:") or ends at an expression ("Interfaces ({n}):"), so the
 * shared JSX-text gate skipped them and a Spanish operator read the whole
 * detail panel in English. Each case asserts the Spanish text and that the
 * English is gone.
 */

import { render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import i18n from '../../i18n';
import type { DiscoveredDevice } from '../../types/generated/engine-discovery-response';
import { DeviceRow } from './DiscoveryModalDeviceRow';
import { WifiChannelGraph } from './WiFiChannelGraph';

const device: DiscoveredDevice = {
  ip: '10.44.10.7',
  mac: '00:1b:21:aa:bb:01',
  hostname: 'sw-core-01',
  discoveryMethod: ['lldp', 'cdp', 'snmp'],
  lastSeen: new Date().toISOString(),
  isLocal: true,
  lldpInfo: {
    chassisId: '00:1b:21:aa:bb:00',
    portId: 'Gi1/0/24',
    systemName: 'sw-core-01',
    capabilities: ['bridge', 'router'],
  },
  cdpInfo: { deviceId: 'sw-core-01', portId: 'Gi1/0/24', platform: 'cisco WS-C3850' },
  snmpData: {
    collectedAt: new Date().toISOString(),
    system: {
      sysDescr: 'Cisco IOS XE',
      sysObjectId: '1.3.6.1.4.1.9.1.2066',
      sysName: 'sw-core-01',
      sysContact: 'noc@example.net',
      sysLocation: 'IDF-3B',
      sysUpTime: 8_640_000,
    },
    interfaces: [{ index: 1, name: 'Gi1/0/1', speedMbps: 1000, operStatus: 'up' }],
    inventory: [{ index: 1, class: 'chassis', name: 'Chassis', modelName: 'WS-C3850-48P' }],
  },
};

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('device detail — Spanish, with no English left behind', () => {
  it('labels every LLDP, CDP, SNMP and inventory field', () => {
    render(
      <table>
        <DeviceRow
          device={device}
          isExpanded={true}
          onToggle={() => {}}
          onShowVulnerabilities={() => {}}
          isScanning={false}
        />
      </table>,
    );

    for (const spanish of [
      'Sistema:',
      'Puerto:',
      'Capacidades:',
      'Dispositivo:',
      'Plataforma:',
      'Nombre:',
      'Descripción:',
      'Ubicación:',
      'Contacto:',
      'Tiempo activo:',
      'Interfaces (1):',
      'Hardware:',
    ]) {
      expect(screen.getByText(spanish)).toBeInTheDocument();
    }
    expect(screen.getByText(/Modelo: WS-C3850-48P/)).toBeInTheDocument();

    for (const english of [
      'System:',
      'Port:',
      'Capabilities:',
      'Device:',
      'Platform:',
      'Name:',
      'Description:',
      'Location:',
      'Contact:',
      'Uptime:',
      /Model:/,
    ]) {
      expect(screen.queryByText(english)).not.toBeInTheDocument();
    }
  });

  it('labels the channel graph axis', () => {
    render(
      <WifiChannelGraph
        data={{
          available: true,
          data: {
            networks24Ghz: [
              {
                ssid: 'lab-ap',
                bssid: '74:ac:b9:3b:af:40',
                channel: 6,
                centerFreq: 2437,
                channelWidth: 20,
                signal: -52,
                band: '2.4GHz',
                isConnected: false,
              },
            ],
            networks5Ghz: [],
            networks6Ghz: [],
            scanTime: new Date().toISOString(),
          },
        }}
      />,
    );

    expect(screen.getByText('Canal')).toBeInTheDocument();
    expect(screen.queryByText('Channel')).not.toBeInTheDocument();
  });
});
