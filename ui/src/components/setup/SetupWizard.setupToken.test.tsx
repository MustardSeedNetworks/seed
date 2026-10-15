/**
 * SetupWizard.setupToken.test.tsx — the operator proves host access.
 *
 * The daemon never sends the setup token over HTTP; it writes it to a file on
 * its host. The wizard must say where that file is, refuse to submit without
 * the token, and send exactly what the operator pasted.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { SetupWizard } from './SetupWizard';

const PASSWORD = 'Correct-Horse-9!';
const TOKEN_FILE = '/var/lib/seed/.setup-token';

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(async () => {
  await i18n.changeLanguage('en');
  fetchMock = vi.fn(async () => ({
    ok: true,
    status: 200,
    json: async () => ({ providers: [] }),
  }));
  vi.spyOn(globalThis, 'fetch').mockImplementation(fetchMock as unknown as typeof fetch);
});

afterEach(() => {
  vi.restoreAllMocks();
});

function renderWizard(): void {
  render(
    <SetupWizard
      onComplete={vi.fn()}
      onLogin={vi.fn(async () => ({ status: 'ok' }) as const)}
      username="admin"
      setupTokenFile={TOKEN_FILE}
    />,
  );
}

function setupCompleteBodies(): unknown[] {
  return fetchMock.mock.calls
    .filter(([url]) => String(url).endsWith('/api/v1/setup/complete'))
    .map(([, init]) => JSON.parse(String((init as RequestInit).body)));
}

describe('SetupWizard — setup token', () => {
  it('names the host file the token is in', () => {
    renderWizard();

    expect(screen.getByText(new RegExp(TOKEN_FILE.replaceAll('.', '\\.')))).toBeInTheDocument();
  });

  it('does not submit without the token', async () => {
    const user = userEvent.setup();
    renderWizard();

    await user.type(screen.getByLabelText('Password'), PASSWORD);
    await user.type(screen.getByLabelText('Confirm Password'), PASSWORD);
    await user.click(screen.getByRole('button', { name: 'Complete Setup' }));

    expect(screen.getByLabelText('Setup token')).toBeInvalid();
    expect(setupCompleteBodies()).toHaveLength(0);
  });

  it('sends the pasted token with the password', async () => {
    const user = userEvent.setup();
    renderWizard();

    await user.type(screen.getByLabelText('Setup token'), '  token-from-file  ');
    await user.type(screen.getByLabelText('Password'), PASSWORD);
    await user.type(screen.getByLabelText('Confirm Password'), PASSWORD);
    await user.click(screen.getByRole('button', { name: 'Complete Setup' }));

    await waitFor(() => {
      expect(setupCompleteBodies()).toEqual([
        { password: PASSWORD, setupToken: 'token-from-file' },
      ]);
    });
  });
});
