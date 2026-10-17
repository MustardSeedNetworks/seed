/**
 * TftpSession — start, watch and stop the TFTP server (P-D4, UI-SEED-44).
 *
 * Devices push and pull configuration and firmware over TFTP, which has no
 * authentication: whoever starts a session decides what every host on that
 * segment may read and, with uploads on, write. So the route is admin-only
 * for every method, and an operator is told who can start one instead of
 * being sent to a 403.
 */

import { type FormEvent, type JSX, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../components/ui/Button';
import { useRole } from '../../contexts/RoleContext';
import { useProfileInterfaces } from '../../contexts/useProfileInterfaces';
import { useTftpSession } from '../../hooks/useTftpSession';
import type { TftpSessionRunning, TftpStartRequest } from '../../types/tftp';
import { DetailFacts } from '../../ui/ListDetail';

export function TftpSession(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { isAdmin } = useRole();
  const { session, error, busy, start, stop } = useTftpSession(isAdmin);
  const headingId = useId();

  return (
    <section
      data-testid="tftp-session"
      aria-labelledby={headingId}
      className="stack-sm rounded-2xl border border-surface-border pad-lg"
    >
      <h2 id={headingId} className="text-base font-semibold text-text-primary">
        {t('deviceConfigs.tftp.title')}
      </h2>
      {error ? (
        <p role="alert" className="body-small text-status-error-strong">
          {error}
        </p>
      ) : null}
      {!isAdmin ? (
        <p data-testid="tftp-admin-only" className="body-small">
          {t('deviceConfigs.tftp.adminOnly')}
        </p>
      ) : session === null ? (
        <p className="body-small">{t('common:status.loading')}</p>
      ) : session.running ? (
        <RunningSession session={session} busy={busy} onStop={stop} />
      ) : (
        <StartForm busy={busy} onStart={start} />
      )}
    </section>
  );
}

function RunningSession({
  session,
  busy,
  onStop,
}: {
  session: TftpSessionRunning;
  busy: boolean;
  onStop: () => Promise<void>;
}): JSX.Element {
  const { t } = useTranslation(['pages']);
  return (
    <div className="stack-sm">
      <p data-testid="tftp-running" className="body-small">
        {t('deviceConfigs.tftp.running', {
          minutes: Math.round(session.idleTimeoutSeconds / 60),
          until: new Date(session.expiresAt).toLocaleTimeString(),
        })}
      </p>
      <DetailFacts
        items={[
          {
            label: t('deviceConfigs.tftp.labelListening'),
            testId: 'tftp-listening',
            value: `${session.interface} · ${session.address}:${session.port}`,
          },
          {
            label: t('deviceConfigs.tftp.labelServes'),
            value: session.directory,
          },
          {
            label: t('deviceConfigs.tftp.labelUploads'),
            testId: 'tftp-uploads',
            prose: true,
            value: session.allowUpload
              ? t('deviceConfigs.tftp.uploadsInto', { directory: session.uploadDirectory })
              : t('deviceConfigs.tftp.uploadsOff'),
          },
          {
            label: t('deviceConfigs.tftp.labelAllowFrom'),
            prose: !session.allowFrom?.length,
            value: session.allowFrom?.length
              ? session.allowFrom.join(', ')
              : t('deviceConfigs.tftp.anyHost'),
          },
          {
            label: t('deviceConfigs.tftp.labelStarted'),
            value: new Date(session.startedAt).toLocaleString(),
          },
          {
            label: t('deviceConfigs.tftp.labelTransfers'),
            testId: 'tftp-transfers',
            value: session.activeTransfers,
          },
        ]}
      />
      <div>
        <Button
          data-testid="tftp-stop"
          variant="outline"
          tone="red"
          loading={busy}
          onClick={(): void => void onStop()}
        >
          {t('deviceConfigs.tftp.stop')}
        </Button>
      </div>
    </div>
  );
}

function StartForm({
  busy,
  onStart,
}: {
  busy: boolean;
  onStart: (request: TftpStartRequest) => Promise<void>;
}): JSX.Element {
  const { t } = useTranslation(['pages']);
  const id = useId();
  const { getAllEthernetInterfaces, getAllWifiInterfaces, getEthernetInterface } =
    useProfileInterfaces();
  const known = [...getAllEthernetInterfaces(), ...getAllWifiInterfaces()].map((i) => i.name);
  const [iface, setIface] = useState(getEthernetInterface()?.name ?? '');
  const [allowUpload, setAllowUpload] = useState(false);

  const submit = (e: FormEvent): void => {
    e.preventDefault();
    void onStart({ interface: iface.trim(), allowUpload });
  };

  return (
    <form onSubmit={submit} className="stack-sm">
      <p className="body-small">{t('deviceConfigs.tftp.stopped')}</p>
      <div className="stack-xs">
        <label htmlFor={`${id}-iface`} className="caption">
          {t('deviceConfigs.tftp.labelInterface')}
        </label>
        <input
          id={`${id}-iface`}
          list={`${id}-ifaces`}
          value={iface}
          onChange={(e): void => setIface(e.target.value)}
          required
          data-testid="tftp-interface"
          className="w-full max-w-xs rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary focus:border-brand-primary focus:outline-none"
        />
        <datalist id={`${id}-ifaces`}>
          {known.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
      </div>
      <label className="flex items-start gap-compact text-sm text-text-secondary">
        <input
          type="checkbox"
          checked={allowUpload}
          onChange={(e): void => setAllowUpload(e.target.checked)}
          data-testid="tftp-allow-upload"
          className="mt-tight"
        />
        <span>
          {t('deviceConfigs.tftp.allowUpload')}
          <span className="block text-xs text-text-muted">
            {t('deviceConfigs.tftp.allowUploadWarning')}
          </span>
        </span>
      </label>
      <div>
        <Button type="submit" data-testid="tftp-start" loading={busy} disabled={!iface.trim()}>
          {t('deviceConfigs.tftp.start')}
        </Button>
      </div>
    </form>
  );
}
