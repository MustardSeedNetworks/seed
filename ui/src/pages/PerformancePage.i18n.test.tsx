/**
 * PerformancePage.i18n.test.tsx — the performance page renders real locale copy.
 *
 * S1-14b. The page is three cards and owns no copy, so the suite renders the
 * real cards through it. Writing it found three things a Spanish operator saw
 * in English: a failed health check reading `fail`, the ping detail line
 * `0% loss, 1.2ms jitter` built by string concatenation, and all five HTTP
 * timing tooltips, which came from an English-only constant table
 * (`HelpContent.tsx`) rather than the locale files — so a Spanish label
 * carried an English explanation.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { AppContext, type AppContextValue } from '../contexts/AppContext';
import i18n from '../i18n';
import { must } from '../test/must';

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

// The DSCP check gates its form on the licence and the role and runs as a
// job; DscpCheckCard.test.tsx covers those. Here it only has to render copy.
vi.mock('../contexts/LicenseContext', () => ({
  useLicense: () => ({ loading: false, hasFeature: () => true }),
}));
vi.mock('../contexts/RoleContext', () => ({ useRole: () => ({ canWrite: true }) }));
vi.mock('../hooks/useDscpCheck', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../hooks/useDscpCheck')>()),
  useDscpCheck: () => ({ state: { phase: 'idle' }, start: vi.fn(), stop: vi.fn() }),
}));

// The Health Check card reads its run through the api client; the rest of the
// page's reads only need to resolve.
vi.mock('../api', () => ({
  api: {
    get: vi.fn((url: string) => Promise.resolve(url.endsWith('/probes/run') ? health : {})),
    post: vi.fn(() => Promise.resolve({})),
  },
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
  currentInterface: 'eth0',
} as unknown as AppContextValue;

const { PerformancePage } = await import('./PerformancePage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <AppContext.Provider value={appContext}>
        <PerformancePage />
      </AppContext.Provider>
    </QueryClientProvider>,
  );
  fireEvent.click(screen.getByTestId('health-check-run'));
  await waitFor(() => expect(screen.getByText('8.8.8.8', notTheTooltip)).toBeVisible());
}

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn(() => Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(health) })),
  );
});

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

// Tooltip mirrors its trigger's text into a portal bubble on document.body, so
// a bare text query matches the value twice. Only the value on the page counts.
const notTheTooltip = { ignore: '[role="tooltip"], script, style' } as const;

describe('PerformancePage — real locale copy', () => {
  it('labels both cards and their sections in English', async () => {
    await renderIn('en');

    expect(screen.getByText('Health Checks')).toBeVisible();
    expect(screen.getByText('Ping')).toBeVisible();
    expect(screen.getByText('TCP Ports')).toBeVisible();
    expect(screen.getByText('Performance Tests')).toBeVisible();
    expect(screen.getByText('Internet Speed')).toBeVisible();
    expect(screen.getByText('No results yet')).toBeVisible();
    expect(screen.getByText('DSCP check')).toBeVisible();
    expect(screen.getByText('Start listening')).toBeVisible();
  });

  it('says in English what a failed check and a lossy ping did', async () => {
    await renderIn('en');

    expect(screen.getByText('fail')).toBeVisible();
    expect(screen.getByText('0% loss, 1.2ms jitter')).toBeVisible();
  });

  it('explains the HTTP timing phases in English', async () => {
    await renderIn('en');

    fireEvent.focus(must(screen.getAllByTestId('http-timing-segment')[0], 'DNS timing'));
    expect(
      screen.getByText(
        'Time to resolve the hostname to an IP address via DNS lookup. Shows 0 when connection is reused from pool.',
      ),
    ).toBeVisible();
    fireEvent.blur(must(screen.getAllByTestId('http-timing-segment')[0], 'DNS timing'));
    fireEvent.focus(must(screen.getAllByTestId('http-timing-segment')[4], 'download timing'));
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
      'DSCP check',
      'Start listening',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Verificaciones de salud')).toBeVisible();
    expect(screen.getByText('Puertos TCP')).toBeVisible();
    expect(screen.getByText('Pruebas de rendimiento')).toBeVisible();
    expect(screen.getByText('fallo')).toBeVisible();
    expect(screen.getByText('0% pérdida, 1.2ms jitter')).toBeVisible();
    expect(screen.getByText('Comprobación DSCP')).toBeVisible();
    expect(screen.getByText('Empezar a escuchar')).toBeVisible();
    fireEvent.focus(must(screen.getAllByTestId('http-timing-segment')[4], 'download timing'));
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
    expect(screen.getByText('8.8.8.8', notTheTooltip)).toBeVisible();
    expect(screen.getByText(/db:5432/, notTheTooltip)).toBeVisible();
  });
});
