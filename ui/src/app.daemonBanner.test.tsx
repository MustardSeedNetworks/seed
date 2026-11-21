/**
 * The shell's daemon banner, through the real App: /__version unreachable shows
 * "Can't reach Seed daemon, retrying"; /__version answering shows nothing.
 * Split from app.test.tsx, which is at its file-size baseline.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './app';
import './app/AppShell';
import './app/LoginForm';
import { ProfileProvider } from './contexts/profileContext';

class StubEventSource {
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: (() => void) | null = null;
  close(): void {}
  addEventListener(): void {}
  removeEventListener(): void {}
}

const mockFetch = vi.fn();

function respond(daemonUp: boolean) {
  return (url: string): Promise<unknown> => {
    if (url.includes('/__version')) {
      return daemonUp
        ? Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
        : Promise.reject(new TypeError('Failed to fetch'));
    }
    if (url.includes('/api/v1/setup/status')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ needsSetup: false, username: 'admin' }),
      });
    }
    if (url.includes('/api/v1/status')) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ authenticated: true }),
      });
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
  };
}

function renderApp(): void {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ProfileProvider>
        <App />
      </ProfileProvider>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch);
  vi.stubGlobal('EventSource', StubEventSource);
});
afterEach(() => {
  mockFetch.mockReset();
  vi.unstubAllGlobals();
});

describe('daemon banner in the shell', () => {
  it('shows the banner when /__version is unreachable', async () => {
    mockFetch.mockImplementation(respond(false));
    renderApp();
    expect(await screen.findByTestId('daemon-banner')).toHaveTextContent(
      "Can't reach Seed daemon, retrying",
    );
  });

  it('shows no banner when /__version answers', async () => {
    mockFetch.mockImplementation(respond(true));
    renderApp();
    await waitFor(() => expect(screen.getAllByText('Seed').length).toBeGreaterThan(0));
    await waitFor(() =>
      expect(mockFetch.mock.calls.some(([u]) => String(u).includes('/__version'))).toBe(true),
    );
    expect(screen.queryByTestId('daemon-banner')).toBeNull();
  });
});
