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
import { Checkbox } from '../../shared/Checkbox';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Modal } from '../../shared/Modal';
import { Textarea } from '../../shared/Textarea';
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
          <FormField
            id="profile-name"
            label={t('profile.name')}
            required
            {...(errors.name ? { error: errors.name.message } : {})}
          >
            {(field) => (
              <Input {...field} {...register('name')} placeholder={t('profile.namePlaceholder')} />
            )}
          </FormField>

          <FormField
            id="profile-description"
            label={t('profile.description')}
            {...(errors.description ? { error: errors.description.message } : {})}
          >
            {(field) => (
              <Input
                {...field}
                {...register('description')}
                placeholder={t('profile.descriptionPlaceholder')}
              />
            )}
          </FormField>

          <FormField id="profile-notes" label={t('profile.notes')}>
            {(field) => (
              <Textarea
                {...field}
                {...register('notes')}
                rows={3}
                placeholder={t('profile.notesPlaceholder')}
              />
            )}
          </FormField>

          <Checkbox
            label={t('profile.setAsDefault')}
            checked={isDefault}
            onChange={(checked): void => setValue('isDefault', checked, { shouldDirty: true })}
          />
        </div>
      </form>
    </Modal>
  );
}

export default ProfileEditor;
