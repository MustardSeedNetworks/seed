/**
 * PathAnalysisPage.i18n.test.tsx — the path analysis page renders real locale
 * copy, including its `<Trans>` licence gate.
 *
 * S1-14b. A `<Trans>` needs its own assertion: when the key is missing in a
 * locale, react-i18next renders the component's own children rather than
 * throwing, so the English sentence survives and a single-locale test cannot
 * see it. Both halves are asserted here — the Spanish sentence is present, and
 * the `seed license` commands inside it stay verbatim because they are typed
 * at a shell.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
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

vi.mock('../hooks/useEngineScan', () => ({
  useEngineScan: () => ({
    running: false,
    status: { state: 'idle', jobId: '', percentComplete: 0, error: null },
    startScan: vi.fn(),
    cancelScan: vi.fn(),
  }),
}));
vi.mock('../hooks/useEnginePhase', () => ({ useEnginePhase: () => ({ phase: '' }) }));
vi.mock('../hooks/useNetworkDiscoveryAutoScan', () => ({
  useNetworkDiscoveryAutoScan: () => ({ handleDeepScan: vi.fn() }),
}));

const context = {
  loading: false,
  isWifi: false,
  cards: { gateway: { gateway: '192.0.2.1' }, dns: { servers: ['192.0.2.53'] }, wifi: null },
  cardSettings: { networkDiscovery: { enabled: true } },
  networkDiscovery: null,
  scanError: false,
  triggerDeviceScan: vi.fn(),
  registerTraceHopHandler: vi.fn(() => () => undefined),
} as unknown as AppContextValue;

const { PathAnalysisPage } = await import('./PathAnalysisPage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <ProfileProvider>
        <AppContext.Provider value={context}>
          <PathAnalysisPage />
        </AppContext.Provider>
      </ProfileProvider>
    </QueryClientProvider>,
  );
  await waitFor(() => expect(document.body.textContent).not.toBe(''));
}

beforeEach(() => {
  license.features = [];
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('PathAnalysisPage — real locale copy', () => {
  it('states the Pro gate and both ways out of it in English', async () => {
    await renderIn('en');

    expect(screen.getByText(/Path Analysis is a Pro-tier feature/)).toBeVisible();
    expect(screen.getByText('seed license trial')).toBeVisible();
    expect(screen.getByText(/seed license activate -k/)).toBeVisible();
  });

  it('renders the licensed cards in English', async () => {
    license.features = ['path_analysis'];
    await renderIn('en');

    expect(screen.getByText('Path Discovery')).toBeVisible();
    expect(screen.getByText('Enter an IP or hostname to trace the network path.')).toBeVisible();
    expect(screen.getByText('Network Discovery')).toBeVisible();
    expect(screen.getByText('Start Scan')).toBeVisible();
  });

  it('renders the gate sentence in Spanish, with the commands still verbatim', async () => {
    await renderIn('es');

    expect(screen.queryByText(/Path Analysis is a Pro-tier feature/)).toBeNull();
    expect(screen.getByText(/El análisis de rutas es una función del nivel Pro/)).toBeVisible();
    // The two <code> children of the <Trans> are shell commands, not copy.
    expect(screen.getByText('seed license trial')).toBeVisible();
    expect(screen.getByText(/seed license activate -k/)).toBeVisible();
  });

  it('renders the licensed cards in Spanish, with no English left behind', async () => {
    license.features = ['path_analysis'];
    await renderIn('es');

    for (const english of [
      'Path Discovery',
      'Enter an IP or hostname to trace the network path.',
      'Network Discovery',
      'Enter target',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Descubrimiento de ruta')).toBeVisible();
    expect(
      screen.getByText('Ingrese una IP o nombre de host para trazar la ruta de red.'),
    ).toBeVisible();
    expect(screen.getByText('Descubrimiento de red')).toBeVisible();
    // ICMP, UDP and TCP are protocol names in both locales.
    for (const protocol of ['ICMP', 'UDP', 'TCP']) {
      expect(screen.getByText(protocol)).toBeVisible();
    }
  });
});
