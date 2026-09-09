/**
 * settings-sections.fixtures.tsx — one mounting fixture per settings section.
 *
 * Two suites read this table: the role-gate suite (#2467) and the i18n suite
 * (S1-14c). Keeping it in one place is the point — a section added to only one
 * of them is exactly the gap both were written to close.
 */

import type { ReactElement } from 'react';

import {
  DEFAULT_CABLE_TEST_SETTINGS,
  DEFAULT_CARD_SETTINGS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_IPERF_SETTINGS,
  DEFAULT_LINK_SETTINGS,
  DEFAULT_NETWORK_DISCOVERY_SETTINGS,
  DEFAULT_SNMP_SETTINGS,
  DEFAULT_TESTS_SETTINGS,
  DEFAULT_THRESHOLDS,
  DEFAULT_VULNERABILITY_SETTINGS,
  type IpSettings,
} from '../../../types/settings';
import { SettingsDrawerNetworkSection } from '../SettingsDrawerNetworkSection';
import { AppearanceSettings } from './AppearanceSettings';
import { CableTestSettings } from './CableTestSettings';
import { ConfigBackupsSection } from './ConfigBackupsSection';
import { DiscoverySettings } from './DiscoverySettings';
import { DnsSettings } from './DnsSettings';
import { HealthChecksSettings } from './HealthChecksSettings';
import { InterfacesSettings } from './InterfacesSettings';
import { LinkSettings } from './LinkSettings';
import { PerformanceSettings } from './PerformanceSettings';
import { SsoSettings } from './SsoSettings';
import { ThresholdsSettings } from './ThresholdsSettings';
import { UsersSettings } from './UsersSettings';
import { VulnerabilitySettings } from './VulnerabilitySettings';
import { WiFiSettings } from './WiFiSettings';

/** A section that is all write controls: one fieldset covers the whole body. */
export interface SectionFixture {
  name: string;
  header: RegExp;
  render: () => ReactElement;
}

/** A section that also carries a read, so it gates per control or per group. */
export interface MixedSectionFixture extends SectionFixture {
  /** The control that reads and must stay usable for a viewer. */
  readControl: RegExp;
  /** Further read-only controls that must survive the gate, by accessible name. */
  alsoUsable?: RegExp[];
}

export const noop = (): void => undefined;
const ipSettings: IpSettings = {
  mode: 'dhcp',
  address: '',
  netmask: '',
  gateway: '',
  dns: [],
};

/**
 * Each section's props are its own; the fixture list keeps the assertion
 * identical across all of them, which is the point — one missed section is
 * exactly the defect this covers.
 */
export const SECTIONS: SectionFixture[] = [
  {
    name: 'LinkSettings',
    header: /link/i,
    render: () => (
      <LinkSettings
        linkSettings={DEFAULT_LINK_SETTINGS}
        setLinkSettings={noop}
        linkStatus="idle"
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={noop}
      />
    ),
  },
  {
    name: 'SettingsDrawerNetworkSection',
    header: /network/i,
    render: () => (
      <SettingsDrawerNetworkSection
        ipSettings={ipSettings}
        setIpSettings={noop}
        dnsInput=""
        setDnsInput={noop}
        saveIpSettings={(): Promise<void> => Promise.resolve()}
        savingIp={false}
        ipMessage={null}
        displayOptions={DEFAULT_DISPLAY_OPTIONS}
        setDisplayOptions={noop}
        displayStatus="idle"
        isValidIp={(): boolean => true}
      />
    ),
  },
  {
    name: 'DnsSettings',
    header: /dns/i,
    render: () => (
      <DnsSettings
        testsSettings={DEFAULT_TESTS_SETTINGS}
        setTestsSettings={noop}
        testsStatus="idle"
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={noop}
      />
    ),
  },
  {
    name: 'HealthChecksSettings',
    header: /health/i,
    render: () => (
      <HealthChecksSettings
        testsSettings={DEFAULT_TESTS_SETTINGS}
        setTestsSettings={noop}
        testsStatus="idle"
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={noop}
      />
    ),
  },
  {
    name: 'AppearanceSettings',
    header: /appearance/i,
    render: () => (
      <AppearanceSettings
        theme="dark"
        setTheme={noop}
        isDark={true}
        unitSystem="sae"
        setUnitSystem={noop}
      />
    ),
  },
  { name: 'SsoSettings', header: /sign-on|sso/i, render: () => <SsoSettings /> },
  {
    name: 'ConfigBackupsSection',
    header: /configuration backups/i,
    render: () => <ConfigBackupsSection />,
  },
  { name: 'InterfacesSettings', header: /interface/i, render: () => <InterfacesSettings /> },
  {
    name: 'ThresholdsSettings',
    header: /thresholds/i,
    render: () => (
      <ThresholdsSettings
        thresholds={DEFAULT_THRESHOLDS}
        setThresholds={noop}
        thresholdsStatus="idle"
      />
    ),
  },
];

