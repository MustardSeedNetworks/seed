/**
 * AlertEscalationSettings.test.tsx — the escalation ladder editor (#3186).
 *
 * The server replaces alerts.escalations whole and refuses a ladder that could
 * not run, so what matters here is the list the section sends: minutes become
 * seconds, list keys never leave the browser, a removed ladder is gone from
 * the save, and a refusal is shown in the server's own words.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPut = vi.fn<(path: string, body: unknown) => Promise<unknown>>();
// The section reads through `api`, RoleContext through `api/client`.
vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));

const { AlertEscalationSettings } = await import('./AlertEscalationSettings');

const STORED = {
  rule: 'iface.down',
  stages: [
    { afterSeconds: 300, channels: ['webhook'] },
    { afterSeconds: 900, channels: ['webhook', 'email'] },
  ],
  repeatSeconds: 1800,
};

function serve(role: CurrentUser['role'], escalations: unknown[]): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }
    if (path === '/api/v1/settings') {
      return Promise.resolve({ alerts: { escalations } });
    }
    if (path === '/api/v1/alerts?limit=1000') {
      return Promise.resolve({
        count: 3,
        alerts: [{ rule: 'bgp.flap' }, { rule: 'iface.down' }, { rule: 'bgp.flap' }, {}],
      });
    }
    return Promise.resolve({});
  });
}

async function open(): Promise<void> {
  render(
    <RoleProvider isAuthenticated={true}>
      <AlertEscalationSettings />
    </RoleProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /alert escalation/i }));
}

function sentEscalations(): unknown {
  expect(mockPut).toHaveBeenCalledTimes(1);
  const [path, body] = mockPut.mock.calls[0] ?? [];
  expect(path).toBe('/api/v1/settings');
  return (body as { alerts: { escalations: unknown } }).alerts.escalations;
}

beforeEach(() => {
  mockGet.mockReset();
  mockPut.mockReset();
  mockPut.mockResolvedValue({});
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('AlertEscalationSettings', () => {
  it('shows a stored ladder in minutes with its channels', async () => {
    serve('operator', [STORED]);
    await open();

    expect(await screen.findByTestId('escalation-rule-0')).toHaveValue('iface.down');
    expect(screen.getByTestId('escalation-after-0-0')).toHaveValue(5);
    expect(screen.getByTestId('escalation-after-0-1')).toHaveValue(15);
    expect(screen.getByTestId('escalation-channel-0-1-email')).toBeChecked();
    expect(screen.getByTestId('escalation-channel-0-0-email')).not.toBeChecked();
    expect(screen.getByTestId('escalation-repeat-0')).toHaveValue(30);
  });

  it('suggests each rule the inbox carries, once', async () => {
    serve('operator', []);
    await open();

    await waitFor(() => {
      const options = [...document.querySelectorAll('datalist option')].map((o) =>
        o.getAttribute('value'),
      );
      expect(options).toEqual(['bgp.flap', 'iface.down']);
    });
  });

  it('saves a new two-stage ladder as seconds, without list keys', async () => {
    serve('operator', []);
    await open();
    expect(await screen.findByTestId('escalation-empty')).toBeVisible();

    await userEvent.click(screen.getByTestId('escalation-add'));
    await userEvent.type(screen.getByTestId('escalation-rule-0'), 'bgp.flap');
    await userEvent.click(screen.getByTestId('escalation-add-stage-0'));
    await userEvent.click(screen.getByTestId('escalation-channel-0-1-syslog'));
    await userEvent.click(screen.getByTestId('escalation-channel-0-1-webhook'));
    await userEvent.clear(screen.getByTestId('escalation-after-0-1'));
    await userEvent.type(screen.getByTestId('escalation-after-0-1'), '20');
    await userEvent.click(screen.getByTestId('escalation-save'));

    await waitFor(() => expect(mockPut).toHaveBeenCalled());
    expect(sentEscalations()).toEqual([
      {
        rule: 'bgp.flap',
        stages: [
          { afterSeconds: 300, channels: ['webhook'] },
          { afterSeconds: 1200, channels: ['syslog'] },
        ],
        repeatSeconds: 0,
      },
    ]);
  });

  it('removes a stage and then the whole ladder', async () => {
    serve('operator', [STORED]);
    await open();

    await userEvent.click(await screen.findByTestId('escalation-remove-stage-0-0'));
    expect(screen.getByTestId('escalation-after-0-0')).toHaveValue(15);
    // The last stage cannot go: a ladder with none could not run.
    expect(screen.getByTestId('escalation-remove-stage-0-0')).toBeDisabled();

    await userEvent.click(screen.getByTestId('escalation-remove-0'));
    await userEvent.click(screen.getByTestId('escalation-save'));

    await waitFor(() => expect(mockPut).toHaveBeenCalled());
    expect(sentEscalations()).toEqual([]);
  });

  it('stops adding stages at five', async () => {
    serve('operator', [STORED]);
    await open();

    const add = await screen.findByTestId('escalation-add-stage-0');
    await userEvent.click(add);
    await userEvent.click(add);
    await userEvent.click(add);
    expect(screen.getByTestId('escalation-after-0-4')).toHaveValue(30);
    expect(add).toBeDisabled();
  });

  it("shows the server's reason when it refuses a ladder", async () => {
    serve('operator', [STORED]);
    mockPut.mockRejectedValue(
      new Error(
        'invalid escalation ladder: iface.down stage 1 must come at least 1m0s after the alert',
      ),
    );
    await open();

    await userEvent.click(await screen.findByTestId('escalation-save'));

    expect(await screen.findByTestId('escalation-error')).toHaveTextContent(
      'iface.down stage 1 must come at least 1m0s after the alert',
    );
  });

  it('is read-only for a viewer', async () => {
    serve('viewer', [STORED]);
    await open();

    // The tooltip keeps a disabled button focusable so the reason is readable.
    await waitFor(() =>
      expect(screen.getByTestId('escalation-save')).toHaveAttribute('aria-disabled', 'true'),
    );
    expect(screen.getByTestId('escalation-add')).toHaveAttribute('aria-disabled', 'true');
    expect(screen.getByTestId('escalation-rule-0')).toBeDisabled();
    expect(screen.getByTestId('escalation-remove-0')).toBeDisabled();
  });
});
