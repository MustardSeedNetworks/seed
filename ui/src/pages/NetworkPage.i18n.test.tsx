/**
 * NetworkPage.i18n.test.tsx — the network page renders real locale copy.
 *
 * S1-14b. The six cards were already translated (three of them by the
 * DashboardCards suite), but everything the page itself says was English in
 * both locales: all four rollup headlines, both explanatory bodies, the two
 * figures and their Up/None values, and the reason the switch card is absent
 * on a wireless interface.
 *
 * It also found the shared `StatusRollup` state word — `All clear`,
 * `Degraded`, `Critical`, `No data` — hardcoded. That one is read on the link
 * page and every List + detail page too, so slice 1's three pages were still
 * showing an English word above a Spanish headline.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AppContext, type AppContextValue } from '../contexts/AppContext';
import { ProfileProvider } from '../contexts/profileContext';
import i18n from '../i18n';

vi.mock('../hooks/useNeighbourCache', () => ({
  useNeighbourCache: () => ({ entries: [], loading: false, error: null, refresh: vi.fn() }),
}));

const cards = {
  dhcp: {
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
  },
  gateway: {
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
  },
  dns: {
    server: '192.0.2.53',
    servers: ['192.0.2.53'],
    testHostname: 'example.com',
    forward: { status: 'success', time: 12, timeMs: 12, result: '192.0.2.10' },
    reverse: null,
  },
  publicip: { ip: '203.0.113.9', city: 'Austin', country: 'US', isp: 'Example ISP' },
  switch: { name: 'core-sw1', model: 'C9300', ports: 48, vlans: 12 },
  vlan: null,
};

function context(overrides: Record<string, unknown> = {}): AppContextValue {
  return {
    loading: false,
    isWifi: false,
    displayOptions: { showPublicIp: true },
    cards,
    ...overrides,
  } as unknown as AppContextValue;
}

const { NetworkPage } = await import('./NetworkPage');

async function renderIn(language: string, ctx: AppContextValue = context()): Promise<void> {
  await i18n.changeLanguage(language);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <ProfileProvider>
        <AppContext.Provider value={ctx}>
          <NetworkPage />
        </AppContext.Provider>
      </ProfileProvider>
    </QueryClientProvider>,
  );
  await waitFor(() => expect(document.body.textContent).not.toBe(''));
}

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('NetworkPage — real locale copy', () => {
  it('answers the page question in English when the link is healthy', async () => {
    await renderIn('en');

    expect(screen.getByText('All clear')).toBeVisible();
    expect(screen.getByText('The upstream link is answering')).toBeVisible();
    expect(screen.getByText('Gateway', { selector: 'dt, dt *' })).toBeVisible();
    expect(screen.getAllByText('Up')).toHaveLength(2);
  });

  it('says in English what a missing gateway means, not just that it is missing', async () => {
    await renderIn('en', context({ cards: { ...cards, gateway: null, dns: null } }));

    expect(screen.getByText('Critical')).toBeVisible();
    expect(screen.getByText('No default gateway was found on this interface')).toBeVisible();
    expect(
      screen.getByText(
        'Without a gateway nothing beyond this segment is reachable. Check the interface selection and the DHCP lease below.',
      ),
    ).toBeVisible();
    expect(screen.getAllByText('None')).toHaveLength(2);
  });

  it('says in English why the switch card is absent on a radio', async () => {
    await renderIn('en', context({ isWifi: true, cards: { ...cards, wifi: { ssid: 'msn-lab' } } }));

    expect(screen.getByText('Switch and VLAN')).toBeVisible();
    expect(
      screen.getByText(
        'Neighbour discovery reads LLDP and CDP from the wire. A wireless interface has no switch port to ask.',
      ),
    ).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    for (const english of [
      'All clear',
      'The upstream link is answering',
      'Up',
      'Neighbour Cache',
      'Public IP',
      'Nearest Switch',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Todo correcto')).toBeVisible();
    expect(screen.getByText('El enlace ascendente responde')).toBeVisible();
    expect(screen.getAllByText('Activo')).toHaveLength(2);
    expect(screen.getByText('Puerta de enlace', { selector: 'dt, dt *' })).toBeVisible();
    expect(screen.getByText('IP pública')).toBeVisible();
  });

  it('degrades in Spanish when the resolver is the thing that is missing', async () => {
    await renderIn('es', context({ cards: { ...cards, dns: null } }));

    expect(screen.getByText('Degradado')).toBeVisible();
    expect(
      screen.getByText('Hay una puerta de enlace, pero ningún resolutor DNS respondió'),
    ).toBeVisible();
    expect(screen.queryByText('Degraded')).toBeNull();
  });

  it('leaves addresses, DNS and the switch model alone under es', async () => {
    await renderIn('es');

    // Addresses and protocol names are not copy.
    expect(screen.getAllByText('192.0.2.10').length).toBeGreaterThan(0);
    expect(screen.getAllByText('DNS').length).toBeGreaterThan(0);
    expect(screen.getAllByText('IPv4').length).toBeGreaterThan(0);
    expect(screen.getByText('02:00:5e:10:00:00')).toBeVisible();
  });
});
