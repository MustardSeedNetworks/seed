/**
 * SetupWizard.firstRunSnmp.test.tsx — the first-run SNMP community step (#2722).
 *
 * The last unbuilt clause of the owner's "scan by default" decision (#2674).
 * Discovery resolves its SNMP credentials from the vault per scan
 * (`internal/discovery/snmp_credentials.go`), and nothing else in the UI ever
 * writes to it, so a fresh install profiles nothing until a community exists.
 *
 * What these pin is the shape of the offer, not the wording: setup does not
 * finish behind the operator's back, saving reaches the vault endpoint,
 * skipping reaches nothing, and a failed save is never a dead end.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '../../api';
import i18n from '../../i18n';
import { SetupWizard } from './SetupWizard';

const PASSWORD = 'Correct-Horse-9!';

function renderWizard(overrides: { onComplete?: () => void } = {}) {
  const onComplete = overrides.onComplete ?? vi.fn();
  const onLogin = vi.fn(async () => ({ status: 'ok' }) as const);
  render(
    <SetupWizard onComplete={onComplete} onLogin={onLogin} username="admin" setupToken="t-1" />,
  );
  return { onComplete, onLogin };
}

/** Fills the password step and submits it; leaves the wizard wherever it lands. */
async function completePasswordStep(user: ReturnType<typeof userEvent.setup>): Promise<void> {
  await user.type(screen.getByLabelText('Password'), PASSWORD);
  await user.type(screen.getByLabelText('Confirm Password'), PASSWORD);
  await user.click(screen.getByRole('button', { name: 'Complete Setup' }));
}

beforeEach(async () => {
  await i18n.changeLanguage('en');
  // The wizard's own probes: SSO providers on mount, /setup/complete on submit.
  vi.spyOn(globalThis, 'fetch').mockImplementation((async () => ({
    ok: true,
    status: 200,
    json: async () => ({ providers: [] }),
  })) as unknown as typeof fetch);
});

afterEach(async () => {
  vi.restoreAllMocks();
  await i18n.changeLanguage('en');
});

describe('SetupWizard — first-run SNMP community', () => {
  it('offers the step instead of finishing setup once the password is set', async () => {
    const user = userEvent.setup();
    const { onComplete, onLogin } = renderWizard();

    await completePasswordStep(user);

    await waitFor(() => {
      expect(screen.getByLabelText('SNMP community string')).toBeInTheDocument();
    });
    expect(onLogin).toHaveBeenCalledWith('admin', PASSWORD);
    expect(onComplete).not.toHaveBeenCalled();
  });

  it('ships the field empty — no default community is ever suggested', async () => {
    const user = userEvent.setup();
    renderWizard();

    await completePasswordStep(user);

    const field = await screen.findByLabelText<HTMLInputElement>('SNMP community string');
    expect(field.value).toBe('');
    expect(field.placeholder).not.toMatch(/public/i);
  });

  it('saves the community to the credential vault, then completes setup', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue({ id: 'cred-0123456789ab' });
    const user = userEvent.setup();
    const { onComplete } = renderWizard();

    await completePasswordStep(user);
    await user.type(await screen.findByLabelText('SNMP community string'), 's3cret-ro');
    await user.click(screen.getByRole('button', { name: 'Save and continue' }));

    await waitFor(() => {
      expect(onComplete).toHaveBeenCalledTimes(1);
    });
    expect(post).toHaveBeenCalledWith(
      '/api/v1/device-credentials',
      expect.objectContaining({ community: 's3cret-ro' }),
    );
  });

  it('skipping completes setup and writes no credential', async () => {
    const post = vi.spyOn(api, 'post');
    const user = userEvent.setup();
    const { onComplete } = renderWizard();

    await completePasswordStep(user);
    await user.click(await screen.findByRole('button', { name: 'Skip for now' }));

    await waitFor(() => {
      expect(onComplete).toHaveBeenCalledTimes(1);
    });
    expect(post).not.toHaveBeenCalled();
  });

  it('renders the step in Spanish, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    const user = userEvent.setup();
    renderWizard();

    await user.type(screen.getByLabelText('Contraseña'), PASSWORD);
    await user.type(screen.getByLabelText('Confirmar Contraseña'), PASSWORD);
    await user.click(screen.getByRole('button', { name: 'Completar Configuración' }));

    expect(await screen.findByLabelText('Cadena de comunidad SNMP')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Guardar y continuar' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Omitir por ahora' })).toBeInTheDocument();
    expect(screen.queryByText(/SNMP community string|Save and continue|Skip for now/)).toBeNull();
  });

  it('a failed save reports the error and still lets the operator move on', async () => {
    vi.spyOn(api, 'post').mockRejectedValue(new Error('vault unavailable'));
    const user = userEvent.setup();
    const { onComplete } = renderWizard();

    await completePasswordStep(user);
    await user.type(await screen.findByLabelText('SNMP community string'), 's3cret-ro');
    await user.click(screen.getByRole('button', { name: 'Save and continue' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(/credential/i);
    expect(onComplete).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Skip for now' }));
    await waitFor(() => {
      expect(onComplete).toHaveBeenCalledTimes(1);
    });
  });
});
