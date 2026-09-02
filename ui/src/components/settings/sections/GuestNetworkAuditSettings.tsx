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
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../../contexts/RoleContext';
import { type GuestAuditTarget, useGuestNetworkAudit } from '../../../hooks/useGuestNetworkAudit';
import { cn, layout, radius, spacing } from '../../../styles/theme';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { ShieldAlert } from '../../ui/icons';

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

  const addTarget = useCallback((): void => {
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
  }, [newIp, newLabel, settings, setSettings, t]);

  const removeTarget = useCallback(
    (ip: string): void => {
      setSettings({
        ...settings,
        targets: settings.targets.filter((target) => target.ip !== ip),
      });
    },
    [settings, setSettings],
  );

  const commitPorts = useCallback((): void => {
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
  }, [portsField, settings, setSettings, t]);

  const save = useCallback((): void => {
    setSaving(true);
    saveSettings()
      .catch(() => undefined)
      .finally(() => setSaving(false));
  }, [saveSettings]);

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

        <label className={cn(layout.flex.between, 'cursor-pointer')} htmlFor="guest-audit-enabled">
          <span className="body-small text-text-primary">{t('guestAudit.enabled')}</span>
          <input
            id="guest-audit-enabled"
            type="checkbox"
            checked={settings.enabled}
            disabled={!canWrite || loading}
            title={readOnlyReason}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              setSettings({ ...settings, enabled: e.target.checked })
            }
          />
        </label>

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
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={(): void => removeTarget(target.ip)}
                  disabled={!canWrite}
                  title={readOnlyReason}
                  // Every row's button would otherwise announce the same words,
                  // so the name carries the address (axe label-title-only, and
                  // the same reasoning as the subnet list).
                  aria-label={t('guestAudit.removeTargetNamed', { ip: target.ip })}
                >
                  {t('guestAudit.removeTarget')}
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <p className="caption text-text-muted">{t('guestAudit.noTargets')}</p>
        )}

        <div className="stack-sm">
          <label className="caption text-text-muted" htmlFor="guest-audit-ip">
            {t('guestAudit.targetIp')}
          </label>
          <input
            id="guest-audit-ip"
            type="text"
            value={newIp}
            disabled={!canWrite}
            title={readOnlyReason}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void => {
              setNewIp(e.target.value);
              setTargetError(null);
            }}
            placeholder={t('guestAudit.targetIpPlaceholder')}
            className={cn(
              'w-full',
              spacing.chip.lg,
              'bg-surface-base border border-surface-border',
              radius.default,
              'body-small text-text-primary',
            )}
          />
          <label className="caption text-text-muted" htmlFor="guest-audit-label">
            {t('guestAudit.targetLabel')}
          </label>
          <input
            id="guest-audit-label"
            type="text"
            value={newLabel}
            disabled={!canWrite}
            title={readOnlyReason}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void => setNewLabel(e.target.value)}
            placeholder={t('guestAudit.targetLabelPlaceholder')}
            className={cn(
              'w-full',
              spacing.chip.lg,
              'bg-surface-base border border-surface-border',
              radius.default,
              'body-small text-text-primary',
            )}
          />
          {targetError ? <p className="caption text-status-error">{targetError}</p> : null}
          <Button
            onClick={addTarget}
            disabled={!canWrite}
            title={readOnlyReason}
            className="w-full"
          >
            {t('guestAudit.addTarget')}
          </Button>
        </div>

        <div className="stack-sm">
          <label className="caption text-text-muted" htmlFor="guest-audit-ports">
            {t('guestAudit.ports')}
          </label>
          <input
            id="guest-audit-ports"
            type="text"
            value={portsValue}
            disabled={!canWrite}
            title={readOnlyReason}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void => {
              setPortsField(e.target.value);
              setPortsError(null);
            }}
            onBlur={commitPorts}
            className={cn(
              'w-full',
              spacing.chip.lg,
              'bg-surface-base border border-surface-border',
              radius.default,
              'body-small text-text-primary',
            )}
          />
          {portsError ? <p className="caption text-status-error">{portsError}</p> : null}
          <Button
            variant="ghost"
            size="sm"
            onClick={(): void => {
              setPortsField(DEFAULT_PORTS.join(', '));
              setPortsError(null);
              setSettings({ ...settings, ports: undefined });
            }}
            disabled={!canWrite}
            title={readOnlyReason}
            className="self-start"
          >
            {t('guestAudit.restoreDefaultPorts')}
          </Button>
        </div>

        {error ? <p className="caption text-status-error">{error}</p> : null}

        <Button
          onClick={save}
          disabled={!canWrite || loading}
          loading={saving}
          title={readOnlyReason}
          className="w-full"
        >
          {saving ? t('guestAudit.saving') : t('guestAudit.save')}
        </Button>
      </div>
    </CollapsibleSection>
  );
}
