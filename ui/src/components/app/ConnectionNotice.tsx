/**
 * ConnectionNotice — the reconnect affordance at phone width.
 *
 * The rail's product mark carries the connection state and the reconnect
 * click, which is the whole shell at 1440px. At 390px the rail is behind the
 * menu button, so a dot nobody can see is not an affordance: this renders the
 * state and the reconnect action as a page notice at the top of the content,
 * which the top-of-shell decision explicitly allows (owner 2026-09-15).
 *
 * `lg:hidden` — above that breakpoint the rail is on screen and a second
 * indicator would just repeat it.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, radius, spacing } from '../../styles/theme';
import { Loader } from '../ui/Icons';

interface ConnectionNoticeProps {
  status: 'connecting' | 'connected' | 'disconnected' | 'error';
  onReconnect: () => void;
}

export function ConnectionNotice({
  status,
  onReconnect,
}: ConnectionNoticeProps): JSX.Element | null {
  const { t } = useTranslation();
  if (status === 'connected') return null;

  const connecting = status === 'connecting';
  return (
    <div
      data-testid="connection-notice"
      role="status"
      className={cn(
        'lg:hidden flex items-center justify-between',
        spacing.gap.tight,
        spacing.pad.sm,
        radius.lg,
        'mb-default border',
        connecting
          ? 'border-status-warning/40 bg-status-warning/10 text-status-warning'
          : 'border-status-error/40 bg-status-error/10 text-status-error',
      )}
    >
      <span className="body-small flex items-center gap-tight">
        {connecting ? (
          <Loader className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          <span aria-hidden="true">●</span>
        )}
        {connecting ? t('status.connecting') : t('status.disconnected')}
      </span>
      {connecting ? null : (
        <button
          type="button"
          onClick={onReconnect}
          className="body-small font-medium underline min-h-11 px-2"
        >
          {t('status.tapToReconnect')}
        </button>
      )}
    </div>
  );
}
