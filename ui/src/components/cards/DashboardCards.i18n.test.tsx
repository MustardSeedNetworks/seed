/**
 * DashboardCards.i18n.test.tsx — the three cards on the Network page render
 * real locale copy.
 *
 * #1942: wrecking both locale trees fails only 26 of 313 tests, because the
 * suite asserts on testids and on English hardcoded in components. These are
 * the first three cards an operator looks at and none of them had a copy
 * assertion.
 *
 * The pattern is stem's (MustardSeedNetworks/stem#778): assert in both
 * locales, and assert that no English is left behind under `es` — a key that
 * silently falls back renders English, which a single-locale test cannot see.
 * Writing these found nineteen Spanish strings still carrying the English
 * word "latency", several of them mid-sentence.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it } from 'vitest';

import { ProfileProvider } from '../../contexts/profileContext';
import i18n from '../../i18n';
import { DnsCard, type DnsData } from './DnsCard';
import { GatewayCard, type GatewayData } from './GatewayCard';
import { type DhcpData, NetworkCard } from './NetworkCard';

/** GatewayCard reads its thresholds from the profile context. */
function renderCard(node: ReactNode): ReturnType<typeof render> {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ProfileProvider>{node}</ProfileProvider>
    </QueryClientProvider>,
  );
}

const dhcp: DhcpData = {
  mac: '02:00:5e:10:00:00',
  mode: 'dhcp',
  ipv4: {
    address: '192.0.2.10',
    subnet: '24',
    gateway: '192.0.2.1',
    dhcpServer: '192.0.2.1',
    leaseTime: 3600,
  },
  ipv6: [],
  dns: ['192.0.2.53'],
  timing: null,
};

const gateway: GatewayData = {
  gateway: '192.0.2.1',
  reachable: true,
  sent: 4,
  received: 4,
  lossPercent: 0,
  minTime: 1,
  maxTime: 3,
  avgTime: 2,
  lastTime: 2,
  status: 'success',
};

const dns: DnsData = {
  server: '192.0.2.53',
  servers: ['192.0.2.53'],
  testHostname: 'example.com',
  forward: { status: 'success', time: 12, timeMs: 12, result: '192.0.2.10' },
  reverse: null,
};

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('Network page cards — real locale copy', () => {
  it('labels the Network card in English', async () => {
    await i18n.changeLanguage('en');
    renderCard(<NetworkCard data={dhcp} />);

    expect(screen.getByText('Network')).toBeInTheDocument();
    expect(screen.getByText('MAC')).toBeInTheDocument();
    expect(screen.getByText('Mode')).toBeInTheDocument();
    expect(screen.getByText('Gateway')).toBeInTheDocument();
    expect(screen.getByText('DNS')).toBeInTheDocument();
  });

  it('labels the Network card in Spanish, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    renderCard(<NetworkCard data={dhcp} />);

    expect(screen.getByText('Red')).toBeInTheDocument();
    expect(screen.getByText('Modo')).toBeInTheDocument();
    expect(screen.getByText('Puerta de enlace')).toBeInTheDocument();
    expect(screen.queryByText('Network')).not.toBeInTheDocument();
    expect(screen.queryByText('Mode')).not.toBeInTheDocument();

    // MAC and DNS are protocol names and stay verbatim in both locales.
    expect(screen.getByText('MAC')).toBeInTheDocument();
    expect(screen.getByText('DNS')).toBeInTheDocument();
  });

  it('labels the Gateway card in English', async () => {
    await i18n.changeLanguage('en');
    renderCard(<GatewayCard data={gateway} />);

    expect(screen.getByText('Gateway')).toBeInTheDocument();
    expect(screen.getByText('Min')).toBeInTheDocument();
    expect(screen.getByText('Avg')).toBeInTheDocument();
    expect(screen.getByText('Max')).toBeInTheDocument();
    expect(screen.getByText('Packets')).toBeInTheDocument();
  });

  it('labels the Gateway card in Spanish, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    renderCard(<GatewayCard data={gateway} />);

    expect(screen.getByText('Puerta de enlace')).toBeInTheDocument();
    expect(screen.getByText('Mín')).toBeInTheDocument();
    expect(screen.getByText('Prom')).toBeInTheDocument();
    expect(screen.getByText('Máx')).toBeInTheDocument();
    expect(screen.getByText('Paquetes')).toBeInTheDocument();
    expect(screen.queryByText('Packets')).not.toBeInTheDocument();
  });

  it('reports packet loss in Spanish when there is any', async () => {
    await i18n.changeLanguage('es');
    renderCard(<GatewayCard data={{ ...gateway, received: 2, lossPercent: 50 }} />);

    expect(screen.getByText('Pérdida de paquetes')).toBeInTheDocument();
    expect(screen.queryByText('Packet Loss')).not.toBeInTheDocument();
  });

  it('labels the DNS card in both locales', async () => {
    await i18n.changeLanguage('en');
    const { unmount } = renderCard(<DnsCard data={dns} />);
    expect(screen.getByText('DNS Servers')).toBeInTheDocument();
    expect(screen.getByText('Testing: example.com')).toBeInTheDocument();
    expect(screen.getByText('Forward (A)')).toBeInTheDocument();
    unmount();

    await i18n.changeLanguage('es');
    renderCard(<DnsCard data={dns} />);
    expect(screen.getByText('Servidores DNS')).toBeInTheDocument();
    // Interpolated, not glued together in JSX -- word order is the locale's.
    expect(screen.getByText('Probando: example.com')).toBeInTheDocument();
    expect(screen.getByText('Directa (A)')).toBeInTheDocument();
    expect(screen.queryByText('DNS Servers')).not.toBeInTheDocument();
  });
});
