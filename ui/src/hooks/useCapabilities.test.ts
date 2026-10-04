/**
 * useCapabilities reads GET /api/v1/status once signed in, and carries whether
 * the platform can run a cable test at all (#2691). Before this, it read once
 * on mount, which on a fresh sign-in was the login screen's 401.
 */
import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useCapabilities } from './useCapabilities';

function statusWith(level: string): ReturnType<typeof vi.fn> {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: () =>
      Promise.resolve({
        icmpAvailable: true,
        capabilities: [{ capability: 'cable_diagnostics', title: 'Cable', level }],
      }),
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('useCapabilities', () => {
  it('does not read the status before sign-in, and reads it after', async () => {
    const fetchMock = statusWith('full');
    vi.stubGlobal('fetch', fetchMock);
    const { result, rerender } = renderHook(
      ({ signedIn }: { signedIn: boolean }) => useCapabilities(signedIn),
      { initialProps: { signedIn: false } },
    );
    expect(fetchMock).not.toHaveBeenCalled();

    rerender({ signedIn: true });

    await waitFor(() =>
      expect(result.current.capabilities).toEqual({ icmpAvailable: true, cableDiagnostics: true }),
    );
  });

  it.each([
    { level: 'full', cableDiagnostics: true },
    { level: 'partial', cableDiagnostics: true },
    { level: 'none', cableDiagnostics: false },
  ])(
    'reports cable diagnostics $cableDiagnostics at level $level',
    async ({ level, cableDiagnostics }) => {
      vi.stubGlobal('fetch', statusWith(level));
      const { result } = renderHook(() => useCapabilities(true));

      await waitFor(() =>
        expect(result.current.capabilities?.cableDiagnostics).toBe(cableDiagnostics),
      );
    },
  );
});
