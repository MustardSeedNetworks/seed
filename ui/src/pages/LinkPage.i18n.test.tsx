/**
 * LinkPage.i18n.test.tsx — the link page renders real locale copy.
 *
 * S1-14b. This page carried the most untranslated copy of the eight: seven
 * rollup headlines, six explanatory bodies, four figure labels and the reason
 * the wired cards are absent on a radio were all English literals in
 * `describeLink`, so the sentence a Spanish operator reads first — the one
 * the rollup exists to deliver — was in English.
 *
 * The existing `LinkPage.test.tsx` asserts the same sentences, but it mocks
 * every card and runs only under `en`, so it stayed green throughout.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { LinkData } from '../components/cards/LinkCard';
import { AppContext, type AppContextValue } from '../contexts/AppContext';
import { ProfileProvider } from '../contexts/profileContext';
import i18n from '../i18n';

vi.mock('../hooks/useDriverStats', () => ({
  useDriverStats: () => ({ counters: [], total: 0, loading: false, error: null, refresh: vi.fn() }),
}));

function link(over: Partial<LinkData> = {}): LinkData {
  return {
    linkUp: true,
    carrier: true,
    hasIp: true,
    speed: '1000Mb/s',
    duplex: 'full',
    advertisedSpeeds: [],
    mtu: 1500,
    flapCount24h: 0,
    ...over,
  } as LinkData;
}

function context(over: Record<string, unknown> = {}): AppContextValue {
  return {
    loading: false,
    isWifi: false,
    currentInterface: 'en0',
    displayOptions: { unitSystem: 'metric' },
    cards: { link: link(), cable: null, wifi: null },
    ...over,
  } as unknown as AppContextValue;
}

const { LinkPage } = await import('./LinkPage');

async function renderIn(language: string, ctx: AppContextValue = context()): Promise<void> {
  await i18n.changeLanguage(language);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <ProfileProvider>
        <AppContext.Provider value={ctx}>
          <LinkPage />
        </AppContext.Provider>
      </ProfileProvider>
    </QueryClientProvider>,
  );
  await waitFor(() => expect(document.body.textContent).not.toBe(''));
}

/** The rollup band, which labels Speed/Duplex/MTU the link card labels too. */
function rollup(): HTMLElement {
  const el = document.querySelector<HTMLElement>('section[aria-live="polite"][data-state]');
  if (el === null) {
    throw new Error('status rollup not rendered');
  }
  return el;
}

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('LinkPage — real locale copy', () => {
  it('leads with the healthy sentence and its figures in English', async () => {
    await renderIn('en');

    expect(screen.getByText('All clear')).toBeVisible();
    expect(screen.getByText('The link is up at 1000Mb/s')).toBeVisible();
    for (const label of ['Speed', 'Duplex', 'MTU', 'Flaps 24h']) {
      expect(within(rollup()).getByText(label)).toBeVisible();
    }
  });

  it('says in English what no carrier means and what to do about it', async () => {
    await renderIn(
      'en',
      context({
        cards: { link: link({ carrier: false, linkUp: false }), cable: null, wifi: null },
      }),
    );

    expect(screen.getByText('Critical')).toBeVisible();
    expect(screen.getByText('No carrier on this interface')).toBeVisible();
    expect(
      screen.getByText(
        'Nothing is detected on the wire. Check the cable and the far-end port; the cable test below reports where the fault is.',
      ),
    ).toBeVisible();
  });

  it('distinguishes half duplex from a healthy link in English', async () => {
    await renderIn(
      'en',
      context({ cards: { link: link({ duplex: 'half' }), cable: null, wifi: null } }),
    );

    expect(screen.getByText('Degraded')).toBeVisible();
    expect(screen.getByText('The link negotiated half duplex')).toBeVisible();
  });

  it('says in English why the wired cards are absent on a radio', async () => {
    await renderIn(
      'en',
      context({
        isWifi: true,
        cards: {
          link: null,
          cable: null,
          wifi: {
            ssid: 'msn-lab',
            bssid: '02:00:5e:00:00:01',
            channel: 36,
            signal: -55,
            frequency: 5180,
            security: 'WPA2',
          },
        },
      }),
    );

    expect(screen.getByText('This interface is wireless')).toBeVisible();
    expect(screen.getByText('Wired link')).toBeVisible();
    expect(
      screen.getByText(
        'This interface is wireless — carrier, duplex and cable diagnostics come from a wired port.',
      ),
    ).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn(
      'es',
      context({
        cards: { link: link({ carrier: false, linkUp: false }), cable: null, wifi: null },
      }),
    );

    for (const english of [
      'Critical',
      'No carrier on this interface',
      'Nothing is detected on the wire. Check the cable and the far-end port; the cable test below reports where the fault is.',
      'Speed',
      'Duplex',
      'Flaps 24h',
      'Link Status',
      'Cable Test',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Crítico')).toBeVisible();
    expect(screen.getByText('Sin portadora en esta interfaz')).toBeVisible();
    expect(within(rollup()).getByText('Velocidad')).toBeVisible();
    expect(within(rollup()).getByText('Dúplex')).toBeVisible();
    expect(screen.getByText('Caídas 24 h')).toBeVisible();
    expect(screen.getByText('Estado del enlace')).toBeVisible();
  });

  it('names the speed and MTU verbatim under es', async () => {
    await renderIn('es');

    // The negotiated speed and MTU are the interface's own numbers.
    expect(screen.getByText('El enlace está activo a 1000Mb/s')).toBeVisible();
    expect(within(rollup()).getByText('MTU')).toBeVisible();
    expect(within(rollup()).getByText('1500')).toBeVisible();
  });
});
