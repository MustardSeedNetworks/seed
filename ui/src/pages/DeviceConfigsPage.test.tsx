/**
 * Device configs (UI-SEED-25): the newest saved backup opens with its change
 * highlighted line by line, a failed attempt explains itself without asking
 * for a diff, the first configuration says there is nothing to compare, and
 * a viewer is never sent to the operator-only reads.
 */
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  DeviceConfigBackup,
  DeviceConfigDiff,
  DeviceConfigTarget,
} from '../types/deviceConfigs';
import { parseUnifiedDiff } from './deviceConfigs/DiffView';

const license = { features: ['compliance_advanced'] };
let canWrite = true;
const requested: string[] = [];

vi.mock('../contexts/LicenseContext', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useLicense: () => ({
    loading: false,
    hasFeature: (feature: string) => license.features.includes(feature),
  }),
}));

vi.mock('../contexts/RoleContext', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useRole: () => ({ canWrite, role: canWrite ? 'operator' : 'viewer', loading: false }),
}));

const targets: DeviceConfigTarget[] = [
  {
    id: 'dct-00000000000a',
    name: 'core-01',
    host: '10.44.10.1',
    port: 22,
    platform: 'cisco-ios',
    updatedAt: '2026-10-15T10:00:00Z',
  },
];

const backups: DeviceConfigBackup[] = [
  {
    id: 'dcb-000000000003',
    targetId: 'dct-00000000000a',
    takenAt: '2026-10-15T12:00:00Z',
    status: 'unreachable',
    error: 'dial tcp 10.44.10.1:22: i/o timeout',
  },
  {
    id: 'dcb-000000000002',
    targetId: 'dct-00000000000a',
    takenAt: '2026-10-15T11:00:00Z',
    status: 'ok',
    sha256: 'b',
  },
  {
    id: 'dcb-000000000001',
    targetId: 'dct-00000000000a',
    takenAt: '2026-10-15T10:00:00Z',
    status: 'ok',
    sha256: 'a',
  },
];

const diff: DeviceConfigDiff = {
  targetId: 'dct-00000000000a',
  backupId: 'dcb-000000000002',
  takenAt: '2026-10-15T11:00:00Z',
  previousBackupId: 'dcb-000000000001',
  previousTakenAt: '2026-10-15T10:00:00Z',
  added: 1,
  removed: 1,
  unified: [
    '--- dcb-000000000001\t2026-10-15T10:00:00Z',
    '+++ dcb-000000000002\t2026-10-15T11:00:00Z',
    '@@ -1,3 +1,3 @@',
    ' hostname core-01',
    '-ntp server 10.0.0.1',
    '+ntp server 10.0.0.2',
    ' !',
    '',
  ].join('\n'),
};

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    api: {
      get: async (path: string): Promise<unknown> => {
        requested.push(path);
        if (path === '/api/v1/device-config-targets') {
          return { count: targets.length, targets };
        }
        if (path.startsWith('/api/v1/device-configs?')) {
          return { count: backups.length, backups };
        }
        if (path === '/api/v1/device-configs/dcb-000000000002/diff') {
          return diff;
        }
        // The target's first configuration has nothing before it.
        throw new actual.ApiError(404, { error: 'Not found' }, undefined, null);
      },
    },
  };
});

const { DeviceConfigsPage } = await import('./DeviceConfigsPage');

describe('parseUnifiedDiff', () => {
  it('drops the file headers and classifies every other line by its prefix', () => {
    expect(parseUnifiedDiff(diff.unified).map((line) => [line.kind, line.text])).toEqual([
      ['hunk', '@@ -1,3 +1,3 @@'],
      ['context', ' hostname core-01'],
      ['removed', '-ntp server 10.0.0.1'],
      ['added', '+ntp server 10.0.0.2'],
      ['context', ' !'],
    ]);
  });
});

describe('DeviceConfigsPage', () => {
  beforeEach(() => {
    license.features = ['compliance_advanced'];
    canWrite = true;
    requested.length = 0;
  });

  it('opens the newest saved backup with added and removed lines highlighted', async () => {
    render(<DeviceConfigsPage />);

    const view = await screen.findByTestId('config-diff');
    const removed = within(view).getByText('-ntp server 10.0.0.1');
    const added = within(view).getByText('+ntp server 10.0.0.2');
    expect(removed).toHaveAttribute('data-kind', 'removed');
    expect(removed.className).toContain('text-status-error-strong');
    expect(added).toHaveAttribute('data-kind', 'added');
    expect(added.className).toContain('text-status-success-strong');
    expect(within(view).queryByText(/^\+\+\+ /)).toBeNull();
    expect(screen.getByTestId('config-diff-counts')).toHaveTextContent('+1 −1 lines');
    // The newest attempt failed; the newest saved one is what opens.
    expect(screen.getByTestId('config-backup-dcb-000000000002')).toHaveAttribute(
      'aria-pressed',
      'true',
    );
  });

  it('explains a failed attempt without asking for its diff', async () => {
    const user = userEvent.setup();
    render(<DeviceConfigsPage />);
    await screen.findByTestId('config-diff');

    await user.click(screen.getByTestId('config-backup-dcb-000000000003'));

    expect(screen.getByTestId('config-backup-failed')).toHaveTextContent(
      'Unreachable: dial tcp 10.44.10.1:22: i/o timeout. This attempt saved no configuration.',
    );
    expect(screen.queryByTestId('config-diff')).toBeNull();
    expect(requested).not.toContain('/api/v1/device-configs/dcb-000000000003/diff');
  });

  it('says the first configuration has nothing to compare with', async () => {
    const user = userEvent.setup();
    render(<DeviceConfigsPage />);
    await screen.findByTestId('config-diff');

    await user.click(screen.getByTestId('config-backup-dcb-000000000001'));

    expect(await screen.findByTestId('config-diff-first')).toHaveTextContent(
      "This is the device's first saved configuration",
    );
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('tells a viewer the reads are operator-only and requests nothing', async () => {
    canWrite = false;
    render(<DeviceConfigsPage />);

    expect(await screen.findByText(/Operator role required/)).toBeInTheDocument();
    await waitFor(() => expect(requested).toEqual([]));
  });

  it('shows the sample, not the live reads, without the licence', () => {
    license.features = [];
    render(<DeviceConfigsPage />);

    expect(screen.getByTestId('gated-preview')).toHaveAttribute(
      'data-feature',
      'compliance_advanced',
    );
    expect(screen.getByTestId('config-diff')).toBeInTheDocument();
    expect(requested).toEqual([]);
  });
});
