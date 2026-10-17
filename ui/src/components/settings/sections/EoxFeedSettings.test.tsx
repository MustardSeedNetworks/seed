/**
 * EoX feed setting (UI-SEED-47): an unanswered feed is said to be off and
 * edits as off, turning it on sends the client, a stored secret is kept or
 * removed only on request, and the server's refusal is shown.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LifecycleFeedResponse } from '../../../types/generated/lifecycle-feed-response';
import type { LifecycleFeedUpdate } from '../../../types/generated/lifecycle-feed-update';

const { ApiError } = await import('../../../api/client');

const UNANSWERED: LifecycleFeedResponse = {
  choice: 'unanswered',
  endpoint: '',
  tokenUrl: '',
  clientId: '',
  hasClientSecret: false,
  fetched: 0,
};

let stored: LifecycleFeedResponse = UNANSWERED;
let refusal: InstanceType<typeof ApiError> | null = null;
const puts: LifecycleFeedUpdate[] = [];

vi.mock('../../../api', () => ({
  api: {
    get: async (): Promise<unknown> => stored,
    put: async (_path: string, body: LifecycleFeedUpdate): Promise<unknown> => {
      puts.push(body);
      if (refusal) {
        throw refusal;
      }
      stored = {
        ...stored,
        choice: body.choice,
        clientId: body.clientId ?? '',
        hasClientSecret:
          body.clientSecret !== undefined || (stored.hasClientSecret && !body.clearClientSecret),
      };
      return stored;
    },
  },
}));

const { EoxFeedSettings } = await import('./EoxFeedSettings');

async function openSection(user: ReturnType<typeof userEvent.setup>): Promise<void> {
  render(<EoxFeedSettings />);
  await user.click(screen.getByText('EoX feed'));
  await waitFor(() => expect(screen.getByTestId('eox-feed-save')).toBeEnabled());
}

describe('EoxFeedSettings', () => {
  beforeEach(() => {
    stored = UNANSWERED;
    refusal = null;
    puts.length = 0;
  });

  it('says an unanswered feed is off, defaults the answer to off and saves it', async () => {
    const user = userEvent.setup();
    await openSection(user);

    expect(screen.getByTestId('eox-feed-unanswered')).toHaveTextContent(
      'The feed stays off until you turn it on.',
    );
    expect(screen.getByTestId('eox-feed-off')).toBeChecked();
    expect(screen.queryByTestId('eox-feed-client-id')).toBeNull();

    await user.click(screen.getByTestId('eox-feed-save'));
    await waitFor(() => expect(screen.queryByTestId('eox-feed-unanswered')).toBeNull());
    expect(puts).toEqual([{ choice: 'off', endpoint: '', tokenUrl: '', clientId: '' }]);
  });

  it('turns the feed on with a client', async () => {
    const user = userEvent.setup();
    await openSection(user);
    await user.click(screen.getByTestId('eox-feed-on'));
    await user.type(screen.getByTestId('eox-feed-client-id'), ' seed-client ');
    await user.type(screen.getByTestId('eox-feed-client-secret'), 's3cret');
    await user.click(screen.getByTestId('eox-feed-save'));

    await waitFor(() => expect(puts).toHaveLength(1));
    expect(puts[0]).toEqual({
      choice: 'on',
      endpoint: '',
      tokenUrl: '',
      clientId: 'seed-client',
      clientSecret: 's3cret',
    });
    // The secret has left the browser; the field now says one is stored.
    await waitFor(() =>
      expect(screen.getByTestId('eox-feed-client-secret')).toHaveAttribute(
        'placeholder',
        'A client secret is stored; type to replace it',
      ),
    );
    expect(screen.getByTestId('eox-feed-client-secret')).toHaveValue('');
  });

  it('keeps a stored secret unless asked to remove it', async () => {
    stored = { ...UNANSWERED, choice: 'on', clientId: 'seed-client', hasClientSecret: true };
    const user = userEvent.setup();
    await openSection(user);
    expect(screen.queryByTestId('eox-feed-unanswered')).toBeNull();

    await user.click(screen.getByTestId('eox-feed-save'));
    await waitFor(() => expect(puts).toHaveLength(1));
    expect(puts[0]).not.toHaveProperty('clientSecret');
    expect(puts[0]).not.toHaveProperty('clearClientSecret');

    await user.click(screen.getByTestId('eox-feed-clear-secret'));
    await user.click(screen.getByTestId('eox-feed-off'));
    await user.click(screen.getByTestId('eox-feed-save'));
    await waitFor(() => expect(puts).toHaveLength(2));
    expect(puts[1]).toMatchObject({ choice: 'off', clearClientSecret: true });
  });

  it('shows the server refusal and keeps the draft', async () => {
    refusal = new ApiError(
      400,
      { error: 'invalid EoX feed settings: the Cisco EoX API needs a client ID and secret' },
      undefined,
      null,
    );
    const user = userEvent.setup();
    await openSection(user);
    await user.click(screen.getByTestId('eox-feed-on'));
    await user.click(screen.getByTestId('eox-feed-save'));

    expect(await screen.findByTestId('eox-feed-error')).toHaveTextContent(
      'the Cisco EoX API needs a client ID and secret',
    );
    expect(screen.getByTestId('eox-feed-on')).toBeChecked();
  });

  it('reports the last refresh', async () => {
    stored = {
      ...UNANSWERED,
      choice: 'on',
      clientId: 'seed-client',
      hasClientSecret: true,
      lastAttempt: '2026-10-17T10:00:00Z',
      lastSuccess: '2026-10-16T10:00:00Z',
      lastError: 'token request: 401',
      fetched: 12,
    };
    const user = userEvent.setup();
    await openSection(user);

    expect(screen.getByTestId('eox-feed-last-success')).toHaveTextContent('12 products');
    expect(screen.getByTestId('eox-feed-last-error')).toHaveTextContent('token request: 401');
  });
});
