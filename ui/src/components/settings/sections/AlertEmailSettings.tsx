/**
 * AlertEmailSettings Component
 *
 * Purpose: name the mail relay alerts are emailed through (P-B1, #3209).
 * Before this the relay existed only in the settings API and the config file.
 *
 * It saves on an explicit action, like the webhook section: the server
 * refuses a relay that could never deliver with its reason, shown as given.
 * The password is write-only; an empty field on save keeps the stored one.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { type AlertEmailTLS, useAlertEmailSettings } from '../../../hooks/useAlertEmailSettings';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Select } from '../../../shared/Select';
import { Textarea } from '../../../shared/Textarea';
import { Tooltip } from '../../../shared/Tooltip';
import { button as buttonTokens, cn, icon as iconTokens, layout } from '../../../styles/theme';
import { Mail } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

export function AlertEmailSettings(): JSX.Element {
  const { email, setEmail, status, error, save } = useAlertEmailSettings();
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Mail className={iconTokens.size.sm} />
          <span>{t('sections.alertEmail')}</span>
          <AutoSaveIndicator status={status} />
        </div>
      }
      defaultOpen={false}
      data-testid="alert-email-section"
    >
      <div className="stack">
        <p className="body-small text-text-muted">{t('alertEmail.description')}</p>

        <fieldset disabled={!canWrite} className="stack">
          <FormField
            id="alert-email-host"
            label={t('alertEmail.host')}
            hint={t('alertEmail.hostHelp')}
          >
            {(control) => (
              <Input
                {...control}
                data-testid="alert-email-host"
                autoComplete="off"
                value={email.host}
                placeholder={t('alertEmail.hostPlaceholder')}
                onChange={(e): void => {
                  const host = e.target.value;
                  setEmail((current) => ({ ...current, host }));
                }}
              />
            )}
          </FormField>

          <div className="flex flex-wrap gap-default">
            <FormField id="alert-email-tls" label={t('alertEmail.tls')}>
              {(control) => (
                <Select
                  {...control}
                  data-testid="alert-email-tls"
                  value={email.tls}
                  onValueChange={(value): void => {
                    const tls: AlertEmailTLS = value === 'tls' ? 'tls' : 'starttls';
                    setEmail((current) => ({ ...current, tls }));
                  }}
                  options={[
                    { value: 'starttls', label: t('alertEmail.tlsStartTls') },
                    { value: 'tls', label: t('alertEmail.tlsImplicit') },
                  ]}
                />
              )}
            </FormField>
            <div className="w-28">
              <FormField id="alert-email-port" label={t('alertEmail.port')}>
                {(control) => (
                  <Input
                    {...control}
                    data-testid="alert-email-port"
                    type="number"
                    min={1}
                    max={65535}
                    step={1}
                    inputMode="numeric"
                    value={email.port === 0 ? '' : email.port}
                    placeholder={email.tls === 'tls' ? '465' : '587'}
                    onChange={(e): void => {
                      const port = Number(e.target.value);
                      setEmail((current) => ({ ...current, port }));
                    }}
                  />
                )}
              </FormField>
            </div>
          </div>

          <FormField
            id="alert-email-username"
            label={t('alertEmail.username')}
            hint={t('alertEmail.usernameHelp')}
          >
            {(control) => (
              <Input
                {...control}
                data-testid="alert-email-username"
                autoComplete="off"
                value={email.username}
                onChange={(e): void => {
                  const username = e.target.value;
                  setEmail((current) => ({ ...current, username }));
                }}
              />
            )}
          </FormField>

          <FormField
            id="alert-email-password"
            label={t('alertEmail.password')}
            hint={t('alertEmail.passwordHelp')}
          >
            {(control) => (
              <Input
                {...control}
                data-testid="alert-email-password"
                type="password"
                autoComplete="off"
                value={email.password}
                placeholder={email.passwordSet ? t('alertEmail.passwordStored') : ''}
                onChange={(e): void => {
                  const password = e.target.value;
                  setEmail((current) => ({ ...current, password }));
                }}
              />
            )}
          </FormField>

          <FormField id="alert-email-from" label={t('alertEmail.from')}>
            {(control) => (
              <Input
                {...control}
                data-testid="alert-email-from"
                autoComplete="off"
                value={email.from}
                placeholder={t('alertEmail.fromPlaceholder')}
                onChange={(e): void => {
                  const from = e.target.value;
                  setEmail((current) => ({ ...current, from }));
                }}
              />
            )}
          </FormField>

          <FormField id="alert-email-to" label={t('alertEmail.to')} hint={t('alertEmail.toHelp')}>
            {(control) => (
              <Textarea
                {...control}
                data-testid="alert-email-to"
                rows={2}
                value={email.to}
                placeholder={t('alertEmail.toPlaceholder')}
                onChange={(e): void => {
                  const to = e.target.value;
                  setEmail((current) => ({ ...current, to }));
                }}
              />
            )}
          </FormField>
        </fieldset>

        {error === '' ? null : (
          <p data-testid="alert-email-error" className="body-small text-status-error">
            {error}
          </p>
        )}

        <div className={layout.inline.default}>
          <Tooltip text={readOnlyReason}>
            <button
              type="button"
              data-testid="alert-email-save"
              disabled={!canWrite || status === 'saving'}
              onClick={(): void => {
                save().catch(() => undefined);
              }}
              className={cn(buttonTokens.base, buttonTokens.variant.primary, buttonTokens.size.sm)}
            >
              {t('alertEmail.save')}
            </button>
          </Tooltip>
        </div>
      </div>
    </CollapsibleSection>
  );
}
