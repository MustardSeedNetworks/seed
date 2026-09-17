/**
 * DiscoveryEmptyState — why the discovered-device list is empty (seed#2674).
 *
 * An empty list has three causes and the operator needs different words for
 * each. The card used to render one line for all of them ("No devices
 * discovered. Click Scan"), so a fresh install that had simply not finished its
 * first sweep read as an empty network, and an install with discovery switched
 * off read the same way again — the owner's report.
 *
 * The discriminator is `lastScan`: the backend leaves it at Go's zero time
 * until a sweep has completed, which is pinned in devices_status_test.go.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { button, cn, spacing } from '../../styles/theme';
import { RefreshCw } from '../ui/icons';
import type { DiscoveryStatus } from './networkDiscoveryCardTypes';

export type DiscoveryPhase = 'off' | 'discovering' | 'empty';

/**
 * hasSwept reports whether `lastScan` names a sweep that actually ran.
 *
 * Go's zero time serialises as year 1, and a status that predates the first
 * sweep carries exactly that. Parsing rather than string-matching the zero
 * literal keeps this honest if the encoding ever changes.
 */
export function hasSwept(lastScan: string | undefined): boolean {
  if (!lastScan) {
    return false;
  }
  const at = Date.parse(lastScan);
  return !Number.isNaN(at) && new Date(at).getUTCFullYear() > 1;
}

/** discoveryPhase names why the device list is empty. */
export function discoveryPhase(
  enabled: boolean,
  status: DiscoveryStatus | null | undefined,
): DiscoveryPhase {
  if (!enabled) {
    return 'off';
  }
  if (!status || status.scanning || !hasSwept(status.lastScan)) {
    return 'discovering';
  }
  return 'empty';
}

interface DiscoveryEmptyStateProps {
  phase: DiscoveryPhase;
  /** Opens the settings drawer. Omitted where the caller has no drawer. */
  onOpenSettings?: () => void;
}

export function DiscoveryEmptyState({
  phase,
  onOpenSettings,
}: DiscoveryEmptyStateProps): JSX.Element {
  const { t } = useTranslation('cards');

  // "Discovering" is a state that passes on its own; sending the operator to
  // the options for it would be advice to change something that is working.
  const offerOptions = phase !== 'discovering' && Boolean(onOpenSettings);

  const testId =
    phase === 'off'
      ? 'discovery-empty-off'
      : phase === 'discovering'
        ? 'discovery-empty-discovering'
        : 'discovery-empty-none';

  const message =
    phase === 'off'
      ? t('discovery.emptyOff')
      : phase === 'discovering'
        ? t('discovery.emptyDiscovering')
        : t('discovery.emptyNone');

  return (
    <div className={cn('text-center stack-sm', spacing.pad.default)} data-testid={testId}>
      <p className="body-small text-text-secondary flex-center gap-compact">
        {phase === 'discovering' ? (
          <RefreshCw className="w-4 h-4 animate-spin" aria-hidden="true" />
        ) : null}
        {message}
      </p>
      {offerOptions ? (
        <button
          type="button"
          onClick={onOpenSettings}
          className={cn(button.base, button.variant.secondary, button.size.sm)}
          data-testid="discovery-open-options"
        >
          {t('discovery.openOptions')}
        </button>
      ) : null}
    </div>
  );
}
