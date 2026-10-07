/**
 * LearnedNetworksNotice tests (seed#3108).
 *
 * The notice is the only place a learned network is put to the operator, so
 * these pin what it promises: each pending network shows its CIDR and where it
 * was learned, Add and Dismiss post that decision for that CIDR, an answered
 * network leaves the list, and a viewer is never asked (the endpoint is
 * operator+, so asking would only 403).
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';

import { must } from '../../test/must';
import type { SubnetResponse } from '../../types/generated/subnet-response';
import { LearnedNetworksNotice } from './LearnedNetworksNotice';

const role = vi.hoisted(() => ({ canWrite: true }));
vi.mock('../../contexts/RoleContext', () => ({
  useRole: () => ({ canWrite: role.canWrite }),
}));

const server = vi.hoisted(() => ({
  pending: [] as SubnetResponse[],
  get: vi.fn<(path: string) => Promise<unknown>>(),
  post: vi.fn<(path: string, body: unknown) => Promise<unknown>>(),
}));
vi.mock('../../api', () => ({
  api: {
    get: (path: string) => server.get(path),
    post: (path: string, body: unknown) => server.post(path, body),
  },
}));

const PENDING_PATH = '/api/v1/security/devices/subnets/pending';

const routed: SubnetResponse = {
  cidr: '10.51.2.0/24',
  name: 'Learned from 10.51.0.1 (routing table)',
  enabled: false,
  learned: true,
};
const hostRoute: SubnetResponse = {
  cidr: '10.51.3.0/24',
  name: "Learned from this host's route via 10.51.0.1",
  enabled: false,
  learned: true,
};

function renderNotice(): void {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui: ReactElement = (
    <QueryClientProvider client={client}>
      <LearnedNetworksNotice />
    </QueryClientProvider>
  );
  render(ui);
}

beforeEach(() => {
  role.canWrite = true;
  server.pending = [routed, hostRoute];
  server.get.mockReset().mockImplementation(async () => server.pending);
  server.post.mockReset().mockImplementation(async (_path, body) => {
    const { cidr } = body as { cidr: string };
    server.pending = server.pending.filter((network) => network.cidr !== cidr);
    return {};
  });
});

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('LearnedNetworksNotice', () => {
  it('names each pending network and where it was learned', async () => {
    renderNotice();

    const rows = await screen.findAllByTestId('learned-network');
    expect(rows.map((row) => row.dataset.cidr)).toEqual(['10.51.2.0/24', '10.51.3.0/24']);
    expect(
      within(must(rows[0], 'first row')).getByText('Learned from 10.51.0.1 (routing table)'),
    ).toBeVisible();
    expect(screen.getByText('Seed learned 2 new networks')).toBeVisible();
    expect(server.get).toHaveBeenCalledWith(PENDING_PATH);
  });

  it.each([
    ['learned-network-add', 'added'],
    ['learned-network-dismiss', 'dismissed'],
  ])('%s records %s for that network and drops it from the list', async (testId, decision) => {
    renderNotice();
    const first = must((await screen.findAllByTestId('learned-network'))[0], 'first row');

    await userEvent.click(within(first).getByTestId(testId));

    expect(server.post).toHaveBeenCalledWith(PENDING_PATH, { cidr: '10.51.2.0/24', decision });
    await vi.waitFor(() => expect(screen.getAllByTestId('learned-network')).toHaveLength(1));
    expect(screen.getByTestId('learned-network').dataset.cidr).toBe('10.51.3.0/24');
    expect(screen.getByText('Seed learned a new network')).toBeVisible();
  });

  it('keeps the network and says so when the decision is refused', async () => {
    server.post.mockRejectedValue(new Error('forbidden'));
    renderNotice();
    const first = must((await screen.findAllByTestId('learned-network'))[0], 'first row');

    await userEvent.click(within(first).getByTestId('learned-network-add'));

    expect(await screen.findByTestId('learned-networks-error')).toBeVisible();
    expect(screen.getAllByTestId('learned-network')).toHaveLength(2);
  });

  it('renders nothing when no network is pending', async () => {
    server.pending = [];
    renderNotice();

    await vi.waitFor(() => expect(server.get).toHaveBeenCalled());
    expect(screen.queryByTestId('learned-networks')).toBeNull();
  });

  it('never asks a viewer', () => {
    role.canWrite = false;
    renderNotice();

    expect(server.get).not.toHaveBeenCalled();
    expect(screen.queryByTestId('learned-networks')).toBeNull();
  });

  it('speaks Spanish under es, with the CIDR in each action name', async () => {
    await i18n.changeLanguage('es');
    renderNotice();

    expect(await screen.findByText('Seed detectó 2 redes nuevas')).toBeVisible();
    expect(
      screen.getByRole('button', { name: 'Agregar 10.51.2.0/24 a la detección' }),
    ).toBeVisible();
    expect(screen.getByRole('button', { name: 'Descartar 10.51.3.0/24' })).toBeVisible();
  });
});
