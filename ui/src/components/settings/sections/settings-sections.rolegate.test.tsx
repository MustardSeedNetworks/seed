/**
 * Read-only settings for a viewer (#1254, owner decision 2026-09-04).
 *
 * Every section here is backed by routes the server registers `minRole: op`.
 * `writeGated` passes GET for every role, so a viewer's data does arrive and
 * the section is readable — what must not happen is a viewer reaching a
 * control whose request can only 403.
 *
 * One case per section, both halves: a viewer opens it and reads it with every
 * control disabled; an operator gets the same section usable.
 */

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import {
  DEFAULT_CABLE_TEST_SETTINGS,
  DEFAULT_CARD_SETTINGS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_LINK_SETTINGS,
  DEFAULT_NETWORK_DISCOVERY_SETTINGS,
  DEFAULT_SNMP_SETTINGS,
  DEFAULT_TESTS_SETTINGS,
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
import { SsoSettings } from './SsoSettings';

const READ_ONLY = 'Read-only — operator role required to change these settings.';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    put: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));
// SsoSettings and InterfacesSettings render their controls only when the
// licence carries the feature, so the stub grants both: the question here is
// role, not tier.
vi.mock('../../../contexts/LicenseContext', () => ({
  useLicense: (): { status: { features: string[] } } => ({
    status: { features: ['sso', 'multi_interface'] },
  }),
}));
// InterfacesSettings reads its interface lists from the profile store; the
// stub gives it one of each so its per-interface controls render.
vi.mock('../../../contexts/profileContext', () => ({
  useProfileContext: (): Record<string, () => unknown> => ({
    getAllEthernetInterfaces: () => [{ name: 'eth0' }, { name: 'eth1' }],
    getAllWifiInterfaces: () => [{ name: 'wlan0' }],
    getEthernetInterface: () => ({ name: 'eth0' }),
    getWifiInterface: () => ({ name: 'wlan0' }),
    addEthernetInterface: () => undefined,
    addWifiInterface: () => undefined,
    removeEthernetInterface: () => undefined,
    removeWifiInterface: () => undefined,
    setActiveEthernetInterface: () => undefined,
    setActiveWifiInterface: () => undefined,
  }),
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    put: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));

const noop = (): void => undefined;
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
const SECTIONS: { name: string; header: RegExp; render: () => ReactElement }[] = [
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
];

function asUser(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }
    if (path.includes('/sso/settings')) {
      return Promise.resolve({ providers: [] });
    }

    return Promise.resolve({});
  });
}

async function openSection(header: RegExp): Promise<HTMLElement> {
  const button = await screen.findByRole('button', { name: header });
  await userEvent.click(button);

  return button.closest('section') as HTMLElement;
}

/**
 * Three sections read their status banner with the global `fetch` rather than
 * the api client (a GET, so the raw-fetch gate allows it). Without the banner
 * the Refresh button never renders and the read half of the case cannot be
 * asserted at all, so the stub answers each status route with a running one.
 */
const RAW_GET_BODIES: Record<string, unknown> = {
  '/api/v1/security/discovery/service/status': { running: true, scanning: false, deviceCount: 1 },
  '/api/v1/config/backups': {
    backups: [{ name: 'b1', createdAt: '2026-09-07T00:00:00Z', size: 1 }],
  },
  '/api/v1/config/version': { current: '1', needsMigration: false },
};

