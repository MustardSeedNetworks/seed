/**
 * Context value identity, held by the React Compiler rather than by
 * hand-written memos (UI-SEED-41, #3066).
 *
 * A provider's value object and its functions must keep their identity
 * across a re-render that changed nothing they derive from; otherwise every
 * consumer re-renders and every effect that lists them re-runs. These
 * tests fail when the compiler is removed from the vitest config, so they
 * measure the compiler, not luck.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, render, renderHook, screen, waitFor } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useProfileStore } from '../stores/profileStore';
import type { Profile } from '../types/profile';
import { LicenseProvider, useLicense } from './LicenseContext';
import { ProfileProvider, useProfileContext } from './profileContext';
import { RoleProvider, useRole } from './RoleContext';
import { useProfileInterfaces } from './useProfileInterfaces';
import { useSettings } from './useSettings';

const { mockGet, apiMock } = vi.hoisted(() => {
  const get = vi.fn<(path: string) => Promise<unknown>>();
  return {
    mockGet: get,
    apiMock: {
      api: {
        get: (path: string): Promise<unknown> => get(path),
        post: (): Promise<unknown> => Promise.resolve({}),
        put: (): Promise<unknown> => Promise.resolve({}),
        patch: (): Promise<unknown> => Promise.resolve({}),
        delete: (): Promise<unknown> => Promise.resolve({}),
      },
    },
  };
});
vi.mock('../api/client', () => apiMock);
vi.mock('../api', () => apiMock);

beforeEach(() => {
  mockGet.mockReset();
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role: 'operator', isActive: true });
    }
    if (path.includes('/license')) {
      return Promise.resolve({ tier: 'pro', features: ['multi_interface'] });
    }
    return Promise.resolve({});
  });
});

function functionsOf(value: object): [string, unknown][] {
  return Object.entries(value).filter(([, v]) => typeof v === 'function');
}

describe('provider value identity across an unrelated re-render', () => {
  it('RoleProvider keeps its value and refresh', async () => {
    const wrapper = ({ children }: { children: ReactNode }): ReactElement => (
      <RoleProvider isAuthenticated={true}>{children}</RoleProvider>
    );
    const { result, rerender } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.user?.role).toBe('operator'));
    const before = result.current;
    rerender();
    expect(result.current).toBe(before);
  });

  it('LicenseProvider keeps its value, refresh and hasFeature', async () => {
    const wrapper = ({ children }: { children: ReactNode }): ReactElement => (
      <LicenseProvider isAuthenticated={true}>{children}</LicenseProvider>
    );
    const { result, rerender } = renderHook(() => useLicense(), { wrapper });
    await waitFor(() => expect(result.current.hasFeature('multi_interface')).toBe(true));
    const before = result.current;
    rerender();
    expect(result.current).toBe(before);
  });

  it('ProfileProvider keeps its value and every action, and useSettings its adapter', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }): ReactElement => (
      <QueryClientProvider client={queryClient}>
        <ProfileProvider>{children}</ProfileProvider>
      </QueryClientProvider>
    );
    const { result, rerender } = renderHook(
      () => ({ profile: useProfileContext(), settings: useSettings() }),
      { wrapper },
    );
    await waitFor(() => expect(queryClient.isFetching()).toBe(0));
    const before = result.current;
    rerender();
    for (const [name, fn] of functionsOf(before.profile)) {
      expect(result.current.profile[name as keyof typeof before.profile], name).toBe(fn);
    }
    expect(result.current.profile).toBe(before.profile);
    expect(result.current.settings).toBe(before.settings);
  });
});

function profileWith(ethernet: string): Profile {
  return {
    id: 'p1',
    name: 'p',
    description: '',
    isDefault: true,
    createdAt: '2026-10-04T00:00:00Z',
    updatedAt: '2026-10-04T00:00:00Z',
    config: { interfaces: { ethernet: [{ name: ethernet, enabled: true }], wifi: [] } },
  };
}

// Reads a getter while rendering, as InterfacesSettings does.
function EthernetNames(): ReactElement {
  const { getAllEthernetInterfaces } = useProfileInterfaces();
  return (
    <span data-testid="names">
      {getAllEthernetInterfaces()
        .map((i) => i.name)
        .join(',')}
    </span>
  );
}

describe('useProfileInterfaces', () => {
  it('a getter read during render follows the active profile', () => {
    act(() => useProfileStore.getState().setActiveProfile(profileWith('eth0')));
    render(<EthernetNames />);
    expect(screen.getByTestId('names')).toHaveTextContent('eth0');
    act(() => useProfileStore.getState().setActiveProfile(profileWith('eth1')));
    expect(screen.getByTestId('names')).toHaveTextContent('eth1');
  });
});
