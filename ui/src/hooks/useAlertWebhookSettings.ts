/**
 * useAlertWebhookSettings
 *
 * Loads and saves the outbound alert receiver (#2605) through the main
 * settings endpoint. Unlike the other settings hooks this one does not
 * auto-save: half a signing key is not a signing key, and a debounced write of
 * one would re-point the live receiver at a secret the operator is still
 * typing. The section saves on an explicit action.
 *
 * The secret is write-only by design. The server reports whether one is stored,
 * never the material, so the input starts empty on every load and an empty
 * input on save means "leave the stored secret alone".
 */

import { useCallback, useState } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { AlertWebhookSettings, SaveStatus } from '../types/settings';

/** The alerts slice of GET /api/v1/settings. */
interface AlertsSettingsResponse {
  alerts?: { webhook?: { url?: string; secretSet?: boolean } };
}

interface UseAlertWebhookSettingsResult {
  webhook: AlertWebhookSettings;
  setWebhook: React.Dispatch<React.SetStateAction<AlertWebhookSettings>>;
  status: SaveStatus;
  /** The receiver's own reason when a save was refused, for display. */
  error: string;
  fetchWebhook: () => Promise<void>;
  saveWebhook: () => Promise<void>;
}

export const EMPTY_ALERT_WEBHOOK: AlertWebhookSettings = {
  url: '',
  secret: '',
  secretSet: false,
};

export function useAlertWebhookSettings(): UseAlertWebhookSettingsResult {
  const [webhook, setWebhook] = useState<AlertWebhookSettings>(EMPTY_ALERT_WEBHOOK);
  const [status, setStatus] = useState<SaveStatus>('idle');
  const [error, setError] = useState('');

  const fetchWebhook = useCallback(async (): Promise<void> => {
    try {
      const data = await api.get<AlertsSettingsResponse>('/api/v1/settings');
      setWebhook({
        url: data.alerts?.webhook?.url ?? '',
        secret: '',
        secretSet: data.alerts?.webhook?.secretSet ?? false,
      });
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch alert webhook settings', err);
    }
  }, []);

  const saveWebhook = useCallback(async (): Promise<void> => {
    setStatus('saving');
    setError('');
    // An untouched secret input is omitted, not sent empty: the server reads an
    // empty string as "no change", and sending one on every save would be the
    // same thing said less clearly.
    const payload: { url: string; secret?: string } = { url: webhook.url };
    if (webhook.secret !== '') {
      payload.secret = webhook.secret;
    }
    try {
      await api.put('/api/v1/settings', { alerts: { webhook: payload } });
      setStatus('saved');
      // The secret has left the browser; drop it rather than keep it in state
      // where a later save would resend it.
      setWebhook((current) => ({
        url: current.url,
        secret: '',
        secretSet: current.url !== '' && (current.secret !== '' || current.secretSet),
      }));
      setTimeout(() => setStatus('idle'), 2000);
    } catch (err) {
      setStatus('error');
      setError(err instanceof Error ? err.message : '');
    }
  }, [webhook]);

  return { webhook, setWebhook, status, error, fetchWebhook, saveWebhook };
}
