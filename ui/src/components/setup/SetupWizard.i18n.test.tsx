/**
 * SetupWizard.i18n.test.tsx — first-run setup renders real locale copy.
 *
 * #1942: wrecking both locale trees fails only 21 of 267 tests, because the
 * suite asserts on testids and on English hardcoded in components. Setup is
 * the first screen an operator ever sees and had no copy assertions at all.
 *
 * The pattern is stem's (MustardSeedNetworks/stem#778): assert in both
 * locales, and assert that no English is left behind under `es` — a key that
 * silently falls back renders English, which a single-locale test cannot see.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { SetupWizard } from './SetupWizard';

function renderWizard() {
  return render(
    <SetupWizard
      onComplete={() => {}}
      onLogin={async () => true}
      suggestedPassword="Correct-Horse-Battery-9"
    />,
  );
}

beforeEach(() => {
  // The wizard probes setup status on mount; without a stub the promise is
  // undefined and it throws before rendering anything.
  vi.spyOn(globalThis, 'fetch').mockImplementation((async () => ({
    ok: true,
    status: 200,
    json: async () => ({ needsSetup: true, username: 'admin' }),
  })) as unknown as typeof fetch);
});

afterEach(async () => {
  vi.restoreAllMocks();
  await i18n.changeLanguage('en');
});

describe('SetupWizard — real locale copy', () => {
  it('renders the English welcome step', async () => {
    await i18n.changeLanguage('en');
    renderWizard();

    await waitFor(() => {
      expect(screen.getByText('Welcome to The Seed')).toBeInTheDocument();
    });
    expect(screen.getByText('Set up your admin password to get started')).toBeInTheDocument();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    renderWizard();

    await waitFor(() => {
      expect(screen.getByText('Bienvenido a The Seed')).toBeInTheDocument();
    });
    expect(
      screen.getByText('Configure su contraseña de administrador para comenzar'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Set up your admin password to get started')).toBeNull();
  });

  it('keeps the product name verbatim in both locales, per the glossary', async () => {
    await i18n.changeLanguage('es');
    renderWizard();

    // "The Seed" is a glossary term: the copy around it translates, the name
    // itself must survive.
    await waitFor(() => {
      expect(screen.getAllByText(/The Seed/).length).toBeGreaterThan(0);
    });
  });
});
