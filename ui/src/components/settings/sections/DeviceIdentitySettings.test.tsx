/**
 * DeviceIdentitySettings tests (#195): the inputs start from what is stored,
 * a save sends what was typed and then shows what the server kept, and a
 * refused save says why.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../../../contexts/RoleContext';
import { DeviceIdentitySettings } from './DeviceIdentitySettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPut = vi.fn<(path: string, body: unknown) => Promise<unknown>>();

vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));
vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));

let stored = { name: 'seed-idf-3b', location: 'Main Office, IDF 3B' };

function renderSection(): void {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <RoleProvider isAuthenticated={true}>
        <DeviceIdentitySettings />
      </RoleProvider>
    </QueryClientProvider>,
  );
}

async function open(): Promise<void> {
  await userEvent.click(await screen.findByRole('button', { name: /device identity/i }));
}

describe('DeviceIdentitySettings', () => {
  beforeEach(() => {
    stored = { name: 'seed-idf-3b', location: 'Main Office, IDF 3B' };
    mockGet.mockReset();
    mockPut.mockReset();
    mockGet.mockImplementation((path: string) => {
      if (path.includes('/users/me')) {
        return Promise.resolve({ username: 'op', role: 'operator', isActive: true });
      }
      return Promise.resolve({ identity: { ...stored } });
    });
    mockPut.mockImplementation((_path: string, body: unknown) => {
      const { identity } = body as { identity: typeof stored };
      // The server trims; the section must show what it kept, not what it sent.
      stored = { name: identity.name.trim(), location: identity.location.trim() };
      return Promise.resolve({ status: 'updated' });
    });
  });

  it('starts from the stored identity', async () => {
    renderSection();
    await open();

    await waitFor(() => {
      expect(screen.getByTestId('device-identity-name')).toHaveValue('seed-idf-3b');
    });
    expect(screen.getByTestId('device-identity-location')).toHaveValue('Main Office, IDF 3B');
  });

  it('saves both fields and then shows what the server kept', async () => {
    renderSection();
    await open();
    const name = screen.getByTestId('device-identity-name');
    await waitFor(() => {
      expect(name).toHaveValue('seed-idf-3b');
    });

    await userEvent.clear(name);
    await userEvent.type(name, '  seed-rack-12  ');
    await userEvent.click(screen.getByTestId('device-identity-save'));

    expect(mockPut).toHaveBeenCalledWith('/api/v1/settings', {
      identity: { name: '  seed-rack-12  ', location: 'Main Office, IDF 3B' },
    });
    await waitFor(() => {
      expect(name).toHaveValue('seed-rack-12');
    });
  });

  it('shows the server reason when a save is refused', async () => {
    mockPut.mockRejectedValue(new Error('identity.name must be one line of text'));
    renderSection();
    await open();
    await waitFor(() => {
      expect(screen.getByTestId('device-identity-name')).toHaveValue('seed-idf-3b');
    });

    await userEvent.click(screen.getByTestId('device-identity-save'));

    expect(await screen.findByTestId('device-identity-error')).toHaveTextContent(
      'identity.name must be one line of text',
    );
  });
});
