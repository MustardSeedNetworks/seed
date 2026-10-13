import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { useDeviceCredentials } from './useDeviceCredentials';

afterEach(() => {
  vi.restoreAllMocks();
});

describe('useDeviceCredentials', () => {
  it('lists the vault', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      count: 1,
      credentials: [
        { id: 'cred-1', clientId: 'c', name: 'core', kind: 'v2c', createdAt: '', updatedAt: '' },
      ],
    });

    const { result } = renderHook(() => useDeviceCredentials());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(get).toHaveBeenCalledWith('/api/v1/device-credentials');
    expect(result.current.credentials.map((c) => c.name)).toEqual(['core']);
    expect(result.current.error).toBeNull();
  });

  it('reports a failed read rather than an empty vault', async () => {
    vi.spyOn(api, 'get').mockRejectedValue(new Error('Forbidden'));

    const { result } = renderHook(() => useDeviceCredentials());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('Forbidden');
    expect(result.current.credentials).toEqual([]);
  });
});
