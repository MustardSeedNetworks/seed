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

import type { FormEvent, JSX } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError, api } from '../../api';
import { isPasskeySupported, registerPasskey } from '../../lib/webauthn';
import { icon as iconTokens } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { Shield } from '../ui/Icons';
import { Input } from '../ui/Input';

/** Backend response shape for GET /api/v1/auth/mfa/status. */
interface MfaStatus {
  totpEnabled: boolean;
  webauthnEnabled: boolean;
  webauthnCredentialCount: number;
}

/** Backend response shape for POST /api/v1/auth/totp/setup. */
interface TotpSetup {
  secret: string;
  provisioningUri: string;
  qrCodePngBase64: string;
}

export function MfaCard(): JSX.Element {
  const { t } = useTranslation(['cards', 'common']);
  const [status, setStatus] = useState<MfaStatus | null>(null);
  const [setup, setSetup] = useState<TotpSetup | null>(null);
  const [code, setCode] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [busy, setBusy] = useState<boolean>(false);
  const [disabling, setDisabling] = useState<boolean>(false);
  const [password, setPassword] = useState<string>('');
  const [disableCode, setDisableCode] = useState<string>('');

  // A wrong factor names itself with a code, so the reason reads in the UI's
  // language rather than the browser's Accept-Language the server answered in.
  const failureText = (err: unknown): string => {
    if (err instanceof ApiError) {
      switch (err.code) {
        case 'INVALID_PASSWORD':
          return t('mfa.wrongPassword');
        case 'INVALID_MFA_CODE':
          return t('mfa.wrongCode');
        case 'RATE_LIMIT_EXCEEDED':
          return t('mfa.tooManyAttempts');
        default:
          break;
      }
    }
    return (err as Error).message;
  };

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
      setError(failureText(err));
    } finally {
      setBusy(false);
    }
  };

  const closeDisable = (): void => {
    setDisabling(false);
    setPassword('');
    setDisableCode('');
  };

  const disableTotp = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();
    setError('');
    setBusy(true);
    try {
      await api.post<unknown>('/api/v1/auth/totp/disable', { password, code: disableCode });
      closeDisable();
      await refresh();
    } catch (err) {
      // The factor stays enrolled, so keep the form open for another try.
      setError(failureText(err));
      setDisableCode('');
    } finally {
      setBusy(false);
    }
  };

  const addPasskey = async (): Promise<void> => {
    setError('');
    setBusy(true);
    try {
      // registerPasskey runs the whole ceremony: begin, navigator.credentials
      // .create, finish. This used to call begin and then look for a
      // window.seedWebAuthnRegister helper that was defined nowhere, so the
      // guard was always false, nothing was enrolled and the card still
      // reported success.
      await registerPasskey();
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const statusLine = ((): string => {
    if (!status) {
      return t('mfa.loading');
    }
    if (status.totpEnabled && status.webauthnEnabled) {
      return t('mfa.bothEnabled');
    }
    if (status.totpEnabled) {
      return t('mfa.totpEnabled');
    }
    if (status.webauthnEnabled) {
      return t('mfa.passkeyEnabled');
    }
    return t('mfa.none');
  })();

  return (
    <Card
      title={t('mfa.title')}
      icon={<Shield className={iconTokens.size.md} />}
      status={status?.totpEnabled || status?.webauthnEnabled ? 'success' : 'unknown'}
    >
      <div className="stack-sm">
        <p className="body-small text-text-muted">{statusLine}</p>
        {error ? (
          <p className="body-small text-status-error" data-testid="mfa-error">
            {error}
          </p>
        ) : null}

        {setup ? (
          <div className="flex flex-col items-start gap-compact">
            <img
              alt={t('mfa.qrAlt')}
              src={`data:image/png;base64,${setup.qrCodePngBase64}`}
              width={200}
              height={200}
            />
            <Input
              id="mfa-totp-code"
              label={t('mfa.scanAndEnter')}
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={code}
              onChange={(event) => setCode(event.target.value)}
              placeholder="123456"
              data-testid="mfa-totp-code"
            />
            <Button
              size="sm"
              disabled={busy || code.length !== 6}
              data-testid="mfa-verify-totp"
              onClick={() => {
                verifyTotp().catch(() => undefined);
              }}
            >
              {t('mfa.verifyAndEnable')}
            </Button>
          </div>
        ) : null}

        {disabling ? (
          <form
            className="flex flex-col items-start gap-compact"
            data-testid="mfa-disable-form"
            onSubmit={(event) => {
              disableTotp(event).catch(() => undefined);
            }}
          >
            <p className="body-small text-text-muted">{t('mfa.disablePrompt')}</p>
            <Input
              id="mfa-disable-password"
              label={t('mfa.password')}
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              data-testid="mfa-disable-password"
            />
            <Input
              id="mfa-disable-code"
              label={t('mfa.currentCode')}
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={disableCode}
              onChange={(event) => setDisableCode(event.target.value)}
              placeholder="123456"
              data-testid="mfa-disable-code"
            />
            <div className="flex flex-wrap gap-compact">
              <Button
                type="submit"
                tone="red"
                size="sm"
                disabled={busy || password === '' || disableCode.length !== 6}
                data-testid="mfa-disable-confirm"
              >
                {t('mfa.disableTotp')}
              </Button>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={busy}
                data-testid="mfa-disable-cancel"
                onClick={() => {
                  closeDisable();
                  setError('');
                }}
              >
                {t('common:buttons.cancel')}
              </Button>
            </div>
          </form>
        ) : null}

        {/*
          Both enrolment choices sit in one flex row. `Button` wraps itself in a
          Tooltip span with `display: contents`, which generates no box, so the
          `stack-sm` margins that used to separate these were dropped on the
          floor and the two unstyled buttons read as one word (#2641). A flex
          `gap-compact` spaces the buttons themselves and survives that wrapper.
        */}
        <div className="flex flex-wrap gap-compact">
          {!(status?.totpEnabled || setup) ? (
            <Button
              variant="secondary"
              size="sm"
              disabled={busy}
              data-testid="mfa-setup-totp"
              onClick={() => {
                startTotp().catch(() => undefined);
              }}
            >
              {t('mfa.setupTotp')}
            </Button>
          ) : null}

          {status?.totpEnabled && !disabling ? (
            <Button
              variant="outline"
              tone="red"
              size="sm"
              disabled={busy}
              data-testid="mfa-disable-totp"
              onClick={() => {
                setError('');
                setDisabling(true);
              }}
            >
              {t('mfa.disableTotp')}
            </Button>
          ) : null}

          <Button
            variant="secondary"
            size="sm"
            // Button renders its own Tooltip from `title`, which keeps an
            // unavailable action in the tab order so the reason can be read. A
            // browser without WebAuthn cannot enrol, and an enabled button that
            // silently does nothing is what this card shipped with.
            title={isPasskeySupported() ? undefined : t('mfa.passkeyUnsupported')}
            disabled={busy || !isPasskeySupported()}
            data-testid="mfa-add-passkey"
            onClick={() => {
              addPasskey().catch(() => undefined);
            }}
          >
            {t('mfa.addPasskey')}
          </Button>
        </div>
      </div>
    </Card>
  );
}
