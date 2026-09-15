/**
 * useAlertWebhookSettings tests (seed#2605).
 *
 * The rules here are the ones a secret field lives or dies by: the material
 * never comes back from the server, an untouched field never overwrites what is
 * stored, and what the operator typed leaves the browser's state once it has
 * been saved.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useAlertWebhookSettings } from './useAlertWebhookSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPut = vi.fn<(path: string, body: unknown) => Promise<unknown>>();

vi.mock('../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));

describe('useAlertWebhookSettings', () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPut.mockReset();
    mockPut.mockResolvedValue({});
  });

  it('loads the url and that a secret is stored, never the secret', async () => {
    // The server does not serve the signing material, and this asserts the
    // browser would not hold it even if one arrived: a secret in React state
    // is a secret in a heap snapshot and in every later save payload.
    mockGet.mockResolvedValue({
      alerts: {
        webhook: {
          url: 'https://receiver.example.com/hook',
          secretSet: true,
          secret: 'should-never-be-served',
        },
      },
    });
    const { result } = renderHook(() => useAlertWebhookSettings());

    // The hook loads on mount; no caller has to ask it to.
    await waitFor(() => {
      expect(result.current.webhook.url).not.toBe('');
    });

    expect(result.current.webhook.url).toBe('https://receiver.example.com/hook');
    expect(result.current.webhook.secretSet).toBe(true);
    // The server has no secret to send and the field must start empty, so an
    // operator who saves without touching it keeps the stored one.
    expect(result.current.webhook.secret).toBe('');
  });

  it('omits the secret from a save the operator did not type one into', async () => {
    mockGet.mockResolvedValue({
      alerts: { webhook: { url: 'https://first.example.com/hook', secretSet: true } },
    });
    const { result } = renderHook(() => useAlertWebhookSettings());
    await waitFor(() => {
      expect(result.current.webhook.secretSet).toBe(true);
    });

    act(() => {
      result.current.setWebhook((current) => ({ ...current, url: 'https://second.example.com/h' }));
    });
    await act(async () => {
      await result.current.saveWebhook();
    });

    expect(mockPut).toHaveBeenCalledWith('/api/v1/settings', {
      alerts: { webhook: { url: 'https://second.example.com/h' } },
    });
  });

  it('sends a secret the operator typed, then drops it from state', async () => {
    const { result } = renderHook(() => useAlertWebhookSettings());
    act(() => {
      result.current.setWebhook({
        url: 'https://receiver.example.com/hook',
        secret: 's3cret',
        secretSet: false,
      });
    });
    await act(async () => {
      await result.current.saveWebhook();
    });

    expect(mockPut).toHaveBeenCalledWith('/api/v1/settings', {
      alerts: { webhook: { url: 'https://receiver.example.com/hook', secret: 's3cret' } },
    });
    // Keeping it would resend it on the next save of an unrelated field.
    expect(result.current.webhook.secret).toBe('');
    expect(result.current.webhook.secretSet).toBe(true);
  });

  it('surfaces the receiver’s own reason when a save is refused', async () => {
    mockPut.mockRejectedValue(new Error('alerts.webhook.url scheme "ftp" is not http or https'));
    const { result } = renderHook(() => useAlertWebhookSettings());
    act(() => {
      result.current.setWebhook({ url: 'ftp://nope', secret: 's3cret', secretSet: false });
    });
    await act(async () => {
      await result.current.saveWebhook();
    });

    await waitFor(() => {
      expect(result.current.status).toBe('error');
    });
    expect(result.current.error).toContain('not http or https');
  });
});
