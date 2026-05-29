/**
 * ProfileEditor Component - Modal for creating/editing profiles.
 * Migrated to react-hook-form + valibot per seed#1201.
 */

import { valibotResolver } from '@hookform/resolvers/valibot';
import type React from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { ProfileEditorSchema } from '../../schemas/auth';
import { cn, radius, spacing } from '../../styles/theme';
import type { Profile, ProfileRequest } from '../../types/profile';

interface ProfileEditorProps {
  profile: Profile | null;
  onSave: (data: ProfileRequest) => Promise<void>;
  onCancel: () => void;
  isLoading: boolean;
}

interface ProfileFormFields {
  name: string;
  description: string;
  isDefault: boolean;
  notes: string;
}

/**
 * Modal dialog for creating or editing a client profile.
 */
export function ProfileEditor({
  profile,
  onSave,
  onCancel,
  isLoading,
}: ProfileEditorProps): React.JSX.Element {
  const { t } = useTranslation();
  const isEditing = profile !== null;

  const initialNotes = (profile?.config as { notes?: string })?.notes || '';

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isValid },
  } = useForm<ProfileFormFields>({
    resolver: valibotResolver(ProfileEditorSchema),
    defaultValues: {
      name: profile?.name || '',
      description: profile?.description || '',
      isDefault: Boolean(profile?.isDefault),
      notes: initialNotes,
    },
    mode: 'onBlur',
  });

  const isDefault = watch('isDefault');

  const onSubmit: SubmitHandler<ProfileFormFields> = async (values) => {
    await onSave({
      name: values.name,
      description: values.description,
      isDefault: values.isDefault,
      config: { notes: values.notes },
    });
  };

  const getButtonLabel = (): string => {
    if (isLoading) {
      return t('common.saving', 'Saving...');
    }
    if (isEditing) {
      return t('common.save', 'Save');
    }
    return t('common.create', 'Create');
  };

  return (
    <div className="fixed inset-0 z-50 flex-center pad">
      <div className="fixed inset-0 bg-scrim/50" onClick={onCancel} aria-hidden="true" />
      <div
        className={cn(
          'relative w-full max-w-lg',
          radius.lg,
          'bg-surface-raised shadow-xl overflow-hidden',
        )}
      >
        {/* Header */}
        <div className={cn(spacing.pad.default, 'border-b border-surface-border')}>
          <h2 className="heading-2 text-text-primary">
            {isEditing ? t('profile.edit', 'Edit Profile') : t('profile.create', 'Create Profile')}
          </h2>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit(onSubmit)}>
          <div className={cn(spacing.pad.default, 'stack-lg')}>
            {/* Name */}
            <div>
              <label
                htmlFor="profile-name"
                className="block body-small font-medium text-text-primary mb-tight"
              >
                {t('profile.name', 'Name')} *
              </label>
              <input
                id="profile-name"
                type="text"
                {...register('name')}
                className={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary',
                )}
                placeholder={t('profile.namePlaceholder', 'e.g., Client A')}
              />
              {errors.name ? (
                <p className="caption mt-tight text-status-error">{errors.name.message}</p>
              ) : null}
            </div>

            {/* Description */}
            <div>
              <label
                htmlFor="profile-description"
                className="block body-small font-medium text-text-primary mb-tight"
              >
                {t('profile.description', 'Description')}
              </label>
              <input
                id="profile-description"
                type="text"
                {...register('description')}
                className={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary',
                )}
                placeholder={t('profile.descriptionPlaceholder', 'Brief description')}
              />
              {errors.description ? (
                <p className="caption mt-tight text-status-error">{errors.description.message}</p>
              ) : null}
            </div>

            {/* Notes */}
            <div>
              <label
                htmlFor="profile-notes"
                className="block body-small font-medium text-text-primary mb-tight"
              >
                {t('profile.notes', 'Notes')}
              </label>
              <textarea
                id="profile-notes"
                {...register('notes')}
                rows={3}
                className={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary resize-none',
                )}
                placeholder={t('profile.notesPlaceholder', 'Contact info, VPN requirements, etc.')}
              />
            </div>

            {/* Default checkbox */}
            <label className="flex items-center gap-compact cursor-pointer">
              <input
                type="checkbox"
                checked={isDefault}
                onChange={(e) => setValue('isDefault', e.target.checked, { shouldDirty: true })}
                className="w-4 h-4 rounded border-surface-border text-brand-primary focus:ring-brand-primary"
              />
              <span className="body-small text-text-primary">
                {t('profile.setAsDefault', 'Set as default profile')}
              </span>
            </label>
          </div>

          {/* Footer */}
          <div
            className={cn(
              spacing.pad.default,
              'border-t border-surface-border flex justify-end gap-default',
            )}
          >
            <button
              type="button"
              onClick={onCancel}
              disabled={isLoading}
              className={cn(
                spacing.pad.sm,
                'px-4',
                radius.md,
                'border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary body-small font-medium disabled:opacity-50',
              )}
            >
              {t('common.cancel', 'Cancel')}
            </button>
            <button
              type="submit"
              disabled={isLoading || !isValid}
              className={cn(
                spacing.pad.sm,
                'px-4',
                radius.md,
                'bg-brand-primary hover:bg-brand-primary-hover text-on-brand body-small font-medium disabled:opacity-50',
              )}
            >
              {getButtonLabel()}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default ProfileEditor;
