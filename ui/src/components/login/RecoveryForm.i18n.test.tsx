/**
 * RecoveryForm.i18n.test.tsx — password recovery renders real locale copy.
 *
 * #1942. Recovery sits on the login path, reachable before anyone has
 * authenticated, and is one of the few screens an operator meets while already
 * having a problem — the worst place to render an unexpected language.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { RecoveryForm } from './RecoveryForm';

function renderForm() {
  return render(<RecoveryForm onRecoveryComplete={() => {}} onBackToLogin={() => {}} />);
}

beforeEach(() => {
  vi.spyOn(globalThis, 'fetch').mockImplementation((async () => ({
    ok: true,
    status: 200,
    // The instructions panel maps over `steps`; an empty object renders and
    // then throws on undefined.map.
    json: async () => ({
      triggerFile: '/var/lib/seed/recover',
      tokenFile: '/var/lib/seed/recovery-token',
      expiryTime: '2026-01-01T00:00:00Z',
      steps: ['Create the trigger file', 'Read the token'],
    }),
  })) as unknown as typeof fetch);
});

afterEach(async () => {
  vi.restoreAllMocks();
  await i18n.changeLanguage('en');
});

describe('RecoveryForm — real locale copy', () => {
  it('renders the English recovery prompt', async () => {
    await i18n.changeLanguage('en');
    renderForm();

    await waitFor(() => {
      expect(screen.getByText('Reset your password')).toBeInTheDocument();
    });
    expect(screen.getByText('Back to login')).toBeInTheDocument();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    renderForm();

    await waitFor(() => {
      expect(screen.getByText('Restablezca su contraseña')).toBeInTheDocument();
    });
    expect(screen.getByText('Volver a iniciar sesión')).toBeInTheDocument();
    expect(screen.queryByText('Reset your password')).toBeNull();
    expect(screen.queryByText('Back to login')).toBeNull();
  });
});
