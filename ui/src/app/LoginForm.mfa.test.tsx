/**
 * LoginForm second-factor tests (seed#2391).
 *
 * A user who enrolled TOTP from the Security page could not get back in: the
 * login answered 200 with `mfaRequired` and no access token, and the form had
 * nowhere to go — `mfaRequired` and `mfaToken` appeared only in the generated
 * type, read by nothing. useAuth then marked the session authenticated with an
 * empty token, so the app looked logged in and every request failed.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { LoginOutcome } from '../hooks/useAuth';
import { LoginForm } from './LoginForm';

// The form probes recovery status and SSO providers on mount; neither is what
// these tests are about.
beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ active: false, providers: [] }),
      } as Response),
    ),
  );
});

function renderForm(
  onLogin: (u: string, p: string) => Promise<LoginOutcome>,
  onSecondFactor: (t: string, c: string) => Promise<boolean>,
): void {
  render(
    <LoginForm onLogin={onLogin} onSecondFactor={onSecondFactor} isLoading={false} error={null} />,
  );
}

async function submitPassword(): Promise<void> {
  await userEvent.type(screen.getByLabelText(/user/i), 'admin');
  await userEvent.type(screen.getByLabelText(/password/i), 'hunter2hunter2');
  await userEvent.click(screen.getByRole('button', { name: /log ?in|sign ?in/i }));
}

describe('LoginForm second factor', () => {
  it('asks for a code when the login reports one is required', async () => {
    const onLogin = vi.fn(
      async (): Promise<LoginOutcome> => ({
        status: 'mfa-required',
        mfaToken: 'mfa-token-1',
        username: 'admin',
      }),
    );
    const onSecondFactor = vi.fn(async () => true);

    renderForm(onLogin, onSecondFactor);
    await submitPassword();

    expect(await screen.findByTestId('mfa-code-input')).toBeInTheDocument();
  });

  it('exchanges the code with the mfaToken it was given', async () => {
    const onLogin = vi.fn(
      async (): Promise<LoginOutcome> => ({
        status: 'mfa-required',
        mfaToken: 'mfa-token-1',
        username: 'admin',
      }),
    );
    const onSecondFactor = vi.fn(async () => true);

    renderForm(onLogin, onSecondFactor);
    await submitPassword();

    await userEvent.type(await screen.findByTestId('mfa-code-input'), '123456');
    await userEvent.click(screen.getByTestId('mfa-submit'));

    await waitFor(() => {
      expect(onSecondFactor).toHaveBeenCalledWith('mfa-token-1', '123456');
    });
  });

  it('stays on the password step when no second factor is required', async () => {
    const onLogin = vi.fn(async (): Promise<LoginOutcome> => ({ status: 'ok' }));
    const onSecondFactor = vi.fn(async () => true);

    renderForm(onLogin, onSecondFactor);
    await submitPassword();

    await waitFor(() => expect(onLogin).toHaveBeenCalled());
    expect(screen.queryByTestId('mfa-code-input')).not.toBeInTheDocument();
  });
});
