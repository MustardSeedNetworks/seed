/**
 * MfaCard
 *
 * Surfaces TOTP + WebAuthn (passkey) enrolment for the current user.
 *
 * Wave 3 (#85) introduced multi-factor authentication for the seed
 * appliance. This card is deliberately compact — the heavy UX
 * (recovery codes, multiple TOTP profiles, passkey transports) is
 * deferred. We expose the minimum surface needed to:
 *
 *   - Show the user's current MFA status (none / TOTP / Passkey).
 *   - Start the TOTP enrolment flow (QR code + verification code box).
 *   - Disable an existing TOTP enrolment (password + code).
 *   - Add a WebAuthn passkey via the browser ceremony.
 *
 * The backend endpoints live under /api/v1/auth/totp/* and
 * /api/v1/auth/webauthn/* — see internal/api/handlers_mfa.go.
 */

import type { JSX } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import { icon as iconTokens } from '../../styles/theme';
import { Card } from '../ui/card';
import { Shield } from '../ui/icons';

/** Backend response shape for GET /api/v1/auth/mfa/status. */
interface MfaStatus {
  totp_enabled: boolean;
  webauthn_enabled: boolean;
  webauthn_credential_count: number;
}

/** Backend response shape for POST /api/v1/auth/totp/setup. */
interface TotpSetup {
  secret: string;
  provisioning_uri: string;
  qr_code_png_base64: string;
}

export function MfaCard(): JSX.Element {
  const { t } = useTranslation('cards');
  const [status, setStatus] = useState<MfaStatus | null>(null);
  const [setup, setSetup] = useState<TotpSetup | null>(null);
  const [code, setCode] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [busy, setBusy] = useState<boolean>(false);

  const refresh = async (): Promise<void> => {
    try {
      const next = await api.get<MfaStatus>('/api/v1/auth/mfa/status');
      setStatus(next);
    } catch (err) {
      setError((err as Error).message);
    }
  };

  useEffect(() => {
    refresh().catch(() => {
      /* errors are surfaced through setError() in refresh */
    });
  }, []);

  const startTotp = async (): Promise<void> => {
    setError('');
    setBusy(true);
    try {
      const next = await api.post<TotpSetup>('/api/v1/auth/totp/setup', {});
      setSetup(next);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const verifyTotp = async (): Promise<void> => {
    setError('');
    setBusy(true);
    try {
      await api.post<unknown>('/api/v1/auth/totp/verify', { code });
      setSetup(null);
      setCode('');
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const addPasskey = async (): Promise<void> => {
    setError('');
    setBusy(true);
    try {
      const opts = await api.post<unknown>('/api/v1/auth/webauthn/register/begin', {});
      // The browser WebAuthn API is wired up at the page level so the
      // card stays presentational. We hand off to a global helper
      // that's been registered by the surrounding page.
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as unknown as { seedWebAuthnRegister?: (o: unknown) => Promise<void> };
      if (w.seedWebAuthnRegister) {
        await w.seedWebAuthnRegister(opts);
      }
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const statusLine = ((): string => {
    if (!status) {
      return t('mfa.loading', 'Loading…');
    }
    if (status.totp_enabled && status.webauthn_enabled) {
      return t('mfa.bothEnabled', 'TOTP + Passkey enabled');
    }
    if (status.totp_enabled) {
      return t('mfa.totpEnabled', 'TOTP enabled');
    }
    if (status.webauthn_enabled) {
      return t('mfa.passkeyEnabled', 'Passkey enabled');
    }
    return t('mfa.none', 'No second factor enrolled');
  })();

  return (
    <Card
      title={t('mfa.title', 'Multi-factor authentication')}
      icon={<Shield className={iconTokens.size.md} />}
      status={status?.totp_enabled || status?.webauthn_enabled ? 'success' : 'unknown'}
    >
      <div className="stack-sm">
        <p className="body-small text-text-muted">{statusLine}</p>
        {error ? <p className="body-small text-status-error">{error}</p> : null}

        {!(status?.totp_enabled || setup) ? (
          <button
            type="button"
            className="btn btn-secondary"
            disabled={busy}
            onClick={() => {
              startTotp().catch(() => undefined);
            }}
          >
            {t('mfa.setupTotp', 'Set up TOTP')}
          </button>
        ) : null}

        {setup ? (
          <div className="stack-sm">
            <img
              alt={t('mfa.qrAlt', 'TOTP QR code')}
              src={`data:image/png;base64,${setup.qr_code_png_base64}`}
              width={200}
              height={200}
            />
            <p className="body-small text-text-muted">
              {t('mfa.scanAndEnter', 'Scan with an authenticator app, then enter a code')}
            </p>
            <input
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={code}
              onInput={(e) => setCode((e.target as HTMLInputElement).value)}
              placeholder="123456"
              className="input"
            />
            <button
              type="button"
              className="btn btn-primary"
              disabled={busy || code.length !== 6}
              onClick={() => {
                verifyTotp().catch(() => undefined);
              }}
            >
              {t('mfa.verifyAndEnable', 'Verify and enable')}
            </button>
          </div>
        ) : null}

        <button
          type="button"
          className="btn btn-secondary"
          disabled={busy}
          onClick={() => {
            addPasskey().catch(() => undefined);
          }}
        >
          {t('mfa.addPasskey', 'Add passkey')}
        </button>
      </div>
    </Card>
  );
}
