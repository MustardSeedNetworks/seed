/**
 * DeviceIdentitySettings Component
 *
 * Purpose: name this Seed and record where it is installed (#195), so an
 * operator with several open can tell which one a tab belongs to. The name is
 * shown in the header and the tab title.
 *
 * It saves on an explicit action rather than the drawer's auto-save debounce:
 * a debounced write of free text would rename the header mid-word.
 */

import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import {
  type DeviceIdentity,
  useDeviceIdentity,
  useSaveDeviceIdentity,
} from '../../../hooks/useDeviceIdentity';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
} from '../../../styles/theme';
import type { SaveStatus } from '../../../types/settings';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Server } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';

const NAME_MAX = 64;
const LOCATION_MAX = 128;

export function DeviceIdentitySettings() {
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const stored = useDeviceIdentity();
  const save = useSaveDeviceIdentity();
  const [draft, setDraft] = useState<DeviceIdentity>(stored);
  const [saved, setSaved] = useState(false);

  // The stored identity arrives after the first render and again after each
  // save (the server trims what it was sent); either way the inputs follow it.
  useEffect(() => {
    setDraft(stored);
  }, [stored]);

  useEffect(() => {
    if (!saved) {
      return;
    }
    const timer = setTimeout(() => setSaved(false), 2000);
    return (): void => clearTimeout(timer);
  }, [saved]);

  let status: SaveStatus = 'idle';
  if (save.isPending) {
    status = 'saving';
  } else if (save.isError) {
    status = 'error';
  } else if (saved) {
    status = 'saved';
  }

  const fields = [
    {
      key: 'name',
      max: NAME_MAX,
      label: t('identity.name'),
      help: t('identity.nameHelp'),
      placeholder: t('identity.namePlaceholder'),
    },
    {
      key: 'location',
      max: LOCATION_MAX,
      label: t('identity.location'),
      help: t('identity.locationHelp'),
      placeholder: t('identity.locationPlaceholder'),
    },
  ] as const;

  return (
    <CollapsibleSection
      data-testid="device-identity-settings-section"
      title={
        <div className={layout.inline.default}>
          <Server className={iconTokens.size.sm} />
          <span>{t('sections.identity')}</span>
          <AutoSaveIndicator status={status} />
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack">
        <p className="body-small text-text-muted">{t('identity.description')}</p>

        {fields.map((field) => (
          <label key={field.key} className="stack-xs" htmlFor={`device-identity-${field.key}`}>
            <span className="body-small font-medium text-text-primary">{field.label}</span>
            <input
              id={`device-identity-${field.key}`}
              data-testid={`device-identity-${field.key}`}
              type="text"
              maxLength={field.max}
              value={draft[field.key]}
              disabled={!canWrite}
              placeholder={field.placeholder}
              onChange={(e): void => {
                const { value } = e.target;
                setDraft((current) => ({ ...current, [field.key]: value }));
              }}
              className={cn(inputTokens.base, 'w-full')}
            />
            <span className="caption text-text-muted">{field.help}</span>
          </label>
        ))}

        {save.isError ? (
          <p data-testid="device-identity-error" className="body-small text-status-error">
            {save.error.message}
          </p>
        ) : null}

        {/* A disabled control takes no hover and no focus, so a tooltip on
              it would never open: the reason is plain text instead. */}
        {canWrite ? (
          <div className={layout.inline.default}>
            <button
              type="button"
              data-testid="device-identity-save"
              disabled={save.isPending}
              onClick={(): void => {
                save.mutate(draft, { onSuccess: () => setSaved(true) });
              }}
              className={cn(buttonTokens.base, buttonTokens.variant.primary, buttonTokens.size.sm)}
            >
              {t('identity.save')}
            </button>
          </div>
        ) : (
          <p data-testid="device-identity-read-only" className="caption text-text-muted">
            {t('common.readOnly')}
          </p>
        )}
      </div>
    </CollapsibleSection>
  );
}
