/**
 * FirstRunEoxStep.test.tsx — the first-run EoX feed question (UI-SEED-47, #3460).
 *
 * Owner 2026-10-14: the feed is off until an admin answers on, and "no" is the
 * default answer. These pin that the question opens on Off and saving it
 * records off, that an answered or unreadable feed is not asked, that a
 * refusal is shown and never a dead end, and that the wizard asks it after
 * the SNMP step rather than finishing setup.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import type { LifecycleFeedResponse } from '../../types/generated/lifecycle-feed-response';
import type { LifecycleFeedUpdate } from '../../types/generated/lifecycle-feed-update';

const UNANSWERED: LifecycleFeedResponse = {
  choice: 'unanswered',
  endpoint: '',
  tokenUrl: '',
  clientId: '',
  hasClientSecret: false,
  fetched: 0,
};

let stored: LifecycleFeedResponse = UNANSWERED;
let loadError: Error | null = null;
let refusal: Error | null = null;
const puts: LifecycleFeedUpdate[] = [];

vi.mock('../../api', () => ({
  api: {
    get: async (): Promise<unknown> => {
      if (loadError) {
        throw loadError;
      }
      return stored;
    },
    put: async (_path: string, body: LifecycleFeedUpdate): Promise<unknown> => {
      puts.push(body);
      if (refusal) {
        throw refusal;
      }
      stored = { ...stored, choice: body.choice, clientId: body.clientId ?? '' };
      return stored;
    },
    post: async (): Promise<unknown> => ({}),
  },
}));

const { FirstRunEoxStep } = await import('./FirstRunEoxStep');
const { SetupWizard } = await import('./SetupWizard');

function renderStep(): { onDone: () => void } {
  const onDone = vi.fn();
  render(<FirstRunEoxStep onDone={onDone} />);
  return { onDone };
}

beforeEach(async () => {
  await i18n.changeLanguage('en');
  stored = UNANSWERED;
  loadError = null;
  refusal = null;
  puts.length = 0;
});

afterEach(async () => {
  vi.restoreAllMocks();
  await i18n.changeLanguage('en');
});

describe('FirstRunEoxStep', () => {
  it('opens on Off and saving it unchanged records off, then leaves setup', async () => {
    const user = userEvent.setup();
    const { onDone } = renderStep();

    await waitFor(() => expect(screen.getByTestId('setup-eox-save')).toBeEnabled());
    expect(screen.getByTestId('eox-feed-off')).toBeChecked();
    expect(screen.queryByTestId('eox-feed-client-id')).toBeNull();

    await user.click(screen.getByTestId('setup-eox-save'));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(puts).toEqual([{ choice: 'off', endpoint: '', tokenUrl: '', clientId: '' }]);
  });

  it('turning it on sends the Cisco API client with the answer', async () => {
    const user = userEvent.setup();
    const { onDone } = renderStep();

    await waitFor(() => expect(screen.getByTestId('setup-eox-save')).toBeEnabled());
    await user.click(screen.getByTestId('eox-feed-on'));
    await user.type(screen.getByTestId('eox-feed-client-id'), 'client-1');
    await user.type(screen.getByTestId('eox-feed-client-secret'), 's3cret');
    await user.click(screen.getByTestId('setup-eox-save'));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(puts).toEqual([
      { choice: 'on', endpoint: '', tokenUrl: '', clientId: 'client-1', clientSecret: 's3cret' },
    ]);
  });

  it('shows a refusal with its reason, stays put, and skipping still leaves', async () => {
    refusal = new Error('a feed needs a client ID');
    const user = userEvent.setup();
    const { onDone } = renderStep();

    await waitFor(() => expect(screen.getByTestId('setup-eox-save')).toBeEnabled());
    await user.click(screen.getByTestId('eox-feed-on'));
    await user.click(screen.getByTestId('setup-eox-save'));

    expect(await screen.findByTestId('setup-eox-error')).toHaveTextContent(
      'The answer could not be saved: a feed needs a client ID',
    );
    expect(onDone).not.toHaveBeenCalled();

    await user.click(screen.getByTestId('setup-eox-skip'));
    expect(onDone).toHaveBeenCalledTimes(1);
    expect(puts).toHaveLength(1);
  });

  it('skipping saves nothing, so the feed stays off and unanswered', async () => {
    const user = userEvent.setup();
    const { onDone } = renderStep();

    await user.click(screen.getByTestId('setup-eox-skip'));

    expect(onDone).toHaveBeenCalledTimes(1);
    expect(puts).toHaveLength(0);
  });

  it('does not ask again once the feed is answered', async () => {
    stored = { ...UNANSWERED, choice: 'off' };
    const { onDone } = renderStep();

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(puts).toHaveLength(0);
  });

  it('leaves the question to Settings when the setting cannot be read', async () => {
    loadError = new Error('service unavailable');
    const { onDone } = renderStep();

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(puts).toHaveLength(0);
  });

  it('renders in Spanish, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    renderStep();

    expect(await screen.findByText('Fechas de fin de vida del hardware')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Guardar y continuar' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Omitir por ahora' })).toBeInTheDocument();
    expect(
      screen.queryByText(/Hardware end-of-life dates|Save and continue|Skip for now/),
    ).toBeNull();
  });
});

describe('SetupWizard — EoX question after the SNMP step', () => {
  it('asks the EoX question instead of finishing setup when SNMP is skipped', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((async () => ({
      ok: true,
      status: 200,
      json: async () => ({ providers: [] }),
    })) as unknown as typeof fetch);
    const onComplete = vi.fn();
    const password = 'Correct-Horse-9!';
    const user = userEvent.setup();
    render(
      <SetupWizard
        onComplete={onComplete}
        onLogin={vi.fn(async () => ({ status: 'ok' }) as const)}
        username="admin"
        setupTokenFile="/var/lib/seed/.setup-token"
      />,
    );

    for (const [label, value] of [
      ['Setup token', 't-1'],
      ['Password', password],
      ['Confirm Password', password],
    ] as const) {
      await user.click(screen.getByLabelText(label));
      await user.paste(value);
    }
    await user.click(screen.getByRole('button', { name: 'Complete Setup' }));
    await user.click(await screen.findByRole('button', { name: 'Skip for now' }));

    expect(await screen.findByTestId('setup-eox-step')).toBeInTheDocument();
    expect(onComplete).not.toHaveBeenCalled();

    await user.click(screen.getByTestId('setup-eox-skip'));
    expect(onComplete).toHaveBeenCalledTimes(1);
  });
});
