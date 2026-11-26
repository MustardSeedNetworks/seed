import { Tooltip } from '../../../shared/Tooltip';
import { Info } from '../../ui/Icons';
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

import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { useAlertWebhookSettings } from '../../../hooks/useAlertWebhookSettings';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { button as buttonTokens, cn, icon as iconTokens, layout } from '../../../styles/theme';
import { Bell } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

export function AlertDeliverySettings() {
  const { webhook, setWebhook, status, error, saveWebhook } = useAlertWebhookSettings();
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

        <FormField
          id="alert-webhook-url"
          label={t('alertDelivery.url')}
          hint={t('alertDelivery.urlHelp')}
        >
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
                    data-testid="alert-webhook-url"
                    type="url"
                    value={webhook.url}
                    disabled={!canWrite}
                    placeholder={t('alertDelivery.urlPlaceholder')}
                    onChange={(e): void => {
                      setWebhook((current) => ({ ...current, url: e.target.value }));
                    }}
                  />
                  {!canWrite && readOnlyReason ? (
                    <button
                      type="button"
                      {...description}
                      aria-label={readOnlyReason}
                      onClick={(event) => event.preventDefault()}
                      className="inline-flex text-text-muted rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
                    >
                      <Info className="h-4 w-4" aria-hidden="true" />
                    </button>
                  ) : null}
                </span>
              )}
            </Tooltip>
          )}
        </FormField>

        <FormField
          id="alert-webhook-secret"
          label={t('alertDelivery.secret')}
          hint={t('alertDelivery.secretHelp')}
        >
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
                    data-testid="alert-webhook-secret"
                    type="password"
                    autoComplete="off"
                    value={webhook.secret}
                    disabled={!canWrite}
                    placeholder={
                      webhook.secretSet
                        ? t('alertDelivery.secretStored')
                        : t('alertDelivery.secretPlaceholder')
                    }
                    onChange={(e): void => {
                      setWebhook((current) => ({ ...current, secret: e.target.value }));
                    }}
                  />
                  {!canWrite && readOnlyReason ? (
                    <button
                      type="button"
                      {...description}
                      aria-label={readOnlyReason}
                      onClick={(event) => event.preventDefault()}
                      className="inline-flex text-text-muted rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
                    >
                      <Info className="h-4 w-4" aria-hidden="true" />
                    </button>
                  ) : null}
                </span>
              )}
            </Tooltip>
          )}
        </FormField>

        {error === '' ? null : (
          <p data-testid="alert-webhook-error" className="body-small text-status-error">
            {error}
          </p>
        )}

        <div className={layout.inline.default}>
          <Tooltip text={readOnlyReason}>
            <button
              type="button"
              data-testid="alert-webhook-save"
              disabled={!canWrite || status === 'saving'}
              onClick={(): void => {
                saveWebhook().catch(() => undefined);
              }}
              className={cn(buttonTokens.base, buttonTokens.variant.primary, buttonTokens.size.sm)}
            >
              {t('alertDelivery.save')}
            </button>
          </Tooltip>
        </div>
      </div>
    </CollapsibleSection>
  );
}
