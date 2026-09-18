/**
 * ReportsPage.i18n.test.tsx — the reports page renders real locale copy,
 * including the `<Trans>` actions in its <GatedPreview> pitch.
 *
 * S1-14b. Same reason the path analysis suite asserts its gate: a `<Trans>`
 * whose key is missing in a locale falls back to the component's children and
 * keeps rendering English, so only a both-locales assertion sees it. The
 * `seed license` commands inside the sentence stay verbatim — they are typed
 * at a shell, not read as prose.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../i18n';

const license = { features: [] as string[] };

vi.mock('../contexts/LicenseContext', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useLicense: () => ({
    loading: false,
    hasFeature: (feature: string) => license.features.includes(feature),
  }),
}));

vi.mock('../contexts/RoleContext', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useRole: () => ({ canWrite: true, role: 'admin', loading: false }),
}));

vi.mock('../hooks/useReports', () => ({
  useReports: () => ({
    reports: [],
    loading: false,
    error: null,
    generating: false,
    generate: vi.fn(),
    remove: vi.fn(),
  }),
}));

const { ReportsPage } = await import('./ReportsPage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <ReportsPage />
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

describe('ReportsPage — real locale copy', () => {
  it('pitches the feature and both ways out of the gate in English', async () => {
    await renderIn('en');

    expect(screen.getByText('Reports is a Starter feature')).toBeVisible();
    expect(screen.getByText(/Generate executive summaries and device inventories/)).toBeVisible();
    expect(screen.getByText('seed license trial')).toBeVisible();
    expect(screen.getByText('seed license activate -k <KEY>')).toBeVisible();
  });

  it('renders the licensed card and its empty state in English', async () => {
    license.features = ['export_csv_json'];
    await renderIn('en');

    expect(screen.getByText('Reports')).toBeVisible();
    expect(screen.getByText('No reports yet. Generate one to get started.')).toBeVisible();
    expect(screen.getByText('Generate')).toBeVisible();
  });

  it('renders the pitch in Spanish, with the commands still verbatim', async () => {
    await renderIn('es');

    expect(screen.queryByText('Reports is a Starter feature')).toBeNull();
    expect(screen.getByText('Informes es una función del nivel Starter')).toBeVisible();
    expect(
      screen.getByText(/Genere resúmenes ejecutivos e inventarios de dispositivos/),
    ).toBeVisible();
    expect(screen.getByText('seed license trial')).toBeVisible();
    expect(screen.getByText('seed license activate -k <KEY>')).toBeVisible();
  });

  it('renders the licensed card in Spanish, with no English left behind', async () => {
    license.features = ['export_csv_json'];
    await renderIn('es');

    for (const english of [
      'Reports',
      'No reports yet. Generate one to get started.',
      'Generate',
      'Active Anomalies',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Informes')).toBeVisible();
    expect(screen.getByText('Aún no hay informes. Genera uno para empezar.')).toBeVisible();
    expect(screen.getByText('Generar')).toBeVisible();
    expect(screen.getByText('Anomalías activas')).toBeVisible();
  });
});
