/**
 * GuestNetworkAuditCard
 *
 * Surfaces the on-demand Guest Network isolation audit (#397).
 *
 * The user connects the appliance to a guest network and clicks "Run audit".
 * The backend probes each configured sensitive internal IP (EMR, PACS, etc.)
 * via ICMP + TCP connect on a default port set. If any target is reachable,
 * the card flips to a Critical state and lists the broken paths.
 *
 * Configuration of the target list lives in Settings -> Security.
 */

import { useTranslation } from 'react-i18next';
import { useGuestNetworkAudit } from '../../hooks/useGuestNetworkAudit';
import { button, cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import { Card, type Status } from '../ui/card';
import { AlertTriangle, CheckCircle, Shield } from '../ui/icons';

export function GuestNetworkAuditCard(): JSX.Element | null {
  const { t } = useTranslation('cards');
  const { settings, runAudit, report, running, error } = useGuestNetworkAudit();

  // Hide entirely when the feature is disabled - matches the pattern of
  // other security cards that don't render until the user opts in.
  if (!settings.enabled) {
    return null;
  }

  const status: Status = (() => {
    if (running) {
      return 'loading';
    }
    if (error) {
      return 'error';
    }
    if (!report) {
      return 'unknown';
    }
    return report.isolationFailed ? 'error' : 'success';
  })();

  const hasTargets = settings.targets.length > 0;

  return (
    <Card
      title={t('guestAudit.title', 'Guest Network Audit')}
      icon={<Shield class={iconTokens.size.md} />}
      status={status}
    >
      <div class="stack-sm">
        {!hasTargets ? (
          <p class="body-small text-text-muted">
            {t(
              'guestAudit.noTargets',
              'Add sensitive internal IP addresses (EMR, PACS, etc.) in Settings → Security to enable this audit.',
            )}
          </p>
        ) : (
          <>
            <p class="body-small text-text-muted">
              {t(
                'guestAudit.description',
                'Probe configured internal hosts to confirm guest-network isolation. Connect to the guest network before running.',
              )}
            </p>
            <button
              type="button"
              onClick={(): void => {
                runAudit().catch(() => undefined);
              }}
              disabled={running}
              class={cn(
                button.size.md,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'font-medium hover:bg-brand-primary/90 disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              {running
                ? t('guestAudit.running', 'Running audit...')
                : t('guestAudit.runButton', `Run audit (${settings.targets.length} targets)`)}
            </button>

            {error ? (
              <div class={cn(spacing.pad.sm, 'bg-status-error/10', radius.md)} role="alert">
                <span class="body-small text-status-error">{error}</span>
              </div>
            ) : null}

            {report?.isolationFailed ? (
              <div
                class={cn(
                  spacing.pad.sm,
                  'bg-status-error/10 border border-status-error',
                  radius.md,
                  'stack-xs',
                )}
                role="alert"
              >
                <div class={cn('flex items-center', spacing.gap.compact)}>
                  <AlertTriangle class={cn(iconTokens.size.sm, 'text-status-error')} />
                  <span class="body-small font-semibold text-status-error">
                    {t(
                      'guestAudit.criticalAlert',
                      'Critical: Guest network isolation is not configured correctly.',
                    )}
                  </span>
                </div>
                <p class="caption text-text-secondary">
                  {t(
                    'guestAudit.criticalDetail',
                    '{{count}} of {{total}} internal hosts are reachable from the guest network.',
                    { count: report.reachableTargets, total: report.totalTargets },
                  )}
                </p>
                <ul class="stack-xs caption">
                  {report.results
                    .filter((r) => r.reachable)
                    .map((r) => (
                      <li key={r.target.ip}>
                        <strong>{r.target.label || r.target.ip}</strong>
                        {r.target.label ? <> ({r.target.ip})</> : null}
                        {r.pingResponded ? <> · {t('guestAudit.ping', 'ping')}</> : null}
                        {r.openPorts.length > 0
                          ? ` · ${t('guestAudit.openPorts', 'open ports')}: ${r.openPorts.join(', ')}`
                          : null}
                      </li>
                    ))}
                </ul>
              </div>
            ) : null}

            {report && !report.isolationFailed ? (
              <div
                class={cn(
                  spacing.pad.sm,
                  'bg-status-success/10 border border-status-success',
                  radius.md,
                )}
                role="status"
              >
                <div class={cn('flex items-center', spacing.gap.compact)}>
                  <CheckCircle class={cn(iconTokens.size.sm, 'text-status-success')} />
                  <span class="body-small font-medium text-status-success">
                    {t(
                      'guestAudit.passed',
                      'Isolation verified - none of the {{total}} internal hosts are reachable.',
                      { total: report.totalTargets },
                    )}
                  </span>
                </div>
              </div>
            ) : null}
          </>
        )}
      </div>
    </Card>
  );
}
