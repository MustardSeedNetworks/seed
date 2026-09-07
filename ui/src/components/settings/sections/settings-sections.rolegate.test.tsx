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
  DEFAULT_CARD_SETTINGS,
  DEFAULT_DISPLAY_OPTIONS,
  DEFAULT_LINK_SETTINGS,
  DEFAULT_TESTS_SETTINGS,
  type IpSettings,
} from '../../../types/settings';
import { SettingsDrawerNetworkSection } from '../SettingsDrawerNetworkSection';
import { AppearanceSettings } from './AppearanceSettings';
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

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
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
