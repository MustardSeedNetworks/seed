import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, radius, spacing } from '../../styles/theme';

/**
 * Shown at every width while the daemon does not answer /__version. The
 * poller keeps retrying on its own, so there is no action to offer.
 */
export function DaemonBanner(): JSX.Element {
  const { t } = useTranslation();
  return (
    <div
      data-testid="daemon-banner"
      role="alert"
      className={cn(
        spacing.pad.sm,
        radius.lg,
        'mb-default border border-status-error/40 bg-status-error/10 body-small text-status-error-strong',
      )}
    >
      {t('status.daemonUnreachable')}
    </div>
  );
}
