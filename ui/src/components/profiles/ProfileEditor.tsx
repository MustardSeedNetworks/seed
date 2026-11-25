/**
 * ProfileEditor Component - Modal for creating/editing profiles.
 * Migrated to react-hook-form + valibot per seed#1201.
 */

import { valibotResolver } from '@hookform/resolvers/valibot';
import type React from 'react';
import { type SubmitHandler, useForm, useWatch } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { ProfileEditorSchema } from '../../schemas/auth';
import { Button } from '../../shared/Button';
import { Modal } from '../../shared/Modal';
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
    control,
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

  const isDefault = useWatch({ control, name: 'isDefault' });

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
      return t('common.saving');
    }
    if (isEditing) {
      return t('common.save');
    }
    return t('common.create');
  };

  return (
    <Modal
      open
      onClose={onCancel}
      title={isEditing ? t('profile.edit') : t('profile.create')}
      closeLabel={t('common.close')}
      busy={isLoading}
      closeOnBackdrop={false}
      data-testid="profile-editor"
      footer={
        <>
          <Button onClick={onCancel} disabled={isLoading}>
            {t('common.cancel')}
          </Button>
          <Button
            type="submit"
            form="profile-editor-form"
            variant="primary"
            disabled={isLoading || !isValid}
          >
            {getButtonLabel()}
          </Button>
        </>
      }
    >
      <form id="profile-editor-form" onSubmit={handleSubmit(onSubmit)}>
        <div className="stack-lg">
          {/* Name */}
          <div>
            <label
              htmlFor="profile-name"
              className="block body-small font-medium text-text-primary mb-tight"
            >
              {t('profile.name')} *
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
              placeholder={t('profile.namePlaceholder')}
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
              {t('profile.description')}
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
              placeholder={t('profile.descriptionPlaceholder')}
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
              {t('profile.notes')}
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
              placeholder={t('profile.notesPlaceholder')}
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
            <span className="body-small text-text-primary">{t('profile.setAsDefault')}</span>
          </label>
        </div>
      </form>
    </Modal>
  );
}

export default ProfileEditor;
