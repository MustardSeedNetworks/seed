/**
 * Help modal advanced sections: health checks, security, troubleshooting.
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { cn, radius, spacing } from '../../styles/theme';

function _healthChecksSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const commonIssues = t('content.healthChecks.commonIssues', { returnObjects: true }) as {
    title: string;
    timeout: string;
    highLatency: string;
    packetLoss: string;
    connectionRefused: string;
  };
  return (
    <helpContentSection title={t('sections.healthChecks')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.healthChecks.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.healthChecks.terms.pingTest.term'),
            description: t('content.healthChecks.terms.pingTest.description'),
          },
          {
            term: t('content.healthChecks.terms.tcpTest.term'),
            description: t('content.healthChecks.terms.tcpTest.description'),
          },
          {
            term: t('content.healthChecks.terms.httpTest.term'),
            description: t('content.healthChecks.terms.httpTest.description'),
          },
          {
            term: t('content.healthChecks.terms.customTargets.term'),
            description: t('content.healthChecks.terms.customTargets.description'),
          },
          {
            term: t('content.healthChecks.terms.thresholds.term'),
            description: t('content.healthChecks.terms.thresholds.description'),
          },
        ]}
      />
      <div
        className={cn(
          spacing.margin.top.content,
          'bg-status-info/10 border border-status-info/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          {commonIssues.title}
        </h4>
        <ul
          className={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-disc',
          )}
        >
          <li>
            <strong>Timeout:</strong> {commonIssues.timeout}
          </li>
          <li>
            <strong>High Latency:</strong> {commonIssues.highLatency}
          </li>
          <li>
            <strong>Packet Loss:</strong> {commonIssues.packetLoss}
          </li>
          <li>
            <strong>Connection Refused:</strong> {commonIssues.connectionRefused}
          </li>
        </ul>
      </div>
    </helpContentSection>
  );
}

function _securitySection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const recovery = t('content.security.passwordRecovery', { returnObjects: true }) as {
    title: string;
    description: string;
    steps: string[];
    note: string;
  };
  const portDetails = t('content.security.portScanDetails', { returnObjects: true }) as {
    title: string;
    description: string;
    levels: Record<string, string>;
    commonPorts: Record<string, string>;
  };
  return (
    <helpContentSection title={t('sections.security')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.security.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.security.terms.portScan.term'),
            description: t('content.security.terms.portScan.description'),
          },
          {
            term: t('content.security.terms.vulnScan.term'),
            description: t('content.security.terms.vulnScan.description'),
          },
          {
            term: t('content.security.terms.devicePosture.term'),
            description: t('content.security.terms.devicePosture.description'),
          },
          {
            term: t('content.security.terms.rogueDhcp.term'),
            description: t('content.security.terms.rogueDhcp.description'),
          },
        ]}
      />
      {/* Password Recovery Section */}
      <div
        className={cn(
          spacing.margin.top.section,
          'bg-status-warning/10 border border-status-warning/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {recovery.title}
        </h4>
        <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
          {recovery.description}
        </p>
        <ol
          className={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-decimal',
          )}
        >
          {recovery.steps.map((step) => (
            <li
              key={step}
              className={
                step.startsWith('User mode:') || step.startsWith('System mode:')
                  ? 'font-mono text-xs bg-surface-base px-2 py-1 rounded'
                  : ''
              }
            >
              {step}
            </li>
          ))}
        </ol>
        <p className={cn('caption text-status-warning', spacing.margin.top.content)}>
          <strong>Note:</strong> {recovery.note}
        </p>
      </div>
      {/* Port Scan Details */}
      <div className={cn(spacing.margin.top.section)}>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {portDetails.title}
        </h4>
        <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
          {portDetails.description}
        </p>
        <div className="grid grid-cols-2 gap-2 body-small">
          {Object.entries(portDetails.levels).map(([level, desc]) => (
            <div key={level} className={cn('border-l-2 border-surface-border', spacing.pad.sm)}>
              <dt className="font-semibold text-text-primary capitalize">{level}</dt>
              <dd className="text-text-secondary">{desc}</dd>
            </div>
          ))}
        </div>
      </div>
      {/* Common Ports Reference */}
      <div className={cn(spacing.margin.top.content)}>
        <h5 className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          Common Ports Reference
        </h5>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2 body-small font-mono">
          {Object.entries(portDetails.commonPorts).map(([port, desc]) => (
            <div key={port} className="flex items-baseline gap-2">
              <span className="text-brand-primary font-bold">{port}</span>
              <span className="text-text-muted">{desc}</span>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _troubleshootingSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const categories = t('content.troubleshooting.categories', { returnObjects: true }) as Record<
    string,
    {
      title: string;
      [key: string]: unknown;
    }
  >;

  return (
    <helpContentSection title={t('sections.troubleshooting')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.troubleshooting.description')}
      </p>
      {/* Link Issues */}
      <troubleshootingCategory
        title={categories.linkIssues.title}
        issues={[
          {
            symptom: (
              categories.linkIssues as {
                noCarrier: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).noCarrier.symptom,
            causes: (
              categories.linkIssues as {
                noCarrier: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).noCarrier.causes,
            solutions: (
              categories.linkIssues as {
                noCarrier: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).noCarrier.solutions,
          },
          {
            symptom: (
              categories.linkIssues as {
                slowSpeed: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowSpeed.symptom,
            causes: (
              categories.linkIssues as {
                slowSpeed: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowSpeed.causes,
            solutions: (
              categories.linkIssues as {
                slowSpeed: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowSpeed.solutions,
          },
        ]}
      />
      {/* Cable Issues */}
      <troubleshootingCategory
        title={categories.cableIssues.title}
        issues={[
          {
            symptom: (
              categories.cableIssues as {
                open: { symptom: string; meaning: string; solutions: string[] };
              }
            ).open.symptom,
            causes: [
              (
                categories.cableIssues as {
                  open: { symptom: string; meaning: string; solutions: string[] };
                }
              ).open.meaning,
            ],
            solutions: (
              categories.cableIssues as {
                open: { symptom: string; meaning: string; solutions: string[] };
              }
            ).open.solutions,
          },
          {
            symptom: (
              categories.cableIssues as {
                short: { symptom: string; meaning: string; solutions: string[] };
              }
            ).short.symptom,
            causes: [
              (
                categories.cableIssues as {
                  short: { symptom: string; meaning: string; solutions: string[] };
                }
              ).short.meaning,
            ],
            solutions: (
              categories.cableIssues as {
                short: { symptom: string; meaning: string; solutions: string[] };
              }
            ).short.solutions,
          },
        ]}
      />
      {/* DNS Issues */}
      <troubleshootingCategory
        title={categories.dnsIssues.title}
        issues={[
          {
            symptom: (
              categories.dnsIssues as {
                slowResolution: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowResolution.symptom,
            causes: (
              categories.dnsIssues as {
                slowResolution: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowResolution.causes,
            solutions: (
              categories.dnsIssues as {
                slowResolution: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowResolution.solutions,
          },
        ]}
      />
      {/* Gateway Issues */}
      <troubleshootingCategory
        title={categories.gatewayIssues.title}
        issues={[
          {
            symptom: (
              categories.gatewayIssues as {
                unreachable: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).unreachable.symptom,
            causes: (
              categories.gatewayIssues as {
                unreachable: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).unreachable.causes,
            solutions: (
              categories.gatewayIssues as {
                unreachable: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).unreachable.solutions,
          },
          {
            symptom: (
              categories.gatewayIssues as {
                highLatency: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).highLatency.symptom,
            causes: (
              categories.gatewayIssues as {
                highLatency: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).highLatency.causes,
            solutions: (
              categories.gatewayIssues as {
                highLatency: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).highLatency.solutions,
          },
        ]}
      />
      {/* Performance Issues */}
      <troubleshootingCategory
        title={categories.performanceIssues.title}
        issues={[
          {
            symptom: (
              categories.performanceIssues as {
                slowInternet: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowInternet.symptom,
            causes: (
              categories.performanceIssues as {
                slowInternet: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowInternet.causes,
            solutions: (
              categories.performanceIssues as {
                slowInternet: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowInternet.solutions,
          },
          {
            symptom: (
              categories.performanceIssues as {
                slowLan: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowLan.symptom,
            causes: (
              categories.performanceIssues as {
                slowLan: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowLan.causes,
            solutions: (
              categories.performanceIssues as {
                slowLan: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowLan.solutions,
          },
        ]}
      />
    </helpContentSection>
  );
}
