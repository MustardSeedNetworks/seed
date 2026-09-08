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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import { MIXED_SECTIONS, RAW_GET_BODIES, SECTIONS } from './settings-sections.fixtures';

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
