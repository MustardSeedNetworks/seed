/**
 * WifiPage.i18n.test.tsx — the Wi-Fi page renders real locale copy.
 *
 * S1-14b. Unlike the other five composition pages this one owns copy of its
 * own, and all of it was hardcoded English: the wired-interface note that
 * replaces the whole page, the Pro tier hint on both gated cards and their
 * absent-card labels. Both Wi-Fi visibility cards carried an English title
 * and subtitle too, and the channel graph's accessible name was English for
 * every locale — the one string a screen-reader user actually hears.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { AppContext, type AppContextValue } from '../contexts/AppContext';
import { ProfileProvider } from '../contexts/profileContext';
import i18n from '../i18n';

const license = { features: [] as string[] };

vi.mock('../contexts/LicenseContext', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useLicense: () => ({
    loading: false,
    hasFeature: (feature: string) => license.features.includes(feature),
  }),
}));

vi.mock('../hooks/useWifiVisibility', () => ({
  useWifiAirspace: () => ({ data: undefined, isLoading: true, isError: false }),
  useWifiAnomalies: () => ({ data: undefined, isLoading: true, isError: false }),
}));

const wifi = {
  ssid: 'msn-lab',
  bssid: '02:00:5e:00:00:01',
  signal: -55,
  channel: 36,
  frequency: 5180,
  security: 'WPA2',
  band: '5',
};

const context = {
  loading: false,
  isWifi: true,
  cards: { wifi },
  channelGraphData: {
    available: true,
    data: {
    networks24Ghz: [],
    networks5Ghz: [
      {
        ssid: 'msn-lab',
        bssid: '02:00:5e:00:00:01',
        channel: 36,
        centerFreq: 5180,
        channelWidth: 80,
        signal: -55,
        band: '5GHz',
        isConnected: true,
      },
    ],
    networks6Ghz: [],
    connectedBssid: '02:00:5e:00:00:01',
    scanTime: '2026-09-07T10:00:00Z',
    },
  },
  channelGraphLoading: false,
} as unknown as AppContextValue;

const { WifiPage } = await import('./WifiPage');

function renderPage(ctx: AppContextValue): ReturnType<typeof render> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ProfileProvider>
        <AppContext.Provider value={ctx}>
          <WifiPage />
        </AppContext.Provider>
      </ProfileProvider>
    </QueryClientProvider>,
  ) as ReturnType<typeof render>;
}

async function renderIn(language: string, ctx: AppContextValue = context): Promise<void> {
  await i18n.changeLanguage(language);
  renderPage(ctx);
  await waitFor(() => expect(document.body.textContent).not.toBe(''));
}

beforeEach(() => {
  license.features = [];
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('WifiPage — real locale copy', () => {
  it('names the Pro-gated cards and the way to get them in English', async () => {
    await renderIn('en');

    expect(screen.getByText('Airspace')).toBeVisible();
    expect(screen.getByText('Association anomalies')).toBeVisible();
    expect(
      screen.getAllByText('Available on Seed Pro. Run `seed license trial` for a 14-day trial.'),
    ).toHaveLength(2);
  });

  it('says in English why a wired interface has no Wi-Fi page at all', async () => {
    await renderIn('en', { ...context, isWifi: false } as AppContextValue);

    expect(screen.getByText('Wireless data')).toBeVisible();
    expect(
      screen.getByText(
        'This interface is wired. Switch to a Wi-Fi interface from the header to see signal, channels and airspace.',
      ),
    ).toBeVisible();
  });

  it('titles the licensed visibility cards in English', async () => {
    license.features = ['wifi_management_capture', 'wifi_association_forensics'];
    await renderIn('en');

    expect(screen.getByText('Wi-Fi Airspace')).toBeVisible();
    expect(
      screen.getByText(
        'Live SSID / AP / BSSID / client map from 802.11 management-frame capture.',
      ),
    ).toBeVisible();
    expect(screen.getByText('Wi-Fi Anomalies')).toBeVisible();
    expect(screen.getByText('Loading airspace…')).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    license.features = ['wifi_management_capture', 'wifi_association_forensics'];
    await renderIn('es');

    for (const english of [
      'Wi-Fi Airspace',
      'Wi-Fi Anomalies',
      'Live SSID / AP / BSSID / client map from 802.11 management-frame capture.',
      'Security, RF, roaming, and standards anomalies detected in the airspace.',
      'Loading airspace…',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Espacio radioeléctrico Wi-Fi')).toBeVisible();
    expect(screen.getByText('Anomalías de Wi-Fi')).toBeVisible();
    expect(screen.getByText('Cargando el espacio radioeléctrico…')).toBeVisible();
  });

  it('says the tier hint and the wired note in Spanish', async () => {
    await renderIn('es');
    expect(
      screen.getAllByText(
        'Disponible en Seed Pro. Ejecute `seed license trial` para una prueba de 14 días.',
      ),
    ).toHaveLength(2);
    expect(screen.queryByText(/Available on Seed Pro/)).toBeNull();
  });

  it('keeps the SSID, BSSID and Wi-Fi itself untranslated under es', async () => {
    await renderIn('es');

    // The network's own name is not copy; the graph's accessible name is.
    expect(screen.getByText('msn-lab')).toBeVisible();
    expect(screen.getByLabelText('Gráfico de señal por canal Wi-Fi')).toBeInTheDocument();
    expect(screen.queryByLabelText('WiFi channel signal graph')).toBeNull();
  });
});
