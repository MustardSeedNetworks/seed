/**
 * MfaCard passkey enrolment test.
 *
 * "Add a passkey" called /auth/webauthn/register/begin and then looked for a
 * `window.seedWebAuthnRegister` helper that was defined nowhere. The guard was
 * always false, so the ceremony never ran, nothing was enrolled, and the card
 * refreshed and reported no error. Verified against a running daemon:
 *
 *   POST /api/v1/auth/webauthn/register/begin -> 200
 *   GET  /api/v1/auth/mfa/status              -> webauthnCredentialCount: 0
 *
 * No test covered the button, which is how it shipped. This one asserts the
 * ceremony actually runs.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { MfaCard } from './MfaCard';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();
const mockRegisterPasskey = vi.fn<() => Promise<void>>();

vi.mock('../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (path: string, body?: unknown): Promise<unknown> => mockPost(path, body),
  },
}));

vi.mock('../../lib/webauthn', () => ({
  isPasskeySupported: (): boolean => true,
  registerPasskey: (): Promise<void> => mockRegisterPasskey(),
}));

beforeEach(() => {
  mockGet.mockResolvedValue({
    totpEnabled: false,
    webauthnEnabled: false,
    webauthnCredentialCount: 0,
  });
  mockRegisterPasskey.mockResolvedValue(undefined);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe('MfaCard passkey enrolment', () => {
  it('runs the browser ceremony when Add a passkey is pressed', async () => {
    render(<MfaCard />);

    const button = await screen.findByRole('button', { name: /passkey/i });
    await userEvent.click(button);

    await waitFor(() => {
      expect(mockRegisterPasskey).toHaveBeenCalled();
    });
  });

  it('surfaces a failed ceremony instead of reporting success', async () => {
    mockRegisterPasskey.mockRejectedValue(new Error('Passkey creation was cancelled.'));
    render(<MfaCard />);

    await userEvent.click(await screen.findByRole('button', { name: /passkey/i }));

    expect(await screen.findByText(/cancelled/i)).toBeInTheDocument();
  });
});
