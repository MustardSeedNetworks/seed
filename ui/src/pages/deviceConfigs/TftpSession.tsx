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
import { useRole } from '../../contexts/RoleContext';
import { useProfileInterfaces } from '../../contexts/useProfileInterfaces';
import { useTftpSession } from '../../hooks/useTftpSession';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card } from '../../shared/Card';
import { Checkbox } from '../../shared/Checkbox';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import type { TftpSessionRunning, TftpStartRequest } from '../../types/tftp';
import { DetailFacts } from '../../ui/ListDetail';

export function TftpSession(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { isAdmin } = useRole();
  const { session, error, busy, start, stop } = useTftpSession(isAdmin);

  return (
    <div data-testid="tftp-session">
      <Card data-testid="card" title={t('deviceConfigs.tftp.title')}>
        <div className="space-y-3">
          {error ? <Alert tone="error">{error}</Alert> : null}
          {!isAdmin ? (
            <p data-testid="tftp-admin-only" className="text-sm text-text-secondary">
              {t('deviceConfigs.tftp.adminOnly')}
            </p>
          ) : session === null ? (
            <p className="text-sm text-text-secondary">{t('common:status.loading')}</p>
          ) : session.running ? (
            <RunningSession session={session} busy={busy} onStop={stop} />
          ) : (
            <StartForm busy={busy} onStart={start} />
          )}
        </div>
      </Card>
    </div>
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
    <div className="space-y-3">
      <p data-testid="tftp-running" className="text-sm text-text-secondary">
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
        <Button data-testid="tftp-stop" loading={busy} onClick={(): void => void onStop()}>
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
    <form onSubmit={submit} className="space-y-3">
      <p className="text-sm text-text-secondary">{t('deviceConfigs.tftp.stopped')}</p>
      <FormField label={t('deviceConfigs.tftp.labelInterface')} required>
        {(control) => (
          <>
            <Input
              {...control}
              list={`${id}-ifaces`}
              value={iface}
              onChange={(e): void => setIface(e.target.value)}
              required
              data-testid="tftp-interface"
            />
            <datalist id={`${id}-ifaces`}>
              {known.map((name) => (
                <option key={name} value={name} />
              ))}
            </datalist>
          </>
        )}
      </FormField>
      <Checkbox
        data-testid="tftp-allow-upload"
        checked={allowUpload}
        onChange={setAllowUpload}
        label={t('deviceConfigs.tftp.allowUpload')}
        description={t('deviceConfigs.tftp.allowUploadWarning')}
      />
      <div>
        <Button type="submit" data-testid="tftp-start" loading={busy} disabled={!iface.trim()}>
          {t('deviceConfigs.tftp.start')}
        </Button>
      </div>
    </form>
  );
}
