/**
 * endpoint-options.i18n.test.tsx — the FHIR auth and OPC UA security choices,
 * and the discovery interface fallback, speak the operator's language.
 *
 * .github#100: "None", "Sign" and `'auto'` are single words, which the shared
 * copy gates skipped, so a Spanish operator picked "None" from an otherwise
 * Spanish form. Each case asserts the Spanish option and that the English is
 * gone.
 */

import { render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import i18n from '../../../i18n';
import type { TestsSettings } from '../../../types/settings';
import { DiscoveryServiceStatus } from './discovery/DiscoveryServiceStatus';
import { ClinicalEndpoints } from './specialty/ClinicalEndpoints';
import { IndustrialEndpoints } from './specialty/IndustrialEndpoints';

const settings: TestsSettings = {
  dnsHostname: '',
  dnsServers: [],
  pingTargets: [],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [],
  fhirEndpoints: [
    { id: 'f1', name: 'ehr', baseUrl: 'https://ehr.example', authType: 'none', enabled: true },
  ],
  opcuaEndpoints: [
    {
      id: 'o1',
      name: 'plc',
      endpointUrl: 'opc.tcp://plc.example:4840',
      securityMode: 'None',
      enabled: true,
    },
  ],
  runPerformance: false,
  runSpeedtest: false,
  runIperf: false,
  runDiscovery: false,
  speedtest: { serverId: '', autoRunOnLink: false },
  iperf: { autoRunOnLink: false },
};

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  await i18n.changeLanguage('en');
});

function optionLabels(): string[] {
  return screen.getAllByRole('option').map((o) => o.textContent ?? '');
}

describe('endpoint options — Spanish, with no English left behind', () => {
  it('names the FHIR auth types', () => {
    render(<ClinicalEndpoints testsSettings={settings} setTestsSettings={() => {}} />);

    expect(optionLabels()).toEqual(expect.arrayContaining(['Ninguna', 'Basic', 'OAuth2']));
    expect(optionLabels()).not.toContain('None');
  });

  it('names the OPC UA security modes', () => {
    render(<IndustrialEndpoints testsSettings={settings} setTestsSettings={() => {}} />);

    expect(optionLabels()).toEqual(['Ninguna', 'Firmar', 'Firmar+Cifrar']);
  });

  it('says the discovery interface is chosen automatically', () => {
    render(
      <DiscoveryServiceStatus
        status={{
          running: true,
          scanning: false,
          deviceCount: 3,
          lastScan: '',
          subnet: '10.44.10.0/24',
          localIP: '10.44.10.2',
          interface: '',
          activeMethods: [],
          rescanInterval: 0,
        }}
        loading={false}
        onRefresh={() => {}}
      />,
    );

    expect(screen.getByText('automática')).toBeInTheDocument();
    expect(screen.queryByText('auto')).not.toBeInTheDocument();
  });
});
