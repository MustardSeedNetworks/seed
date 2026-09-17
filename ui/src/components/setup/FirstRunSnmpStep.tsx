/**
 * FirstRunSnmpStep — the second step of first-run setup (#2722, #2674).
 *
 * Discovery reads its SNMP credentials from the encrypted vault and re-reads
 * it per scan (`internal/discovery/snmp_credentials.go`), so a community
 * saved here is used by the next sweep without a restart. Until one exists
 * the sweep finds hosts it cannot identify, which is what a fresh install
 * looks like to an operator.
 *
 * The field ships empty. Suggesting `public` would put a credential nobody
 * chose on the wire, which is the default-credential rule in another form.
 * Skipping is a first-class answer: setup must not become a gate in front of
 * the product because one optional secret is unavailable.
 */

import { KeyRound } from 'lucide-react';
import type React from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import { LogComponents, logger } from '../../lib/logger';
import {
  button,
  card,
  cn,
  input,
  layout,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';

interface FirstRunSnmpStepProps {
  /** Leaves setup — called once the operator has saved or skipped. */
  onDone: () => void;
}

// The vault entry's name. A slug rather than a sentence: it is stored
// server-side and read back by every operator afterwards, so it must not
// change with whoever's locale happened to run setup.
const CREDENTIAL_NAME = 'discovery-first-run';

export function FirstRunSnmpStep({ onDone }: FirstRunSnmpStepProps): React.JSX.Element {
  const { t } = useTranslation('setup');
  const [community, setCommunity] = useState('');
  const [saveError, setSaveError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async (): Promise<void> => {
    const trimmed = community.trim();
    if (trimmed === '') {
      onDone();
      return;
    }

    setSaveError(null);
    setIsSaving(true);
    try {
      await api.post('/api/v1/device-credentials', {
        name: CREDENTIAL_NAME,
        community: trimmed,
      });
      logger.info(LogComponents.SETUP, 'First-run SNMP credential saved');
      onDone();
    } catch (err) {
      // The secret never reaches the log line, only that the save failed.
      logger.error(LogComponents.SETUP, 'First-run SNMP credential save failed', err);
      setSaveError(t('snmp.errors.saveFailed'));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className={cn('min-h-screen bg-surface-base', layout.flex.center, 'pad')}>
      <div className="w-full max-w-md">
        <div className={cn('text-center', spacing.margin.bottom.sectionLg)}>
          <div className="w-16 h-16 mx-auto flex-center rounded-2xl bg-brand-primary text-on-brand">
            <KeyRound className="w-8 h-8" />
          </div>
          <h1 className={cn('heading-2', spacing.margin.top.heading)}>{t('snmp.title')}</h1>
          <p className={cn('body-small', spacing.margin.top.inline)}>{t('snmp.subtitle')}</p>
        </div>

        <div className={cn(card.base, card.variant.default, card.padding.lg)}>
          <div className={spacing.margin.bottom.content}>
            <label
              htmlFor="setup-snmp-community"
              className={cn(
                'block body-small font-medium text-text-primary',
                spacing.margin.bottom.inline,
              )}
            >
              {t('snmp.label')}
            </label>
            <input
              id="setup-snmp-community"
              type="password"
              autoComplete="off"
              value={community}
              onChange={(e): void => setCommunity(e.target.value)}
              className={cn(input.base, input.state.default, input.size.md)}
              placeholder={t('snmp.placeholder')}
            />
            <p className={cn('caption text-text-muted', spacing.margin.top.inline)}>
              {t('snmp.help')}
            </p>
          </div>

          {saveError ? (
            <div
              role="alert"
              aria-live="assertive"
              className={cn(
                spacing.margin.bottom.content,
                'pad-sm bg-status-error/10 border border-status-error/20',
                radius.md,
                statusColor.text.error,
                'body-small',
              )}
            >
              {saveError}
            </div>
          ) : null}

          <div className="stack-sm">
            <button
              type="button"
              disabled={isSaving}
              onClick={(): void => {
                void handleSave();
              }}
              className={cn(button.base, button.variant.primary, button.size.md, 'w-full')}
            >
              {isSaving ? t('snmp.buttons.saving') : t('snmp.buttons.save')}
            </button>
            <button
              type="button"
              onClick={onDone}
              className={cn(button.base, button.variant.secondary, button.size.md, 'w-full')}
            >
              {t('snmp.buttons.skip')}
            </button>
          </div>

          <p className={cn('caption text-text-muted text-center', spacing.margin.top.inline)}>
            {t('snmp.later')}
          </p>
        </div>
      </div>
    </div>
  );
}

export default FirstRunSnmpStep;
