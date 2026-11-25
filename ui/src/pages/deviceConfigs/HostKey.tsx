/**
 * HostKey — the SSH host key a target's backups are pinned to (UI-SEED-45).
 *
 * A backup never trusts a key on its own: the first one records the key the
 * device presented as pending and saves nothing, and an operator confirms it
 * here after checking it on the device. Forgetting a pinned key starts that
 * over, which is how a replaced device is accepted again.
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  confirmDeviceConfigHostKey,
  forgetDeviceConfigHostKey,
} from '../../hooks/useDeviceConfigs';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { ConfirmDialog } from '../../shared/ConfirmDialog';
import type { DeviceConfigTarget } from '../../types/deviceConfigs';

export function HostKey({
  target,
  onChanged,
}: {
  target: DeviceConfigTarget;
  onChanged: () => void;
}): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [forgetting, setForgetting] = useState(false);

  const act = async (request: () => Promise<void>): Promise<void> => {
    setBusy(true);
    setError(null);
    try {
      await request();
      onChanged();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const pinned = target.hostKeySha256;
  const pending = target.pendingHostKeySha256;

  return (
    <div data-testid="config-host-key" className="space-y-1.5">
      <p className="kicker">{t('deviceConfigs.hostKey.heading')}</p>
      {pinned ? (
        <>
          <p className="text-sm text-text-secondary">{t('deviceConfigs.hostKey.pinned')}</p>
          <code data-testid="config-host-key-pinned" className="figure break-all text-sm">
            {pinned}
          </code>
          <div>
            <Button
              data-testid="config-host-key-forget"
              variant="destructive"
              size="sm"
              loading={busy}
              onClick={(): void => setForgetting(true)}
            >
              {t('deviceConfigs.hostKey.forget')}
            </Button>
          </div>
        </>
      ) : pending ? (
        <>
          <Alert tone="warning">{t('deviceConfigs.hostKey.pending')}</Alert>
          <code data-testid="config-host-key-pending" className="figure break-all text-sm">
            {pending}
          </code>
          <div>
            <Button
              data-testid="config-host-key-confirm"
              size="sm"
              loading={busy}
              onClick={(): void => void act(() => confirmDeviceConfigHostKey(target.id, pending))}
            >
              {t('deviceConfigs.hostKey.confirm')}
            </Button>
          </div>
        </>
      ) : (
        <p data-testid="config-host-key-none" className="text-sm text-text-secondary">
          {t('deviceConfigs.hostKey.none')}
        </p>
      )}
      {error ? <Alert tone="error">{error}</Alert> : null}
      <ConfirmDialog
        open={forgetting}
        title={t('deviceConfigs.hostKey.forgetTitle', { name: target.name })}
        message={t('deviceConfigs.hostKey.forgetMessage')}
        confirmLabel={t('deviceConfigs.hostKey.forget')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={(): void => {
          setForgetting(false);
          void act(() => forgetDeviceConfigHostKey(target.id));
        }}
        onCancel={(): void => setForgetting(false)}
      />
    </div>
  );
}
