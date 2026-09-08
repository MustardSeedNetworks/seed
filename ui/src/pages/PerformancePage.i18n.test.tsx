/**
 * PerformancePage.i18n.test.tsx — the performance page renders real locale copy.
 *
 * S1-14b. The page is two cards and owns no copy, so the suite renders the
 * real cards through it. Writing it found three things a Spanish operator saw
 * in English: a failed health check reading `fail`, the ping detail line
 * `0% loss, 1.2ms jitter` built by string concatenation, and all five HTTP
 * timing tooltips, which came from an English-only constant table
 * (`HelpContent.tsx`) rather than the locale files — so a Spanish label
 * carried an English explanation.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { AppContext, type AppContextValue } from '../contexts/AppContext';
import i18n from '../i18n';

const cardSettings = {
  healthCheck: { enabled: true },
  healthChecks: { autoRunOnLink: false },
  performance: {
    enabled: true,
    speedtest: { enabled: true, autoRunOnLink: false },
    iperf: { enabled: true, autoRunOnLink: false },
  },
};

vi.mock('../contexts/useSettings', () => ({
  useSettings: () => ({
    cardSettings,
    iperfSettings: {
      serverHost: '',
      serverPort: 5201,
      duration: 10,
      parallel: 1,
      protocol: 'tcp',
      reverse: false,
    },
  }),
}));

vi.mock('../hooks/useIperfServerSync', () => ({ useIperfServerSync: () => undefined }));

vi.mock('../api', () => ({
  api: { get: vi.fn(() => Promise.resolve({})), post: vi.fn(() => Promise.resolve({})) },
}));

/** One ping that answered, one that did not, and an HTTP test with phase timings. */
const health = {
  hasTests: true,
  pingResults: [
    {
      name: '8.8.8.8',
      success: true,
      latency: 12,
      testStatus: 'success',
      packetLoss: 0,
      jitter: 1.2,
    },
    { name: '10.0.0.9', success: false, latency: 0, testStatus: 'error' },
  ],
  tcpResults: [{ name: 'db:5432', success: true, latency: 3, testStatus: 'success' }],
  udpResults: [],
  httpResults: [
    {
      name: 'https://example.com',
      success: true,
      latency: 120,
      status: 200,
      testStatus: 'success',
      dnsLatency: 10,
      tcpConnect: 20,
      tlsLatency: 30,
      ttfbLatency: 40,
    },
  ],
};

const appContext = {
  loading: false,
  isWifi: false,
  cards: { wifi: false },
  cardSettings,
} as unknown as AppContextValue;

const { PerformancePage } = await import('./PerformancePage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  render(
    <AppContext.Provider value={appContext}>
      <PerformancePage />
    </AppContext.Provider>,
  );
  await waitFor(() => expect(screen.getByText('8.8.8.8')).toBeVisible());
}

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(health) }),
    ),
  );
});

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('PerformancePage — real locale copy', () => {
  it('labels both cards and their sections in English', async () => {
    await renderIn('en');

    expect(screen.getByText('Health Checks')).toBeVisible();
    expect(screen.getByText('Ping')).toBeVisible();
    expect(screen.getByText('TCP Ports')).toBeVisible();
    expect(screen.getByText('Performance Tests')).toBeVisible();
    expect(screen.getByText('Internet Speed')).toBeVisible();
    expect(screen.getByText('No results yet')).toBeVisible();
  });

  it('says in English what a failed check and a lossy ping did', async () => {
    await renderIn('en');

    expect(screen.getByText('fail')).toBeVisible();
    expect(screen.getByText('0% loss, 1.2ms jitter')).toBeVisible();
  });

  it('explains the HTTP timing phases in English', async () => {
    await renderIn('en');

    expect(
      screen.getByText(
        'Time to resolve the hostname to an IP address via DNS lookup. Shows 0 when connection is reused from pool.',
      ),
    ).toBeVisible();
    expect(
      screen.getByText('Time to download the full response body after receiving the first byte.'),
    ).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    for (const english of [
      'Health Checks',
      'TCP Ports',
      'Performance Tests',
      'Internet Speed',
      'No results yet',
      'fail',
      '0% loss, 1.2ms jitter',
      'Time to resolve the hostname to an IP address via DNS lookup. Shows 0 when connection is reused from pool.',
      'Time to download the full response body after receiving the first byte.',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Verificaciones de salud')).toBeVisible();
    expect(screen.getByText('Puertos TCP')).toBeVisible();
    expect(screen.getByText('Pruebas de rendimiento')).toBeVisible();
    expect(screen.getByText('fallo')).toBeVisible();
    expect(screen.getByText('0% pérdida, 1.2ms jitter')).toBeVisible();
    expect(
      screen.getByText(
        'Tiempo para descargar el cuerpo completo de la respuesta tras recibir el primer byte.',
      ),
    ).toBeVisible();
  });

  it('leaves the protocol names and the tested hosts alone under es', async () => {
    await renderIn('es');

    // Ping, HTTP and iperf3 are the names of the things being run.
    expect(screen.getByText('Ping')).toBeVisible();
    expect(screen.getByText('HTTP')).toBeVisible();
    expect(screen.getByText('8.8.8.8')).toBeVisible();
    expect(screen.getByText(/db:5432/)).toBeVisible();
  });
});
