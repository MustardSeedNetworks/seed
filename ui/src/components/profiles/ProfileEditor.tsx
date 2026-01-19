/**
 * ProfileEditor Component - Modal for creating/editing profiles
 */

import type React from 'react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, radius, spacing } from '../../styles/theme';
import type { Profile, ProfileRequest } from '../../types/profile';

interface ProfileEditorProps {
  profile: Profile | null;
  onSave: (data: ProfileRequest) => Promise<void>;
  onCancel: () => void;
  isLoading: boolean;
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

  const [name, setName] = useState(profile?.name || '');
  const [description, setDescription] = useState(profile?.description || '');
  const [isDefault, setIsDefault] = useState(profile?.is_default);
  const [notes, setNotes] = useState((profile?.config as { notes?: string })?.notes || '');

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      await onSave({
        name,
        description,
        // biome-ignore lint/style/useNamingConvention: API requires snake_case for this field
        is_default: isDefault,
        config: { notes },
      });
    },
    [name, description, isDefault, notes, onSave],
  );

  // Helper to get button label (avoids nested ternary)
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
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div class="fixed inset-0 bg-black/50" onClick={onCancel} aria-hidden="true" />
      <div
        class={cn(
          'relative w-full max-w-lg',
          radius.lg,
          'bg-surface-raised shadow-xl overflow-hidden',
        )}
      >
        {/* Header */}
        <div class={cn(spacing.pad.md, 'border-b border-surface-border')}>
          <h2 class="heading-2 text-text-primary">
            {isEditing ? t('profile.edit', 'Edit Profile') : t('profile.create', 'Create Profile')}
          </h2>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit}>
          <div class={cn(spacing.pad.md, 'space-y-4')}>
            {/* Name */}
            <div>
              <label for="profile-name" class="block body-small font-medium text-text-primary mb-1">
                {t('profile.name', 'Name')} *
              </label>
              <input
                id="profile-name"
                type="text"
                value={name}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setName(e.target.value)
                }
                required={true}
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary',
                )}
                placeholder={t('profile.namePlaceholder', 'e.g., Client A')}
              />
            </div>

            {/* Description */}
            <div>
              <label
                for="profile-description"
                class="block body-small font-medium text-text-primary mb-1"
              >
                {t('profile.description', 'Description')}
              </label>
              <input
                id="profile-description"
                type="text"
                value={description}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setDescription(e.target.value)
                }
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary',
                )}
                placeholder={t('profile.descriptionPlaceholder', 'Brief description')}
              />
            </div>

            {/* Notes */}
            <div>
              <label
                for="profile-notes"
                class="block body-small font-medium text-text-primary mb-1"
              >
                {t('profile.notes', 'Notes')}
              </label>
              <textarea
                id="profile-notes"
                value={notes}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setNotes(e.target.value)
                }
                rows={3}
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary resize-none',
                )}
                placeholder={t('profile.notesPlaceholder', 'Contact info, VPN requirements, etc.')}
              />
            </div>

            {/* Default checkbox */}
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={isDefault}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  setIsDefault(e.target.checked)
                }
                class="w-4 h-4 rounded border-surface-border text-brand-primary focus:ring-brand-primary"
              />
              <span class="body-small text-text-primary">
                {t('profile.setAsDefault', 'Set as default profile')}
              </span>
            </label>
          </div>

          {/* Footer */}
          <div class={cn(spacing.pad.md, 'border-t border-surface-border flex justify-end gap-3')}>
            <button
              type="button"
              onClick={onCancel}
              disabled={isLoading}
              class={cn(
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
              disabled={isLoading || !name.trim()}
              class={cn(
                spacing.pad.sm,
                'px-4',
                radius.md,
                'bg-brand-primary hover:bg-brand-primary-hover text-text-inverse body-small font-medium disabled:opacity-50',
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
