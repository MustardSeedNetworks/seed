/**
 * SettingsDrawer role-gating tests (#1254, #2467, owner decision 2026-09-04).
 *
 * The drawer used to hide every panel from a viewer behind one notice, on the
 * premise that its loader routes gate GET. They do not: `minRole` is applied
 * through `writeGated`, which passes GET for every role, and
 * `TestViewerCanReadEveryRoleGatedRoute` asserts that server-side. So a viewer
 * reads the whole drawer and every write control inside it is disabled — the
 * gating each section now carries itself.
 *
 * This renders the real SettingsDrawer, not a stand-in for its gate: the defect
 * this covers is a section reaching a viewer with a live control, which only
 * the real composition can show.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ProfileProvider } from '../../contexts/profileContext';
import { type CurrentUser, RoleProvider } from '../../contexts/RoleContext';
import { SettingsDrawer } from './SettingsDrawer';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockWrite = vi.fn<(method: string, path: string) => void>();
vi.mock('../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    put: (): Promise<unknown> => Promise.resolve({}),
    patch: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));
vi.mock('../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (path: string): Promise<unknown> => {
      mockWrite('POST', path);

      return Promise.resolve({});
    },
    put: (path: string): Promise<unknown> => {
      mockWrite('PUT', path);

      return Promise.resolve({});
    },
    patch: (path: string): Promise<unknown> => {
      mockWrite('PATCH', path);

      return Promise.resolve({});
    },
    delete: (path: string): Promise<unknown> => {
      mockWrite('DELETE', path);

      return Promise.resolve({});
    },
  },
}));
vi.mock('../../contexts/LicenseContext', () => ({
  useLicense: (): { status: { features: string[] } } => ({
    status: { features: ['sso', 'multi_interface', 'multi_user', 'api_tokens'] },
  }),
}));

function asUser(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }

    return Promise.resolve({});
  });
}

function renderDrawer(): void {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const node: ReactElement = (
    <QueryClientProvider client={queryClient}>
      <ProfileProvider>
        <RoleProvider isAuthenticated={true}>
          <SettingsDrawer isOpen={true} onClose={(): void => undefined} version="1.0.0" />
        </RoleProvider>
      </ProfileProvider>
    </QueryClientProvider>
  );
  render(node);
}

beforeEach(() => {
  mockGet.mockReset();
  mockWrite.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

/**
 * Every section the drawer mounts for a viewer. The two this slice unwrapped —
 * SSO and the guest-network audit — are in the list with the rest, because a
 * gate coming back in another shape would drop any of them silently.
 */
const VIEWER_SECTIONS = [
  'Link',
  'Cable Test',
  'Network',
  'DNS',
  'Health Checks',
  'Performance',
  'Discovery',
  'Vulnerability Scanning',
  'Thresholds',
  'Appearance',
  'API Tokens',
  'Guest Network Audit',
  'Single Sign-On',
  'Network Interfaces',
  'Configuration Backups',
];

function sectionHeaders(): string[] {
  return screen.queryAllByRole('button').map((el) => {
    const heading = el.querySelector('h3, h4, span');

    return (heading?.textContent ?? '').trim();
  });
}

describe('SettingsDrawer — viewer gating', () => {
  it('gives a viewer every panel, not a notice in place of them', async () => {
    asUser('viewer');
    renderDrawer();

    await waitFor(() => {
      expect(screen.getByText('Link')).toBeInTheDocument();
    });

    expect(screen.queryByTestId('settings-viewer-notice')).not.toBeInTheDocument();
    const headers = sectionHeaders();
    for (const name of VIEWER_SECTIONS) {
      expect(headers).toContain(name);
    }
  });

  it('marks the sections a viewer cannot change read-only', async () => {
    asUser('viewer');
    renderDrawer();

    await waitFor(() => {
      expect(screen.getByText('Link')).toBeInTheDocument();
    });

    // The all-write sections carry the badge on the header itself; the ones
    // with a read of their own gate per control and are covered by
    // sections/settings-sections.rolegate.test.tsx.
    expect(screen.getAllByText(/read-only/i).length).toBeGreaterThan(0);
  });

  it('sends no write of its own while a viewer has the drawer open', async () => {
    asUser('viewer');
    renderDrawer();

    await waitFor(() => {
      expect(screen.getByText('Link')).toBeInTheDocument();
    });
    // Long enough for every debounced auto-save the drawer arms (800ms).
    await new Promise((resolve) => setTimeout(resolve, 1200));

    expect(mockWrite.mock.calls).toEqual([]);
  });

  it('keeps user management out of a viewer drawer', async () => {
    asUser('viewer');
    renderDrawer();

    await waitFor(() => {
      expect(screen.getByText('Link')).toBeInTheDocument();
    });

    expect(sectionHeaders()).not.toContain('User Management');
  });

  it('gives an operator the same sections with no read-only marking', async () => {
    asUser('operator');
    renderDrawer();

    await waitFor(() => {
      expect(screen.getByText('Link')).toBeInTheDocument();
    });

    const headers = sectionHeaders();
    for (const name of VIEWER_SECTIONS) {
      expect(headers).toContain(name);
    }
    expect(screen.queryByText(/read-only/i)).not.toBeInTheDocument();
  });
});