beforeEach(() => {
  mockGet.mockReset();
  vi.stubGlobal('fetch', (input: RequestInfo | URL) => {
    const url = String(input);
    const key = Object.keys(RAW_GET_BODIES).find((path) => url.includes(path));

    return Promise.resolve({
      ok: key !== undefined,
      status: key === undefined ? 404 : 200,
      json: () => Promise.resolve(key === undefined ? {} : RAW_GET_BODIES[key]),
    } as Response);
  });
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe.each(SECTIONS)('$name — viewer read-only', ({ header, render: renderSection }) => {
  it('opens for a viewer with every control disabled and says why', async () => {
    asUser('viewer');
    render(<RoleProvider isAuthenticated={true}>{renderSection()}</RoleProvider>);

    const section = await openSection(header);
    await waitFor(() => {
      expect(within(section).getByText(READ_ONLY)).toBeInTheDocument();
    });

    const controls = [
      ...within(section).queryAllByRole('textbox'),
      ...within(section).queryAllByRole('checkbox'),
      ...within(section).queryAllByRole('combobox'),
      ...within(section).queryAllByRole('spinbutton'),
      // The header toggle is the one button outside the fieldset.
      ...within(section)
        .queryAllByRole('button')
        .filter((el) => el !== section.querySelector('button')),
    ];
    expect(controls.length).toBeGreaterThan(0);
    for (const control of controls) {
      expect(control).toBeDisabled();
    }
  });

  it('leaves the same section usable for an operator', async () => {
    asUser('operator');
    render(<RoleProvider isAuthenticated={true}>{renderSection()}</RoleProvider>);

    const section = await openSection(header);
    await waitFor(() => {
      expect(within(section).queryByText(READ_ONLY)).not.toBeInTheDocument();
    });
  });
});

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
const MIXED_SECTIONS: {
  name: string;
  header: RegExp;
  /** The control that reads and must stay usable for a viewer. */
  readControl: RegExp;
  /** Further read-only controls that must survive the gate, by accessible name. */
  alsoUsable?: RegExp[];
  render: () => ReactElement;
}[] = [
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
    alsoUsable: [/lab-v3/],
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
        snmpSettings={{ ...DEFAULT_SNMP_SETTINGS, v3Credentials: [V3_CREDENTIAL] }}
        setSnmpSettings={noop}
        snmpStatus="idle"
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={noop}
      />
    ),
  },
];

/**
 * One SNMPv3 credential, so the credential row renders. Its accordion header is
 * a `<div role="button">` — the row that named it a "non-form clickable the
 * fieldset would never catch" read it as a missed write control; it is a read
 * (expand/collapse), and it cannot become a real `<button>` because it contains
 * one. Leaving it enabled is correct and is asserted below; the Remove button
 * inside it is the write, and the fieldset does disable that.
 */
const V3_CREDENTIAL = {
  id: 'cred-1',
  name: 'lab-v3',
  username: 'operator',
  authProtocol: 'SHA',
  authPassword: '',
  privProtocol: 'AES',
  privPassword: '',
  contextName: '',
  securityLevel: 'authPriv',
};

/** Write controls of a section: everything interactive but the header and the read. */
function writeControls(section: HTMLElement, reads: HTMLElement[]): HTMLElement[] {
  const header = section.querySelector('button');

  return [
    ...within(section).queryAllByRole('textbox'),
    ...within(section).queryAllByRole('checkbox'),
    ...within(section).queryAllByRole('combobox'),
    ...within(section).queryAllByRole('spinbutton'),
    ...within(section).queryAllByRole('button'),
  ].filter((el) => el !== header && !reads.includes(el));
}

describe.each(MIXED_SECTIONS)(
  '$name — viewer read-only with a live read',
  ({ header, readControl, alsoUsable = [], render: renderSection }) => {
    async function reads(section: HTMLElement): Promise<HTMLElement[]> {
      const found = [await within(section).findByRole('button', { name: readControl })];
      for (const name of alsoUsable) {
        found.push(within(section).getByRole('button', { name }));
      }
      for (const control of found) {
        expect(control).toBeEnabled();
      }

      return found;
    }

    it('keeps the reads usable for a viewer and disables every write control', async () => {
      asUser('viewer');
      render(<RoleProvider isAuthenticated={true}>{renderSection()}</RoleProvider>);

      const section = await openSection(header);
      const controls = writeControls(section, await reads(section));
      expect(controls.length).toBeGreaterThan(0);
      for (const control of controls) {
        expect(control).toBeDisabled();
      }
    });

    it('leaves every control usable for an operator', async () => {
      asUser('operator');
      render(<RoleProvider isAuthenticated={true}>{renderSection()}</RoleProvider>);

      const section = await openSection(header);
      for (const control of writeControls(section, await reads(section))) {
        expect(control).toBeEnabled();
      }
    });
  },
);
