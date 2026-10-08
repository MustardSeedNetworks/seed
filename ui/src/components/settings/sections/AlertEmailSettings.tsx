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

import type { JSX, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { type AlertEmailTLS, useAlertEmailSettings } from '../../../hooks/useAlertEmailSettings';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
} from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Mail } from '../../ui/Icons';
import { Tooltip } from '../../ui/Tooltip';
import { AutoSaveIndicator } from './AutoSaveIndicator';

interface FieldProps {
  id: string;
  label: string;
  help?: string;
  children: ReactNode;
}

function Field({ id, label, help, children }: FieldProps): JSX.Element {
  return (
    <label className="stack-xs" htmlFor={id}>
      <span className="body-small font-medium text-text-primary">{label}</span>
      {children}
      {help === undefined ? null : <span className="caption text-text-muted">{help}</span>}
    </label>
  );
}

export function AlertEmailSettings(): JSX.Element {
  const { email, setEmail, status, error, save } = useAlertEmailSettings();
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');
  const inputClass = cn(inputTokens.base, 'w-full');

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
          <Field id="alert-email-host" label={t('alertEmail.host')} help={t('alertEmail.hostHelp')}>
            <input
              id="alert-email-host"
              data-testid="alert-email-host"
              type="text"
              autoComplete="off"
              value={email.host}
              placeholder={t('alertEmail.hostPlaceholder')}
              onChange={(e): void => {
                const host = e.target.value;
                setEmail((current) => ({ ...current, host }));
              }}
              className={inputClass}
            />
          </Field>

          <div className="flex flex-wrap gap-default">
            <Field id="alert-email-tls" label={t('alertEmail.tls')}>
              <select
                id="alert-email-tls"
                data-testid="alert-email-tls"
                value={email.tls}
                onChange={(e): void => {
                  const tls = e.target.value as AlertEmailTLS;
                  setEmail((current) => ({ ...current, tls }));
                }}
                className={inputTokens.base}
              >
                <option value="starttls">{t('alertEmail.tlsStartTls')}</option>
                <option value="tls">{t('alertEmail.tlsImplicit')}</option>
              </select>
            </Field>
            <Field id="alert-email-port" label={t('alertEmail.port')}>
              <input
                id="alert-email-port"
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
                className={cn(inputTokens.base, 'w-24')}
              />
            </Field>
          </div>

          <Field
            id="alert-email-username"
            label={t('alertEmail.username')}
            help={t('alertEmail.usernameHelp')}
          >
            <input
              id="alert-email-username"
              data-testid="alert-email-username"
              type="text"
              autoComplete="off"
              value={email.username}
              onChange={(e): void => {
                const username = e.target.value;
                setEmail((current) => ({ ...current, username }));
              }}
              className={inputClass}
            />
          </Field>

          <Field
            id="alert-email-password"
            label={t('alertEmail.password')}
            help={t('alertEmail.passwordHelp')}
          >
            <input
              id="alert-email-password"
              data-testid="alert-email-password"
              type="password"
              autoComplete="off"
              value={email.password}
              placeholder={email.passwordSet ? t('alertEmail.passwordStored') : ''}
              onChange={(e): void => {
                const password = e.target.value;
                setEmail((current) => ({ ...current, password }));
              }}
              className={inputClass}
            />
          </Field>

          <Field id="alert-email-from" label={t('alertEmail.from')}>
            <input
              id="alert-email-from"
              data-testid="alert-email-from"
              type="text"
              autoComplete="off"
              value={email.from}
              placeholder={t('alertEmail.fromPlaceholder')}
              onChange={(e): void => {
                const from = e.target.value;
                setEmail((current) => ({ ...current, from }));
              }}
              className={inputClass}
            />
          </Field>

          <Field id="alert-email-to" label={t('alertEmail.to')} help={t('alertEmail.toHelp')}>
            <textarea
              id="alert-email-to"
              data-testid="alert-email-to"
              rows={2}
              value={email.to}
              placeholder={t('alertEmail.toPlaceholder')}
              onChange={(e): void => {
                const to = e.target.value;
                setEmail((current) => ({ ...current, to }));
              }}
              className={inputClass}
            />
          </Field>
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
