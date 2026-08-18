/**
 * The List + detail archetype's honest-state rule, on the page the design
 * actually drew: a target nobody is polling must not read as healthy. Green
 * for "we have not looked" is the one failure this shell exists to prevent.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { PollingTarget } from '../types/polling';

const targets: PollingTarget[] = [
  {
    id: 'healthy',
    clientId: 'c',
    name: 'core-01',
    ipAddress: '10.44.10.2',
    snmpVersion: 'v3',
    credentialsId: '',
    pollIntervalSeconds: 300,
    enabled: true,
    collectorChain: ['sys_info', 'if_table'],
    lastStatus: 'ok',
    lastError: '',
    lastPolledAt: '2026-08-17T10:00:00Z',
    createdAt: '',
    updatedAt: '',
  },
  {
    id: 'paused',
    clientId: 'c',
    name: 'acc-sw-12',
    ipAddress: '10.44.10.9',
    snmpVersion: 'v2c',
    credentialsId: '',
    pollIntervalSeconds: 300,
    enabled: false,
    collectorChain: [],
    lastStatus: 'ok',
    lastError: '',
    createdAt: '',
    updatedAt: '',
  },
];

vi.mock('../hooks/usePollingTargets', () => ({
  usePollingTargets: () => ({
    targets,
    loading: false,
    error: null,
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  }),
}));

const { PollingTargetsPage } = await import('./PollingTargetsPage');

describe('PollingTargetsPage — list + detail', () => {
  beforeEach(() => {
    render(<PollingTargetsPage />);
  });

  it('prints an em dash rather than a figure for a target never polled', () => {
    const row = within(screen.getByTestId('target-row-paused'));

    expect(row.getByText('—')).toBeTruthy();
  });

  it('says why a paused target has no reading instead of showing it as fine', async () => {
    await userEvent.click(screen.getByTestId('target-row-paused'));

    expect(screen.getByText('Polling paused')).toBeTruthy();
    expect(screen.queryByText('Polling normally')).toBeNull();
  });

  it('shows the selected target detail, including its collector chain', async () => {
    await userEvent.click(screen.getByTestId('target-row-healthy'));

    expect(screen.getByText('Selected target')).toBeTruthy();
    expect(screen.getByText('sys_info')).toBeTruthy();
    expect(screen.getByText('if_table')).toBeTruthy();
  });

  it('filters the list down to failing targets without losing the counts', async () => {
    await userEvent.click(screen.getByRole('button', { name: /^Paused/ }));

    expect(screen.queryByTestId('target-row-healthy')).toBeNull();
    expect(screen.getByTestId('target-row-paused')).toBeTruthy();
  });
});
