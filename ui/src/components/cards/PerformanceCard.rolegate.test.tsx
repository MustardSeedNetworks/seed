/**
 * PerformanceCard role gating (#1254).
 *
 * Not a button: the card reconciles the iperf3 listener with the operator's
 * setting on every mount, and POST /api/v1/telemetry/iperf/server is
 * `minRole: op`. A viewer's dashboard therefore fired a silent 403 on load
 * against a listener that is not theirs to manage. The sync belongs to a
 * session that can perform it.
 */

import { render, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../contexts/RoleContext';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();

vi.mock('../../api/client', () => ({
  api: {
    get: (p: string): Promise<unknown> => mockGet(p),
    post: (p: string, b?: unknown): Promise<unknown> => mockPost(p, b),
  },
}));
vi.mock('../../api', () => ({
  api: {
    get: (p: string): Promise<unknown> => mockGet(p),
    post: (p: string, b?: unknown): Promise<unknown> => mockPost(p, b),
  },
}));

// The card reads the setting that says the server should run; the settings
// context itself is not what is under test.
vi.mock('../../contexts/useSettings', () => ({
  useSettings: () => ({ iperfSettings: { enableServer: true, serverPort: 5201 } }),
}));

const { PerformanceCard } = await import('./PerformanceCard');

/** Resolves once the card has tried to start or stop the listener. */
function waitForServerPost(): Promise<void> {
  return waitFor(
    () => {
      expect(
        mockPost.mock.calls.filter(([path]) => path === '/api/v1/telemetry/iperf/server'),
      ).not.toHaveLength(0);
    },
    { timeout: 500 },
  );
}

/** iperf is installed and the listener is down, so the sync wants to start it. */
function stubStatusEndpoints(): void {
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      const body = url.includes('/iperf/info')
        ? { installed: true }
        : url.includes('/iperf/server/status')
          ? { running: false, port: 5201, pid: 0 }
          : {};
      return Promise.resolve({ ok: true, json: () => Promise.resolve(body) } as Response);
    }),
  );
}

function renderAs(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role, isActive: true })
      : Promise.resolve({ installed: true }),
  );
  render(
    <RoleProvider isAuthenticated={true}>
      <PerformanceCard />
    </RoleProvider>,
  );
}

beforeEach(() => {
  mockGet.mockReset();
  mockPost.mockReset();
  stubStatusEndpoints();
});

describe('PerformanceCard — viewer gating', () => {
  it('never posts to the operator-gated iperf server route as a viewer', async () => {
    renderAs('viewer');

    // Asserting an absence needs a window, not a tick: the same waitFor the
    // operator case below satisfies must time out here. A bare assertion after
    // one flush passes whether or not the post is coming.
    await expect(waitForServerPost()).rejects.toThrow();
  });

  it('still reconciles the listener for an operator, so the check above means something', async () => {
    renderAs('operator');

    await waitForServerPost();
  });
});
