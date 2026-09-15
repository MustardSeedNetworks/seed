/**
 * AlertDeliverySettings Component
 *
 * Purpose: name the receiver Seed POSTs alerts to (#2605). Until this existed
 * the only way to configure the webhook was to edit the daemon's environment
 * and restart it, so nothing in the product said the feature was there.
 *
 * Two things make this section different from its neighbours:
 *
 *   - It saves on an explicit action, not on the drawer's auto-save debounce.
 *     Half a signing key is not a signing key, and writing one mid-keystroke
 *     would re-point the live receiver at a secret the operator is still
 *     typing.
 *   - The secret is write-only. The server reports that one is stored, never
 *     the material, so the field is empty on every load and an empty field on
 *     save means "leave the stored secret alone".
 */

import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
} from '../../../styles/theme';
import type { AlertWebhookSettings, SaveStatus } from '../../../types/settings';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Bell } from '../../ui/icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

interface AlertDeliverySettingsProps {
  webhook: AlertWebhookSettings;
  setWebhook: React.Dispatch<React.SetStateAction<AlertWebhookSettings>>;
  status: SaveStatus;
  /** The server's own reason when a save was refused. */
  error: string;
  saveWebhook: () => Promise<void>;
}

export const AlertDeliverySettings: React.NamedExoticComponent<AlertDeliverySettingsProps> = memo(
  function alertDeliverySettings({
    webhook,
    setWebhook,
    status,
    error,
    saveWebhook,
  }: AlertDeliverySettingsProps) {
    const { t } = useTranslation('settings');
    const { canWrite } = useRole();
    const readOnlyReason = canWrite ? undefined : t('common.readOnly');

    return (
      <CollapsibleSection
        title={
          <div className={layout.inline.default}>
            <Bell className={iconTokens.size.sm} />
            <span>{t('sections.alertDelivery')}</span>
            <AutoSaveIndicator status={status} />
          </div>
        }
        defaultOpen={false}
      >
        <div className="stack">
          <p className="body-small text-text-muted">{t('alertDelivery.description')}</p>

          <label className="stack-xs" htmlFor="alert-webhook-url">
            <span className="body-small font-medium text-text-primary">
              {t('alertDelivery.url')}
            </span>
            <input
              id="alert-webhook-url"
              data-testid="alert-webhook-url"
              type="url"
              value={webhook.url}
              disabled={!canWrite}
              title={readOnlyReason}
              placeholder={t('alertDelivery.urlPlaceholder')}
              onChange={(e): void => {
                setWebhook((current) => ({ ...current, url: e.target.value }));
              }}
              className={cn(inputTokens.base, 'w-full')}
            />
            <span className="caption text-text-muted">{t('alertDelivery.urlHelp')}</span>
          </label>

          <label className="stack-xs" htmlFor="alert-webhook-secret">
            <span className="body-small font-medium text-text-primary">
              {t('alertDelivery.secret')}
            </span>
            <input
              id="alert-webhook-secret"
              data-testid="alert-webhook-secret"
              type="password"
              autoComplete="off"
              value={webhook.secret}
              disabled={!canWrite}
              title={readOnlyReason}
              placeholder={
                webhook.secretSet
                  ? t('alertDelivery.secretStored')
                  : t('alertDelivery.secretPlaceholder')
              }
              onChange={(e): void => {
                setWebhook((current) => ({ ...current, secret: e.target.value }));
              }}
              className={cn(inputTokens.base, 'w-full')}
            />
            <span className="caption text-text-muted">{t('alertDelivery.secretHelp')}</span>
          </label>

          {error === '' ? null : (
            <p data-testid="alert-webhook-error" className="body-small text-status-error">
              {error}
            </p>
          )}

          <div className={layout.inline.default}>
            <button
              type="button"
              data-testid="alert-webhook-save"
              disabled={!canWrite || status === 'saving'}
              title={readOnlyReason}
              onClick={(): void => {
                saveWebhook().catch(() => undefined);
              }}
              className={cn(buttonTokens.base, buttonTokens.variant.primary, buttonTokens.size.sm)}
            >
              {t('alertDelivery.save')}
            </button>
          </div>
        </div>
      </CollapsibleSection>
    );
  },
);