export const MIXED_SECTIONS: MixedSectionFixture[] = [
  {
    name: 'CableTestSettings',
    header: /cable test/i,
    readControl: /^refresh$/i,
    render: () => (
      <CableTestSettings
        cableTestSettings={DEFAULT_CABLE_TEST_SETTINGS}
        setCableTestSettings={noop}
        cableTestStatus="idle"
      />
    ),
  },
  {
    name: 'DiscoverySettings',
    header: /^discovery$/i,
    readControl: /^refresh$/i,
    render: () => (
      <DiscoverySettings
        networkDiscoverySettings={DEFAULT_NETWORK_DISCOVERY_SETTINGS}
        setNetworkDiscoverySettings={noop}
        networkDiscoveryStatus="idle"
        subnets={[]}
        subnetsStatus="idle"
        newSubnetCidr=""
        setNewSubnetCidr={noop}
        newSubnetName=""
        setNewSubnetName={noop}
        subnetError={null}
        setSubnetError={noop}
        addSubnet={noop}
        toggleSubnet={noop}
        deleteSubnet={noop}
        snmpSettings={DEFAULT_SNMP_SETTINGS}
        setSnmpSettings={noop}
        snmpStatus="idle"
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={noop}
      />
    ),
  },
  {
    name: 'PerformanceSettings',
    header: /^performance$/i,
    // A viewer may still scan the LAN for public iperf servers: the button is
    // a read. It sits between the server-address input and the suggestion
    // chips, both writes, which is why this section gates per group.
    readControl: /find iperf hosts/i,
    render: () => (
      <PerformanceSettings
        testsSettings={DEFAULT_TESTS_SETTINGS}
        setTestsSettings={noop}
        iperfSettings={DEFAULT_IPERF_SETTINGS}
        setIperfSettings={noop}
        iperfStatus="idle"
        iperfSuggestions={[{ host: '10.0.0.5', hostname: 'lab-iperf' }]}
        iperfSuggestionsStatus="idle"
        iperfSuggestionsError={null}
        fetchIperfSuggestions={noop}
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={noop}
      />
    ),
  },
  {
    name: 'VulnerabilitySettings',
    header: /vulnerability/i,
    // Two reads: the scanner-status Refresh, and the API-key help toggle that
    // sits directly above the key input. The second is why the NVD group gates
    // per control rather than behind a fieldset.
    readControl: /^refresh$/i,
    alsoUsable: [/requires an api key/i],
    render: () => (
      <VulnerabilitySettings
        settings={DEFAULT_VULNERABILITY_SETTINGS}
        setSettings={noop}
        status="idle"
      />
    ),
  },
];
/**
 * Slice 2 (#2467): the sections that also carry a *read* action.
 *
 * A disabled `<fieldset>` over the whole body would take the read with it —
 * a viewer could no longer refresh the scanner status or the discovery
 * service. These sections gate their write controls only, and the case below
 * asserts both halves at once: the read control stays usable, every write
 * control does not. Asserting only "everything is disabled" is what would let
 * the read regress silently.
 */

/**
 * Three sections read their status banner with the global `fetch` rather than
 * the api client (a GET, so the raw-fetch gate allows it). Without the banner
 * the Refresh button never renders and the read half of the case cannot be
 * asserted at all, so the stub answers each status route with a running one.
 */
export const RAW_GET_BODIES: Record<string, unknown> = {
  '/api/v1/security/discovery/service/status': { running: true, scanning: false, deviceCount: 1 },
  '/api/v1/config/backups': {
    backups: [{ name: 'b1', createdAt: '2026-09-07T00:00:00Z', size: 1 }],
  },
  '/api/v1/config/version': { current: '1', needsMigration: false },
  // Without a status body the vulnerability banner never renders, and with it
  // the Refresh button that is the read half of that case.
  '/api/v1/security/vulnerabilities/status': { running: false, lastScan: null },
};

/**
 * Sections the role-gate suite covers with a dedicated file of its own, and the
 * i18n suite still has to read. WiFiSettings gates per control (its Forget and
 * Disconnect buttons are writes beside a live scan) and UsersSettings is
 * admin-only, so neither belongs in the two tables above — but a section left
 * out of the copy assertion is exactly the gap S1-14c exists to close.
 */
export const COPY_ONLY_SECTIONS: SectionFixture[] = [
  {
    name: 'WiFiSettings',
    header: /wi-?fi/i,
    render: () => (
      <WiFiSettings
        wifiSettings={{ interface: 'wlan0', availableWifi: ['wlan0'], isWireless: true }}
        setWifiSettings={noop}
        wifiStatus="idle"
      />
    ),
  },
  { name: 'UsersSettings', header: /users|usuarios/i, render: () => <UsersSettings /> },
];
