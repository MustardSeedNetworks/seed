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
import { Button } from '../../components/ui/Button';
import {
  confirmDeviceConfigHostKey,
  forgetDeviceConfigHostKey,
} from '../../hooks/useDeviceConfigs';
import type { DeviceConfigTarget } from '../../types/deviceConfigs';

export function HostKey({
  target,
  onChanged,
}: {
  target: DeviceConfigTarget;
  onChanged: () => void;
}): JSX.Element {
  const { t } = useTranslation(['pages']);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
    <div data-testid="config-host-key" className="stack-xs">
      <p className="caption">{t('deviceConfigs.hostKey.heading')}</p>
      {pinned ? (
        <>
          <p className="body-small">{t('deviceConfigs.hostKey.pinned')}</p>
          <code data-testid="config-host-key-pinned" className="figure break-all text-sm">
            {pinned}
          </code>
          <div>
            <Button
              data-testid="config-host-key-forget"
              variant="outline"
              tone="red"
              size="sm"
              loading={busy}
              onClick={(): void => void act(() => forgetDeviceConfigHostKey(target.id))}
            >
              {t('deviceConfigs.hostKey.forget')}
            </Button>
          </div>
        </>
      ) : pending ? (
        <>
          <p className="body-small text-status-warning-strong">
            {t('deviceConfigs.hostKey.pending')}
          </p>
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
        <p data-testid="config-host-key-none" className="body-small">
          {t('deviceConfigs.hostKey.none')}
        </p>
      )}
      {error ? (
        <p role="alert" className="body-small text-status-error-strong">
          {error}
        </p>
      ) : null}
    </div>
  );
}
