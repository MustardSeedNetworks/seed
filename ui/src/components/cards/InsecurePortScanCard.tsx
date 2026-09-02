/**
 * InsecurePortScanCard — the on-demand insecure-port audit (#347).
 *
 * A technician points this at a host and gets back the legacy, cleartext and
 * remote-administration services it is exposing, each with the reason it
 * matters. Finding port 23 open is only useful if the tool also says that
 * Telnet carries credentials in the clear.
 *
 * The port list is the backend's: `profile: "insecure"` runs
 * `config.PortsInsecureTCP`, the same set the discovery preset uses, so this
 * card and a discovery sweep agree about what "insecure" means. The card owns
 * only the explanations.
 *
 * The route is `minRole: op` and rate-limited, so the control is role-gated —
 * a viewer's scan could only 403.
 */

import type { JSX } from 'react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../contexts/RoleContext';
import { type ScannedService, useInsecurePortScan } from '../../hooks/useInsecurePortScan';
import { cn, radius, spacing, status as statusColor } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/card';
import { AlertTriangle, CheckCircle, ShieldAlert } from '../ui/icons';

/**
 * Ports that carry their own explanation. A port the scanner reports without
 * an entry here still shows, named by whatever service the banner grab
 * identified — the list is a courtesy, not a filter, and a port with no copy
 * is not evidence that it is safe.
 */
const RISK_KEYS: Record<number, string> = {
  21: 'ftp',
  23: 'telnet',
  25: 'smtp',
  69: 'tftp',
  80: 'http',
  110: 'pop3',
  111: 'rpcbind',
  135: 'msrpc',
  139: 'netbios',
  143: 'imap',
  445: 'smb',
  512: 'rexec',
  513: 'rlogin',
  514: 'rsh',
  1099: 'rmi',
  2049: 'nfs',
  3389: 'rdp',
  5800: 'vncHttp',
  5900: 'vnc',
};

/**
 * Every explanation, resolved through a literal t() call.
 *
 * A template key -- t(`insecurePorts.risks.${name}`) -- reads better and is
 * invisible to scripts/i18n/check-keys.py, which cross-references locale keys
 * against the t() calls that use them. It reported all twenty as unreferenced.
 * Spelling them out is what keeps a renamed or deleted key a build failure.
 */
function riskTexts(t: (key: string) => string): Record<string, string> {
  return {
    ftp: t('insecurePorts.risks.ftp'),
    telnet: t('insecurePorts.risks.telnet'),
    smtp: t('insecurePorts.risks.smtp'),
    tftp: t('insecurePorts.risks.tftp'),
    http: t('insecurePorts.risks.http'),
    pop3: t('insecurePorts.risks.pop3'),
    rpcbind: t('insecurePorts.risks.rpcbind'),
    msrpc: t('insecurePorts.risks.msrpc'),
    netbios: t('insecurePorts.risks.netbios'),
    imap: t('insecurePorts.risks.imap'),
    smb: t('insecurePorts.risks.smb'),
    rexec: t('insecurePorts.risks.rexec'),
    rlogin: t('insecurePorts.risks.rlogin'),
    rsh: t('insecurePorts.risks.rsh'),
    rmi: t('insecurePorts.risks.rmi'),
    nfs: t('insecurePorts.risks.nfs'),
    rdp: t('insecurePorts.risks.rdp'),
    vncHttp: t('insecurePorts.risks.vncHttp'),
    vnc: t('insecurePorts.risks.vnc'),
    x11: t('insecurePorts.risks.x11'),
  };
}

/** X11 listens across 6000-6009; one explanation covers the range. */
const X11_FIRST = 6000;
const X11_LAST = 6009;

function riskKeyFor(port: number): string | null {
  if (port >= X11_FIRST && port <= X11_LAST) {
    return 'x11';
  }

  return RISK_KEYS[port] ?? null;
}

function isOpen(service: ScannedService): boolean {
  return service.state === 'open';
}

export function InsecurePortScanCard(): JSX.Element {
  const { t } = useTranslation('cards');
  const { canWrite } = useRole();
  const { result, scanning, error, scan } = useInsecurePortScan();
  const [target, setTarget] = useState('');

  const run = useCallback((): void => {
    const trimmed = target.trim();
    if (!trimmed) {
      return;
    }
    scan(trimmed).catch(() => undefined);
  }, [scan, target]);

  const risks = riskTexts(t as unknown as (key: string) => string);
  const openPorts = result?.services.filter(isOpen) ?? [];
  const scanned = result !== null;
  const readOnlyReason = canWrite ? undefined : t('insecurePorts.readOnly');

  let cardStatus: Status = 'unknown';
  if (scanned) {
    cardStatus = openPorts.length > 0 ? 'error' : 'success';
  }

  return (
    <Card
      title={t('insecurePorts.title')}
      icon={<ShieldAlert className="w-4 h-4" />}
      status={cardStatus}
      ariaLabel={t('insecurePorts.title')}
    >
      <div className="stack-sm">
        <p className="caption text-text-muted">{t('insecurePorts.description')}</p>

        <label className="caption text-text-muted" htmlFor="insecure-scan-target">
          {t('insecurePorts.target')}
        </label>
        <input
          id="insecure-scan-target"
          type="text"
          value={target}
          disabled={!canWrite || scanning}
          title={readOnlyReason}
          onChange={(e): void => setTarget(e.target.value)}
          placeholder={t('insecurePorts.targetPlaceholder')}
          className={cn(
            'w-full',
            spacing.chip.lg,
            'bg-surface-base border border-surface-border',
            radius.default,
            'body-small text-text-primary',
          )}
        />
        <Button
          onClick={run}
          disabled={!canWrite || target.trim() === ''}
          loading={scanning}
          title={readOnlyReason}
          className="w-full"
        >
          {scanning ? t('insecurePorts.scanning') : t('insecurePorts.scan')}
        </Button>

        {error ? <p className="caption text-status-error">{error}</p> : null}
        {result?.error ? <p className="caption text-status-error">{result.error}</p> : null}

        {scanned && openPorts.length === 0 && !result?.error ? (
          <p className={cn('body-small', statusColor.text.success)}>
            <CheckCircle className="w-4 h-4 inline mr-1" />
            {t('insecurePorts.none', { target: result?.ip ?? target })}
          </p>
        ) : null}

        {openPorts.length > 0 ? (
          <div className="stack-sm" data-testid="insecure-port-findings">
            <p className={cn('body-small font-medium', statusColor.text.error)}>
              <AlertTriangle className="w-4 h-4 inline mr-1" />
              {t('insecurePorts.found', { count: openPorts.length })}
            </p>
            {openPorts.map((service) => {
              const key = riskKeyFor(service.port);

              return (
                <div
                  key={`${service.port}-${service.protocol ?? 'tcp'}`}
                  className={cn(
                    spacing.pad.xs,
                    'bg-surface-base',
                    radius.default,
                    'border border-status-error/30',
                  )}
                >
                  <div className="body-small font-medium text-text-primary">
                    {t('insecurePorts.portLabel', {
                      port: service.port,
                      service: service.service || t('insecurePorts.unknownService'),
                    })}
                  </div>
                  <div className="caption text-text-secondary">
                    {(key === null ? undefined : risks[key]) ?? t('insecurePorts.risks.generic')}
                  </div>
                  {service.banner ? (
                    <div className="caption text-text-muted font-mono truncate">
                      {service.banner}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        ) : null}
      </div>
    </Card>
  );
}
