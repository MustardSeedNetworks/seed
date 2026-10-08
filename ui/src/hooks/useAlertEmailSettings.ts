/**
 * useAlertEmailSettings
 *
 * Loads and saves the mail relay alerts are emailed through (P-B1, #3209)
 * through the main settings endpoint. Like the webhook it saves on an explicit
 * action: a relay is only usable once host, sender and recipients agree, and
 * the server refuses one that could never deliver with its reason.
 *
 * The password is write-only. The server reports whether one is stored, never
 * the material, so the input starts empty on every load and an omitted
 * password on save means "leave the stored one alone".
 */

import { useEffect, useState } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { SaveStatus } from '../types/settings';

/** How the relay connection is encrypted. There is no plaintext mode. */
export type AlertEmailTLS = 'starttls' | 'tls';

/** The relay as the section edits it. */
export interface AlertEmailSettings {
  /** Bare host name; empty turns email off. */
  host: string;
  /** 0 means the TLS mode's submission port (587 or 465). */
  port: number;
  tls: AlertEmailTLS;
  /** Empty for a relay that admits by address. */
  username: string;
  /** Newly entered password; empty means leave the stored one alone. */
  password: string;
  /** Whether a password is already stored. */
  passwordSet: boolean;
  from: string;
  /** Recipients as typed: comma or newline separated. */
  to: string;
}

/** The alerts slice of GET /api/v1/settings. */
interface AlertsSettingsResponse {
  alerts?: {
    email?: {
      host?: string;
      port?: number;
      tls?: string;
      username?: string;
      passwordSet?: boolean;
      from?: string;
      to?: string[];
    };
  };
}

interface UseAlertEmailSettingsResult {
  email: AlertEmailSettings;
  setEmail: React.Dispatch<React.SetStateAction<AlertEmailSettings>>;
  status: SaveStatus;
  /** The server's reason when a save was refused, for display. */
  error: string;
  save: () => Promise<void>;
}

export const EMPTY_ALERT_EMAIL: AlertEmailSettings = {
  host: '',
  port: 0,
  tls: 'starttls',
  username: '',
  password: '',
  passwordSet: false,
  from: '',
  to: '',
};

/** Splits the recipients field into addresses, dropping empty entries. */
export function parseRecipients(to: string): string[] {
  return to
    .split(/[,\n]/)
    .map((addr) => addr.trim())
    .filter((addr) => addr !== '');
}

export function useAlertEmailSettings(): UseAlertEmailSettingsResult {
  const [email, setEmail] = useState<AlertEmailSettings>(EMPTY_ALERT_EMAIL);
  const [status, setStatus] = useState<SaveStatus>('idle');
  const [error, setError] = useState('');

  const load = async (): Promise<void> => {
    await api
      .get<AlertsSettingsResponse>('/api/v1/settings')
      .then((data) => {
        const stored = data.alerts?.email;
        setEmail({
          host: stored?.host ?? '',
          port: stored?.port ?? 0,
          // The server stores an empty mode as starttls.
          tls: stored?.tls === 'tls' ? 'tls' : 'starttls',
          username: stored?.username ?? '',
          password: '',
          passwordSet: stored?.passwordSet ?? false,
          from: stored?.from ?? '',
          to: (stored?.to ?? []).join(', '),
        });
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch alert email settings', err);
      });
  };

  const save = async (): Promise<void> => {
    setStatus('saving');
    setError('');
    const payload: {
      host: string;
      port: number;
      tls: AlertEmailTLS;
      username: string;
      from: string;
      to: string[];
      password?: string;
    } = {
      host: email.host.trim(),
      port: email.port,
      tls: email.tls,
      username: email.username.trim(),
      from: email.from.trim(),
      to: parseRecipients(email.to),
    };
    // An untouched password is omitted: the server keeps the stored one.
    if (email.password !== '') {
      payload.password = email.password;
    }
    await api
      .put('/api/v1/settings', { alerts: { email: payload } })
      .then(() => {
        setStatus('saved');
        // The password has left the browser; drop it so a later save does not
        // resend it. The server keeps a password only beside a username on a
        // configured relay.
        setEmail((current) => ({
          ...current,
          password: '',
          passwordSet:
            payload.host !== '' &&
            payload.username !== '' &&
            (current.password !== '' || current.passwordSet),
        }));
        setTimeout(() => setStatus('idle'), 2000);
      })
      .catch((err: unknown) => {
        setStatus('error');
        setError(err instanceof Error ? err.message : '');
      });
  };

  // Loaded on mount, like the other explicit-save sections: it has nothing to
  // synchronise with the drawer's auto-saving ones.
  useEffect((): void => {
    load().catch(() => undefined);
  }, [load]);

  return { email, setEmail, status, error, save };
}
