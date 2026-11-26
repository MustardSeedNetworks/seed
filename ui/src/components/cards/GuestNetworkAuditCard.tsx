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

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useCardStatusLabel } from '../../hooks/useCardStatusLabel';
import { useGuestNetworkAudit } from '../../hooks/useGuestNetworkAudit';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card, type CardStatus } from '../../shared/Card';
import { icon as iconTokens } from '../../styles/theme';
import { Shield } from '../ui/Icons';

export function GuestNetworkAuditCard(): JSX.Element | null {
  const { t } = useTranslation('cards');
  const labelFor = useCardStatusLabel();
  const { settings, runAudit, report, running, error } = useGuestNetworkAudit();

  // Hide entirely when the feature is disabled - matches the pattern of
  // other security cards that don't render until the user opts in.
  if (!settings.enabled) {
    return null;
  }

  const status: CardStatus = (() => {
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
      title={t('guestAudit.title')}
      icon={<Shield className={iconTokens.size.md} />}
      status={status}
      statusLabel={labelFor(status)}
      data-testid="card"
    >
      <div className="stack-sm">
        {!hasTargets ? (
          <p className="body-small text-text-muted">{t('guestAudit.noTargets')}</p>
        ) : (
          <>
            <p className="body-small text-text-muted">{t('guestAudit.description')}</p>
            <Button
              variant="primary"
              loading={running}
              onClick={(): void => {
                runAudit().catch(() => undefined);
              }}
            >
              {running
                ? t('guestAudit.running')
                : t('guestAudit.runButton', { count: settings.targets.length })}
            </Button>

            {error ? (
              <Alert tone="error" size="sm">
                {error}
              </Alert>
            ) : null}

            {report?.isolationFailed ? (
              <Alert tone="error" size="sm" title={t('guestAudit.criticalAlert')}>
                <p>
                  {t('guestAudit.criticalDetail', {
                    count: report.reachableTargets,
                    total: report.totalTargets,
                  })}
                </p>
                <ul className="mt-1 space-y-1">
                  {report.results
                    .filter((r) => r.reachable)
                    .map((r) => (
                      <li key={r.target.ip}>
                        <strong>{r.target.label || r.target.ip}</strong>
                        {r.target.label ? <> ({r.target.ip})</> : null}
                        {r.pingResponded ? <> · {t('guestAudit.ping')}</> : null}
                        {r.openPorts.length > 0
                          ? ` · ${t('guestAudit.openPorts')}: ${r.openPorts.join(', ')}`
                          : null}
                      </li>
                    ))}
                </ul>
              </Alert>
            ) : null}

            {report && !report.isolationFailed ? (
              <Alert tone="success" size="sm">
                {t('guestAudit.passed', { total: report.totalTargets })}
              </Alert>
            ) : null}
          </>
        )}
      </div>
    </Card>
  );
}
