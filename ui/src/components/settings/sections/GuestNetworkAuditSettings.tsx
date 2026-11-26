/**
 * GuestNetworkAuditSettings — the in-app editor for the guest-isolation audit's
 * target list (#1004).
 *
 * The audit shipped end to end in #397 except for this: the operator had to PUT
 * /api/v1/security/guest-audit/settings by hand, and GuestNetworkAuditCard only
 * renders once `enabled` is true and a target exists, so without an editor the
 * feature was unreachable from the UI.
 *
 * Every control is role-gated. The route is registered `minRole: op`, so a
 * viewer's save can only 403 — disabled with a reason, matching the API-tokens
 * and Wi-Fi sections (#1254).
 */

import type React from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../../contexts/RoleContext';
import { type GuestAuditTarget, useGuestNetworkAudit } from '../../../hooks/useGuestNetworkAudit';
import { Button } from '../../../shared/Button';
import { Checkbox } from '../../../shared/Checkbox';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Tooltip, type TooltipDescription } from '../../../shared/Tooltip';
import { cn, layout, radius, spacing } from '../../../styles/theme';
import { Info, ShieldAlert } from '../../ui/Icons';

/** The Info button that carries the read-only reason beside a disabled control. */
function ReadOnlyInfo({
  description,
  reason,
}: {
  description: TooltipDescription;
  reason: string | undefined;
}) {
  if (!reason) return null;
  return (
    <button
      type="button"
      {...description}
      aria-label={reason}
      onClick={(event) => event.preventDefault()}
      className="inline-flex text-text-muted rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
    >
      <Info className="h-4 w-4" aria-hidden="true" />
    </button>
  );
}

/**
 * The ports the backend probes when the settings carry no override.
 * `guestaudit.DefaultPorts()` is the source of truth; this mirrors it so the
 * field can be restored without a round trip.
 */
const DEFAULT_PORTS = [80, 443, 445, 3389, 22, 3306, 5432, 1433, 8080, 8443];

const MIN_PORT = 1;
const MAX_PORT = 65535;

/** parsePorts turns the comma-separated field into a port list. */
function parsePorts(value: string): { ports: number[]; invalid: string | null } {
  const parts = value
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean);

  const ports: number[] = [];
  for (const part of parts) {
    const port = Number(part);
    if (!Number.isInteger(port) || port < MIN_PORT || port > MAX_PORT) {
      return { ports: [], invalid: part };
    }
    ports.push(port);
  }

  return { ports, invalid: null };
}

/** isIPv4 accepts a dotted quad; the backend validates again before probing. */
function isIPv4(value: string): boolean {
  const octets = value.split('.');
  if (octets.length !== 4) {
    return false;
  }

  return octets.every((octet) => {
    const parsed = Number(octet);

    return /^\d{1,3}$/.test(octet) && Number.isInteger(parsed) && parsed >= 0 && parsed <= 255;
  });
}

