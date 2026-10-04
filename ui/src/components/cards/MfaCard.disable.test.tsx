/**
 * MfaCard TOTP disable tests (seed#2731).
 *
 * `POST /api/v1/auth/totp/disable` had no caller: an operator who enrolled TOTP
 * could only remove it with a direct API call. A wrong factor also answered
 * 401, which the client reads as an expired session — it refreshed, replayed
 * the request against the attempt limiter and signed the user out. The route
 * now answers 403 with a code naming the factor, and the card localizes it.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError } from '../../api';
import i18n from '../../i18n';
import { MfaCard } from './MfaCard';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();

vi.mock('../../api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api')>()),
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (path: string, body?: unknown): Promise<unknown> => mockPost(path, body),
  },
}));

vi.mock('../../lib/webauthn', () => ({
  isPasskeySupported: (): boolean => true,
  registerPasskey: (): Promise<void> => Promise.resolve(),
}));

const enrolled = { totpEnabled: true, webauthnEnabled: false, webauthnCredentialCount: 0 };
const notEnrolled = { totpEnabled: false, webauthnEnabled: false, webauthnCredentialCount: 0 };

function rejected(code: string): ApiError {
  return new ApiError(403, { error: 'server copy', code }, undefined, undefined);
}

async function submitDisable(password: string, code: string): Promise<void> {
  await userEvent.click(await screen.findByTestId('mfa-disable-totp'));
  await userEvent.type(screen.getByTestId('mfa-disable-password'), password);
  await userEvent.type(screen.getByTestId('mfa-disable-code'), code);
  await userEvent.click(screen.getByTestId('mfa-disable-confirm'));
}

beforeEach(() => {
  mockGet.mockResolvedValue(enrolled);
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('MfaCard TOTP disable', () => {
  it('offers no disable control when TOTP is not enrolled', async () => {
    mockGet.mockResolvedValue(notEnrolled);
    render(<MfaCard />);

    expect(await screen.findByTestId('mfa-setup-totp')).toBeInTheDocument();
    expect(screen.queryByTestId('mfa-disable-totp')).not.toBeInTheDocument();
  });

  it('sends the password and code, then shows TOTP off', async () => {
    mockPost.mockResolvedValue({ status: 'disabled', enabled: false });
    render(<MfaCard />);
    mockGet.mockResolvedValue(notEnrolled);

    await submitDisable('hunter2-long', '123456');

    expect(mockPost).toHaveBeenCalledWith('/api/v1/auth/totp/disable', {
      password: 'hunter2-long',
      code: '123456',
    });
    expect(await screen.findByText('No second factor enrolled')).toBeInTheDocument();
    expect(screen.queryByTestId('mfa-disable-form')).not.toBeInTheDocument();
    expect(screen.getByTestId('mfa-setup-totp')).toBeInTheDocument();
  });

  it('keeps the confirm button off until both factors are entered', async () => {
    render(<MfaCard />);

    await userEvent.click(await screen.findByTestId('mfa-disable-totp'));
    await userEvent.type(screen.getByTestId('mfa-disable-password'), 'hunter2-long');
    await userEvent.type(screen.getByTestId('mfa-disable-code'), '12345');

    expect(screen.getByTestId('mfa-disable-confirm')).toBeDisabled();
  });

  it.each([
    ['INVALID_PASSWORD', 'en', 'Wrong password. TOTP is still on.'],
    ['INVALID_MFA_CODE', 'en', 'Wrong or expired code. Try the next one.'],
    ['INVALID_PASSWORD', 'es', 'Contraseña incorrecta. TOTP sigue activado.'],
    ['INVALID_MFA_CODE', 'es', 'Código incorrecto o expirado. Pruebe con el siguiente.'],
  ])('names a rejected %s in %s and leaves TOTP on', async (code, language, message) => {
    await i18n.changeLanguage(language);
    mockPost.mockRejectedValue(rejected(code));
    render(<MfaCard />);

    await submitDisable('hunter2-long', '123456');

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.queryByText('server copy')).not.toBeInTheDocument();
    // The form stays open for another try, with the spent code cleared, and the
    // status was not re-read: nothing changed on the server.
    expect(screen.getByTestId('mfa-disable-form')).toBeInTheDocument();
    expect(screen.getByTestId('mfa-disable-code')).toHaveValue('');
    expect(mockGet).toHaveBeenCalledTimes(1);
  });

  it('cancel closes the form and clears what was typed', async () => {
    render(<MfaCard />);

    await userEvent.click(await screen.findByTestId('mfa-disable-totp'));
    await userEvent.type(screen.getByTestId('mfa-disable-password'), 'hunter2-long');
    await userEvent.click(screen.getByTestId('mfa-disable-cancel'));

    expect(screen.queryByTestId('mfa-disable-form')).not.toBeInTheDocument();
    await userEvent.click(screen.getByTestId('mfa-disable-totp'));
    await waitFor(() => {
      expect(screen.getByTestId('mfa-disable-password')).toHaveValue('');
    });
    expect(mockPost).not.toHaveBeenCalled();
  });
});
