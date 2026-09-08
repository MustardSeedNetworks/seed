/**
 * ReportsPage.i18n.test.tsx — the reports page renders real locale copy,
 * including its `<Trans>` licence gate.
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
  it('states the tier gate and both ways out of it in English', async () => {
    await renderIn('en');

    expect(screen.getByText(/Reports require the Starter tier or higher/)).toBeVisible();
    expect(screen.getByText('seed license trial')).toBeVisible();
    expect(screen.getByText(/seed license activate -k/)).toBeVisible();
  });

  it('renders the licensed card and its empty state in English', async () => {
    license.features = ['export_csv_json'];
    await renderIn('en');

    expect(screen.getByText('Reports')).toBeVisible();
    expect(screen.getByText('No reports yet. Generate one to get started.')).toBeVisible();
    expect(screen.getByText('Generate')).toBeVisible();
  });

  it('renders the gate sentence in Spanish, with the commands still verbatim', async () => {
    await renderIn('es');

    expect(screen.queryByText(/Reports require the Starter tier or higher/)).toBeNull();
    expect(screen.getByText(/Los informes requieren el nivel Starter o superior/)).toBeVisible();
    expect(screen.getByText('seed license trial')).toBeVisible();
    expect(screen.getByText(/seed license activate -k/)).toBeVisible();
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