export function GuestNetworkAuditSettings(): React.ReactElement {
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const { settings, setSettings, saveSettings, loading, error } = useGuestNetworkAudit();

  const [newIp, setNewIp] = useState('');
  const [newLabel, setNewLabel] = useState('');
  const [targetError, setTargetError] = useState<string | null>(null);
  const [portsField, setPortsField] = useState<string | null>(null);
  const [portsError, setPortsError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const readOnlyReason = canWrite ? undefined : t('guestAudit.readOnly');
  const ports = settings.ports ?? DEFAULT_PORTS;
  const portsValue = portsField ?? ports.join(', ');

  const addTarget = (): void => {
    const ip = newIp.trim();
    if (!isIPv4(ip)) {
      setTargetError(t('guestAudit.invalidIp'));

      return;
    }
    if (settings.targets.some((existing) => existing.ip === ip)) {
      setTargetError(t('guestAudit.duplicateIp'));

      return;
    }

    const target: GuestAuditTarget = newLabel.trim() ? { ip, label: newLabel.trim() } : { ip };
    setSettings({ ...settings, targets: [...settings.targets, target] });
    setNewIp('');
    setNewLabel('');
    setTargetError(null);
  };

  const removeTarget = (ip: string): void => {
    setSettings({
      ...settings,
      targets: settings.targets.filter((target) => target.ip !== ip),
    });
  };

  const commitPorts = (): void => {
    if (portsField === null) {
      return;
    }
    const { ports: parsed, invalid } = parsePorts(portsField);
    if (invalid !== null) {
      setPortsError(t('guestAudit.invalidPort', { value: invalid }));

      return;
    }
    setPortsError(null);
    setSettings({ ...settings, ports: parsed.length > 0 ? parsed : undefined });
  };

  const save = (): void => {
    setSaving(true);
    saveSettings()
      .catch(() => undefined)
      .finally(() => setSaving(false));
  };

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-compact">
          <ShieldAlert className="w-4 h-4" />
          <span>{t('guestAudit.title')}</span>
        </div>
      }
      defaultOpen={false}
      data-testid="guest-audit-settings-section"
    >
      <div className="stack-sm">
        <p className="caption text-text-muted">{t('guestAudit.description')}</p>

        <Tooltip text={readOnlyReason}>
          {(description) => (
            <span className="flex items-center gap-2">
              <Checkbox
                id="guest-audit-enabled"
                label={t('guestAudit.enabled')}
                aria-describedby={description['aria-describedby']}
                onClick={description.onClick}
                checked={settings.enabled}
                disabled={!canWrite || loading}
                onChange={(enabled): void => setSettings({ ...settings, enabled })}
              />
              {!canWrite || loading ? (
                <ReadOnlyInfo description={description} reason={readOnlyReason} />
              ) : null}
            </span>
          )}
        </Tooltip>

        {settings.targets.length > 0 ? (
          <div className="stack-sm">
            {settings.targets.map((target) => (
              <div
                key={target.ip}
                className={cn(
                  layout.flex.between,
                  spacing.pad.xs,
                  'bg-surface-base',
                  radius.default,
                  'border border-surface-border',
                )}
              >
                <div className="flex-1 min-w-0">
                  <div className="body-small text-text-primary truncate">
                    {target.label || target.ip}
                  </div>
                  <div className="caption text-text-muted">{target.ip}</div>
                </div>
                <Tooltip text={readOnlyReason}>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={(): void => removeTarget(target.ip)}
                    disabled={!canWrite}
                    // Every row's button would otherwise announce the same words,
                    // so the name carries the address (axe label-title-only, and
                    // the same reasoning as the subnet list).
                    aria-label={t('guestAudit.removeTargetNamed', { ip: target.ip })}
                  >
                    {t('guestAudit.removeTarget')}
                  </Button>
                </Tooltip>
              </div>
            ))}
          </div>
        ) : (
          <p className="caption text-text-muted">{t('guestAudit.noTargets')}</p>
        )}

        <div className="stack-sm">
          <FormField id="guest-audit-ip" label={t('guestAudit.targetIp')} error={targetError}>
            {(control) => (
              <Tooltip text={readOnlyReason}>
                {(description) => (
                  <span className="flex items-center gap-2">
                    <Input
                      {...control}
                      aria-describedby={
                        [control['aria-describedby'], description['aria-describedby']]
                          .filter(Boolean)
                          .join(' ') || undefined
                      }
                      onClick={description.onClick}
                      type="text"
                      disabled={!canWrite}
                      value={newIp}
                      onChange={(e): void => {
                        setNewIp(e.target.value);
                        setTargetError(null);
                      }}
                      placeholder={t('guestAudit.targetIpPlaceholder')}
                    />
                    <ReadOnlyInfo
                      description={description}
                      reason={!canWrite ? readOnlyReason : undefined}
                    />
                  </span>
                )}
              </Tooltip>
            )}
          </FormField>
          <FormField id="guest-audit-label" label={t('guestAudit.targetLabel')}>
            {(control) => (
              <Tooltip text={readOnlyReason}>
                {(description) => (
                  <span className="flex items-center gap-2">
                    <Input
                      {...control}
                      aria-describedby={
                        [control['aria-describedby'], description['aria-describedby']]
                          .filter(Boolean)
                          .join(' ') || undefined
                      }
                      onClick={description.onClick}
                      type="text"
                      disabled={!canWrite}
                      value={newLabel}
                      onChange={(e): void => setNewLabel(e.target.value)}
                      placeholder={t('guestAudit.targetLabelPlaceholder')}
                    />
                    <ReadOnlyInfo
                      description={description}
                      reason={!canWrite ? readOnlyReason : undefined}
                    />
                  </span>
                )}
              </Tooltip>
            )}
          </FormField>
          <Tooltip text={readOnlyReason}>
            <Button variant="secondary" onClick={addTarget} disabled={!canWrite} className="w-full">
              {t('guestAudit.addTarget')}
            </Button>
          </Tooltip>
        </div>

        <div className="stack-sm">
          <FormField id="guest-audit-ports" label={t('guestAudit.ports')} error={portsError}>
            {(control) => (
              <Tooltip text={readOnlyReason}>
                {(description) => (
                  <span className="flex items-center gap-2">
                    <Input
                      {...control}
                      aria-describedby={
                        [control['aria-describedby'], description['aria-describedby']]
                          .filter(Boolean)
                          .join(' ') || undefined
                      }
                      onClick={description.onClick}
                      type="text"
                      disabled={!canWrite}
                      value={portsValue}
                      onChange={(e): void => {
                        setPortsField(e.target.value);
                        setPortsError(null);
                      }}
                      onBlur={commitPorts}
                    />
                    <ReadOnlyInfo
                      description={description}
                      reason={!canWrite ? readOnlyReason : undefined}
                    />
                  </span>
                )}
              </Tooltip>
            )}
          </FormField>
          <Tooltip text={readOnlyReason}>
            <Button
              variant="ghost"
              size="sm"
              onClick={(): void => {
                setPortsField(DEFAULT_PORTS.join(', '));
                setPortsError(null);
                setSettings({ ...settings, ports: undefined });
              }}
              disabled={!canWrite}
              className="self-start"
            >
              {t('guestAudit.restoreDefaultPorts')}
            </Button>
          </Tooltip>
        </div>

        {error ? <p className="caption text-status-error">{error}</p> : null}

        <Tooltip text={readOnlyReason}>
          <Button
            variant="primary"
            onClick={save}
            disabled={!canWrite || loading}
            loading={saving}
            className="w-full"
          >
            {saving ? t('guestAudit.saving') : t('guestAudit.save')}
          </Button>
        </Tooltip>
      </div>
    </CollapsibleSection>
  );
}
